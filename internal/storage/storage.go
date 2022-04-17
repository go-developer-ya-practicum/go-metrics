package storage

import (
	"context"

	"github.com/hikjik/go-musthave-devops-tpl.git/internal/config"
)

type Storage interface {
	PutGauge(name string, value float64)
	UpdateCounter(name string, value int64)
	GetGauge(name string) (value float64, ok bool)
	GetCounter(name string) (value int64, ok bool)
	GetGaugeMetrics() map[string]float64
	GetCounterMetrics() map[string]int64
}

func New(ctx context.Context, cfg config.StorageConfig) (Storage, error) {
	if cfg.DatabaseDNS != "" {
		return newDBStorage(ctx, cfg)
	}
	return newFileStorage(ctx, cfg)
}
