#!/bin/bash

RG_NAME=YOUR_RESOURCE_GROUP_NAME
ACR_NAME=YOUR_ACR_NAME
CONTAINER_IMAGE_NAME=containerapp-and-k8s-keda-ext-scaler:v0.4 # this is the image name with tag
LOCATION=YOUR_LOCATION
BASE_NAME=test1


az group create --name ${RG_NAME} --location ${LOCATION}

cd external-scaler-containerapps-and-k8s/infra/bicep

az deployment group create \
        --resource-group ${RG_NAME} \
        --template-file ./base.bicep \
        --parameters location=${LOCATION} \
        --parameters baseName=${BASE_NAME}

az acr build --registry ${ACR_NAME} --image ${CONTAINER_IMAGE_NAME} --file ../../../external-scaler-containerapps-and-k8s/Dockerfile ../../../external-scaler-containerapps-and-k8s/ 

az deployment group create \
        --resource-group ${RG_NAME} \
        --template-file ./main.bicep \
        --parameters location=${LOCATION} \
        --parameters baseName=${BASE_NAME} \
        --parameters logLevel=debug \
        --parameters kedaExternalScalerImageTag=${CONTAINER_IMAGE_NAME}

