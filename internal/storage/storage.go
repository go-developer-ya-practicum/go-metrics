package storage

import (
	"context"
	"errors"

	"github.com/hikjik/go-metrics/internal/config"
	"github.com/hikjik/go-metrics/internal/metrics"
)

// Возможные ошибки при работе с хранилищем метрик
var (
	ErrNotFound          = errors.New("not found")
	ErrUnknownMetricType = errors.New("unknown metric type")
	ErrBadArgument       = errors.New("bad argument")
)

// Storage определяет интерфейс для хранения метрик
type Storage interface {
	// Put сохраняет значение метрики
	Put(ctx context.Context, metric *metrics.Metric) error

	// Get возвращает значение метрики
	Get(ctx context.Context, metric *metrics.Metric) error

	// List возвращает список всех сохраненных метрик
	List(ctx context.Context) ([]*metrics.Metric, error)
}

// New вовращает объект типа Storage
func New(ctx context.Context, cfg config.StorageConfig) (Storage, error) {
	if cfg.DatabaseDNS != "" {
		return newDBStorage(ctx, cfg)
	}
	return newFileStorage(ctx, cfg)
}
