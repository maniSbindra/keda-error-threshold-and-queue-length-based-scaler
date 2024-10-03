targetScope = 'resourceGroup'

@description('The supported Azure location (region) where the resources will be deployed')
param location string

@description('The base name for the deployment')
param baseName string

param logLevel string

param kedaExternalScalerImageTag string

// param currentUserPrincipalId string

// extract these to a common module to have a single, shared place for these across base/infra?
var containerRegistryName = replace('aoaiscaler-${baseName}', '-', '')
var containerAppEnvName = 'aoaiscaler-${baseName}'
var logAnalyticsName = 'aoaiscaler-${baseName}'
var appInsightsName = 'aoaiscaler-${baseName}'

var serviceBusNamespaceName = 'aoai-servicebus-${baseName}'
var serviceBusQueueName = 'aoai-queue-${baseName}'

var extScalerAppName = 'aoai-external-scaler'
var workloadAppName = 'aoai-workload-app'

var queueMessageCountPerReplicas = '10'
var rate429ErrorThreshold = '5'
var metricsBackend = 'azure'
var instanceComputeBackend = 'containerApps'
var timeBetweenScaleDownRequestsMinutes = '1'
var msgQueueLengthMetricName = 'Messages'
var rate429ErrorMetricName = 'rate_429_error'

var minReplicas = '1'
var maxReplicas = '7'



resource containerRegistry 'Microsoft.ContainerRegistry/registries@2021-12-01-preview' existing = {
  name: containerRegistryName
}


resource acrPullRoleDefinition 'Microsoft.Authorization/roleDefinitions@2018-01-01-preview' existing = {
  scope: subscription()
  name: '7f951dda-4ed3-4680-a7ca-43fe172d538d' // https://learn.microsoft.com/en-us/azure/role-based-access-control/built-in-roles#acrpull
}

resource monitoringReaderRoleDefintion 'Microsoft.Authorization/roleDefinitions@2018-01-01-preview' existing = {
  scope: subscription()
  name: '43d0d8ad-25c7-4714-9337-8ba259a9fe05' // https://learn.microsoft.com/en-us/azure/role-based-access-control/built-in-roles#acrpull
}

resource containerAppReaderRoleDefintion 'Microsoft.Authorization/roleDefinitions@2018-01-01-preview' existing = {
  scope: subscription()
  name: 'ad2dd5fb-cd4b-4fd4-a9b6-4fed3630980b' // https://learn.microsoft.com/en-us/azure/role-based-access-control/built-in-roles#acrpull
}



resource assignAcrPullToAca 'Microsoft.Authorization/roleAssignments@2020-04-01-preview' = {
  name: guid(resourceGroup().id, containerRegistry.name, managedIdentity.name, 'AssignAcrPullToAca')
  scope: containerRegistry
  properties: {
    description: 'Assign AcrPull role to ACA identity'
    principalId: managedIdentity.properties.principalId
    principalType: 'ServicePrincipal'
    roleDefinitionId: acrPullRoleDefinition.id
  }
}

resource logAnalytics 'Microsoft.OperationalInsights/workspaces@2021-12-01-preview' = {
  name: logAnalyticsName
  location: location
  properties: {
    sku: {
      name: 'PerGB2018'
    }
  }
}
resource appInsights 'Microsoft.Insights/components@2020-02-02' = {
  name: appInsightsName
  location: location
  kind: 'web'
  properties: {
    Application_Type: 'web'
    WorkspaceResourceId: logAnalytics.id
  }
}

resource managedIdentity 'Microsoft.ManagedIdentity/userAssignedIdentities@2023-01-31' = {
  name: '${containerAppEnvName}-identity'
  location: location
}



resource containerAppEnv 'Microsoft.App/managedEnvironments@2023-11-02-preview' = {
  name: containerAppEnvName
  location: location
  properties: {
    appLogsConfiguration: {
      destination: 'log-analytics'
      logAnalyticsConfiguration: {
        customerId: logAnalytics.properties.customerId
        sharedKey: logAnalytics.listKeys().primarySharedKey
      }
    }
  }
}

resource serviceBusNamespace 'Microsoft.ServiceBus/namespaces@2022-01-01-preview' = {
  name: serviceBusNamespaceName
  location: location
  sku: {
    name: 'Standard'
  }
  properties: {}
}

resource serviceBusQueue 'Microsoft.ServiceBus/namespaces/queues@2022-01-01-preview' = {
  parent: serviceBusNamespace
  name: serviceBusQueueName
  properties: {
    status: 'Active'
  }
}

resource assignMonitoringReader 'Microsoft.Authorization/roleAssignments@2020-04-01-preview' = {
  name: guid(resourceGroup().id, managedIdentity.name, 'AssignMonitoringReader')
  scope: resourceGroup()
  properties: {
    description: 'Monitoring Reader Role to ACA identity'
    principalId: managedIdentity.properties.principalId
    principalType: 'ServicePrincipal'
    roleDefinitionId: monitoringReaderRoleDefintion.id
  }
}

resource assignContainerAppsReader 'Microsoft.Authorization/roleAssignments@2020-04-01-preview' = {
  name: guid(resourceGroup().id, managedIdentity.name, 'AssignContainerAppsReader')
  scope: resourceGroup()
  properties: {
    description: 'Container Apps Reader Role to ACA identity'
    principalId: managedIdentity.properties.principalId
    principalType: 'ServicePrincipal'
    roleDefinitionId: containerAppReaderRoleDefintion.id
  }
}

resource apiExtScaler 'Microsoft.App/containerApps@2023-05-01' = {
  name: extScalerAppName
  location: location
  identity: {
    type: 'UserAssigned'
    userAssignedIdentities: {
      '${managedIdentity.id}': {} 
    }
  }
  properties: {
    managedEnvironmentId: containerAppEnv.id
    configuration: {
      activeRevisionsMode: 'single'
      // setting maxInactiveRevisions to 0 makes it easier when iterating and fixing issues by preventing 
      // old revisions showing in logs etc
      maxInactiveRevisions: 0
      ingress: {
        external: false
        allowInsecure: true
        targetPort: 6000
        clientCertificateMode: 'ignore'
        transport: 'http2'
      }
       registries: [
        {
          identity: managedIdentity.id
          server: containerRegistry.properties.loginServer
        }
      ]
    }
    template: {
      containers: [
        {
          name: 'aoai-external-scaler'
          image: '${containerRegistry.properties.loginServer}/${kedaExternalScalerImageTag}'
          resources: {
            cpu: json('0.25')
            memory: '0.5Gi'
          }
          env: [
            { name: 'QUEUE_MESSAGE_COUNT_PER_REPLICAS', value: queueMessageCountPerReplicas }
            { name: 'RATE_429_ERROR_THRESHOLD', value: rate429ErrorThreshold }
            { name: 'METRICS_BACKEND', value:  metricsBackend}
            { name: 'INSTANCE_COMPUTE_BACKEND', value:  instanceComputeBackend}
            { name: 'AZURE_CLIENT_ID', value:  managedIdentity.properties.clientId }
            { name: 'AZURE_TENANT_ID', value:  managedIdentity.properties.tenantId}
            { name: 'TIME_BETWEEN_SCALE_DOWN_REQUESTS_MINUTES', value: timeBetweenScaleDownRequestsMinutes }
            {name: 'MSG_QUEUE_LENGTH_METRIC_NAME', value: msgQueueLengthMetricName}
            {name: 'RATE_429_ERRORS_METRIC_NAME', value: rate429ErrorMetricName}
            { name: 'LOG_LEVEL', value: logLevel }
          ]
          
        }
      ]
      
      scale: {
        minReplicas: 1
        maxReplicas: 1
      }
    }
  }
}

resource apiWorkloadApp 'Microsoft.App/containerApps@2023-05-01' = {
  name: workloadAppName
  location: location
  identity: {
    type: 'UserAssigned'
    userAssignedIdentities: {
      '${managedIdentity.id}': {} 
    }
  }
  properties: {
    managedEnvironmentId: containerAppEnv.id
    configuration: {
      activeRevisionsMode: 'single'
      maxInactiveRevisions: 0
      ingress: {
        external: false
        targetPort: 80
      }
      registries: [
        {
          identity: managedIdentity.id
          server: containerRegistry.properties.loginServer
        }
      ]
    }
    template: {
      containers: [
        {
          name: 'workload-app'
          image: 'mcr.microsoft.com/k8se/quickstart:latest'
          resources: {
            cpu: json('0.25')
            memory: '0.5Gi'
          }

        }
      ]
      
      scale: {
        minReplicas: 1
        maxReplicas: 7
        rules: [
          {
            name: 'keda-external-scaler'
            custom: {
             type: 'external'
             metadata: {
              azureSubscriptionId: subscription().subscriptionId
              containerApp: workloadAppName
              logAnalyticsWorkspaceId: logAnalytics.properties.customerId
              minReplicas: minReplicas
              maxReplicas: maxReplicas
              resourceGroup: resourceGroup().name
              // scalerAddress: apiExtScaler.properties.configuration.ingress.fqdn
              scalerAddress: '${apiExtScaler.properties.latestRevisionFqdn}:80'
              serviceBusResourceId: serviceBusNamespace.id
              serviceBusQueueName: serviceBusQueue.name

             }
            }
          }
        ]
      }
    }
  }
}

output rgName string = resourceGroup().name
output containerRegistryLoginServer string = containerRegistry.properties.loginServer
output containerRegistryName string = containerRegistry.name

output acaName string = apiExtScaler.name
output acaEnvName string = containerAppEnv.name
output acaIdentityId string = managedIdentity.id
output apiExtScalerFqdn string = apiExtScaler.properties.configuration.ingress.fqdn

output logAnalyticsName string = logAnalytics.name

output kedaScalerIngress string = apiExtScaler.properties.latestRevisionFqdn
