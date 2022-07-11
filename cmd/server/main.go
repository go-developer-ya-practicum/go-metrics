package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"

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
		log.Fatalf("Failed to create storage: %v", err)
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
			log.Errorf("Failed to shutdown HTTP server: %v", err)
		}
		close(idle)
	}()
	if err = srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Errorf("HTTP server ListenAndServe: %v", err)
	}
	<-idle
}
