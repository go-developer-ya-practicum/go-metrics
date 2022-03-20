package main

import (
	"context"
	"github.com/hikjik/go-musthave-devops-tpl.git/internal/handlers"
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

const address = "127.0.0.1:8080"

func main() {
	mux := http.NewServeMux()
	mux.Handle("/", handlers.NewHandler())
	server := &http.Server{
		Addr:    address,
		Handler: mux,
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
