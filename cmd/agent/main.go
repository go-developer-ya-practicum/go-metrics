package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"

	"github.com/hikjik/go-musthave-devops-tpl.git/internal/config"
	"github.com/hikjik/go-musthave-devops-tpl.git/internal/metrics"
	"github.com/hikjik/go-musthave-devops-tpl.git/internal/scheduler"
)

type agent struct {
	collector *metrics.Collector
	key       string
	address   string
}

func (a *agent) sendMetrics() {
	collection := a.collector.ListMetrics()
	if a.key != "" {
		for _, metric := range collection {
			if err := metrics.Sign(metric, a.key); err != nil {
				log.Warnf("Failed to set hash: %v", err)
			}
		}
	}

	data, err := json.Marshal(collection)
	if err != nil {
		log.Warnf("Failed to marshal metrics")
		return
	}
	url := fmt.Sprintf("http://%s/updates/", a.address)
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

	collector := metrics.NewCollector()
	a := &agent{
		collector: collector,
		key:       cfg.Key,
		address:   cfg.Address,
	}

	s := scheduler.New()
	defer s.Stop()
	s.Add(collector.UpdateRuntimeMetrics, cfg.PollInterval)
	s.Add(collector.UpdateUtilizationMetrics, cfg.PollInterval)
	s.Add(a.sendMetrics, cfg.ReportInterval)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	<-sig
}
