package metrics

import (
	"github.com/openlyinc/pointy"
)

const (
	GaugeType   string = "gauge"
	CounterType string = "counter"
)

type Metric struct {
	ID    string   `json:"id"`
	MType string   `json:"type"`
	Delta *int64   `json:"delta,omitempty"`
	Value *float64 `json:"value,omitempty"`
	Hash  string   `json:"hash,omitempty"`
}

func NewGauge(id string, value float64) *Metric {
	return &Metric{
		ID:    id,
		MType: GaugeType,
		Value: pointy.Float64(value),
	}
}

func NewCounter(id string, value int64) *Metric {
	return &Metric{
		ID:    id,
		MType: CounterType,
		Delta: pointy.Int64(value),
	}
}
