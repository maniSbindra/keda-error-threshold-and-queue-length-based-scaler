# KEDA multi metric scaler - using external scaler trigger

This sample demonstrates how to use KEDA to scale either a Kubernetes / container apps deployment, based on multiple metrics. Specifically for not scaling up deployment when the error threshold is exceeded, else scaling based on a queue length metric. This keda external scaler implementation scales down 1 instance at a time during scale down.

## External scaler logical view

![Logical View](./images/logical-view.svg)

The workload app is an app which processes messages from a queue, while processing messages this app may integrate with multiple other apps, which have not been shown in this diagram. The Goal of the external scaler is to scale up the app as the number of messages on the queue increases, unless the error rate as the workload app processes messages of the queue increases beyond a certain threshold. Once the error rate breaches the configured threshold the workload app is scaled down one instance at a time.

### Sample configuration with Kubernetes and prometheus

![Kubernetes and Prometheus](./images/kubernetes-prometheus.svg)

In this configuration, the external scaler, and workload app are deployed to a Kubernetes cluster. Keda and Prometheus are deployed to the same kuberntes cluster as well. The external scaler reads the error rate, and queue length metrics from the Prometheus server, and it reads the deployment replica count via the Kubernetes API.

### Sample configuration with Azure Container Apps, Azure Service Bus and Azure Log Analytics

![Azure Container Apps](./images/container-apps-azure-metrics.svg)

In this configuration, the external scaler, and workload app are deployed to Azure Container Apps. The Native Keda Integration in container apps is used for theexternal scaler integration. The external scaler reads the error rate via an Azure log analytics query. The queue lenght is queries using the Azure API for Service bus metrics. The ARM API is used to read the Azure Container App replica count.

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

