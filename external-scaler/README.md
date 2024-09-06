# KEDA multi metric scaler - using external scaler trigger

This sample demonstrates how to use KEDA to scale a Kubernetes deployment based on multiple metrics. Specifically for not scaling up deployment when the error threshold is exceeded, else scaling based on a queue length metric. This keda external scaler implementation scales down 1 instance at a time during scale down.

## Customization

This keda custom scaler customizations has been written in go.

## Installation

### Base component installation
The inital installation steps from the prometheus scaler can be reused, other than the last line (Which creates keda scaled object with prometheus trigger). Refer the [setup.sh](../prometheus-scaler/setup.sh), this script installs the base required components for this sample. Details of the components have been added as comments in the script. 

Key points to note:

* After the [Line](../prometheus-scaler/setup.sh#L168) in the setup.sh script, which builds the test go application container image, and pushes it to ACR, you should update the ACR_NAME with your created ACR name in [deployment.yaml](../prometheus-scaler//deployment.yaml) file, for both the deployments. This deployment will be used in the tests to modify the metrics in prometheus
* if you want to install the external scaler, do not execute the final line of the [setup.sh](../prometheus-scaler/setup.sh#L194) file (Which creates keda scaled object with prometheus trigger).
* The Prometheus [scrape config](../prometheus-scaler//prometheus.yaml#L795-800) scrapes the test go apps for the msg_queue_length and rate_429_errors metrics.
  
### Additional steps for the extermal scaler

The additional setup steps for external scaler are in the file [additional-setup-for-external-scaler.sh](./additional-setup-for-external-scaler.sh). The $ACR_NAME in the [ext-scaler-deployment-service.yaml](./ext-scaler-deployment-service.yaml) and [to-be-scaled-workload-deployment.yaml](./to-be-scaled-workload-deployment.yaml) files need to be modified with you ACR name.

### Keda scaled object - External scaler configuration

Details of how the keda scaled object integrates with the external scaler please look at [keda-scaled-object-with-ext-scaler.yaml](./keda-scaled-object-with-ext-scaler.yaml#L16)

## KEDA scaling tests

After the installation, you can follow the steps in [tests.sh](./tests.sh) to test the scaling behavior.
