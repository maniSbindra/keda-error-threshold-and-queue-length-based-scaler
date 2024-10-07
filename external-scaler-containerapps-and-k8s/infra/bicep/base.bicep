@description('The base name for the deployment')
param baseName string

@description('The supported Azure location (region) where the resources will be deployed')
param location string


param containerRegistryName string

resource containerRegistry 'Microsoft.ContainerRegistry/registries@2021-12-01-preview' = {
  name: containerRegistryName
  location: location
  sku: {
    name: 'Basic'
  }

}

output rgName string = resourceGroup().name
output containerRegistryName string = containerRegistry.name
