package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net"
	"strconv"

	pb "github.com/manisbindra/kedaQueueLengthAndErrorRateExternalScaler/externalscaler"

	// "log"
	// "net"

	"os"
	"time"

	"google.golang.org/grpc"
	// "google.golang.org/grpc/codes"
	// "google.golang.org/grpc/status"
)

type ExternalScaler struct {
	pb.UnimplementedExternalScalerServer
	lastScaleDownRequestTime               time.Time
	replicaCountDuringLastScaleDownRequest int
	// deploymentName string
	// deploymentNamespace string
}

// const TIME_BETWEEN_SCALE_DOWN_REQUESTS_MINUTES = 4

var (
	QUEUE_MESSAGE_COUNT_PER_REPLICA          = getEnvInt("QUEUE_MESSAGE_COUNT_PER_REPLICA", 10)
	RATE_429_ERROR_THRESHOLD                 = getEnvInt("RATE_429_ERROR_THRESHOLD", 5)
	TIME_BETWEEN_SCALE_DOWN_REQUESTS_MINUTES = getEnvInt("TIME_BETWEEN_SCALE_DOWN_REQUESTS_MINUTES", 1)
	MSG_QUEUE_LENGTH_METRIC_NAME             = getEnvString("MSG_QUEUE_LENGTH_METRIC_NAME", "msg_queue_length")
	RATE_429_ERRORS_METRIC_NAME              = getEnvString("RATE_429_ERRORS_METRIC_NAME", "rate_429_errors")
)

// log all configuration values set via environment variables
func init() {
	log.Printf("Loading Environment Configurations\n")
	log.Printf("QUEUE_MESSAGE_COUNT_PER_REPLICA: %d\n", QUEUE_MESSAGE_COUNT_PER_REPLICA)
	log.Printf("RATE_429_ERROR_THRESHOLD: %d\n", RATE_429_ERROR_THRESHOLD)
	log.Printf("TIME_BETWEEN_SCALE_DOWN_REQUESTS_MINUTES: %d\n", TIME_BETWEEN_SCALE_DOWN_REQUESTS_MINUTES)
	log.Printf("MSG_QUEUE_LENGTH_METRIC_NAME: %s\n", MSG_QUEUE_LENGTH_METRIC_NAME)
	log.Printf("RATE_429_ERRORS_METRIC_NAME: %s\n", RATE_429_ERRORS_METRIC_NAME)
	log.Printf("PROMETHEUS_ENDPOINT: %s\n", getEnvString("PROMETHEUS_ENDPOINT", "http://prometheus-server.prometheus:80"))
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

	slog.Info("GetMetricSpec called - setting threshold to QUEUE_MESSAGE_COUNT_PER_REPLICA")

	return &pb.GetMetricSpecResponse{
		MetricSpecs: []*pb.MetricSpec{{
			MetricName: "qThreshold",
			TargetSize: int64(QUEUE_MESSAGE_COUNT_PER_REPLICA),
		}},
	}, nil
}

func (e *ExternalScaler) GetMetrics(_ context.Context, metricRequest *pb.GetMetricsRequest) (*pb.GetMetricsResponse, error) {

	fmt.Println("GetMetrics called")

	deploymentName := metricRequest.ScaledObjectRef.ScalerMetadata["deploymentName"]
	deploymentNamespace := metricRequest.ScaledObjectRef.ScalerMetadata["deploymentNamespace"]
	minReplicasStr := metricRequest.ScaledObjectRef.ScalerMetadata["minReplicas"]
	maxReplicasStr := metricRequest.ScaledObjectRef.ScalerMetadata["maxReplicas"]
	minReplicas, err := strconv.Atoi(minReplicasStr)

	if err != nil {
		fmt.Printf("failed to convert minReplicas to int: %v\n", err)
		return nil, err
	}

	maxReplicas, err := strconv.Atoi(maxReplicasStr)
	if err != nil {
		fmt.Printf("Failed to convert maxReplicas to int: %v\n", err)
		return nil, err
	}

	replicas, err := getDeploymentInstanceCount(deploymentName, deploymentNamespace)
	if err != nil {
		fmt.Printf("Failed to get deployment instance count: %v\n", err)
		return nil, err
	}
	fmt.Printf("number of current workload replicas: %d\n", replicas)

	rate429Errors, err := getMetric(RATE_429_ERRORS_METRIC_NAME)
	if err != nil {
		fmt.Printf("Failed to get rate_429_errors: %v\n", err)
		return nil, err
	}
	fmt.Printf("rate_429_errors: %d\n", rate429Errors)

	msgQueueLength, err := getMetric(MSG_QUEUE_LENGTH_METRIC_NAME)
	if err != nil {
		fmt.Printf("Failed to get msg_queue_length: %v\n", err)
		return nil, err
	}
	fmt.Printf("msg_queue_length: %d\n", msgQueueLength)

	revisedMetricValue := e.getRevisedMetricValue(msgQueueLength, rate429Errors, replicas, minReplicas, maxReplicas, time.Since(e.lastScaleDownRequestTime))

	return &pb.GetMetricsResponse{
		MetricValues: []*pb.MetricValue{{
			MetricName:  "qThreshold",
			MetricValue: int64(revisedMetricValue),
		}},
	}, nil
}

func (e *ExternalScaler) getRevisedMetricValue(msgQueueLength int, rate429Errors int, workloadReplicaCount int, minReplicas int, maxReplicas int, timeSinceLastScaleDownRequest time.Duration) int {

	slog.Info(fmt.Sprintf("msgQueueLength: %d, rate429Errors: %d, workloadReplicaCount: %d, minReplicas: %d, maxReplicas: %d, timeSinceLastScaleDownRequest: %v", msgQueueLength, rate429Errors, workloadReplicaCount, minReplicas, maxReplicas, timeSinceLastScaleDownRequest))

	var retVal int
	scaleDownWaitInterval := time.Minute * time.Duration(TIME_BETWEEN_SCALE_DOWN_REQUESTS_MINUTES)

	if e.replicaCountDuringLastScaleDownRequest == -1 {
		e.replicaCountDuringLastScaleDownRequest = workloadReplicaCount
	}

	if rate429Errors < RATE_429_ERROR_THRESHOLD {
		slog.Info("rate429Errors < RATE_429_ERROR_THRESHOLD, returning msgQueueLength")
		return msgQueueLength
	}

	if workloadReplicaCount <= minReplicas {
		retVal = QUEUE_MESSAGE_COUNT_PER_REPLICA * minReplicas
		slog.Info(fmt.Sprintf("workloadReplicaCount <= minReplicas, returning QUEUE_MESSAGE_COUNT_PER_REPLICA * minReplicas: %d", retVal))
		return retVal
	}

	if timeSinceLastScaleDownRequest < scaleDownWaitInterval {
		retVal = e.replicaCountDuringLastScaleDownRequest * QUEUE_MESSAGE_COUNT_PER_REPLICA
		slog.Info(fmt.Sprintf("timeSinceLastScaleDownRequest < scaleDownWaitInterval, returning replicaCountDuringLastScaleDownRequest * QUEUE_MESSAGE_COUNT_PER_REPLICA: %d", retVal))
		return retVal
	}

	// Error Rate Higher than Threshold.
	// Current Replicas more then min replicas.
	// Time since last scale down request is more than the wait time
	// Create scale down request by setting return value appropriately

	e.lastScaleDownRequestTime = time.Now()
	requestedReplicaCount := workloadReplicaCount - 1
	e.replicaCountDuringLastScaleDownRequest = requestedReplicaCount
	retVal = requestedReplicaCount * QUEUE_MESSAGE_COUNT_PER_REPLICA
	slog.Info(fmt.Sprintf("Returning requestedReplicaCount (%d) * QUEUE_MESSAGE_COUNT_PER_REPLICA(%d): %d", requestedReplicaCount, QUEUE_MESSAGE_COUNT_PER_REPLICA, retVal))
	return retVal

}

func (e *ExternalScaler) StreamIsActive(scaledObject *pb.ScaledObjectRef, epsServer pb.ExternalScaler_StreamIsActiveServer) error {

	slog.Info("StreamIsActive called")

	for {
		select {
		case <-epsServer.Context().Done():
			// call cancelled
			return nil
		case <-time.Tick(time.Hour * 1):
			_ = epsServer.Send(&pb.IsActiveResponse{
				Result: true,
			})
		}
	}
}

func main() {
	grpcServer := grpc.NewServer()
	lis, _ := net.Listen("tcp", ":6000")
	pb.RegisterExternalScalerServer(grpcServer, &ExternalScaler{
		lastScaleDownRequestTime:               time.Now(),
		replicaCountDuringLastScaleDownRequest: -1,
	})

	fmt.Println("listenting on :6000")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatal(err)
	}
}
