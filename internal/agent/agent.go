// Package agent содержит реализацию агента,
// собирающего и отправляющего на сервер значения рантайм метрик
package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/hikjik/go-metrics/internal/config"
	"github.com/hikjik/go-metrics/internal/encryption"
	"github.com/hikjik/go-metrics/internal/encryption/rsa"
	"github.com/hikjik/go-metrics/internal/metrics"
	"github.com/hikjik/go-metrics/internal/scheduler"
)

type Agent struct {
	collector      *metrics.Collector
	signer         metrics.Signer
	encrypter      encryption.Encrypter
	address        string
	pollInterval   time.Duration
	reportInterval time.Duration
}

func New(cfg config.AgentConfig) (*Agent, error) {
	collector := metrics.NewCollector()

	encrypter, err := rsa.NewEncrypter(cfg.PublicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to setup rsa encryption")
	}

	signer := metrics.NewHMACSigner(cfg.SignatureKey)

	return &Agent{
		collector:      collector,
		signer:         signer,
		encrypter:      encrypter,
		address:        cfg.Address,
		pollInterval:   cfg.PollInterval,
		reportInterval: cfg.ReportInterval,
	}, nil
}

func (a *Agent) Run(ctx context.Context) {
	s := scheduler.New()

	s.Add(ctx, a.collector.UpdateRuntimeMetrics, a.pollInterval)
	s.Add(ctx, a.collector.UpdateUtilizationMetrics, a.pollInterval)
	s.Add(ctx, a.sendMetrics, a.reportInterval)
}

func (a *Agent) sendMetrics() {
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

	encryptedData, err := a.encryptData(data)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to encrypt metrics data")
		return
	}

	url := fmt.Sprintf("http://%s/updates/", a.address)

	client := http.Client{
		Transport: CustomTransport{http.DefaultTransport},
	}
	response, err := client.Post(url, "application/json", bytes.NewBuffer(encryptedData))
	if err != nil {
		log.Warn().Err(err).Msg("Failed to post metric")
		return
	}
	if err = response.Body.Close(); err != nil {
		log.Warn().Err(err).Msg("Failed to close response body")
	}
}

func (a *Agent) encryptData(data []byte) ([]byte, error) {
	if a.encrypter == nil {
		return data, nil
	}
	return a.encrypter.Encrypt(data)
}
