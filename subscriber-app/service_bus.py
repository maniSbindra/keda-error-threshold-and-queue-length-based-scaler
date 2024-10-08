import asyncio
import logging
from typing import Awaitable, Callable

from azure.servicebus import ServiceBusReceivedMessage
from azure.servicebus.aio import AutoLockRenewer, ServiceBusClient, ServiceBusReceiver

logger = logging.getLogger(__name__)


# enum for message result status
class MessageResult:
    """
    Enum for message processing result status:
    - SUCCESS: message was processed successfully and should be marked as completed
    - RETRY: message processing failed and should be retried
    - DROP: message processing failed and should be dropped (dead-lettered)
    """
    SUCCESS = "success"
    RETRY = "retry"
    DROP = "drop"


class MessageProcessingOptions:
    """
    Options for message processing
    """
    max_message_count: int = 10
    max_wait_time: int = 30
    max_lock_renewal_duration: int = 5 * 60


async def process_subscription_messages(
    service_bus_client: ServiceBusClient,
    topic_name: str,
    subscription_name: str,
    handler: Callable[[ServiceBusReceivedMessage], Awaitable[MessageResult]],
    options: MessageProcessingOptions = None):
    """
    Start a message processing look to receive and process messages from a service bus topic subscription
    """

    options = options or MessageProcessingOptions()
    async with service_bus_client:
        logger.info(
            "👟 Creating service bus receiver... (topic=%s, subscription=%s)",
            topic_name,
            subscription_name
        )
        receiver = service_bus_client.get_subscription_receiver(
            topic_name=topic_name,
            subscription_name=subscription_name,
        )
        async with receiver:
            # AutoLockRenewer performs message lock renewal (for long message processing)
            # TODO - do we want to provide a callback for renewal failure? What action would we take?
            renewer = AutoLockRenewer(max_lock_renewal_duration=options.max_lock_renewal_duration)

            logger.info("👟 Starting message receiver...")
            while True:
                # TODO: Add back-off logic when no messages?
                received_msgs = await receiver.receive_messages(
                    max_message_count=options.max_message_count,
                    max_wait_time=options.max_wait_time
                )

                message_count = len(received_msgs)
                if message_count > 0:
                    logger.info("⚡Got messages: count=%s", message_count)

                    # Set up message renewal for the batch
                    for msg in received_msgs:
                        renewer.register(receiver, msg)

                    # process messages in parallel
                    await asyncio.gather(*[__wrapped_handler(receiver, handler, msg) for msg in received_msgs])


async def __wrapped_handler(receiver: ServiceBusReceiver, handler, msg: ServiceBusReceivedMessage):
    # TODO - add logic to retry the message delivery with back-off before abandoning

    result = None
    try:
        result = await handler(msg)
    except Exception as e:
        logger.error("Error processing message: %s", e)
        result = MessageResult.RETRY

    if result == MessageResult.SUCCESS or result is None:  # default to success if no exception
        await receiver.complete_message(msg)
    elif result == MessageResult.RETRY:
        # TODO: allow setting a reason when retrying/dead-lettering?
        await receiver.abandon_message(msg)
    elif result == MessageResult.DROP:
        await receiver.dead_letter_message(msg)
    else:
        raise ValueError(f"Invalid message result: {result}")


def apply_retry(handler: Callable[[ServiceBusReceivedMessage], Awaitable[MessageResult]], max_attempts: int = 5) -> Callable[[ServiceBusReceivedMessage], Awaitable[MessageResult]]:
    """
    Decorator to wrap a message handler with retry logic
    """
    async def wrapper(msg: ServiceBusReceivedMessage) -> MessageResult:
        retry_count = 0
        message_id = msg.message_id
        delivery_count = msg.delivery_count
        wait_time = 1
        while True:
            try:
                logger.info("[%s, %s] Attempt %s...", message_id,
                            delivery_count, retry_count+1)
                response = await handler(msg)
                if response is not None and response != MessageResult.RETRY:
                    logger.info("[%s, %s] Returning response: %s",
                                message_id, delivery_count, response)
                    return response
            except Exception as e:
                logger.error("Error processing message: %s", e)
            retry_count += 1
            if retry_count >= max_attempts:
                logger.info(
                    "[%s, %s] Max attempts reached, retry message delivery...", message_id, delivery_count)
                return MessageResult.RETRY

            logger.info("[%s, %s] Retrying in %s seconds...",
                        message_id, delivery_count, wait_time)
            await asyncio.sleep(wait_time)
            wait_time *= 2

    return wrapper
