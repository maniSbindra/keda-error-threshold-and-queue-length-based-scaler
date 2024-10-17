from opentelemetry import metrics

meter = metrics.get_meter(__name__)

# dimensions: status
_histogram_open_ai_embeddings_requests = meter.create_histogram(
    name="subscriber-app.openai.embeddings.requests",
    description="Number of OpenAI embeddings requests",
    unit="requests",
)

_histogram_open_ai_embeddings_retries = meter.create_histogram(
    name="subscriber-app.openai.embeddings.retries",
    description="Number of OpenAI embeddings requests that got a 429 retry response",
    unit="requests",
)


def increment_open_ai_retry(status: int):
    _histogram_open_ai_embeddings_requests.record(
        1,
        {"status": str(status)}
    )
    if status == 429:
        # TODO - consider updating the external scaler to enable using the status code on the
        # requests metric rather than adding a new metric
        _histogram_open_ai_embeddings_retries.record(1)
