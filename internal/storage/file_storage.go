package storage

import (
	"context"
	"encoding/json"
	"os"
	"sync"
	"time"

	"github.com/openlyinc/pointy"
	"github.com/rs/zerolog/log"

	"github.com/hikjik/go-metrics/internal/config"
	"github.com/hikjik/go-metrics/internal/metrics"
)

type FileStorage struct {
	Floats   map[string]float64
	Integers map[string]int64
	sync.RWMutex
}

func newFileStorage(ctx context.Context, cfg config.StorageConfig) (Storage, error) {
	storage := &FileStorage{
		Floats:   make(map[string]float64),
		Integers: make(map[string]int64),
	}

	if cfg.Restore {
		if err := storage.load(cfg.StoreFile); err != nil {
			log.Warn().Err(err).Msg("Failed to load metrics storage")
		}
	}

	go func() {
		storeTicker := time.NewTicker(cfg.StoreInterval)
		for {
			select {
			case <-ctx.Done():
				return
			case <-storeTicker.C:
				if err := storage.dump(cfg.StoreFile); err != nil {
					log.Warn().Err(err).Msg("Failed to dump metrics storage")
				} else {
					log.Info().Msg("Dump server metrics")
				}
			}
		}
	}()

	return storage, nil
}

func (s *FileStorage) Put(_ context.Context, metric *metrics.Metric) error {
	s.Lock()
	defer s.Unlock()

	switch metric.MType {
	case metrics.GaugeType:
		if metric.Value == nil {
			return ErrBadArgument
		}
		s.Floats[metric.ID] = *metric.Value
	case metrics.CounterType:
		if metric.Delta == nil {
			return ErrBadArgument
		}
		s.Integers[metric.ID] += *metric.Delta
	default:
		return ErrUnknownMetricType
	}
	return nil
}

func (s *FileStorage) Get(_ context.Context, metric *metrics.Metric) error {
	s.RLock()
	defer s.RUnlock()

	switch metric.MType {
	case metrics.GaugeType:
		value, ok := s.Floats[metric.ID]
		if !ok {
			return ErrNotFound
		}
		metric.Value = pointy.Float64(value)
	case metrics.CounterType:
		delta, ok := s.Integers[metric.ID]
		if !ok {
			return ErrNotFound
		}
		metric.Delta = pointy.Int64(delta)
	default:
		return ErrUnknownMetricType
	}
	return nil
}

func (s *FileStorage) List(_ context.Context) ([]*metrics.Metric, error) {
	s.RLock()
	defer s.RUnlock()

	result := make([]*metrics.Metric, 0)
	for id, value := range s.Floats {
		result = append(result, metrics.NewGauge(id, value))
	}
	for id, delta := range s.Integers {
		result = append(result, metrics.NewCounter(id, delta))
	}
	return result, nil
}

func (s *FileStorage) dump(storeFile string) error {
	s.RLock()
	defer s.RUnlock()

	file, err := os.OpenFile(storeFile, os.O_WRONLY|os.O_CREATE, 0777)
	if err != nil {
		return err
	}

	return json.NewEncoder(file).Encode(s)
}

func (s *FileStorage) load(storeFile string) error {
	s.Lock()
	defer s.Unlock()

	file, err := os.OpenFile(storeFile, os.O_RDONLY|os.O_CREATE, 0777)
	if err != nil {
		return err
	}

	return json.NewDecoder(file).Decode(&s)
}
