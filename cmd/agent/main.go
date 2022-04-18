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
)

func main() {
	cfg := config.GetAgentConfig()

	collector := metrics.NewCollector()

	reportTicker := time.NewTicker(cfg.ReportInterval)
	pollTicker := time.NewTicker(cfg.PollInterval)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)

	for {
		select {
		case <-pollTicker.C:
			collector.Update()
		case <-reportTicker.C:
			metricsBatch := make([]metrics.Metrics, 0)
			for name, value := range collector.CounterMetrics {
				metric := metrics.Metrics{
					ID:    name,
					MType: "counter",
					Delta: new(int64),
				}
				*metric.Delta = value
				if cfg.Key != "" {
					if err := metric.SetHash(cfg.Key); err != nil {
						log.Warnf("Failed to set hash: %v", err)
					}
				}
				metricsBatch = append(metricsBatch, metric)
			}
			for name, value := range collector.GaugeMetrics {
				metric := metrics.Metrics{
					ID:    name,
					MType: "gauge",
					Value: new(float64),
				}
				*metric.Value = value
				if cfg.Key != "" {
					if err := metric.SetHash(cfg.Key); err != nil {
						log.Warnf("Failed to set hash: %v", err)
					}
				}
				metricsBatch = append(metricsBatch, metric)
			}

			data, err := json.Marshal(metricsBatch)
			if err != nil {
				log.Warnf("Failed to marshal metrics")
				continue
			}
			url := fmt.Sprintf("http://%s/updates/", cfg.Address)
			response, err := http.Post(url, "application/json", bytes.NewBuffer(data))
			if err != nil {
				log.Warnf("Failed to post metric: %v", err)
				continue
			}
			if err := response.Body.Close(); err != nil {
				log.Warnf("Failed to close response body: %v", err)
			}
		case <-sig:
			return
		}
	}
}
