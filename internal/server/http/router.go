package http

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Route регистрирует обработчики и возвращает роутер
func (s *Server) Route() *chi.Mux {
	router := chi.NewRouter()
	router.Use(middleware.Compress(5))
	router.Use(middleware.RealIP)
	router.Use(FilterIP(s.TrustedSubnet))
	router.Mount("/debug", middleware.Profiler())
	router.Get("/ping", s.PingDatabase())
	router.Get("/", s.GetAllMetrics())
	router.Get("/value/{metricType}/{metricName}", s.GetMetric())
	router.Post("/update/{metricType}/{metricName}/{metricValue}", s.PutMetric())
	router.Post("/update/", s.PutMetricJSON())
	router.Post("/updates/", s.PutMetricBatchJSON())
	router.Post("/value/", s.GetMetricJSON())
	return router
}
