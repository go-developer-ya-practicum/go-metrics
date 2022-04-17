package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/hikjik/go-musthave-devops-tpl.git/internal/config"
	"github.com/hikjik/go-musthave-devops-tpl.git/internal/handlers"
	"github.com/hikjik/go-musthave-devops-tpl.git/internal/storage"
)

func main() {
	cfg := config.GetServerConfig()

	metricsStorage := storage.NewStorage()
	if cfg.Restore {
		if err := metricsStorage.Load(cfg.StoreFile); err != nil {
			log.Warnf("Failed to load metrics storage: %v", err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func(ctx context.Context) {
		storeTicker := time.NewTicker(cfg.StoreInterval)
		for {
			select {
			case <-storeTicker.C:
				if err := metricsStorage.Dump(cfg.StoreFile); err != nil {
					log.Warnf("Failed to dump metrics storage: %v", err)
				} else {
					log.Infoln("Dump server metrics")
				}
			case <-ctx.Done():
				return
			}
		}
	}(ctx)

	server := &http.Server{
		Addr:    cfg.Address,
		Handler: handlers.NewHandler(metricsStorage, cfg.Key),
	}

	idle := make(chan struct{})
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGQUIT, syscall.SIGINT, syscall.SIGTERM)
		<-sig

		if err := server.Shutdown(context.Background()); err != nil {
			log.Errorf("Failed to shutdown HTTP server: %v", err)
		}
		close(idle)
	}()
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Errorf("HTTP server ListenAndServe: %v", err)
	}
	<-idle
}
