package main

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/rs/zerolog/log"

	"github.com/hikjik/go-metrics/internal/config"
	"github.com/hikjik/go-metrics/internal/greeting"
	"github.com/hikjik/go-metrics/internal/server/grpc"
	"github.com/hikjik/go-metrics/internal/server/http"
)

var (
	buildVersion string
	buildDate    string
	buildCommit  string
)

func main() {
	if err := greeting.PrintBuildInfo(os.Stdout, buildVersion, buildDate, buildCommit); err != nil {
		log.Warn().Err(err).Msg("Failed to print build info")
	}

	ctx, cancel := signal.NotifyContext(
		context.Background(), syscall.SIGQUIT, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	cfg := config.GetServerConfig()

	var wg sync.WaitGroup

	log.Info().Msgf("Start http server: %s", cfg.Address)
	wg.Add(1)
	go func() {
		defer wg.Done()
		http.NewServer(cfg).Run(ctx)
	}()

	if cfg.GRPCAddress != "" {
		log.Info().Msgf("Start grpc server: %s", cfg.GRPCAddress)
		wg.Add(1)
		go func() {
			defer wg.Done()
			grpc.NewServer(cfg).Run(ctx)
		}()
	}

	wg.Wait()
}
