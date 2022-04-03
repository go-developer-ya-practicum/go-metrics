package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/caarlos0/env/v6"
	log "github.com/sirupsen/logrus"

	"github.com/hikjik/go-musthave-devops-tpl.git/internal/handlers"
	"github.com/hikjik/go-musthave-devops-tpl.git/internal/storage"
)

type Config struct {
	Address       string        `env:"ADDRESS"`
	StoreFile     string        `env:"STORE_FILE"`
	StoreInterval time.Duration `env:"STORE_INTERVAL"`
	Restore       bool          `env:"RESTORE"`
}

func main() {
	var config Config
	flag.StringVar(&config.Address, "a", "127.0.0.1:8080", "Server Address")
	flag.StringVar(&config.StoreFile, "f", "/tmp/devops-metrics-db.json", "Store File")
	flag.DurationVar(&config.StoreInterval, "i", time.Second*300, "Store Interval")
	flag.BoolVar(&config.Restore, "r", true, "Restore After Start")
	flag.Parse()

	if err := env.Parse(&config); err != nil {
		log.Fatalf("Failed to parse server config, %v", err)
	}

	metricsStorage := storage.NewStorage()
	if config.Restore {
		if err := metricsStorage.Load(config.StoreFile); err != nil {
			log.Warnf("Failed to load metrics storage: %v", err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func(ctx context.Context) {
		storeTicker := time.NewTicker(config.StoreInterval)
		for {
			select {
			case <-storeTicker.C:
				if err := metricsStorage.Dump(config.StoreFile); err != nil {
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
		Addr:    config.Address,
		Handler: handlers.NewHandler(metricsStorage),
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
