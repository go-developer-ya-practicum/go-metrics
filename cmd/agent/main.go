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

	a, err := agent.New(config.GetAgentConfig())
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to setup agent")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	a.Run(ctx)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	<-sig
}
