// Package agent содержит реализацию агента,
// собирающего и отправляющего на сервер значения метрик
// с заданной периодичностью
package agent

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/hikjik/go-metrics/internal/agent/sender"
	"github.com/hikjik/go-metrics/internal/agent/sender/grpc"
	"github.com/hikjik/go-metrics/internal/agent/sender/http"
	"github.com/hikjik/go-metrics/internal/config"
	"github.com/hikjik/go-metrics/internal/metrics"
	"github.com/hikjik/go-metrics/internal/scheduler"
)

type Agent struct {
	collector      *metrics.Collector
	signer         metrics.Signer
	sender         sender.MetricSender
	pollInterval   time.Duration
	reportInterval time.Duration
}

func New(cfg config.AgentConfig) *Agent {
	agent := &Agent{
		collector:      metrics.NewCollector(),
		signer:         metrics.NewHMACSigner(cfg.SignatureKey),
		sender:         http.New(cfg.Address, cfg.PublicKeyPath),
		pollInterval:   cfg.PollInterval,
		reportInterval: cfg.ReportInterval,
	}
	if cfg.GRPCAddress != "" {
		agent.sender = grpc.New(cfg.GRPCAddress)
	} else {
		agent.sender = http.New(cfg.Address, cfg.PublicKeyPath)
	}
	return agent
}

func (a *Agent) Run(ctx context.Context) {
	s := scheduler.New()

	s.Add(ctx, a.collector.UpdateRuntimeMetrics, a.pollInterval)
	s.Add(ctx, a.collector.UpdateUtilizationMetrics, a.pollInterval)
	s.Add(ctx, a.sendMetrics(ctx), a.reportInterval)
}

func (a *Agent) sendMetrics(ctx context.Context) func() {
	return func() {
		collection := a.collector.ListMetrics()
		for _, metric := range collection {
			if err := a.signer.Sign(metric); err != nil {
				log.Warn().Err(err).Msg("Failed to set hash")
			}
		}
		a.sender.Send(ctx, collection)
	}
}
