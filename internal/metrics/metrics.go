package metrics

import (
	"math/rand"
	"runtime"
)

type Metrics struct {
	GaugeMetrics   map[string]float64
	CounterMetrics map[string]int64
}

func NewMetrics() *Metrics {
	return &Metrics{
		GaugeMetrics: make(map[string]float64),
		CounterMetrics: map[string]int64{
			"PollCount": 0,
		},
	}
}

func (metrics *Metrics) Update() {
	for metricName := range metrics.CounterMetrics {
		metrics.CounterMetrics[metricName] += 1
	}

	memStats := &runtime.MemStats{}
	runtime.ReadMemStats(memStats)
	metrics.GaugeMetrics = map[string]float64{
		"RandomValue":   rand.Float64(),
		"GCCPUFraction": memStats.GCCPUFraction,
		"Alloc":         float64(memStats.Alloc),
		"BuckHashSys":   float64(memStats.BuckHashSys),
		"Frees":         float64(memStats.Frees),
		"GCSys":         float64(memStats.GCSys),
		"HeapAlloc":     float64(memStats.HeapAlloc),
		"HeapIdle":      float64(memStats.HeapIdle),
		"HeapInuse":     float64(memStats.HeapInuse),
		"HeapObjects":   float64(memStats.HeapObjects),
		"HeapReleased":  float64(memStats.HeapReleased),
		"HeapSys":       float64(memStats.HeapSys),
		"LastGC":        float64(memStats.LastGC),
		"Lookups":       float64(memStats.Lookups),
		"MCacheInuse":   float64(memStats.MCacheInuse),
		"MCacheSys":     float64(memStats.MCacheSys),
		"MSpanInuse":    float64(memStats.MSpanInuse),
		"MSpanSys":      float64(memStats.MSpanSys),
		"Mallocs":       float64(memStats.Mallocs),
		"NextGC":        float64(memStats.NextGC),
		"NumForcedGC":   float64(memStats.NumForcedGC),
		"NumGC":         float64(memStats.NumGC),
		"OtherSys":      float64(memStats.OtherSys),
		"PauseTotalNs":  float64(memStats.PauseTotalNs),
		"StackInuse":    float64(memStats.StackInuse),
		"StackSys":      float64(memStats.StackSys),
		"Sys":           float64(memStats.Sys),
		"TotalAlloc":    float64(memStats.TotalAlloc),
	}
}
