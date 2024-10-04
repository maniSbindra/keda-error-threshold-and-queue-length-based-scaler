# subscriber-app

The subscriber app consumes messages from a Service Bus Topic and simulates processing the messages.
For each message, the app makes a call to an OpenAI endpoint to get embeddings for the message text.

## Config

| Environment Variable            | Description                                                                                                |
| ------------------------------- | ---------------------------------------------------------------------------------------------------------- |
| `SERVICE_BUS_NAMESPACE`         | Namespace of the Service Bus Topic - used with managed/workload identity (if connection string is not set) |
| `SERVICE_BUS_CONNECTION_STRING` | Connection string to the Service Bus Topic                                                                 |
| `SERVICE_BUS_TOPIC_NAME`        | Name of the Service Bus Topic to consume messages from                                                     |
| `SERVICE_BUS_SUBSCRIPTION_NAME` | Name of the Service Bus Subscription to consume messages via                                               |
| `OPENAI_ENDPOINT`               | OpenAI endpoint to call for embeddings                                                                     |
| `OPENAI_API_KEY`                | OpenAI API key                                                                                             |
| `OPENAI_EMBEDDING_DEPLOYMENT`   | Name of the OpenAI deployment to use for embeddings                                                        |

## Running the app

To run the app locally, copy the `sample.env` file to `.env` and fill in the required environment variables.
Then run `python app.py` from the `subscriber-app` directory.

