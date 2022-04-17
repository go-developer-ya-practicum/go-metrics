package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/hikjik/go-musthave-devops-tpl.git/internal/config"
	"github.com/hikjik/go-musthave-devops-tpl.git/internal/metrics"
	"github.com/hikjik/go-musthave-devops-tpl.git/internal/types"
)

func postMetric(url string, metric types.Metrics) {
	data, err := json.Marshal(metric)
	if err != nil {
		log.Warnf("Failed to marshal metric")
		return
	}

	response, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		log.Warnf("Failed to post metric: %v", err)
		return
	}
	if err := response.Body.Close(); err != nil {
		log.Warnf("Failed to close response body: %v", err)
	}
}

func main() {
	cfg := config.GetAgentConfig()

	runtimeMetrics := metrics.NewMetrics()

	reportTicker := time.NewTicker(cfg.ReportInterval)
	pollTicker := time.NewTicker(cfg.PollInterval)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)

	for {
		select {
		case <-pollTicker.C:
			runtimeMetrics.Update()
		case <-reportTicker.C:
			url := fmt.Sprintf("http://%s/update/", cfg.Address)
			for name, value := range runtimeMetrics.CounterMetrics {
				metric := types.Metrics{
					ID:    name,
					MType: "counter",
					Delta: &value,
				}
				postMetric(url, metric)
			}

			for name, value := range runtimeMetrics.GaugeMetrics {
				metric := types.Metrics{
					ID:    name,
					MType: "gauge",
					Value: &value,
				}
				postMetric(url, metric)
			}
		case <-sig:
			return
		}
	}
}
