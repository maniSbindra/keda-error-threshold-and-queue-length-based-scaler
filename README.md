# KEDA multi metric scaler sample

This sample demonstrates how to use KEDA to scale a Kubernetes deployment based on multiple metrics. Specifically for not scaling up deployment when the error threshold is exceeded, else scaling based on a queue length metric.

## Installation

The [setup.sh](./setup.sh) script installs the required components for this sample. Details of the components have been added as comments in the script.

Key points to note:

* After the [Line](./setup.sh#L168) in the setup.sh script, which builds the test go application container image, and pushes it to ACR, you should update the ACR_NAME with your created ACR name in [deployment.yaml](./deployment.yaml) file, for both the deployments.
* The Prometheus [scrape config](./prometheus.yaml#L795-800) scrapes the test go apps for the msg_queue_length and rate_429_errors metrics.
  

## KEDA scaling tests

After the installation, you can follow the steps in [tests.sh](./tests.sh) to test the scaling behavior.
