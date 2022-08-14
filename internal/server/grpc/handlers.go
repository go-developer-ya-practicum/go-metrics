package grpc

import (
	"context"
	"errors"
	"io"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/hikjik/go-metrics/internal/proto"
	"github.com/hikjik/go-metrics/internal/storage"
)

func (s *Server) GetMetric(ctx context.Context, r *pb.GetMetricRequest) (*pb.GetMetricResponse, error) {
	metric := pb.FromPb(r.GetMetric())

	if err := s.Storage.Get(ctx, metric); err != nil {
		return nil, handleStorageError(err)
	}

	if s.Signer != nil {
		if err := s.Signer.Sign(metric); err != nil {
			log.Warn().Err(err).Msg("Failed to set hash")
			return nil, status.Error(codes.Internal, "Internal error: failed to set metric hash")
		}
	}

	return &pb.GetMetricResponse{
		Metric: pb.ToPb(metric),
	}, nil
}

func (s *Server) PutMetric(ctx context.Context, r *pb.PutMetricRequest) (*pb.PutMetricResponse, error) {
	metric := pb.FromPb(r.GetMetric())

	if s.Signer != nil {
		ok, err := s.Signer.Validate(metric)
		if err != nil {
			log.Warn().Err(err).Msg("Failed to validate hash")
			return nil, status.Error(codes.Internal, "Failed to validate hash")
		}
		if !ok {
			log.Info().Msgf("Invalid hash: %v", metric)
			return nil, status.Error(codes.InvalidArgument, "Invalid hash")
		}
	}

	if err := s.Storage.Put(ctx, metric); err != nil {
		return nil, handleStorageError(err)
	}

	return &pb.PutMetricResponse{}, nil
}

func (s *Server) PutMetrics(stream pb.Metrics_PutMetricsServer) error {
	for {
		message, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}

		metric := pb.FromPb(message.GetMetric())
		if s.Signer != nil {
			var ok bool
			ok, err = s.Signer.Validate(metric)
			if err != nil {
				log.Warn().Err(err).Msg("Failed to validate hash")
				return status.Error(codes.Internal, "Failed to validate hash")
			}
			if !ok {
				log.Info().Msgf("Invalid hash: %v", metric)
				return status.Error(codes.InvalidArgument, "Invalid hash")
			}
		}

		if err = s.Storage.Put(stream.Context(), metric); err != nil {
			return handleStorageError(err)
		}
	}
	return stream.SendAndClose(&pb.PutMetricResponse{})
}

func handleStorageError(err error) error {
	switch err {
	case storage.ErrUnknownMetricType:
		return status.Error(codes.Unimplemented, "Unknown metric type")
	case storage.ErrBadArgument:
		return status.Error(codes.InvalidArgument, "Invalid request args")
	case storage.ErrNotFound:
		return status.Error(codes.NotFound, "Metric not found")
	default:
		return status.Error(codes.Internal, "Internal storage error")
	}
}
