import os

from dotenv import load_dotenv

load_dotenv()

SERVICE_BUS_CONNECTION_STRING = os.getenv("SERVICE_BUS_CONNECTION_STRING", "")
SERVICE_BUS_NAMESPACE = os.getenv("SERVICE_BUS_NAMESPACE", "")
SERVICE_BUS_TOPIC_NAME = os.getenv("SERVICE_BUS_TOPIC_NAME", "")
SERVICE_BUS_SUBSCRIPTION_NAME = os.getenv("SERVICE_BUS_SUBSCRIPTION_NAME", "")

if not SERVICE_BUS_NAMESPACE and not SERVICE_BUS_CONNECTION_STRING:
    raise ValueError(
        "One of SERVICE_BUS_NAMESPACE or SERVICE_BUS_CONNECTION_STRING must be set")
if not SERVICE_BUS_TOPIC_NAME:
    raise ValueError("SERVICE_BUS_TOPIC_NAME must be set")
if not SERVICE_BUS_SUBSCRIPTION_NAME:
    raise ValueError("SERVICE_BUS_SUBSCRIPTION_NAME must be set")

OPENAI_ENDPOINT = os.getenv("OPENAI_ENDPOINT", "")
OPENAI_API_KEY = os.getenv("OPENAI_API_KEY", "")
OPENAI_EMBEDDING_DEPLOYMENT = os.getenv("OPENAI_EMBEDDING_DEPLOYMENT", "")

if not OPENAI_ENDPOINT:
    raise ValueError("OPENAI_ENDPOINT must be set")
if not OPENAI_API_KEY:
    raise ValueError("OPENAI_API_KEY must be set")
if not OPENAI_EMBEDDING_DEPLOYMENT:
    raise ValueError("OPENAI_EMBEDDING_DEPLOYMENT must be set")


APPLICATION_INSIGHTS_CONNECTION_STRING = os.getenv(
    "APPLICATIONINSIGHTS_CONNECTION_STRING")

MAX_MESSAGES_PER_BATCH = int(os.getenv("MAX_MESSAGEs_PER_BATCH", "10"))
MAX_FAILURES_PER_BATCH = int(os.getenv("MAX_FAILURES_PER_BATCH", "2"))
CIRCUIT_BREAKER_OPEN_SLEEP_TIME = int(os.getenv("CIRCUIT_BREAKER_OPEN_SLEEP_TIME", "5"))
MAX_RETRIES_PER_MESSAGE = int(os.getenv("MAX_RETRIES_PER_MESSAGE", "3"))
