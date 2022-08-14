// Package http содержит реализацию сервера по сбору метрик,
// принимающего данные от агентов по протоколу HTTP.
package http

import (
	"context"
	"errors"
	"net/http"

	"github.com/rs/zerolog/log"

	"github.com/hikjik/go-metrics/internal/config"
	"github.com/hikjik/go-metrics/internal/encryption"
	"github.com/hikjik/go-metrics/internal/encryption/rsa"
	"github.com/hikjik/go-metrics/internal/metrics"
	"github.com/hikjik/go-metrics/internal/storage"
)

type Server struct {
	Storage       storage.Storage
	Signer        metrics.Signer
	Decrypter     encryption.Decrypter
	TrustedSubnet string
	Address       string
}

func NewServer(cfg config.ServerConfig) *Server {
	store, err := storage.New(context.Background(), cfg.StorageConfig)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create storage")
	}

	decrypter, err := rsa.NewDecrypter(cfg.EncryptionKeyPath)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to setup rsa decryption")
	}

	signer := metrics.NewHMACSigner(cfg.SignatureKey)

	return &Server{
		Storage:       store,
		Signer:        signer,
		Decrypter:     decrypter,
		TrustedSubnet: cfg.TrustedSubnet,
		Address:       cfg.Address,
	}
}

func (s *Server) Run(ctx context.Context) {
	srv := &http.Server{
		Addr:    s.Address,
		Handler: s.Route(),
	}

	go func() {
		<-ctx.Done()
		if err := srv.Shutdown(context.Background()); err != nil {
			log.Error().Err(err).Msg("Failed to shutdown HTTP server")
		}
	}()
	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		log.Error().Err(err).Msg("Error on http server ListenAndServe")
	}
}
