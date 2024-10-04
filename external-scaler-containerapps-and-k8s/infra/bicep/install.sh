#!/bin/bash
set -e

SCALER_IMAGE_NAME=containerapp-and-k8s-keda-ext-scaler:v0.4 # this is the image name with tag
WORKLOAD_IMAGE_NAME=subscriber-app:v0.1

script_dir="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

if [[ -f "$script_dir/.env" ]]; then
  echo "Loading .env"
  source "$script_dir/.env"
fi

if [[ -z "$RG_NAME" ]]; then
  echo "RG_NAME is not set"
  exit 1
fi
if [[ -z "$LOCATION" ]]; then
  echo "LOCATION is not set"
  exit 1
fi
if [[ -z "$BASE_NAME" ]]; then
  echo "BASE_NAME is not set"
  exit 1
fi

ACR_NAME="aoaiscaler${BASE_NAME}"
ACR_NAME=${ACR_NAME//[^a-zA-Z0-9]/} # Remove non-alphanumeric characters

script_dir="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

######################################################################
# Create resource group and perform base deployment
######################################################################


az group create --name ${RG_NAME} --location ${LOCATION}

cd "$script_dir"

az deployment group create \
        --resource-group ${RG_NAME} \
        --template-file ./base.bicep \
        --parameters location=${LOCATION} \
        --parameters baseName=${BASE_NAME} \
        --parameters containerRegistryName=${ACR_NAME}

######################################################################
# Build container images
######################################################################

az acr build --registry ${ACR_NAME} --image ${SCALER_IMAGE_NAME} --file ../../../external-scaler-containerapps-and-k8s/Dockerfile ../../../external-scaler-containerapps-and-k8s/ 

az acr build --registry ${ACR_NAME} --image ${WORKLOAD_IMAGE_NAME} --file ../../../subscriber-app/Dockerfile ../../../subscriber-app/

######################################################################
# Clone and build the simulator image
######################################################################


# Clone simulator
simulator_path="$script_dir/../simulator"
simulator_git_tag=${SIMULATOR_GIT_TAG:=v0.5}

if [[ -n "$SIMULATOR_IMAGE_TAG" ]]; then
  simulator_image_tag=$SIMULATOR_IMAGE_TAG
else
  simulator_image_tag=$simulator_git_tag
fi
simulator_image_tag=${simulator_image_tag//\//_} # Replace slashes with underscores
echo "Using simulator git tag: $simulator_git_tag"
echo "Using simulator image tag: $simulator_image_tag"

clone_simulator=true
if [[ -d "$simulator_path" ]]; then
  if [[ -f "$script_dir/.simulator_tag" ]]; then
    previous_tag=$(cat "$script_dir/.simulator_tag")
    if [[ "$previous_tag" == "$simulator_git_tag" ]]; then
      clone_simulator=false
      echo "Simulator folder already exists - skipping clone."
    else
      rm -rf "$simulator_path"
      echo "Cloned simulator has tag ${previous_tag} - re-cloning ${simulator_git_tag}."
    fi
  else
      rm -rf "$simulator_path"
      echo "Cloned simulator exists without tag file - re-cloning ${simulator_git_tag}."
  fi
else
  echo "Simulator folder does not exist - cloning."
fi
if [[ "$clone_simulator" == "true" ]]; then
  echo "Cloning simulator (tag: ${simulator_git_tag})..."
  git clone \
    --depth 1 \
    --branch "$simulator_git_tag" \
    --config advice.detachedHead=false \
    https://github.com/microsoft/aoai-api-simulator \
    "$simulator_path"
  echo "$simulator_git_tag" > "$script_dir/.simulator_tag"
fi

# create a tik_token_cache folder to avoid failure in the build
mkdir -p "$simulator_path/src/aoai-api-simulator/tiktoken_cache"
az acr build --registry ${ACR_NAME} --image "aoai-api-simulator:${simulator_image_tag}" --file "$simulator_path/src/aoai-api-simulator/Dockerfile" "$simulator_path/src/aoai-api-simulator"

######################################################################
# Deploy the main template
######################################################################

# Generate API key for the simulator (save and re-use)
output_generated_keys="$script_dir/.generated-keys.json"

SIMULATOR_API_KEY=""
if [[ -f "$output_generated_keys" ]]; then
  SIMULATOR_API_KEY=$(jq -r '.simulatorApiKey // ""' < "$output_generated_keys")
else 
  echo "{}" > "$output_generated_keys"
fi
if [[ ${#SIMULATOR_API_KEY} -eq 0 ]]; then
  echo 'Generating new SIMULATOR_API_KEY'
  SIMULATOR_API_KEY=$(bash "$script_dir/generate-api-key.sh")
else
  echo "Loaded SIMULATOR_API_KEY from generated-keys.json"
fi
jq ".simulatorApiKey = \"${SIMULATOR_API_KEY}\"" < "$output_generated_keys" > "/tmp/generated-keys.json"
cp "/tmp/generated-keys.json" "$output_generated_keys"


az deployment group create \
        --resource-group ${RG_NAME} \
        --template-file "$script_dir/main.bicep" \
        --parameters location=${LOCATION} \
        --parameters baseName=${BASE_NAME} \
        --parameters scalerLogLevel=debug \
        --parameters kedaExternalScalerImageTag=${SCALER_IMAGE_NAME} \
        --parameters workloadImageTag=${WORKLOAD_IMAGE_NAME} \
        --parameters simulatorImageTag="aoai-api-simulator:${simulator_image_tag}" \
        --parameters simulatorApiKey="${SIMULATOR_API_KEY}" \
        --parameters simulatorLogLevel=INFO 
