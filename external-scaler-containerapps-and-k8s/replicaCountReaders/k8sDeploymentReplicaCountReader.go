package replicaCountReaders

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type K8sDeploymentReplicaCountReader struct {
	DeploymentName      string
	DeploymentNamespace string
}

func NewK8sDeploymentReplicaCountReader() *K8sDeploymentReplicaCountReader {
	return &K8sDeploymentReplicaCountReader{}
}

func (k *K8sDeploymentReplicaCountReader) GetInstanceCount() (int, error) {
	// Get the kubeconfig file path
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	if err != nil {
		return 0, fmt.Errorf("failed to create clientset: %v", err)
	}

	// Get the deployment
	deployment, err := clientset.AppsV1().Deployments(k.DeploymentNamespace).Get(context.Background(), k.DeploymentName, metav1.GetOptions{})
	if err != nil {
		return 0, fmt.Errorf("failed to get deployment: %v", err)
	}

	// Return the number of replicas
	return int(deployment.Status.Replicas), nil
}
