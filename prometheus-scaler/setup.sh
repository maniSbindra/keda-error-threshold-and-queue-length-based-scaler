#!/bin/bash
set -e

script_dir="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

# read RANDOM_ID from .random-id file
id_path="$script_dir/.random-id"
RANDOM_ID=""
if [ -f $id_path ]; then
    RANDOM_ID=$(cat $id_path)
fi
if [ -z "$RANDOM_ID" ]; then
    RANDOM_ID="$(openssl rand -hex 3)"
    echo $RANDOM_ID > $id_path
fi

RESOURCE_GROUP_NAME="keda-mul-scale${RANDOM_ID}-rg"
REGION="uksouth"
AKS_CLUSTER_NAME="keda-mult-scale-${RANDOM_ID}"
AKS_DNS_LABEL="akskedascale$RANDOM_ID"
ACR_NAME="kedamultscaleacr$RANDOM_ID"
VNET_NAME="keda-multi-scale-test-vnet"
VNET_ADDRESS_PREFIX="10.23.0.0/16"
AKS_SUBNET_ADDRESS_PREFIX="10.23.2.0/24"
AKS_SUBNET_NAME="keda-multi-scale-test-subnet"
ACR_SUBNET_NAME="acr-subnet"
ACR_SUBNET_ADDRESS_PREFIX="10.23.3.0/24"
K8s_UID_NAME="k8s-user-assign-identity"
KUBELET_UID_NAME="kubelet-user-assign-identity"

# for KEDA workload identity federation
SERVICE_ACCOUNT_NAMESPACE="default"
SERVICE_ACCOUNT_NAME="workload-identity-sa"
SUBSCRIPTION="$(az account show --query id --output tsv)"
KEDA_FEDERATION_UID_NAME="keda-federation-uid"
FEDERATED_IDENTITY_CREDENTIAL_NAME="keda-fed-id-credential-name"
FED_WORKLOAD="keda-fed-id-workload"
FED_KEDA_CRED_NAME="keda-fed-cred"

# Service bus
SB_NAME="kedascaletestsb$RANDOM_ID"
SB_HOSTNAME="${SB_NAME}.servicebus.windows.net"
SB_QUEUE_NAME=tasksqueue

# create a resource group
az group create --name $RESOURCE_GROUP_NAME --location $REGION

# create a vnet
az network vnet create --resource-group $RESOURCE_GROUP_NAME --name $VNET_NAME --address-prefixes $VNET_ADDRESS_PREFIX --subnet-name $AKS_SUBNET_NAME --subnet-prefix $AKS_SUBNET_ADDRESS_PREFIX


# create k8s user assigned managed identity
az identity create --name $K8s_UID_NAME --resource-group $RESOURCE_GROUP_NAME


# create kubelet user assigned managed identity
az identity create --name $KUBELET_UID_NAME --resource-group $RESOURCE_GROUP_NAME


# create acr
az acr create --name $ACR_NAME --resource-group $RESOURCE_GROUP_NAME --sku basic


# get the subnet id
SUBNET_ID=$(az network vnet subnet show --resource-group $RESOURCE_GROUP_NAME --vnet-name $VNET_NAME --name $AKS_SUBNET_NAME --query id --output tsv)

K8s_CLIENT_ID=$(az identity show --name $K8s_UID_NAME --resource-group $RESOURCE_GROUP_NAME --query id --output tsv)
KUBELET_CLIENT_ID=$(az identity show --name $KUBELET_UID_NAME --resource-group $RESOURCE_GROUP_NAME --query id --output tsv)


# VM_SIZE="Standard_DS3_v2"
VM_SIZE="Standard_D4s_v3"


# create AKS cluster
az aks create \
    --resource-group $RESOURCE_GROUP_NAME \
    --name $AKS_CLUSTER_NAME \
    --node-vm-size $VM_SIZE \
    --node-count 2 \
    --network-plugin azure \
    --vnet-subnet-id $SUBNET_ID \
    --assign-identity $K8s_CLIENT_ID \
    --assign-kubelet-identity $KUBELET_CLIENT_ID \
    --enable-addons monitoring \
    --enable-workload-identity \
    --enable-oidc-issuer \
    --attach-acr $ACR_NAME \
    --enable-keda \
    --location $REGION \
    --generate-ssh-keys

# create AKS cluster (no keda)
# az aks create \
#     --resource-group $RESOURCE_GROUP_NAME \
#     --name $AKS_CLUSTER_NAME \
#     --node-vm-size $VM_SIZE \
#     --node-count 2 \
#     --network-plugin azure \
#     --vnet-subnet-id $SUBNET_ID \
#     --assign-identity $K8s_CLIENT_ID \
#     --assign-kubelet-identity $KUBELET_CLIENT_ID \
#     --enable-addons monitoring \
#     --enable-workload-identity \
#     --enable-oidc-issuer \
#     --attach-acr $ACR_NAME \
#     --location $REGION \
#     --generate-ssh-keys

# get aks creds
az aks get-credentials --resource-group $RESOURCE_GROUP_NAME --name $AKS_CLUSTER_NAME --overwrite-existing


# deploy apps
# kubectl apply -f deployment.yaml

### Uncomment below lines if Service Bus with Managed Keda Authentication needed
# az aks show \
#     --name $AKS_CLUSTER_NAME \
#     --resource-group $RESOURCE_GROUP_NAME \
#     --query "[workloadAutoScalerProfile, securityProfile, oidcIssuerProfile]"


# # https://learn.microsoft.com/en-us/azure/aks/keda-workload-identity

# az servicebus namespace create \
#     --name $SB_NAME \
#     --resource-group $RESOURCE_GROUP_NAME \
#     --disable-local-auth



# az servicebus queue create \
#     --name $SB_QUEUE_NAME \
#     --namespace $SB_NAME \
#     --resource-group $RESOURCE_GROUP_NAME

# KEDA_FEDERATION_UID_CLIENT_ID=$(az identity create \
#     --name $KEDA_FEDERATION_UID_NAME \
#     --resource-group $RESOURCE_GROUP_NAME \
#     --query "clientId" \
#     --output tsv)

# AKS_OIDC_ISSUER=$(az aks show \
#     --name $AKS_CLUSTER_NAME \
#     --resource-group $RESOURCE_GROUP_NAME \
#     --query oidcIssuerProfile.issuerUrl \
#     --output tsv)


# # federated credential for workload
# az identity federated-credential create \
#     --name $FED_WORKLOAD \
#     --identity-name $KEDA_FEDERATION_UID_NAME \
#     --resource-group $RESOURCE_GROUP_NAME \
#     --issuer $AKS_OIDC_ISSUER \
#     --subject system:serviceaccount:default:$KEDA_FEDERATION_UID_NAME \
#     --audience api://AzureADTokenExchange

# # federated credential for keda operator
# az identity federated-credential create \
#     --name $FED_KEDA_CRED_NAME \
#     --identity-name $KEDA_FEDERATION_UID_NAME \
#     --resource-group $RESOURCE_GROUP_NAME \
#     --issuer $AKS_OIDC_ISSUER \
#     --subject system:serviceaccount:kube-system:keda-operator \
#     --audience api://AzureADTokenExchange


# KEDA_FEDERATION_UID_OBJECT_ID=$(az identity show --name $KEDA_FEDERATION_UID_NAME --resource-group $RESOURCE_GROUP_NAME --query "principalId" --output tsv)

# SB_ID=$(az servicebus namespace show --name $SB_NAME --resource-group $RESOURCE_GROUP_NAME --query "id" --output tsv)

# az role assignment create --role "Azure Service Bus Data Owner" --assignee-object-id $KEDA_FEDERATION_UID_OBJECT_ID --assignee-principal-type ServicePrincipal --scope $SB_ID

# kubectl rollout restart deploy keda-operator -n kube-system
# kubectl get pod -n kube-system -lapp=keda-operator -w
# KEDA_POD_ID=$(kubectl get po -n kube-system -l app.kubernetes.io/name=keda-operator -ojsonpath='{.items[0].metadata.name}')\nkubectl describe po $KEDA_POD_ID -n kube-system

# kubectl apply -f - <<EOF
# apiVersion: keda.sh/v1alpha1
# kind: TriggerAuthentication
# metadata:
#   name: azure-servicebus-auth
#   namespace: default  # this must be same namespace as the ScaledObject/ScaledJob that will use it
# spec:
#   podIdentity:
#     provider:  azure-workload
#     identityId: $KEDA_FEDERATION_UID_CLIENT_ID
# EOF


# kubectl apply -f - <<EOF
# apiVersion: v1
# kind: ServiceAccount
# metadata:
#   annotations:
#     azure.workload.identity/client-id: $KEDA_FEDERATION_UID_CLIENT_ID
#   name: $KEDA_FEDERATION_UID_NAME
# EOF



# acr test go app container image
# the go app is used to modify the queue_length and rate_429_errors metrics which are scrated by prometheus
az acr build --registry $ACR_NAME --image keda-test-app:v0.2 --file ./Dockerfile .


# REPLACE ACR_NAME with your acrname in the deployment.yaml
# then apply the deployments and the services
ACR_NAME=$ACR_NAME envsubst < deployment.yaml | kubectl apply -f -
kubectl apply -f service.yaml



# Install Prometheus. Note this uses the prometheus.yaml which has a scrape config for the metrics emitted by the golang app
# following is the scrape config
      # - job_name: 'keda-custom-scaling-metrics'
      #   scrape_interval: 10s
      #   static_configs:
      #   - targets:
      #     - 'metrics-modifier-service.default.svc.cluster.local:80'

# install Prometheus
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update

helm install -f prometheus.yaml prometheus prometheus-community/prometheus --namespace prometheus --create-namespace

# Create KEDA scaled object with custom scaler which checks both msg_queue_length and rate_429_errors
kubectl apply -f keda-scaled-object.yaml



