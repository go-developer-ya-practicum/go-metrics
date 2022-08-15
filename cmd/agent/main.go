package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog/log"

	"github.com/hikjik/go-metrics/internal/agent"
	"github.com/hikjik/go-metrics/internal/config"
	"github.com/hikjik/go-metrics/internal/greeting"
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

	log.Info().Msg("Start agent")
	agent.New(config.GetAgentConfig()).Run(ctx)
	<-ctx.Done()
}
