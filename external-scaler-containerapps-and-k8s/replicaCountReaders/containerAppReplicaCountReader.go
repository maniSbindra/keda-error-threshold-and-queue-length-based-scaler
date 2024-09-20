package replicaCountReaders

// Hit the Azure Api to find the number of current replicas of the container app
// func getContainerAppReplicaCount(resourceGroup string, containerAppEnv string, containerApp string) (int, error) {
// 	// Get the Azure Api token
// 	token, err := getAzureApiToken()
// 	if err != nil {
// 		return 0, err
// 	}

// 	// Get the number of replicas
// 	replicas, err := getContainerAppReplicaCountFromAzure(token, resourceGroup, containerAppEnv, containerApp)
// 	if err != nil {
// 		return 0, err
// 	}

// 	return replicas, nil
// }

import (
	"context"
	"fmt"

	// "github.com/Azure/azure-sdk-for-go/profiles/latest/authorization/mgmt/authorization"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v3"
	// "github.com/Azure/go-autorest/autorest"
	// "github.com/Azure/go-autorest/autorest/azure/auth"
)

type ContainerAppReplicaCountReader struct {
	SubscriptionID string
	ResourceGroup  string
	ContainerApp   string
}

func NewContainerAppReplicaCountReader(subscriptionId, resourceGroup, containerApp string) *ContainerAppReplicaCountReader {
	return &ContainerAppReplicaCountReader{
		SubscriptionID: subscriptionId,
		ResourceGroup:  resourceGroup,
		ContainerApp:   containerApp,
	}
}

func (c *ContainerAppReplicaCountReader) GetInstanceCount() (int, error) {

	// Get the Azure Api token
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		// return wrapped error
		return 0, fmt.Errorf("failed to get Azure credential: %w", err)
	}

	ctx := context.Background()

	clientFactory, err := armappcontainers.NewClientFactory(c.SubscriptionID, cred, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create app containers client factory: %w", err)
	}

	contApp, err := clientFactory.NewContainerAppsClient().Get(ctx, c.ResourceGroup, c.ContainerApp, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to get container app: %w", err)
	}

	latestRevisionName := contApp.ContainerApp.Properties.LatestReadyRevisionName

	revision, err := clientFactory.NewContainerAppsRevisionsClient().GetRevision(ctx, c.ResourceGroup, c.ContainerApp, *latestRevisionName, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to get revision: %w", err)
	}

	return int(*revision.Properties.Replicas), nil

	// Get the number of replicas

	// contAppsClient := clientFactory.NewContainerAppsClient()
	// clientFactory.NewContainerAppsDiagnosticsClient().NewListDetectorsPager()
	// NewContainerAppsRevisionsClient().NewListRevisionsPager(ctx, containerApp, *armappcontainers.ContainerAppsRevisionsClientListRevisionsOptions{
	// 	Filter: ,
	// })
	// contAppsClient := clientFactory.NewContainerAppsRevisionReplicasClient()
	// contAppsClient.GetReplica(ctx, resourceGroup,containerApp)
	// contAppsClient.
	// resourcesClientFactory, err := armresources.NewClientFactory(subscriptionID, cred, nil)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// resourcesClientFactory.

}
