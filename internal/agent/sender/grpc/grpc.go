// Package grpc предназначен для отправки метрик на сервер по grpc
package grpc

import (
	"context"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/hikjik/go-metrics/internal/metrics"
	pb "github.com/hikjik/go-metrics/internal/proto"
)

type Sender struct {
	Conn   *grpc.ClientConn
	Client pb.MetricsClient
}

func New(address string) *Sender {
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal().Err(err)
	}

	return &Sender{
		Conn:   conn,
		Client: pb.NewMetricsClient(conn),
	}
}

func (s *Sender) Send(ctx context.Context, collection []*metrics.Metric) {
	stream, err := s.Client.PutMetrics(ctx)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to open grpc stream")
		return
	}
	defer func() {
		if err = stream.CloseSend(); err != nil {
			log.Error().Err(err).Msg("Failed to close stream")
		}
	}()

	for _, metric := range collection {
		request := pb.PutMetricRequest{
			Metric: pb.ToPb(metric),
		}

		if err = stream.Send(&request); err != nil {
			log.Error().Err(err).Msgf("Failed to send metric %v", metric)
		}
	}
}
