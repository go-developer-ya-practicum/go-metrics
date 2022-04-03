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

	"github.com/caarlos0/env/v6"
	log "github.com/sirupsen/logrus"

	"github.com/hikjik/go-musthave-devops-tpl.git/internal/metrics"
	"github.com/hikjik/go-musthave-devops-tpl.git/internal/types"
)

type Config struct {
	Address        string        `env:"ADDRESS" envDefault:"127.0.0.1:8080"`
	PollInterval   time.Duration `env:"POLL_INTERVAL" envDefault:"2s"`
	ReportInterval time.Duration `env:"REPORT_INTERVAL" envDefault:"10s"`
}

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
	var config Config
	if err := env.Parse(&config); err != nil {
		log.Fatalf("Failed to parse agent config, %v", err)
	}

	runtimeMetrics := metrics.NewMetrics()

	reportTicker := time.NewTicker(config.ReportInterval)
	pollTicker := time.NewTicker(config.PollInterval)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)

	for {
		select {
		case <-pollTicker.C:
			runtimeMetrics.Update()
		case <-reportTicker.C:
			url := fmt.Sprintf("http://%s/update/", config.Address)
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
