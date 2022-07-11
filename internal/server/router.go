package server

import (
	"github.com/go-chi/chi/v5"

	"github.com/hikjik/go-metrics/internal/middleware"
	"github.com/hikjik/go-metrics/internal/storage"
)

// NewRouter регистрирует обработчики и возвращает роутер chi.Mux
func NewRouter(storage storage.Storage, key string) *chi.Mux {
	srv := &server{
		Storage: storage,
		Key:     key,
	}

	router := chi.NewRouter()
	router.Use(middleware.GZIPHandle)
	router.Get("/ping", srv.PingDatabase())
	router.Get("/", srv.GetAllMetrics())
	router.Get("/value/{metricType}/{metricName}", srv.GetMetric())
	router.Post("/update/{metricType}/{metricName}/{metricValue}", srv.PutMetric())
	router.Post("/update/", srv.PutMetricJSON())
	router.Post("/updates/", srv.PutMetricBatchJSON())
	router.Post("/value/", srv.GetMetricJSON())
	return router
}
