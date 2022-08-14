// Package grpc содержит реализацию сервера по сбору метрик,
// принимающего данные от агентов по GRPC.
package grpc

import (
	"context"
	"net"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"

	"github.com/hikjik/go-metrics/internal/config"
	"github.com/hikjik/go-metrics/internal/metrics"
	pb "github.com/hikjik/go-metrics/internal/proto"
	"github.com/hikjik/go-metrics/internal/storage"
)

type Server struct {
	pb.UnimplementedMetricsServer

	Storage storage.Storage
	Signer  metrics.Signer
	Address string
}

var _ pb.MetricsServer = (*Server)(nil)

func NewServer(cfg config.ServerConfig) *Server {
	store, err := storage.New(context.Background(), cfg.StorageConfig)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create storage")
	}

	signer := metrics.NewHMACSigner(cfg.SignatureKey)

	return &Server{
		Storage: store,
		Signer:  signer,
		Address: cfg.GRPCAddress,
	}
}

func (s *Server) Run(ctx context.Context) {
	listen, err := net.Listen("tcp", s.Address)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to start grpc server")
	}
	grpcServer := grpc.NewServer()
	pb.RegisterMetricsServer(grpcServer, s)

	go func() {
		<-ctx.Done()
		grpcServer.GracefulStop()
	}()

	if err = grpcServer.Serve(listen); err != nil {
		log.Error().Err(err).Msg("Error on grpc server Serve")
	}
}
