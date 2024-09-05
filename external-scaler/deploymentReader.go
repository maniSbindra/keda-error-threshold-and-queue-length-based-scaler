package main

import (
	"context"
	"fmt"

	// "k8s.io/client-go/kubernetes"
	// "k8s.io/client-go/tools/clientcmd"
	// "k8s.io/client-go/util/homedir"
	// "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func getDeploymentInstanceCount(deploymentName, deploymentNamespace string) (int, error) {
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
	deployment, err := clientset.AppsV1().Deployments(deploymentNamespace).Get(context.Background(), deploymentName, metav1.GetOptions{})
	if err != nil {
		return 0, fmt.Errorf("failed to get deployment: %v", err)
	}

	// Return the number of replicas
	return int(deployment.Status.Replicas), nil
}
