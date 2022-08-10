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
	"github.com/hikjik/go-metrics/internal/encryption"
	"github.com/hikjik/go-metrics/internal/encryption/rsa"
	"github.com/hikjik/go-metrics/internal/greeting"
	"github.com/hikjik/go-metrics/internal/metrics"
	"github.com/hikjik/go-metrics/internal/scheduler"
)

var (
	buildVersion string
	buildDate    string
	buildCommit  string
)

type agent struct {
	collector *metrics.Collector
	signer    metrics.Signer
	encrypter encryption.Encrypter
	address   string
}

func (a *agent) encryptData(data []byte) ([]byte, error) {
	if a.encrypter == nil {
		return data, nil
	}
	return a.encrypter.Encrypt(data)
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

	encryptedData, err := a.encryptData(data)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to encrypt metrics data")
		return
	}

	url := fmt.Sprintf("http://%s/updates/", a.address)
	response, err := http.Post(url, "application/json", bytes.NewBuffer(encryptedData))
	if err != nil {
		log.Warn().Err(err).Msg("Failed to post metric")
		return
	}
	if err := response.Body.Close(); err != nil {
		log.Warn().Err(err).Msg("Failed to close response body")
	}
}

func main() {
	if err := greeting.PrintBuildInfo(os.Stdout, buildVersion, buildDate, buildCommit); err != nil {
		log.Warn().Err(err).Msg("Failed to print build info")
	}

	cfg := config.GetAgentConfig()

	collector := metrics.NewCollector()

	encrypter, err := rsa.NewEncrypter(cfg.PublicKeyPath)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to setup rsa encryption")
	}

	signer := metrics.NewHMACSigner(cfg.SignatureKey)

	a := &agent{
		collector: collector,
		signer:    signer,
		encrypter: encrypter,
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
