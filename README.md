# KEDA multi metric scaler samples

These samples demonstrate how to use KEDA to scale a Kubernetes deployment based on multiple metrics. Specifically for not scaling up deployment when the error threshold is exceeded, else scaling based on a queue length metric. 

## Samples

This repository shows 2 samples, one with the keda prometheus trigger and the other with the keda external scaler. The main difference between the 2 is that the external scaler is customized to reduce 1 instance at a time when scaling down.

* [Prometheus Scaler](./prometheus-scaler/README.md)
* [External Scaler](./external-scaler/README.md)