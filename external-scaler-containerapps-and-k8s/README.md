# KEDA multi metric scaler - using external scaler trigger

This sample demonstrates how to use KEDA to scale either a Kubernetes / container apps deployment, based on multiple metrics. Specifically for not scaling up deployment when the error threshold is exceeded, else scaling based on a queue length metric. This keda external scaler implementation scales down 1 instance at a time during scale down.

## TODO installation and other details

## keda external scaler configuration

**Environment variables:**
* QUEUE_MESSAGE_COUNT_PER_REPLICAS: This corresponds to the target size property of external scaler. The scaler will try to have 1 replica per QUEUE_MESSAGE_COUNT_PER_REPLICAS messages in the queue.
* RATE_429_ERROR_THRESHOLD: If the error rate exceeds this threshold, the scaler will not scale up the deployment. 
* METRICS_BACKEND: The metrics backend to use. Supported values are prometheus and azure
* INSTANCE_COMPUTE_BACKEND: The instance compute backend to use. Supported values are kubernetes and containerApps
* AZURE_CLIENT_ID: of the managed identity associated with the container apps. This needs to have permissions to read the metrics from the Log Analytics workspace, replica details of the container apps, and service bus queue length.
* AZURE_TENANT_ID: of the managed identity associated with the container apps
* TIME_BETWEEN_SCALE_DOWN_REQUESTS_MINUTES: The time between scale down requests in minutes. Request is sent to keda to scale down only after this time period. This is request is made keda takes about 5 minutes to scale down the replica


## workload app configuration

**Scaler Metadata:**

* containerApp: Name of container app
* logAnalyticsWorkspaceId: Log Analytics workspace ID for the error metric
* azureSubscriptionId: Azure subscription ID for container app
* resourceGroup: Resource group for container app
* minReplicas: Minimum number of replicas
* maxReplicas: Maximum number of replicas
* scalerAddress: Address of the external scaler (such as keda-ext-scaler--uuuuuu.uksouth.azurecontainerapps.io:80) 
* serviceBusResourceId: Azure resource ID of the service bus
* serviceBusQueueName: Name of the service bus queue

