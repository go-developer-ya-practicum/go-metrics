package handlers

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	log "github.com/sirupsen/logrus"

	"github.com/hikjik/go-metrics/internal/metrics"
	"github.com/hikjik/go-metrics/internal/middleware"
	"github.com/hikjik/go-metrics/internal/storage"
)

//go:embed index.html
var fs embed.FS

type Handler struct {
	*chi.Mux
	Storage storage.Storage
	Key     string
}

func NewHandler(storage storage.Storage, key string) *Handler {
	h := &Handler{
		Mux:     chi.NewMux(),
		Storage: storage,
		Key:     key,
	}
	h.Use(middleware.GZIPHandle)
	h.Get("/ping", h.PingDatabase())
	h.Get("/", h.GetAllMetrics())
	h.Get("/value/{metricType}/{metricName}", h.GetMetric())
	h.Post("/update/{metricType}/{metricName}/{metricValue}", h.PutMetric())
	h.Post("/update/", h.PutMetricJSON())
	h.Post("/updates/", h.PutMetricBatchJSON())
	h.Post("/value/", h.GetMetricJSON())
	return h
}

func (h *Handler) PingDatabase() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s, ok := h.Storage.(*storage.DBStorage)
		if !ok {
			log.Warnln("Failed to connect to db")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if err := s.Ping(r.Context()); err != nil {
			log.Warnf("Failed to ping db: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

func (h *Handler) GetAllMetrics() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		t, err := template.ParseFS(fs, "index.html")
		if err != nil {
			log.Warnln("Failed to parse index.html")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		m, err := h.Storage.List(r.Context())
		if err != nil {
			log.Warnf("Failed to list metrics: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if err = t.Execute(w, m); err != nil {
			log.Warnf("Failed to execute template: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func (h *Handler) GetMetric() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m := &metrics.Metric{
			ID:    chi.URLParam(r, "metricName"),
			MType: chi.URLParam(r, "metricType"),
		}

		if err := h.Storage.Get(r.Context(), m); err != nil {
			handleStorageError(w, err)
			return
		}

		var str string
		switch m.MType {
		case metrics.GaugeType:
			str = strconv.FormatFloat(*m.Value, 'f', -1, 64)
		case metrics.CounterType:
			str = fmt.Sprintf("%d", *m.Delta)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(str))
	}
}

func (h *Handler) GetMetricJSON() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Header.Get("Content-Type") != "application/json" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		var m metrics.Metric
		err = json.Unmarshal(body, &m)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if err = h.Storage.Get(r.Context(), &m); err != nil {
			handleStorageError(w, err)
			return
		}

		if h.Key != "" {
			if err = metrics.Sign(&m, h.Key); err != nil {
				log.Warnf("Failed to set hash: %v", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}

		if err = json.NewEncoder(w).Encode(m); err != nil {
			log.Warnf("Failed to encode metric: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

func (h *Handler) PutMetric() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		metricValue := chi.URLParam(r, "metricValue")
		metricName := chi.URLParam(r, "metricName")
		metricType := chi.URLParam(r, "metricType")

		var m *metrics.Metric
		switch metricType {
		case metrics.GaugeType:
			value, err := strconv.ParseFloat(metricValue, 64)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			m = metrics.NewGauge(metricName, value)
		case metrics.CounterType:
			delta, err := strconv.ParseInt(metricValue, 10, 64)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			m = metrics.NewCounter(metricName, delta)
		default:
			w.WriteHeader(http.StatusNotImplemented)
			return
		}

		if err := h.Storage.Put(r.Context(), m); err != nil {
			handleStorageError(w, err)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

func (h *Handler) PutMetricJSON() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/json" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		var m metrics.Metric
		if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if h.Key != "" {
			ok, err := metrics.Validate(&m, h.Key)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				log.Warnf("Failed to validate hash: %v", err)
				return
			}
			if !ok {
				w.WriteHeader(http.StatusBadRequest)
				log.Infof("Invalid hash: %v", m)
				return
			}
		}

		if err := h.Storage.Put(r.Context(), &m); err != nil {
			handleStorageError(w, err)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

func (h *Handler) PutMetricBatchJSON() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/json" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		var metricsBatch []metrics.Metric
		if err := json.NewDecoder(r.Body).Decode(&metricsBatch); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		for _, m := range metricsBatch {
			if h.Key != "" {
				ok, err := metrics.Validate(&m, h.Key)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					log.Warnf("Failed to validate hash: %v", err)
					return
				}
				if !ok {
					w.WriteHeader(http.StatusBadRequest)
					log.Infof("Invalid hash: %v", m)
					return
				}
			}

			if err := h.Storage.Put(r.Context(), &m); err != nil {
				handleStorageError(w, err)
				return
			}
		}
		w.WriteHeader(http.StatusOK)
	}
}

func handleStorageError(w http.ResponseWriter, err error) {
	switch err {
	case storage.ErrUnknownMetricType:
		w.WriteHeader(http.StatusNotImplemented)
	case storage.ErrBadArgument:
		w.WriteHeader(http.StatusBadRequest)
	case storage.ErrNotFound:
		w.WriteHeader(http.StatusNotFound)
	default:
		log.Warnf("Failed to put metric: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}
