
az acr build --registry $ACR_NAME --image keda-queue-length-and-error-rate-externalscaler:v0.9 --file ./Dockerfile .

# Create service account, role and role binding for the external scaler
kubectl apply -f ext-scaler-sa-role-role-binding.yaml

# REPLACE ACR_NAME with your acrname in the ext-scaler-deployment-service.yaml and to-be-scaled-workload-deployment.yaml
# then apply the deployments and the services
kubectl apply -f ext-scaler-deployment-service.yaml
kubectl apply -f to-be-scaled-workload-deployment.yaml

# Create the keda scaled object with external trigger
kubectl apply -f keda-scaled-object-with-ext-scaler.yaml





