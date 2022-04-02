package storage

import "sync"

type Storage struct {
	sync.RWMutex

	GaugeMetrics   map[string]float64
	CounterMetrics map[string]int64
}

func NewStorage() *Storage {
	return &Storage{
		GaugeMetrics:   make(map[string]float64),
		CounterMetrics: make(map[string]int64),
	}
}

func (storage *Storage) PutGauge(name string, value float64) {
	storage.Lock()
	defer storage.Unlock()
	storage.GaugeMetrics[name] = value
}

func (storage *Storage) UpdateCounter(name string, value int64) {
	storage.Lock()
	defer storage.Unlock()
	if storedValue, ok := storage.CounterMetrics[name]; ok {
		storage.CounterMetrics[name] = value + storedValue
	} else {
		storage.CounterMetrics[name] = value
	}
}

func (storage *Storage) GetGauge(name string) (value float64, ok bool) {
	storage.RLock()
	defer storage.RUnlock()
	value, ok = storage.GaugeMetrics[name]
	return
}

func (storage *Storage) GetCounter(name string) (value int64, ok bool) {
	storage.RLock()
	defer storage.RUnlock()
	value, ok = storage.CounterMetrics[name]
	return
}
