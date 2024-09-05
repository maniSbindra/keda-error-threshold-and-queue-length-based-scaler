#!/bin/bash

# port-forward prometheus server
kubectl port-forward svc/prometheus-server -n prometheus 9090:80 

# port forward metrics-modifier (golang app)
kubectl port-forward svc/metrics-modifier-service 5060:80

# modify msg_queue_length metric
curl -X PUT localhost:5060/setQueueLength/20

# modify rate_429_errors metric
curl -X PUT localhost:5060/setRate429Errors/1

# follow external scaler logs
kubectl logs svc/golang-external-scaler -f
