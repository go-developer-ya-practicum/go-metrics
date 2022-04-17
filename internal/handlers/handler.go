package handlers

import (
	"embed"
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/v5"
	log "github.com/sirupsen/logrus"
	"html/template"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/hikjik/go-musthave-devops-tpl.git/internal/metrics"
	"github.com/hikjik/go-musthave-devops-tpl.git/internal/middleware"
	"github.com/hikjik/go-musthave-devops-tpl.git/internal/storage"
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

		if err := s.Ping(); err != nil {
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
			w.WriteHeader(http.StatusNotFound)
			return
		}

		data := struct {
			GaugeMetrics   map[string]float64
			CounterMetrics map[string]int64
		}{
			h.Storage.GetGaugeMetrics(),
			h.Storage.GetCounterMetrics(),
		}
		if err := t.Execute(w, data); err != nil {
			log.Warnf("Failed to execute template: %v", err)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

func (h *Handler) GetMetric() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		metricType := chi.URLParam(r, "metricType")
		metricName := chi.URLParam(r, "metricName")

		switch metricType {
		case "gauge":
			if value, ok := h.Storage.GetGauge(metricName); !ok {
				w.WriteHeader(http.StatusNotFound)
			} else {
				w.WriteHeader(http.StatusOK)
				strValue := strconv.FormatFloat(value, 'f', -1, 64)
				w.Write([]byte(strValue))
			}
		case "counter":
			if value, ok := h.Storage.GetCounter(metricName); !ok {
				w.WriteHeader(http.StatusNotFound)
			} else {
				w.WriteHeader(http.StatusOK)
				strValue := fmt.Sprintf("%d", value)
				w.Write([]byte(strValue))
			}
		default:
			msg := fmt.Sprintf("Unknown metric type '%s'", metricType)
			http.Error(w, msg, http.StatusNotImplemented)
		}
	}
}

func (h *Handler) PutMetric() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		metricType := chi.URLParam(r, "metricType")
		metricName := chi.URLParam(r, "metricName")
		metricValue := chi.URLParam(r, "metricValue")

		switch metricType {
		case "gauge":
			value, err := strconv.ParseFloat(metricValue, 64)
			if err != nil {
				msg := fmt.Sprintf("Failed to parse gauge value '%s'", metricValue)
				http.Error(w, msg, http.StatusBadRequest)
				return
			}

			w.WriteHeader(http.StatusOK)
			h.Storage.PutGauge(metricName, value)
		case "counter":
			value, err := strconv.ParseInt(metricValue, 10, 64)
			if err != nil {
				msg := fmt.Sprintf("Failed to parse counter value '%s'", metricValue)
				http.Error(w, msg, http.StatusBadRequest)
				return
			}

			w.WriteHeader(http.StatusOK)
			h.Storage.UpdateCounter(metricName, value)
		default:
			msg := fmt.Sprintf("Unknown metric type '%s'", metricType)
			http.Error(w, msg, http.StatusNotImplemented)
		}
	}
}

func (h *Handler) PutMetricJSON() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/json" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		var metric metrics.Metrics
		if err := json.NewDecoder(r.Body).Decode(&metric); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if h.Key != "" {
			ok, err := metric.ValidateHash(h.Key)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				log.Warnf("Failed to validate hash: %v", err)
				return
			}
			if !ok {
				w.WriteHeader(http.StatusBadRequest)
				log.Infof("Invalid hash: %v", metric)
				return
			}
		}

		if err := h.storeMetric(metric); err != nil {
			w.WriteHeader(http.StatusNotImplemented)
			return
		}

		w.WriteHeader(http.StatusOK)
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

		var metric metrics.Metrics
		err = json.Unmarshal(body, &metric)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		switch metric.MType {
		case "gauge":
			if value, ok := h.Storage.GetGauge(metric.ID); !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			} else {
				metric.Value = &value
			}
		case "counter":
			if value, ok := h.Storage.GetCounter(metric.ID); !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			} else {
				metric.Delta = &value
			}
		default:
			w.WriteHeader(http.StatusNotImplemented)
			return
		}

		if h.Key != "" {
			if err = metric.SetHash(h.Key); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				log.Warnf("Failed to set hash: %v", err)
				return
			}
		}

		if err = json.NewEncoder(w).Encode(metric); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

func (h *Handler) storeMetric(metric metrics.Metrics) error {
	switch metric.MType {
	case "gauge":
		if metric.Value == nil {
			return fmt.Errorf("empty '%s' metric value", metric.ID)
		}
		h.Storage.PutGauge(metric.ID, *metric.Value)
		return nil
	case "counter":
		if metric.Delta == nil {
			return fmt.Errorf("empty '%s' metric value", metric.ID)
		}
		h.Storage.UpdateCounter(metric.ID, *metric.Delta)
		return nil
	default:
		return fmt.Errorf("unknown metric type %s", metric.MType)
	}
}
