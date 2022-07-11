package server

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
	"github.com/hikjik/go-metrics/internal/storage"
)

//go:embed res
var fs embed.FS

type server struct {
	Storage storage.Storage
	Key     string
}

func (s *server) PingDatabase() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		db, ok := s.Storage.(*storage.DBStorage)
		if !ok {
			log.Warnln("Failed to connect to db")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if err := db.Ping(r.Context()); err != nil {
			log.Warnf("Failed to ping db: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

func (s *server) GetAllMetrics() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		t, err := template.ParseFS(fs, "res/index.html")
		if err != nil {
			log.Warnln("Failed to parse index.html")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		m, err := s.Storage.List(r.Context())
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

func (s *server) GetMetric() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m := &metrics.Metric{
			ID:    chi.URLParam(r, "metricName"),
			MType: chi.URLParam(r, "metricType"),
		}

		if err := s.Storage.Get(r.Context(), m); err != nil {
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

func (s *server) GetMetricJSON() http.HandlerFunc {
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

		if err = s.Storage.Get(r.Context(), &m); err != nil {
			handleStorageError(w, err)
			return
		}

		if s.Key != "" {
			if err = metrics.Sign(&m, s.Key); err != nil {
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

func (s *server) PutMetric() http.HandlerFunc {
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

		if err := s.Storage.Put(r.Context(), m); err != nil {
			handleStorageError(w, err)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

func (s *server) PutMetricJSON() http.HandlerFunc {
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
		if s.Key != "" {
			ok, err := metrics.Validate(&m, s.Key)
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

		if err := s.Storage.Put(r.Context(), &m); err != nil {
			handleStorageError(w, err)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

func (s *server) PutMetricBatchJSON() http.HandlerFunc {
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
			if s.Key != "" {
				ok, err := metrics.Validate(&m, s.Key)
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

			if err := s.Storage.Put(r.Context(), &m); err != nil {
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