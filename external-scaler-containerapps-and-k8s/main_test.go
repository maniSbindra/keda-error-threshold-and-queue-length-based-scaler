package main

import (
	"testing"
	"time"
)

func TestGetRevisedMetricValueErrorsBelowThreshold(t *testing.T) {

	e := &ExternalScaler{
		lastScaleDownRequestTime:                 time.Now(),
		replicaCountDuringLastScaleDownRequest:   -1,
		TIME_BETWEEN_SCALE_DOWN_REQUESTS_MINUTES: 1,
		RATE_429_ERROR_THRESHOLD:                 5,
		QUEUE_MESSAGE_COUNT_PER_REPLICA:          10,
	}

	testCases := []struct {
		msgQueueLength                int
		rate429Errors                 int
		workloadReplicaCount          int
		minReplicas                   int
		maxReplicas                   int
		timeSinceLastScaleDownRequest time.Duration
		expected                      int
	}{
		{
			msgQueueLength:                20,
			rate429Errors:                 0,
			workloadReplicaCount:          1,
			minReplicas:                   1,
			maxReplicas:                   7,
			timeSinceLastScaleDownRequest: time.Minute * 2,
			expected:                      20,
		},
		{
			msgQueueLength:                20,
			rate429Errors:                 3,
			workloadReplicaCount:          2,
			minReplicas:                   2,
			maxReplicas:                   7,
			timeSinceLastScaleDownRequest: time.Minute * 2,
			expected:                      20,
		},
		{
			msgQueueLength:                40,
			rate429Errors:                 4,
			workloadReplicaCount:          6,
			minReplicas:                   2,
			maxReplicas:                   7,
			timeSinceLastScaleDownRequest: time.Second * 20,
			expected:                      40,
		},
		{
			msgQueueLength:                40,
			rate429Errors:                 4,
			workloadReplicaCount:          6,
			minReplicas:                   2,
			maxReplicas:                   7,
			timeSinceLastScaleDownRequest: time.Minute * 2,
			expected:                      40,
		},
		// Add more test cases here...
	}

	for _, tc := range testCases {
		result := e.getRevisedMetricValue(tc.msgQueueLength, tc.rate429Errors, tc.workloadReplicaCount, tc.minReplicas, tc.maxReplicas, tc.timeSinceLastScaleDownRequest)

		if result != tc.expected {
			t.Errorf("Expected %d, but got %d", tc.expected, result)
		}
	}
}

func TestGetRevisedMetricValueErrorsAboveThreshold(t *testing.T) {
	e := &ExternalScaler{
		lastScaleDownRequestTime:                 time.Now(),
		replicaCountDuringLastScaleDownRequest:   -1,
		TIME_BETWEEN_SCALE_DOWN_REQUESTS_MINUTES: 1,
		RATE_429_ERROR_THRESHOLD:                 5,
		QUEUE_MESSAGE_COUNT_PER_REPLICA:          10,
	}

	testCases := []struct {
		name                                                  string
		replicaCountDuringLastScaleDownRequest                int
		msgQueueLength                                        int
		rate429Errors                                         int
		workloadReplicaCount                                  int
		minReplicas                                           int
		maxReplicas                                           int
		timeSinceLastScaleDownRequest                         time.Duration
		expectedRevisedReplicaCountDuringLastScaleDownRequest int
		expected                                              int
	}{
		{
			name:                                   "Last replica count is -1",
			replicaCountDuringLastScaleDownRequest: -1,
			msgQueueLength:                         20,
			rate429Errors:                          6,
			workloadReplicaCount:                   2,
			minReplicas:                            1,
			maxReplicas:                            7,
			timeSinceLastScaleDownRequest:          time.Minute * 2,
			expectedRevisedReplicaCountDuringLastScaleDownRequest: 1,
			expected: 10,
		},
		{
			name:                                   "rate429Errors is one full threshold",
			replicaCountDuringLastScaleDownRequest: 6,
			msgQueueLength:                         60,
			rate429Errors:                          6,
			workloadReplicaCount:                   6,
			minReplicas:                            1,
			maxReplicas:                            7,
			timeSinceLastScaleDownRequest:          time.Minute * 2,
			expectedRevisedReplicaCountDuringLastScaleDownRequest: 5,
			expected: 50, // scale down by 1 replicas
		},

		{
			name:                                   "rate429Errors is twice the threshold",
			replicaCountDuringLastScaleDownRequest: 6,
			msgQueueLength:                         60,
			rate429Errors:                          10,
			workloadReplicaCount:                   6,
			minReplicas:                            1,
			maxReplicas:                            7,
			timeSinceLastScaleDownRequest:          time.Minute * 2,
			expectedRevisedReplicaCountDuringLastScaleDownRequest: 4,
			expected: 40, // scale down by 2 replicas
		},
		{
			name:                                   "rate429Errors is twice the threshold (ignore last scale-down request)",
			replicaCountDuringLastScaleDownRequest: 5,
			msgQueueLength:                         60,
			rate429Errors:                          10,
			workloadReplicaCount:                   6,
			minReplicas:                            1,
			maxReplicas:                            7,
			timeSinceLastScaleDownRequest:          time.Minute * 2,
			expectedRevisedReplicaCountDuringLastScaleDownRequest: 4,
			expected: 40,
		},
		{
			name:                                   "don't scale below min replicas",
			replicaCountDuringLastScaleDownRequest: 2,
			msgQueueLength:                         70,
			rate429Errors:                          10,
			workloadReplicaCount:                   2,
			minReplicas:                            2,
			maxReplicas:                            7,
			timeSinceLastScaleDownRequest:          time.Minute * 2,
			expectedRevisedReplicaCountDuringLastScaleDownRequest: 2,
			expected: 20,
		},
		{
			name:                                   "rate429Errors is twice the threshold (2)",
			replicaCountDuringLastScaleDownRequest: 4,
			msgQueueLength:                         70,
			rate429Errors:                          10,
			workloadReplicaCount:                   5,
			minReplicas:                            2,
			maxReplicas:                            7,
			timeSinceLastScaleDownRequest:          time.Minute * 2,
			expectedRevisedReplicaCountDuringLastScaleDownRequest: 3,
			expected: 30,
		},
		{
			name:                                   "Within scaledown window, don't scale down",
			replicaCountDuringLastScaleDownRequest: 4,
			msgQueueLength:                         70,
			rate429Errors:                          10,
			workloadReplicaCount:                   5,
			minReplicas:                            2,
			maxReplicas:                            7,
			timeSinceLastScaleDownRequest:          time.Second * 20,
			expectedRevisedReplicaCountDuringLastScaleDownRequest: 4,
			expected: 40,
		},
		{
			name:                                   "Within scaledown window, don't scale down (2)",
			replicaCountDuringLastScaleDownRequest: 4,
			msgQueueLength:                         70,
			rate429Errors:                          10,
			workloadReplicaCount:                   4,
			minReplicas:                            2,
			maxReplicas:                            7,
			timeSinceLastScaleDownRequest:          time.Second * 20,
			expectedRevisedReplicaCountDuringLastScaleDownRequest: 4,
			expected: 40,
		},
	}

	for _, tc := range testCases {
		e.replicaCountDuringLastScaleDownRequest = tc.replicaCountDuringLastScaleDownRequest
		result := e.getRevisedMetricValue(tc.msgQueueLength, tc.rate429Errors, tc.workloadReplicaCount, tc.minReplicas, tc.maxReplicas, tc.timeSinceLastScaleDownRequest)

		if e.replicaCountDuringLastScaleDownRequest != tc.expectedRevisedReplicaCountDuringLastScaleDownRequest {
			t.Errorf("Set replica count mismatch, Expected %d, but got %d (%q)", tc.expectedRevisedReplicaCountDuringLastScaleDownRequest, e.replicaCountDuringLastScaleDownRequest, tc.name)
		}

		if result != tc.expected {
			t.Errorf("Expected %d, but got %d (%q)", tc.expected, result, tc.name)
		}

	}
}
