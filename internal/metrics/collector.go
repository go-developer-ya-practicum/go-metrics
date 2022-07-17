package metrics

import (
	"fmt"
	"math/rand"
	"runtime"
	"sync"

	"github.com/rs/zerolog/log"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

// Collector собирает различные рантайм-метрики для их последующей отправки на сервер по протоколу HTTP.
// В качестве источника метрик используются пакеты runtime и gopsutil.
type Collector struct {
	muRuntime      sync.RWMutex
	PollCount      int64
	RuntimeMetrics map[string]float64

	muUtilize          sync.RWMutex
	UtilizationMetrics map[string]float64
}

// NewCollector создает экземпляр Collector
func NewCollector() *Collector {
	return &Collector{
		PollCount:      0,
		RuntimeMetrics: make(map[string]float64),

		UtilizationMetrics: make(map[string]float64),
	}
}

// UpdateRuntimeMetrics обновляет значения метрик RuntimeMetrics, используя пакет runtime,
// а также счетчика PollCount и метрики RandomValue.
func (c *Collector) UpdateRuntimeMetrics() {
	c.muRuntime.Lock()
	defer c.muRuntime.Unlock()

	c.PollCount += 1

	memStats := &runtime.MemStats{}
	runtime.ReadMemStats(memStats)
	c.RuntimeMetrics = map[string]float64{
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

// UpdateUtilizationMetrics обновляет значения метрик UtilizationMetrics, используя пакет gopsutil,
func (c *Collector) UpdateUtilizationMetrics() {
	c.muUtilize.Lock()
	defer c.muUtilize.Unlock()

	v, err := mem.VirtualMemory()
	if err != nil {
		log.Warn().Err(err).Msg("Failed to get memory stats")
		return
	}

	c.UtilizationMetrics = map[string]float64{
		"TotalMemory": float64(v.Total),
		"FreeMemory":  float64(v.Free),
	}

	usage, err := cpu.Percent(0, true)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to get cpu stats")
		return
	}
	for i, value := range usage {
		id := fmt.Sprintf("CPUutilization%d", i)
		c.UtilizationMetrics[id] = value
	}
}

// ListMetrics возвращает список всех собранных значений метрик.
func (c *Collector) ListMetrics() []*Metric {
	metrics := make([]*Metric, 0)

	c.muRuntime.RLock()
	metrics = append(metrics, NewCounter("PollCount", c.PollCount))
	for id, value := range c.RuntimeMetrics {
		metrics = append(metrics, NewGauge(id, value))
	}
	c.muRuntime.RUnlock()

	c.muUtilize.RLock()
	for id, value := range c.UtilizationMetrics {
		metrics = append(metrics, NewGauge(id, value))
	}
	c.muUtilize.RUnlock()

	return metrics
}
