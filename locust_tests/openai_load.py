"""
This locust test script sends messages to the OpenAI endpoint
that the subscriber-app uses.
The level of load can be adjusted to vary the throttling that the 
subscriber-app experiences
"""


import json
import logging
import os

from dotenv import load_dotenv
from locust import HttpUser, constant, task, events

load_dotenv()

logging.getLogger("azure").setLevel(logging.WARNING)


EMBEDDING_DEPLOYMENT_NAME = os.getenv("EMBEDDING_DEPLOYMENT_NAME", "")
if not EMBEDDING_DEPLOYMENT_NAME:
    raise ValueError("EMBEDDING_DEPLOYMENT_NAME must be set")

OPENAI_API_ENDPOINT = os.getenv("OPENAI_API_ENDPOINT", "")
if not OPENAI_API_ENDPOINT:
    raise ValueError("OPENAI_API_ENDPOINT must be set")
if not OPENAI_API_ENDPOINT.endswith("/"):
    OPENAI_API_ENDPOINT += "/"

OPENAI_API_KEY = os.getenv("OPENAI_API_KEY", "")
if not OPENAI_API_KEY:
    raise ValueError("OPENAI_API_KEY must be set")


@events.init.add_listener
def on_locust_init(environment, **kwargs):
    environment.host = OPENAI_API_ENDPOINT


class EmbeddingUser(HttpUser):
    wait_time = constant(0.1)

    @task
    def send_embedding_request(self):
        url = f"openai/deployments/{
            EMBEDDING_DEPLOYMENT_NAME}/completions?api-version=2023-05-15"
        payload = {
            "model": "gpt-5-turbo-1",
            "prompt": "Once upon a time",
            "max_tokens": 10,
        }
        self.client.post(
            url,
            json=payload,
            headers={
                "api-key": OPENAI_API_KEY,
                "ocp-apim-subscription-key": OPENAI_API_KEY,
            },
        )
