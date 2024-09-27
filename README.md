# KEDA multi metric scaler samples

These samples demonstrate how to use KEDA to scale a Kubernetes deployment based on multiple metrics. Specifically for not scaling up deployment when the error threshold is exceeded, else scaling based on a queue length metric. 

## Samples

This repository shows 3 samples, first one with the keda prometheus trigger, the second one with the keda external scaler on Kubernetes, and the third one with keda external scaler which can be customized for deployment either to Kubernetes or Azure Container apps. The external scaler (2 and 3) is customized to reduce 1 instance at a time when scaling down.

* [Prometheus Scaler](./prometheus-scaler/README.md)
* [External Scaler](./external-scaler/README.md) and workload on Kubernetes
* [External Scaler](./external-scaler-containerapps-and-k8s/README.md) and workload on either Azure Container Apps or Kubernetes