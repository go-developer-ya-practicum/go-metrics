// Package server содержит реализацию сервера по сбору рантайм-метрик,
// принимающего данные от агентов по протоколу HTTP.
package server

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/hikjik/go-metrics/internal/encryption"
	"github.com/hikjik/go-metrics/internal/metrics"
	"github.com/hikjik/go-metrics/internal/storage"
)

// NewRouter регистрирует обработчики и возвращает роутер chi.Mux
func NewRouter(
	storage storage.Storage,
	decrypter encryption.Decrypter,
	signer metrics.Signer,
	trustedSubnet string,
) *chi.Mux {
	srv := &Server{
		Storage:   storage,
		Signer:    signer,
		Decrypter: decrypter,
	}

	router := chi.NewRouter()
	router.Use(middleware.Compress(5))
	router.Use(middleware.RealIP)
	router.Use(FilterIP(trustedSubnet))
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
