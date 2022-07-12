package server

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/hikjik/go-metrics/internal/metrics"
	"github.com/hikjik/go-metrics/internal/storage"
)

// NewRouter регистрирует обработчики и возвращает роутер chi.Mux
func NewRouter(storage storage.Storage, key string) *chi.Mux {
	srv := &Server{
		Storage: storage,
		Signer:  metrics.NewSigner(key),
	}

	router := chi.NewRouter()
	router.Use(middleware.Compress(5))
	router.Mount("/debug", middleware.Profiler())
	router.Get("/ping", srv.PingDatabase())
	router.Get("/", srv.GetAllMetrics())
	router.Get("/value/{metricType}/{metricName}", srv.GetMetric())
	router.Post("/update/{metricType}/{metricName}/{metricValue}", srv.PutMetric())
	router.Post("/update/", srv.PutMetricJSON())
	router.Post("/updates/", srv.PutMetricBatchJSON())
	router.Post("/value/", srv.GetMetricJSON())
	return router
}
