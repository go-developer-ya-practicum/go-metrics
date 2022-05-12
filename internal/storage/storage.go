package storage

import (
	"context"
	"errors"

	"github.com/hikjik/go-musthave-devops-tpl.git/internal/config"
	"github.com/hikjik/go-musthave-devops-tpl.git/internal/metrics"
)

var ErrNotFound = errors.New("not found")
var ErrUnknownMetricType = errors.New("unknown metric type")
var ErrBadArgument = errors.New("bad argument")

type Storage interface {
	Put(ctx context.Context, metric *metrics.Metric) error
	Get(ctx context.Context, metric *metrics.Metric) error
	List(ctx context.Context) ([]*metrics.Metric, error)
}

func New(ctx context.Context, cfg config.StorageConfig) (Storage, error) {
	if cfg.DatabaseDNS != "" {
		return newDBStorage(ctx, cfg)
	}
	return newFileStorage(ctx, cfg)
}
