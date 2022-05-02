package storage

import (
	"context"
	"encoding/json"
	"os"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/hikjik/go-musthave-devops-tpl.git/internal/config"
)

type FileStorage struct {
	sync.RWMutex

	GaugeMetrics   map[string]float64
	CounterMetrics map[string]int64
}

func newFileStorage(ctx context.Context, cfg config.StorageConfig) (Storage, error) {
	storage := &FileStorage{
		GaugeMetrics:   make(map[string]float64),
		CounterMetrics: make(map[string]int64),
	}

	if cfg.Restore {
		if err := storage.load(cfg.StoreFile); err != nil {
			log.Warnf("Failed to load metrics storage: %v", err)
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
					log.Warnf("Failed to dump metrics storage: %v", err)
				} else {
					log.Infoln("Dump server metrics")
				}
			}
		}
	}()

	return storage, nil
}

func (s *FileStorage) PutGauge(name string, value float64) {
	s.Lock()
	defer s.Unlock()
	s.GaugeMetrics[name] = value
}

func (s *FileStorage) UpdateCounter(name string, value int64) {
	s.Lock()
	defer s.Unlock()
	if storedValue, ok := s.CounterMetrics[name]; ok {
		s.CounterMetrics[name] = value + storedValue
	} else {
		s.CounterMetrics[name] = value
	}
}

func (s *FileStorage) GetGauge(name string) (value float64, ok bool) {
	s.RLock()
	defer s.RUnlock()
	value, ok = s.GaugeMetrics[name]
	return
}

func (s *FileStorage) GetCounter(name string) (value int64, ok bool) {
	s.RLock()
	defer s.RUnlock()
	value, ok = s.CounterMetrics[name]
	return
}

func (s *FileStorage) GetGaugeMetrics() map[string]float64 {
	s.RLock()
	defer s.RUnlock()
	return s.GaugeMetrics
}

func (s *FileStorage) GetCounterMetrics() map[string]int64 {
	s.RLock()
	defer s.RUnlock()
	return s.CounterMetrics
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
