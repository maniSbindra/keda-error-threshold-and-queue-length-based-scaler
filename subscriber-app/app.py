import asyncio
from typing import Awaitable, Callable

from azure.identity import DefaultAzureCredential
from azure.servicebus import ServiceBusReceivedMessage
from azure.servicebus.aio import ServiceBusClient, AutoLockRenewer, ServiceBusReceiver
import jsons
from openai import AzureOpenAI, RateLimitError

import config

#
#  Helper methods
#


async def process_messages(service_bus_client: ServiceBusClient, handler):
    async with service_bus_client:
        print(f"👟 Creating service bus receiver... (topic={config.SERVICE_BUS_TOPIC_NAME}, " +
              f"subscription={config.SERVICE_BUS_SUBSCRIPTION_NAME})", flush=True)
        receiver = service_bus_client.get_subscription_receiver(
            topic_name=config.SERVICE_BUS_TOPIC_NAME,
            subscription_name=config.SERVICE_BUS_SUBSCRIPTION_NAME,
        )
        async with receiver:
            # AutoLockRenewer performs message lock renewal (for long message processing)
            # TODO - do we want to provide a callback for renewal failure? What action would we take?
            # TODO - make max_lock_renewal_duration configurable
            renewer = AutoLockRenewer(max_lock_renewal_duration=5 * 60)

            print("👟 Starting message receiver...", flush=True)
            while True:
                # TODO: Add back-off logic when no messages?
                # TODO: Add max message count etc to config
                received_msgs = await receiver.receive_messages(max_message_count=10, max_wait_time=30)

                message_count = len(received_msgs)
                if message_count > 0:
                    print(f"⚡Got messages: count {message_count}", flush=True)

                    # Set up message renewal for the batch
                    for msg in received_msgs:
                        renewer.register(receiver, msg)

                    # process messages in parallel
                    await asyncio.gather(*[__wrapped_handler(receiver, handler, msg) for msg in received_msgs])


# enum for message result status
class MessageResult:
    SUCCESS = "success"
    RETRY = "retry"
    DROP = "drop"


async def __wrapped_handler(receiver: ServiceBusReceiver, handler, msg: ServiceBusReceivedMessage):
    # TODO - add logic to retry the message delivery with back-off before abandoning

    result = None
    try:
        result = await handler(msg)
    except Exception as e:
        print(f"Error processing message: {e}")
        result = MessageResult.RETRY

    if result == MessageResult.SUCCESS or result is None:  # default to success if no exception
        await receiver.complete_message(msg)
    elif result == MessageResult.RETRY:
        await receiver.abandon_message(msg) # TODO: allow setting a reason when retrying/dead-lettering?
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
        wait_time = 1
        while True:
            try:
                print(f"[{message_id}] Attempt {retry_count + 1}...", flush=True)
                response = await handler(msg)
                if response is not None and response != MessageResult.RETRY:
                    print(f"[{message_id}] Returning response: {response}", flush=True)
                    return response
            except Exception as e:
                print(f"Error processing message: {e}")
            retry_count += 1
            if retry_count >= max_attempts:
                print(f"[{message_id}] Max attempts reached, retry message delivery...", flush=True)
                return MessageResult.RETRY
            
            print(f"[{message_id}] Retrying in {wait_time} seconds...", flush=True)
            await asyncio.sleep(wait_time)
            wait_time *= 2

    return wrapper


#
# Message processing logic
#


def get_service_bus_client() -> ServiceBusClient:
    if config.SERVICE_BUS_CONNECTION_STRING:
        print("🔗 Connecting to service bus using connection string...", flush=True)
        return ServiceBusClient.from_connection_string(conn_str=config.SERVICE_BUS_CONNECTION_STRING)

    print("🔗 Connecting to service bus using Azure credential...", flush=True)
    credential = DefaultAzureCredential()
    servicebus_client = ServiceBusClient(
        fully_qualified_namespace=config.SERVICE_BUS_NAMESPACE, credential=credential
    )
    return servicebus_client


aoai_client = AzureOpenAI(
    api_key=config.OPENAI_API_KEY,
    api_version="2023-12-01-preview",
    azure_endpoint=config.OPENAI_ENDPOINT,
    max_retries=0,  # disable automatic retries as we want to be aware and handle them
    timeout=10, # TODO set a reasonable timeout for production
)


async def message_processor(msg: ServiceBusReceivedMessage) -> MessageResult:
    message_id = msg.message_id
    print(f"[{message_id}] Processing message...", flush=True)
    body = jsons.loads(str(msg))
    text = body["text"]
    try:
        response = aoai_client.embeddings.create(input=text, model=config.OPENAI_EMBEDDING_DEPLOYMENT)
        print(f"[{message_id}] Got embedding: [{response.data[0].embedding[0]}, {response.data[0].embedding[1]}...]", flush=True)

        # PLACEHOLDER: Add logic to save the embedding or pass back to the originator

        
        return MessageResult.SUCCESS
    
    except RateLimitError as e:
        print(f"[{message_id}] Rate limit error: {e}")
        return MessageResult.RETRY


service_bus_client = get_service_bus_client()
handler = apply_retry(message_processor)
asyncio.run(process_messages(service_bus_client, handler))
