from opentelemetry import metrics

meter = metrics.get_meter(__name__)

# dimensions: status
_histogram_open_ai_embeddings_requests = meter.create_histogram(
    name="subscriber-app.openai.embeddings.requests",
    description="Number of OpenAI embeddings requests",
    unit="requests",
)


def increment_open_ai_retry(status: int):
    _histogram_open_ai_embeddings_requests.record(
        1,
        {"status": str(status)}
    )
