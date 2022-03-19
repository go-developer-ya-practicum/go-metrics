package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/hikjik/go-musthave-devops-tpl.git/internal/metrics"
)

const (
	address        = "127.0.0.1:8080"
	pollInterval   = 2 * time.Second
	reportInterval = 10 * time.Second
)

func postMetric(url string) {
	response, err := http.Post(url, "text/plain", nil)
	if err != nil {
		log.Warnf("Failed to post metric: %v", err)
	}
	if err := response.Body.Close(); err != nil {
		log.Warnf("Failed to close response body: %v", err)
	}
}

func main() {
	runtimeMetrics := metrics.NewMetrics()

	reportTicker := time.NewTicker(reportInterval)
	pollTicker := time.NewTicker(pollInterval)
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)

	for {
		select {
		case <-pollTicker.C:
			runtimeMetrics.Update()
		case <-reportTicker.C:
			url := fmt.Sprintf("http://%s/update/%s/%s/%d", address, "counter", "PollCount", runtimeMetrics.PollCount)
			postMetric(url)

			for name, value := range runtimeMetrics.GaugeMetrics {
				url := fmt.Sprintf("http://%s/update/%s/%s/%f", address, "gauge", name, value)
				postMetric(url)
			}
		case <-sig:
			return
		}
	}
}
