// Package http предназначен для отправки метрик на сервер по протоколу http
package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/rs/zerolog/log"

	"github.com/hikjik/go-metrics/internal/encryption"
	"github.com/hikjik/go-metrics/internal/encryption/rsa"
	"github.com/hikjik/go-metrics/internal/metrics"
)

type Sender struct {
	Encrypter encryption.Encrypter
	Address   string
}

func New(address string, keyPath string) *Sender {
	encrypter, err := rsa.NewEncrypter(keyPath)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to setup rsa encryption")
	}

	return &Sender{
		Address:   address,
		Encrypter: encrypter,
	}
}

func (s *Sender) Send(ctx context.Context, collection []*metrics.Metric) {
	data, err := json.Marshal(collection)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to marshal metrics")
		return
	}

	encryptedData, err := s.encryptData(data)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to encrypt metrics data")
		return
	}

	client := http.Client{
		Transport: CustomTransport{http.DefaultTransport},
	}
	url := fmt.Sprintf("http://%s/updates/", s.Address)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(encryptedData))
	if err != nil {
		log.Warn().Err(err).Msgf("Failed to create req")
		return
	}
	req.Header.Set("Content-Type", "application/json")

	response, err := client.Do(req)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to post metric")
		return
	}
	if err = response.Body.Close(); err != nil {
		log.Warn().Err(err).Msg("Failed to close response body")
	}
}

func (s *Sender) encryptData(data []byte) ([]byte, error) {
	if s.Encrypter == nil {
		return data, nil
	}
	return s.Encrypter.Encrypt(data)
}
