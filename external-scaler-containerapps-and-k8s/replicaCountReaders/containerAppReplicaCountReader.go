package replicaCountReaders

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v3"
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

}
