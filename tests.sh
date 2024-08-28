#!/bin/bash

# port-forward prometheus server
kubectl port-forward svc/prometheus-server -n prometheus 9090:80 

# port forward metrics-modifier (golang app)
kubectl port-forward svc/metrics-modifier-service 5060:80

# get all metrics from metrics-modifier app
curl localhost:5060/metrics

# modify msg_queue_length metric
curl -X PUT localhost:5060/setQueueLength/20

# modify rate_429_errors metric
curl -X PUT localhost:5060/setRate429Errors/1

# continously watch prometheus metrics used by keda
# watch msg_queue_length
watch " curl -s -g --data-urlencode 'query=msg_queue_length' 'http://localhost:9090/api/v1/query' | jq '{queue_len: .data.result[0].value[1]}'"
# watch rate_429_errors
watch " curl -s -g --data-urlencode 'query=rate_429_errors' 'http://localhost:9090/api/v1/query' | jq '{rate_429_errors: .data.result[0].value[1]}'"
# watch value of prometheus query used by keda scaled object
watch " curl -s -g --data-urlencode 'query=(msg_queue_length)*(clamp_min(clamp_max((rate_429_errors < 5),1),1)) OR on() vector(0)' 'http://localhost:9090/api/v1/query' | jq '{query_result: .data.result[0].value[1]}'"




