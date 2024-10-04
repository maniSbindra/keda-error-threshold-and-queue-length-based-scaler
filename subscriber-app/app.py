import asyncio
import jsons

from azure.identity import DefaultAzureCredential
from azure.servicebus import ServiceBusReceivedMessage
from azure.servicebus.aio import ServiceBusClient
from openai import AzureOpenAI, RateLimitError

import config
from service_bus import process_messages, apply_retry, MessageResult


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
    timeout=10,  # TODO set a reasonable timeout for production
)


async def message_processor(msg: ServiceBusReceivedMessage) -> MessageResult:
    message_id = msg.message_id
    delivery_count = msg.delivery_count
    print(f"[{message_id}, {delivery_count}] Processing message...", flush=True)
    body = jsons.loads(str(msg))
    text = body["text"]
    try:
        response = aoai_client.embeddings.create(
            input=text, model=config.OPENAI_EMBEDDING_DEPLOYMENT)
        print(f"[{message_id}, {delivery_count}] Got embedding: [{response.data[0].embedding[0]}, {response.data[0].embedding[1]}...]", flush=True)

        # PLACEHOLDER: Add logic to save the embedding or pass back to the originator

        return MessageResult.SUCCESS

    except RateLimitError as e:
        print(f"[{message_id}, {delivery_count}] Rate limit error: {e}")
        return MessageResult.RETRY


service_bus_client = get_service_bus_client()
handler = apply_retry(message_processor)
asyncio.run(process_messages(service_bus_client, handler))
