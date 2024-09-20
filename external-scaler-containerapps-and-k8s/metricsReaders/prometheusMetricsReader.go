package metricsReaders

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

type PrometheusMetricsReader struct {
	PROMETHEUS_ENDPOINT          string
	MSG_QUEUE_LENGTH_METRIC_NAME string
	RATE_429_ERRORS_METRIC_NAME  string
}

func NewPrometheusMetricsReader(prometheusEndpoint string, msqQueueLengthMetricName string, rate429ErrorMetricName string) *PrometheusMetricsReader {
	return &PrometheusMetricsReader{
		PROMETHEUS_ENDPOINT:          prometheusEndpoint,
		MSG_QUEUE_LENGTH_METRIC_NAME: msqQueueLengthMetricName,
		RATE_429_ERRORS_METRIC_NAME:  rate429ErrorMetricName,
	}
}

func (p *PrometheusMetricsReader) GetMetricValue(metricName string) (int, error) {
	// Create a new Prometheus API client
	client, err := api.NewClient(api.Config{
		Address: p.PROMETHEUS_ENDPOINT,
		Client:  &http.Client{},
	})
	if err != nil {
		fmt.Println("Failed to create Prometheus client:", err)
		return 0, err
	}

	// Create a new API query client
	queryClient := v1.NewAPI(client)

	// Define the metric name you want to query
	// metricName := "your_metric_name"

	// // Define the time range for the query
	// startTime := time.Now().Add(-time.Hour)
	// endTime := time.Now()

	// // Build the query string
	// query := fmt.Sprintf(`your_query_expression{metric="%s"}`, metricName)

	// Execute the query
	res, warnings, err := queryClient.Query(context.Background(), metricName, time.Now())
	// queryClient.Q
	if err != nil {
		fmt.Println("Failed to execute query:", err)
		return 0, err
	}

	// Check for any warnings
	if len(warnings) > 0 {
		fmt.Println("Query warnings:", warnings)
	}

	var resStrVal string
	switch res.Type() {

	// case model.ValScalar:
	case model.ValScalar:
		// Implement model.Scalar
	case model.ValVector:
		vector := res.(model.Vector)
		for _, sample := range vector {
			// log.Println(sample.Value)
			resStrVal = sample.Value.String()
		}
	case model.ValMatrix:
		// Implement model.Matrix
	case model.ValString:
		// Implemenet String
	}

	// metricVal, err := strconv.Atoi(result.String())
	// resultBytes, err := result.Type().MarshalJSON()

	// fetch .data.result[0].value[1] from results json
	// result.

	metricVal, err := strconv.Atoi(resStrVal)
	if err != nil {
		return 0, err
	}
	return metricVal, nil
	// Process the query result
	// ...

	// Print the result
	// fmt.Println(result)
}

func (p *PrometheusMetricsReader) GetQueueLength() (int, error) {
	// Execute the query
	return p.GetMetricValue(p.MSG_QUEUE_LENGTH_METRIC_NAME)
}

func (p *PrometheusMetricsReader) GetRate429Errors() (int, error) {
	// Execute the query
	return p.GetMetricValue(p.RATE_429_ERRORS_METRIC_NAME)
}
