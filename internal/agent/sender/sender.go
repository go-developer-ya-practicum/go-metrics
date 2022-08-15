// Package sender содержит интерфейс для отправки метрик на сервер
package sender

import (
	"context"

	"github.com/hikjik/go-metrics/internal/metrics"
)

type MetricSender interface {
	Send(context.Context, []*metrics.Metric)
}
