package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net"
	"strconv"

	pb "github.com/manisbindra/kedaQueueLengthAndErrorRateExternalScaler/externalscaler"
	"github.com/manisbindra/kedaQueueLengthAndErrorRateExternalScaler/metricsReaders"
	"github.com/manisbindra/kedaQueueLengthAndErrorRateExternalScaler/replicaCountReaders"
	"google.golang.org/grpc"

	// "log"
	// "net"

	"os"
	"time"
	// "google.golang.org/grpc/codes"
	// "google.golang.org/grpc/status"
)

type ReplicaCountReader interface {
	GetInstanceCount() (int, error)
}

type MetricsReader interface {
	GetQueueLength() (int, error)
	GetRate429Errors() (int, error)
}

const (
	METRICS_BACKEND_PROMETHEUS              = "prometheus"
	METRICS_BACKEND_AZURE                   = "azure"
	INSTANCE_COMPUTE_BACKEND_KUBERNETES     = "kubernetes"
	INSTANCE_COMPUTE_BACKEND_CONTAINER_APPS = "containerApps"
)

type ExternalScaler struct {
	pb.UnimplementedExternalScalerServer

	// state variables
	lastScaleDownRequestTime               time.Time
	replicaCountDuringLastScaleDownRequest int

	// common settings
	QUEUE_MESSAGE_COUNT_PER_REPLICA          int
	RATE_429_ERROR_THRESHOLD                 int
	TIME_BETWEEN_SCALE_DOWN_REQUESTS_MINUTES int
	MSG_QUEUE_LENGTH_METRIC_NAME             string
	RATE_429_ERRORS_METRIC_NAME              string

	// common settings passed via metadata
	MIN_REPLICAS int
	MAX_REPLICAS int

	METRICS_BACKEND          string
	INSTANCE_COMPUTE_BACKEND string

	// Prometheus Metrics Reader settings
	PROMETHEUS_ENDPOINT string

	// Kubernetes Replica reader settins set via metadata
	DEPLOYMENT_NAME      string
	DEPLOYMENT_NAMESPACE string

	// Azure Container App settings set via metadata
	AZURE_SUBSCRIPTION_ID string
	RESOURCE_GROUP        string
	CONTAINER_APP         string

	// Azure Service Bus settings set via metadata
	SERVICE_BUS_RESOURCE_ID string
	SERVICE_BUS_QUEUE_NAME  string

	// Azure setting to get rate_429_errors metrics
	LOG_ANALYTICS_WORKSPACE_ID string

	MetricsReader      MetricsReader
	ReplicaCountReader ReplicaCountReader
}

func getEnvInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		log.Printf("Failed to convert %s to int: %v\n", key, err)
		return defaultValue
	}
	return value
}

func getEnvString(key string, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func (e *ExternalScaler) IsActive(ctx context.Context, scaledObject *pb.ScaledObjectRef) (*pb.IsActiveResponse, error) {

	slog.Info("Is Active called")

	return &pb.IsActiveResponse{
		Result: true,
	}, nil
}

func (e *ExternalScaler) GetMetricSpec(context.Context, *pb.ScaledObjectRef) (*pb.GetMetricSpecResponse, error) {

	slog.Info(fmt.Sprintf("GetMetricSpec called - setting threshold to QUEUE_MESSAGE_COUNT_PER_REPLICA - %d", e.QUEUE_MESSAGE_COUNT_PER_REPLICA))

	return &pb.GetMetricSpecResponse{
		MetricSpecs: []*pb.MetricSpec{{
			MetricName: "qThreshold",
			TargetSize: int64(e.QUEUE_MESSAGE_COUNT_PER_REPLICA),
		}},
	}, nil
}

func (e *ExternalScaler) ValidateSetRequiredMetadata(metadata map[string]string) error {
	if e.METRICS_BACKEND == METRICS_BACKEND_PROMETHEUS {
		if e.PROMETHEUS_ENDPOINT == "" && metadata["prometheusEndpoint"] == "" {
			return fmt.Errorf("prometheusEndpoint is required for this configuration and not set")
		}
		if e.PROMETHEUS_ENDPOINT == "" && metadata["prometheusEndpoint"] != "" {
			fmt.Printf("Setting prometheusEndpoint to %s\n", metadata["prometheusEndpoint"])
			e.PROMETHEUS_ENDPOINT = metadata["prometheusEndpoint"]
		}
	}

	if e.METRICS_BACKEND == METRICS_BACKEND_AZURE {
		if e.LOG_ANALYTICS_WORKSPACE_ID == "" && metadata["logAnalyticsWorkspaceId"] == "" {
			return fmt.Errorf("logAnalyticsWorkspaceId is required for this configuration and not set")
		}
		if e.LOG_ANALYTICS_WORKSPACE_ID == "" && metadata["logAnalyticsWorkspaceId"] != "" {
			fmt.Printf("Setting logAnalyticsWorkspaceId to %s\n", metadata["logAnalyticsWorkspaceId"])
			e.LOG_ANALYTICS_WORKSPACE_ID = metadata["logAnalyticsWorkspaceId"]
		}

		if e.SERVICE_BUS_RESOURCE_ID == "" && metadata["serviceBusResourceId"] == "" {
			return fmt.Errorf("serviceBusResourceId is required for this configuration and not set")
		}
		if e.SERVICE_BUS_RESOURCE_ID == "" && metadata["serviceBusResourceId"] != "" {
			fmt.Println("Setting serviceBusResourceId")
			e.SERVICE_BUS_RESOURCE_ID = metadata["serviceBusResourceId"]
		}

		if e.SERVICE_BUS_QUEUE_NAME == "" && metadata["serviceBusQueueName"] == "" {
			return fmt.Errorf("serviceBusQueueName is required for this configuration and not set")
		}
		if e.SERVICE_BUS_QUEUE_NAME == "" && metadata["serviceBusQueueName"] != "" {
			fmt.Printf("Setting serviceBusQueueName to %s\n", metadata["serviceBusQueueName"])
			e.SERVICE_BUS_QUEUE_NAME = metadata["serviceBusQueueName"]
			e.MetricsReader = metricsReaders.NewAzureMetricsReader(e.SERVICE_BUS_RESOURCE_ID, e.SERVICE_BUS_QUEUE_NAME, e.MSG_QUEUE_LENGTH_METRIC_NAME, e.RATE_429_ERRORS_METRIC_NAME, e.LOG_ANALYTICS_WORKSPACE_ID)
		}

	}

	if e.INSTANCE_COMPUTE_BACKEND == INSTANCE_COMPUTE_BACKEND_KUBERNETES {
		if e.DEPLOYMENT_NAME == "" && metadata["deploymentName"] == "" {
			return fmt.Errorf("deploymentName is required for this configuration and not set")
		}
		if e.DEPLOYMENT_NAME == "" && metadata["deploymentName"] != "" {
			fmt.Printf("Setting deploymentName to %s\n", metadata["deploymentName"])
			e.DEPLOYMENT_NAME = metadata["deploymentName"]
		}

		if e.DEPLOYMENT_NAMESPACE == "" && metadata["deploymentNamespace"] == "" {
			return fmt.Errorf("deploymentNamespace is required for this configuration and not set")
		}
		if e.DEPLOYMENT_NAMESPACE == "" && metadata["deploymentNamespace"] != "" {
			fmt.Printf("Setting deploymentNamespace to %s\n", metadata["deploymentNamespace"])
			e.DEPLOYMENT_NAMESPACE = metadata["deploymentNamespace"]
		}

	}

	if e.INSTANCE_COMPUTE_BACKEND == INSTANCE_COMPUTE_BACKEND_CONTAINER_APPS {

		if e.AZURE_SUBSCRIPTION_ID == "" && metadata["azureSubscriptionId"] == "" {
			return fmt.Errorf("azureSubscriptionId is required for this configuration and not set")
		}
		if e.AZURE_SUBSCRIPTION_ID == "" && metadata["azureSubscriptionId"] != "" {
			fmt.Println("Setting azureSubscriptionId")
			e.AZURE_SUBSCRIPTION_ID = metadata["azureSubscriptionId"]
		}

		if e.RESOURCE_GROUP == "" && metadata["resourceGroup"] == "" {
			return fmt.Errorf("resourceGroup is required for this configuration and not set")
		}
		if e.RESOURCE_GROUP == "" && metadata["resourceGroup"] != "" {
			fmt.Printf("Setting resourceGroup to %s\n", metadata["resourceGroup"])
			e.RESOURCE_GROUP = metadata["resourceGroup"]
		}

		if e.CONTAINER_APP == "" && metadata["containerApp"] == "" {
			return fmt.Errorf("containerApp is required for this configuration and not set")
		}
		if e.CONTAINER_APP == "" && metadata["containerApp"] != "" {
			fmt.Printf("Setting containerApp to %s\n", metadata["containerApp"])
			e.CONTAINER_APP = metadata["containerApp"]
			//

			fmt.Printf("Setting Instance compute backend to containerApps")
			e.ReplicaCountReader = replicaCountReaders.NewContainerAppReplicaCountReader(e.AZURE_SUBSCRIPTION_ID, e.RESOURCE_GROUP, e.CONTAINER_APP)
			//

		}
	}

	if e.MIN_REPLICAS == 0 && metadata["minReplicas"] == "" {
		return fmt.Errorf("minReplicas is required for this configuration and not set")
	}
	if e.MIN_REPLICAS == 0 && metadata["minReplicas"] != "" {
		minReplicas, err := strconv.Atoi(metadata["minReplicas"])
		if err != nil {
			return fmt.Errorf("failed to convert minReplicas to int: %v", err)
		}
		fmt.Printf("Setting minReplicas to %d\n", minReplicas)
		e.MIN_REPLICAS = minReplicas
	}

	if e.MAX_REPLICAS == 0 && metadata["maxReplicas"] == "" {
		return fmt.Errorf("maxReplicas is required for this configuration and not set")
	}
	if e.MAX_REPLICAS == 0 && metadata["maxReplicas"] != "" {
		maxReplicas, err := strconv.Atoi(metadata["maxReplicas"])
		if err != nil {
			return fmt.Errorf("failed to convert maxReplicas to int: %v", err)
		}
		fmt.Printf("Setting maxReplicas to %d\n", maxReplicas)
		e.MAX_REPLICAS = maxReplicas
	}

	return nil

}

func (e *ExternalScaler) GetMetrics(_ context.Context, metricRequest *pb.GetMetricsRequest) (*pb.GetMetricsResponse, error) {

	slog.Info("GetMetrics called")

	// Validate the metadata and set the required configurations
	e.ValidateSetRequiredMetadata(metricRequest.ScaledObjectRef.ScalerMetadata)

	replicas, err := e.ReplicaCountReader.GetInstanceCount()
	if err != nil {
		fmt.Printf("Failed to get deployment instance count: %v\n", err)
		return nil, err
	}

	slog.Debug(fmt.Sprintf("number of current workload replicas: %d\n", replicas))

	rate429Errors, err := e.MetricsReader.GetRate429Errors()
	if err != nil {
		fmt.Printf("Failed to get rate_429_errors: %v\n", err)
		return nil, err
	}

	slog.Debug(fmt.Sprintf("rate_429_errors: %d\n", rate429Errors))

	msgQueueLength, err := e.MetricsReader.GetQueueLength()
	if err != nil {
		fmt.Printf("Failed to get msg_queue_length: %v\n", err)
		return nil, err
	}

	slog.Debug(fmt.Sprintf("msg_queue_length: %d\n", msgQueueLength))

	revisedMetricValue := e.getRevisedMetricValue(msgQueueLength, rate429Errors, replicas, e.MIN_REPLICAS, e.MAX_REPLICAS, time.Since(e.lastScaleDownRequestTime))

	return &pb.GetMetricsResponse{
		MetricValues: []*pb.MetricValue{{
			MetricName:  "qThreshold",
			MetricValue: int64(revisedMetricValue),
		}},
	}, nil
}

func (e *ExternalScaler) getRevisedMetricValue(msgQueueLength int, rate429Errors int, workloadReplicaCount int, minReplicas int, maxReplicas int, timeSinceLastScaleDownRequest time.Duration) int {

	slog.Debug(fmt.Sprintf("Current Time UTC: %v", time.Now().UTC()))
	fmt.Println("############################################################################################################")
	slog.Info(fmt.Sprintf("msgQueueLength: %d, rate429Errors: %d, workloadReplicaCount: %d, minReplicas: %d, maxReplicas: %d, timeSinceLastScaleDownRequest: %v", msgQueueLength, rate429Errors, workloadReplicaCount, minReplicas, maxReplicas, timeSinceLastScaleDownRequest))

	var retVal int
	scaleDownWaitInterval := time.Minute * time.Duration(e.TIME_BETWEEN_SCALE_DOWN_REQUESTS_MINUTES)

	if e.replicaCountDuringLastScaleDownRequest == -1 {
		e.replicaCountDuringLastScaleDownRequest = workloadReplicaCount
	}

	if rate429Errors < e.RATE_429_ERROR_THRESHOLD {
		fmt.Printf("rate429Errors < RATE_429_ERROR_THRESHOLD(%d), returning msgQueueLength \n", e.RATE_429_ERROR_THRESHOLD)
		return msgQueueLength
	}

	if workloadReplicaCount <= minReplicas {
		retVal = e.QUEUE_MESSAGE_COUNT_PER_REPLICA * minReplicas
		fmt.Printf("workloadReplicaCount <= minReplicas, returning QUEUE_MESSAGE_COUNT_PER_REPLICA(%d) * minReplicas(%d): %d\n", e.QUEUE_MESSAGE_COUNT_PER_REPLICA, minReplicas, retVal)
		return retVal
	}

	if timeSinceLastScaleDownRequest < scaleDownWaitInterval {
		retVal = e.replicaCountDuringLastScaleDownRequest * e.QUEUE_MESSAGE_COUNT_PER_REPLICA
		fmt.Printf("timeSinceLastScaleDownRequest < scaleDownWaitInterval, returning replicaCountDuringLastScaleDownRequest(%d) * QUEUE_MESSAGE_COUNT_PER_REPLICA(%d): %d\n", e.replicaCountDuringLastScaleDownRequest, e.QUEUE_MESSAGE_COUNT_PER_REPLICA, retVal)
		return retVal
	}

	// Error Rate Higher than Threshold.
	// Current Replicas more then min replicas.
	// Time since last scale down request is more than the wait time
	// Create scale down request by setting return value appropriately

	e.lastScaleDownRequestTime = time.Now()
	requestedReplicaCount := workloadReplicaCount - 1
	e.replicaCountDuringLastScaleDownRequest = requestedReplicaCount
	retVal = requestedReplicaCount * e.QUEUE_MESSAGE_COUNT_PER_REPLICA
	fmt.Printf("Returning requestedReplicaCount (%d) * QUEUE_MESSAGE_COUNT_PER_REPLICA(%d): %d \n", requestedReplicaCount, e.QUEUE_MESSAGE_COUNT_PER_REPLICA, retVal)
	return retVal

}

func (e *ExternalScaler) StreamIsActive(scaledObject *pb.ScaledObjectRef, epsServer pb.ExternalScaler_StreamIsActiveServer) error {

	slog.Info("StreamIsActive called")

	for {
		select {
		case <-epsServer.Context().Done():
			return nil
		case <-time.Tick(time.Hour * 1):
			_ = epsServer.Send(&pb.IsActiveResponse{
				Result: true,
			})
		}
	}
}

func printConfigurationSettings(es *ExternalScaler) {
	fmt.Println("QUEUE_MESSAGE_COUNT_PER_REPLICA: ", es.QUEUE_MESSAGE_COUNT_PER_REPLICA)
	fmt.Println("RATE_429_ERROR_THRESHOLD: ", es.RATE_429_ERROR_THRESHOLD)
	fmt.Println("TIME_BETWEEN_SCALE_DOWN_REQUESTS_MINUTES: ", es.TIME_BETWEEN_SCALE_DOWN_REQUESTS_MINUTES)
	fmt.Println("MSG_QUEUE_LENGTH_METRIC_NAME: ", es.MSG_QUEUE_LENGTH_METRIC_NAME)
	fmt.Println("RATE_429_ERRORS_METRIC_NAME: ", es.RATE_429_ERRORS_METRIC_NAME)
	fmt.Println("PROMETHEUS_ENDPOINT: ", es.PROMETHEUS_ENDPOINT)
}

func main() {
	grpcServer := grpc.NewServer()
	lis, err := net.Listen("tcp", ":6000")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	e := ExternalScaler{
		lastScaleDownRequestTime:                 time.Now(),
		replicaCountDuringLastScaleDownRequest:   -1,
		QUEUE_MESSAGE_COUNT_PER_REPLICA:          getEnvInt("QUEUE_MESSAGE_COUNT_PER_REPLICA", 10),
		RATE_429_ERROR_THRESHOLD:                 getEnvInt("RATE_429_ERROR_THRESHOLD", 5),
		TIME_BETWEEN_SCALE_DOWN_REQUESTS_MINUTES: getEnvInt("TIME_BETWEEN_SCALE_DOWN_REQUESTS_MINUTES", 1),
		MSG_QUEUE_LENGTH_METRIC_NAME:             getEnvString("MSG_QUEUE_LENGTH_METRIC_NAME", "msg_queue_length"),
		RATE_429_ERRORS_METRIC_NAME:              getEnvString("RATE_429_ERRORS_METRIC_NAME", "rate_429_errors"),
		PROMETHEUS_ENDPOINT:                      getEnvString("PROMETHEUS_ENDPOINT", ""),
		METRICS_BACKEND:                          getEnvString("METRICS_BACKEND", ""),
		INSTANCE_COMPUTE_BACKEND:                 getEnvString("INSTANCE_COMPUTE_BACKEND", ""),
	}

	// wire up metrics and compute backends

	if e.METRICS_BACKEND == "" {
		fmt.Println("METRICS_BACKEND not set, defaulting to prometheus")
		e.METRICS_BACKEND = METRICS_BACKEND_PROMETHEUS
	}

	if e.INSTANCE_COMPUTE_BACKEND == "" {
		fmt.Println("INSTANCE_COMPUTE_BACKEND not set, defaulting to kubernetes")
		e.INSTANCE_COMPUTE_BACKEND = INSTANCE_COMPUTE_BACKEND_KUBERNETES
	}

	if e.METRICS_BACKEND == METRICS_BACKEND_PROMETHEUS {
		e.MetricsReader = metricsReaders.NewPrometheusMetricsReader(e.PROMETHEUS_ENDPOINT, e.MSG_QUEUE_LENGTH_METRIC_NAME, e.RATE_429_ERRORS_METRIC_NAME)
	}

	if e.INSTANCE_COMPUTE_BACKEND == INSTANCE_COMPUTE_BACKEND_KUBERNETES {
		fmt.Printf("Setting Instance compute backend to kubernetes")
		e.ReplicaCountReader = replicaCountReaders.NewK8sDeploymentReplicaCountReader()
	}

	pb.RegisterExternalScalerServer(grpcServer, &e)
	fmt.Println("listenting on :6000")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatal(err)
	}
}
