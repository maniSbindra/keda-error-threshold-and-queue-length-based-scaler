# KEDA multi metric scaler samples

These samples demonstrate how to use KEDA to scale a Kubernetes deployment based on multiple metrics. Specifically for not scaling up deployment when the error threshold is exceeded, else scaling based on a queue length metric. 

## Samples

This repository shows 3 samples, first one with the keda prometheus trigger, the second one with the keda external scaler on Kubernetes, and the third one with keda external scaler which can be customized for deployment either to Kubernetes or Azure Container apps. 

### Prometheus Trigger sample

[Prometheus Scaler](./prometheus-scaler/README.md): This uses the keda scaled object, prometheus trigger to achieve the desired behaviour

### External Scaler based Samples

We see 2 samples for the external scaler, one for Kubernetes and the other for Azure Container Apps. The external scaler is a custom implementation which reads metrics from a metrics backend, and scales the deployment based on the metrics.


#### Logical component view for external scaler based samples

![Logical View](./external-scaler-containerapps-and-k8s/images/logical-view.svg)

The workload app is an app which processes messages from a queue, while processing messages this app may integrate with multiple other apps, which have not been shown in this diagram. The Goal of the external scaler is to scale up the app as the number of messages on the queue increases, unless the error rate as the workload app processes messages of the queue increases beyond a certain threshold. Once the error rate breaches the configured threshold the workload app is scaled down one instance at a time.

#### Sample configuration with Kubernetes and prometheus

![Kubernetes and Prometheus](./external-scaler-containerapps-and-k8s/images/kubernetes-prometheus.svg)

In this configuration, the external scaler, and workload app are deployed to a Kubernetes cluster. Keda and Prometheus are deployed to the same kuberntes cluster as well. The external scaler reads the error rate, and queue length metrics from the Prometheus server, and it reads the deployment replica count via the Kubernetes API.

**Sample implementations**

* [External Scaler](./external-scaler/README.md) and workload on Kubernetes Sample
* [External Scaler](./external-scaler-containerapps-and-k8s/README.md) and workload on either Azure Container Apps or Kubernetes Sample


#### Sample configuration with Azure Container Apps, Azure Service Bus and Azure Log Analytics

![Azure Container Apps](./external-scaler-containerapps-and-k8s/images/container-apps-azure-metrics.svg)

In this configuration, the external scaler, and workload app are deployed to Azure Container Apps. The Native Keda Integration in container apps is used for theexternal scaler integration. The external scaler reads the error rate via an Azure log analytics query. The queue lenght is queries using the Azure API for Service bus metrics. The ARM API is used to read the Azure Container App replica count.

**Sample implementation**

* [External Scaler](./external-scaler-containerapps-and-k8s/README.md) and workload on either Azure Container Apps or Kubernetes Sample