package main

import (
	// "fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	log "github.com/sirupsen/logrus"
)

var (
	msgQueueLen = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "msg_queue_length",
		Help: "The message queue length used by Keda for scaling decisions",
	})

	rate429Errors = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "rate_429_errors",
		Help: " The per-second average rate of HTTP 429 errors over a 2 minute window",
	})
)

func recordQueueLength(len float64) {
	msgQueueLen.Set(len)
}

func recordRate429Errors(rate float64) {
	rate429Errors.Set(rate)
}

func main() {

	recordQueueLength(0.0)
	recordRate429Errors(0.0)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /callResult", callResult)
	mux.HandleFunc("GET /delayedCallResult/{delayMS}", delayedCallResult)
	mux.HandleFunc("GET /invokeRequestThrottledWithDelay/{delayMS}", throttleRequestWithDelay)
	mux.HandleFunc("PUT /setQueueLength/{newQueueLength}", setQueueLength)
	mux.HandleFunc("PUT /setRate429Errors/{newErrorRate}", setRate429Errors)
	// mux.HandleFunc("GET /metrics", promhttp.Handler())
	mux.Handle("GET /metrics", promhttp.Handler())

	log.Info("Starting http Server")
	err := http.ListenAndServe(":5050", mux)
	if err != nil {
		log.Fatalf("Error starting http server: %v", err)
	}

}

func setQueueLength(w http.ResponseWriter, r *http.Request) {
	qlen, err := strconv.Atoi(r.PathValue("newQueueLength"))
	if err != nil {
		log.Warn("could not read new queuelength from request")
	}

	recordQueueLength(float64(qlen))
	log.Infof("queue length set to: %d \n", qlen)
}

func setRate429Errors(w http.ResponseWriter, r *http.Request) {
	errRate, err := strconv.Atoi(r.PathValue("newErrorRate"))
	if err != nil {
		log.Warn("could not read new error Rate from request")
	}

	recordRate429Errors(float64(errRate))
	log.Infof("429 Error rate set to: %d \n", errRate)
}

func callResult(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("callResult returned"))
	log.Info("callResult called and returned..")
}

func delayedCallResult(w http.ResponseWriter, r *http.Request) {
	delayMS, err := strconv.Atoi(r.PathValue("delayMS"))
	if err != nil {
		delayMS = 0
		log.Warn("Incorrect response delay value specified, defaulting to 0")
	}
	time.Sleep(time.Millisecond * time.Duration(delayMS))
	w.Write([]byte("delayedCallResult returned"))
	log.Info("delayedCallResult called and returned...")

}

func throttleRequestWithDelay(w http.ResponseWriter, r *http.Request) {
	delayMS, err := strconv.Atoi(r.PathValue("delayMS"))
	if err != nil {
		delayMS = 0
		log.Warn("Incorrect response delay value specified, defaulting to 0")
	}
	// response code 429 is used to indicate that the user has sent too many requests in a given amount of time
	w.WriteHeader(http.StatusTooManyRequests)
	time.Sleep(time.Millisecond * time.Duration(delayMS))
	w.Write([]byte("throttleRequestWithDelay returned"))
	log.Info("throttleRequestWithDelay called and returned...")
}
