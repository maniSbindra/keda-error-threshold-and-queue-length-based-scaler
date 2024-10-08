package metricsReaders

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
)

type AzureMetricsReader struct {
	servicebusResourceID           string
	servBusQueueOrTopicName        string
	serviceBusTopicSubcriptionName string
	error429MetricName             string

	logAnalyticsWorkspaceID string
}

type TokenProvider interface {
	GetToken(ctx context.Context, opts policy.TokenRequestOptions) (azcore.AccessToken, error)
}

func NewAzureMetricsReader(servicebusResourceID string, serBusQueueOrTopicName string, serBusTopicSubcriptionName string, error429MetricName string, logAnalyticsWorkspaceID string) *AzureMetricsReader {
	return &AzureMetricsReader{
		servicebusResourceID:           servicebusResourceID,
		servBusQueueOrTopicName:        serBusQueueOrTopicName,
		serviceBusTopicSubcriptionName: serBusTopicSubcriptionName,
		error429MetricName:             error429MetricName,
		logAnalyticsWorkspaceID:        logAnalyticsWorkspaceID,
	}
}

func (a *AzureMetricsReader) getBearerToken(tp TokenProvider) (bearerToken string, err error) {
	opts := policy.TokenRequestOptions{Scopes: []string{"https://management.azure.com/.default"}}
	tok, err := tp.GetToken(context.Background(), opts)
	if err != nil {
		return "", err
	}

	return tok.Token, nil
}

func (a *AzureMetricsReader) getLogAnalyticsBearerToken(tp TokenProvider) (bearerToken string, err error) {
	opts := policy.TokenRequestOptions{Scopes: []string{"https://api.loganalytics.io/.default"}}
	tok, err := tp.GetToken(context.Background(), opts)
	if err != nil {
		return "", err
	}

	return tok.Token, nil
}

func (a *AzureMetricsReader) GetRate429Errors() (int, error) {
	return a.GetLogAnalyticsQueryResult(fmt.Sprintf("AppMetrics | where Name  == '%s' | top 1 by TimeGenerated desc | project rate_429_errors=(Sum /ItemCount)", a.error429MetricName))
}

func (a *AzureMetricsReader) GetQueueOrTopicLengthRequestUri() string {
	if a.serviceBusTopicSubcriptionName == "" {
		return fmt.Sprintf("https://management.azure.com:443%s/queues/%s?api-version=2023-01-01-preview", a.servicebusResourceID, a.servBusQueueOrTopicName)
	}
	return fmt.Sprintf("https://management.azure.com:443%s/topics/%s/subscriptions/%s?api-version=2023-01-01-preview", a.servicebusResourceID, a.servBusQueueOrTopicName, a.serviceBusTopicSubcriptionName)
}

func (a *AzureMetricsReader) GetQueueLength() (int, error) {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return 0, fmt.Errorf("failed to get Azure credential: %w", err)
	}

	startTime := time.Now().Add(-time.Minute * 5).UTC()

	startTimeString := startTime.Format("2006-01-02T15:04:05")
	startTimeString = fmt.Sprintf("%s.000Z", startTimeString)

	endTime := time.Now().UTC()
	endTimeString := endTime.Format("2006-01-02T15:04:05")
	endTimeString = fmt.Sprintf("%s.000Z", endTimeString)

	timespan := fmt.Sprintf("%s/%s", startTimeString, endTimeString)

	timespan = strings.Trim(timespan, " ")
	fmt.Printf("Time span: %s\n", timespan)
	requestUri := a.GetQueueOrTopicLengthRequestUri()

	// fmt.Printf("Request URI: %s\n", requestUri)
	slog.Debug(fmt.Sprintf("Request URI: %s\n", requestUri))

	bearerToken, err := a.getBearerToken(cred)
	if err != nil {
		return 0, fmt.Errorf("failed to get bearer token: %w", err)
	}

	client := &http.Client{}

	req, err := http.NewRequest("GET", requestUri, nil)
	if err != nil {
		return 0, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Go HTTP Client")

	// add bearer token to header
	req.Header.Add("Authorization", "Bearer "+bearerToken)

	// make request
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}

	// read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	// parse response body
	var result map[string]interface{}
	err = json.Unmarshal(body, &result)
	if err != nil {
		return 0, err
	}

	// fmt.Printf("Response body: %v", result)
	// slog.Debug(fmt.Sprintf("Response body: %v", result))

	// print result json

	// get metric value

	queueOrTopicLength := result["properties"].(map[string]interface{})["countDetails"].(map[string]interface{})["activeMessageCount"].(float64)

	return int(queueOrTopicLength), nil

	// get auth token

}

func (a *AzureMetricsReader) GetLogAnalyticsQueryResult(query string) (int, error) {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return 0, err
	}

	bearerToken, err := a.getLogAnalyticsBearerToken(cred)
	if err != nil {
		return 0, err
	}

	client := &http.Client{}

	queryUri := fmt.Sprintf("https://api.loganalytics.io/v1/workspaces/%s/query?query=%s", a.logAnalyticsWorkspaceID, url.QueryEscape(query))
	slog.Debug(fmt.Sprintf("QueryURI: %s \n", queryUri))

	// fmt.Printf("Query URI: %s\n", queryUri)

	req, err := http.NewRequest("GET", queryUri, nil)
	if err != nil {
		return 0, fmt.Errorf("could not create get request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Go HTTP Client")

	// add bearer token to header
	req.Header.Add("Authorization", "Bearer "+bearerToken)

	// make request
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("could not make request: %w", err)
	}

	// read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("could not read log analytics workspace response body: %w", err)
	}
	defer resp.Body.Close()

	slog.Debug(fmt.Sprintf("Response body: %s\n", string(body)))

	// parse body to get query result
	var result map[string]interface{}
	err = json.Unmarshal(body, &result)
	if err != nil {
		return 0, fmt.Errorf("could not unmarshall log analytics workspace query response body %w", err)
	}
	// fmt.Printf("Result: %v\n", result)

	tables := result["tables"].([]interface{})
	// fmt.Printf("Tables: %v\n", tables)

	// fmt.Printf("Tables[0]: %v\n", tables[0])

	rows := tables[0].(map[string]interface{})["rows"].([]interface{})

	// fmt.Printf("Rows: %v\n", rows)
	if len(rows) == 0 {
		// this implies no 429 error data exists
		return 0, nil
	}
	row := rows[0]
	// fmt.Printf("Row: %v\n", row)

	//row is [15] , we need to get value 15 from it
	res := row.([]interface{})[0]
	// fmt.Printf("Res: %v\n", res)

	// res := rows[0]

	queryResult, err := strconv.Atoi(fmt.Sprintf("%v", res))
	if err != nil {
		return 0, fmt.Errorf("could not parse  log analytics workspace query response: %w", err)
	}

	return queryResult, nil
}
