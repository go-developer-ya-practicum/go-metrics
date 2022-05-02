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
			go collector.UpdateRuntimeMetrics()
			go collector.UpdateUtilizationMetrics()
		case <-reportTicker.C:
			go func() {
				collection := collector.ListMetrics()
				if cfg.Key != "" {
					for _, metric := range collection {
						if err := metrics.Sign(metric, cfg.Key); err != nil {
							log.Warnf("Failed to set hash: %v", err)
						}
					}
				}

				data, err := json.Marshal(collection)
				if err != nil {
					log.Warnf("Failed to marshal metrics")
					return
				}
				url := fmt.Sprintf("http://%s/updates/", cfg.Address)
				response, err := http.Post(url, "application/json", bytes.NewBuffer(data))
				if err != nil {
					log.Warnf("Failed to post metric: %v", err)
					return
				}
				if err := response.Body.Close(); err != nil {
					log.Warnf("Failed to close response body: %v", err)
				}

			}()
		case <-sig:
			return
		}
	}
}
