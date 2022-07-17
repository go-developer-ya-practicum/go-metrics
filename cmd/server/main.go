package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog/log"

	"github.com/hikjik/go-metrics/internal/config"
	"github.com/hikjik/go-metrics/internal/server"
	"github.com/hikjik/go-metrics/internal/storage"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := config.GetServerConfig()

	metricsStorage, err := storage.New(ctx, cfg.StorageConfig)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create storage")
	}

	srv := &http.Server{
		Addr:    cfg.Address,
		Handler: server.NewRouter(metricsStorage, cfg.Key),
	}

	idle := make(chan struct{})
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGQUIT, syscall.SIGINT, syscall.SIGTERM)
		<-sig

		if err = srv.Shutdown(context.Background()); err != nil {
			log.Error().Err(err).Msg("Failed to shutdown HTTP server")
		}
		close(idle)
	}()
	if err = srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Error().Err(err).Msg("HTTP server ListenAndServe")
	}
	<-idle
}
