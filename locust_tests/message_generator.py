import json
import logging
import os

from dotenv import load_dotenv
from locust import User, constant, task
from azure.servicebus import ServiceBusClient, ServiceBusMessage, ServiceBusMessageBatch

load_dotenv()

logging.getLogger("azure").setLevel(logging.WARNING)


SERVICE_BUS_CONNECTION_STRING = os.getenv("SERVICE_BUS_CONNECTION_STRING", "")
if not SERVICE_BUS_CONNECTION_STRING:
    raise ValueError("SERVICE_BUS_CONNECTION_STRING must be set")

TOPIC_NAME = os.getenv("TOPIC_NAME", "")
if not TOPIC_NAME:
    raise ValueError("TOPIC_NAME must be set")

MESSAGE_BATCH_SIZE = int(os.getenv("MESSAGE_BATCH_SIZE", "10"))


class MessageSendUser(User):
    wait_time = constant(1)

    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)
        self.servicebus_client = ServiceBusClient.from_connection_string(
            SERVICE_BUS_CONNECTION_STRING)
        

    @task
    def send_message(self):
        with self.servicebus_client:
            sender = self.servicebus_client.get_topic_sender(
            topic_name=TOPIC_NAME)
            with sender:
                messages = sender.create_message_batch()
                for _ in range(MESSAGE_BATCH_SIZE):
                    messages.add_message(ServiceBusMessage(
                        json.dumps({"text": "Sample message"})
                    ))
                sender.send_messages(messages)
