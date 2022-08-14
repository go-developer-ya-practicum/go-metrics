package proto

import (
	"github.com/rs/zerolog/log"

	"github.com/hikjik/go-metrics/internal/metrics"
)

func FromPb(pbMetric *Metric) *metrics.Metric {
	var metric *metrics.Metric
	switch pbMetric.Type {
	case Metric_COUNTER:
		metric = metrics.NewCounter(pbMetric.Id, pbMetric.Delta)
	case Metric_GAUGE:
		metric = metrics.NewGauge(pbMetric.Id, pbMetric.Value)
	default:
		log.Warn().Msgf("Unknown metric type: %v", pbMetric.Type)
	}
	metric.Hash = pbMetric.Hash
	return metric
}

func ToPb(metric *metrics.Metric) *Metric {
	pbMetric := Metric{
		Id:   metric.ID,
		Hash: metric.Hash,
	}
	switch metric.MType {
	case metrics.CounterType:
		pbMetric.Type = Metric_COUNTER
		pbMetric.Delta = *metric.Delta
	case metrics.GaugeType:
		pbMetric.Type = Metric_GAUGE
		pbMetric.Value = *metric.Value
	default:
		log.Warn().Msgf("Unknown metric type: %v", metric.MType)
	}
	return &pbMetric
}
