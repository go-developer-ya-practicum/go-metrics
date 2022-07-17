package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog/log"

	"github.com/hikjik/go-metrics/internal/config"
	"github.com/hikjik/go-metrics/internal/metrics"
	"github.com/hikjik/go-metrics/internal/scheduler"
)

type agent struct {
	collector *metrics.Collector
	signer    metrics.Signer
	address   string
}

func (a *agent) sendMetrics() {
	collection := a.collector.ListMetrics()
	for _, metric := range collection {
		if err := a.signer.Sign(metric); err != nil {
			log.Warn().Err(err).Msg("Failed to set hash")
		}
	}

	data, err := json.Marshal(collection)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to marshal metrics")
		return
	}
	url := fmt.Sprintf("http://%s/updates/", a.address)
	response, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		log.Warn().Err(err).Msg("Failed to post metric")
		return
	}
	if err := response.Body.Close(); err != nil {
		log.Warn().Err(err).Msg("Failed to close response body")
	}
}

func main() {
	cfg := config.GetAgentConfig()

	collector := metrics.NewCollector()
	a := &agent{
		collector: collector,
		signer:    metrics.NewHMACSigner(cfg.Key),
		address:   cfg.Address,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s := scheduler.New()
	s.Add(ctx, collector.UpdateRuntimeMetrics, cfg.PollInterval)
	s.Add(ctx, collector.UpdateUtilizationMetrics, cfg.PollInterval)
	s.Add(ctx, a.sendMetrics, cfg.ReportInterval)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	<-sig
}
