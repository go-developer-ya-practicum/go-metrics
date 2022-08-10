// Package metrics предоставляет функционал по сбору рантайм-метрик.
package metrics

import (
	"github.com/openlyinc/pointy"
)

// Возможные типы метрик
const (
	GaugeType   string = "gauge"
	CounterType string = "counter"
)

// Metric содержит информацию о метрике
type Metric struct {
	ID    string   `json:"id"`
	MType string   `json:"type"`
	Delta *int64   `json:"delta,omitempty"`
	Value *float64 `json:"value,omitempty"`
	Hash  string   `json:"hash,omitempty"`
}

// NewGauge создает метрику типа GaugeType
func NewGauge(id string, value float64) *Metric {
	return &Metric{
		ID:    id,
		MType: GaugeType,
		Value: pointy.Float64(value),
	}
}

// NewCounter создает метрику типа CounterType
func NewCounter(id string, delta int64) *Metric {
	return &Metric{
		ID:    id,
		MType: CounterType,
		Delta: pointy.Int64(delta),
	}
}
