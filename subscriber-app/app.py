import asyncio
import logging
import os

import jsons
from azure.identity import DefaultAzureCredential
from azure.monitor.opentelemetry import configure_azure_monitor
from azure.servicebus import ServiceBusReceivedMessage
from azure.servicebus.aio import ServiceBusClient
from openai import APIStatusError, AzureOpenAI

import config
import metrics
from service_bus import MessageProcessingOptions, MessageResult, apply_retry,process_subscription_messages

log_level = os.getenv("LOG_LEVEL") or "INFO"

logger = logging.getLogger(__name__)
logging.basicConfig(level=log_level)
logging.getLogger("azure").setLevel(logging.WARNING)


def get_service_bus_client() -> ServiceBusClient:
    if config.SERVICE_BUS_CONNECTION_STRING:
        logger.info("🔗 Connecting to service bus using connection string...")
        return ServiceBusClient.from_connection_string(
            conn_str=config.SERVICE_BUS_CONNECTION_STRING
        )

    logger.info("🔗 Connecting to service bus using Azure credential...")
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
    timeout=10,  # TODO set a reasonable timeout for production
)


async def message_processor(msg: ServiceBusReceivedMessage) -> MessageResult:
    message_id = msg.message_id
    delivery_count = msg.delivery_count
    logger.info("[%s, %s] Processing message...", message_id, delivery_count)
    body = jsons.loads(str(msg))
    text = body["text"]
    try:
        response = aoai_client.embeddings.create(
            input=text, model=config.OPENAI_EMBEDDING_DEPLOYMENT)
        logger.info(
            "[%s, %s] Got embedding: [%s, %s...]",
            message_id,
            delivery_count,
            response.data[0].embedding[0],
            response.data[0].embedding[1]
        )
        metrics.increment_open_ai_retry(200)  # emit success metric

        # PLACEHOLDER: This is where to add logic to save the embedding or pass back to the originator

        return MessageResult.SUCCESS

    except APIStatusError as e:
        logger.info("[%s, %s] API status error: %s",
                    message_id, delivery_count, e.status_code)
        metrics.increment_open_ai_retry(e.status_code)
        return MessageResult.RETRY


if config.APPLICATION_INSIGHTS_CONNECTION_STRING:
    logger.info("🚀 Configuring Azure Monitor telemetry")

    # Options: https://github.com/Azure/azure-sdk-for-python/tree/main/sdk/monitor/azure-monitor-opentelemetry#usage
    configure_azure_monitor(
        connection_string=config.APPLICATION_INSIGHTS_CONNECTION_STRING
    )
else:
    logger.info(
        "🚀 Azure Monitor telemetry not configured (set APPLICATIONINSIGHTS_CONNECTION_STRING)"
    )

options = MessageProcessingOptions()
options.max_messages_per_batch=config.MAX_MESSAGES_PER_BATCH
options.max_failures_per_batch=config.MAX_FAILURES_PER_BATCH
options.circuit_breaker_open_sleep_time=config.CIRCUIT_BREAKER_OPEN_SLEEP_TIME

service_bus_client = get_service_bus_client()
handler = apply_retry(message_processor, max_attempts=config.MAX_RETRIES_PER_MESSAGE + 1)
asyncio.run(
    process_subscription_messages(
        service_bus_client,
        config.SERVICE_BUS_TOPIC_NAME,
        config.SERVICE_BUS_SUBSCRIPTION_NAME,
        handler,
        options)
)
