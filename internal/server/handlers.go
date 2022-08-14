package server

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"

	"github.com/hikjik/go-metrics/internal/encryption"
	"github.com/hikjik/go-metrics/internal/metrics"
	"github.com/hikjik/go-metrics/internal/storage"
)

//go:embed res
var fs embed.FS

type Server struct {
	Storage   storage.Storage
	Signer    metrics.Signer
	Decrypter encryption.Decrypter
}

// PingDatabase хендлер для проверки доступности базы данных
func (s *Server) PingDatabase() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		db, ok := s.Storage.(*storage.DBStorage)
		if !ok {
			log.Warn().Msg("Failed to connect to db")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if err := db.Ping(r.Context()); err != nil {
			log.Warn().Err(err).Msg("Failed to ping db")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

// GetAllMetrics хендлер, возвращающий html-страницу
// с информацией о всех сохраненных метриках
func (s *Server) GetAllMetrics() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		t, err := template.ParseFS(fs, "res/index.html")
		if err != nil {
			log.Warn().Err(err).Msg("Failed to parse index.html")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		m, err := s.Storage.List(r.Context())
		if err != nil {
			log.Warn().Err(err).Msg("Failed to list metrics")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if err = t.Execute(w, m); err != nil {
			log.Warn().Err(err).Msg("Failed to execute template")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

// GetMetric хендлер, возвращающий текущее значение запрашиваемой метрики в текстовом виде.
// Параметры метрики передаются из URL в параметрах запроса
func (s *Server) GetMetric() http.HandlerFunc {
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
		if _, err := io.WriteString(w, str); err != nil {
			log.Warn().Err(err).Msg("Write failed")
		}
	}
}

// GetMetricJSON хендлер, возвращающий текущее значение запрашиваемой метрики.
// Параметры метрики передаются в теле запроса в формате JSON
func (s *Server) GetMetricJSON() http.HandlerFunc {
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

		if s.Signer != nil {
			if err = s.Signer.Sign(&m); err != nil {
				log.Warn().Err(err).Msg("Failed to set hash")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}

		if err = json.NewEncoder(w).Encode(m); err != nil {
			log.Warn().Err(err).Msg("Failed to encode metric")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
}

// PutMetric хендлер принимает и сохраняет переданное значение метрики.
// Параметры метрики передаются из URL параметрах запроса
func (s *Server) PutMetric() http.HandlerFunc {
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

// PutMetricJSON хендлер принимает и сохраняет переданное значение метрики.
// Параметры метрики передаются в теле запроса в формате JSON
func (s *Server) PutMetricJSON() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/json" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		decryptedData, err := s.decryptRequestBody(r)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Warn().Err(err).Msg("Failed to decrypt request body")
			return
		}

		var m metrics.Metric
		if err = json.Unmarshal(decryptedData, &m); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if s.Signer != nil {
			ok, err := s.Signer.Validate(&m)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				log.Warn().Err(err).Msg("Failed to validate hash")
				return
			}
			if !ok {
				w.WriteHeader(http.StatusBadRequest)
				log.Info().Msgf("Invalid hash: %v", m)
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

// PutMetricBatchJSON хендлер принимает и сохраняет переданные значения метрик.
// Параметры метрик передаются в теле запроса в формате JSON
func (s *Server) PutMetricBatchJSON() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/json" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		decryptedData, err := s.decryptRequestBody(r)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Warn().Err(err).Msg("Failed to decrypt request body")
			return
		}

		var metricsBatch []metrics.Metric
		if err = json.Unmarshal(decryptedData, &metricsBatch); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		for _, m := range metricsBatch {
			if s.Signer != nil {
				ok, err := s.Signer.Validate(&m)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					log.Warn().Err(err).Msg("Failed to validate hash")
					return
				}
				if !ok {
					w.WriteHeader(http.StatusBadRequest)
					log.Info().Msgf("Invalid hash: %v", m)
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
		log.Warn().Err(err).Msg("Failed to put metric")
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (s *Server) decryptRequestBody(r *http.Request) ([]byte, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	if s.Decrypter == nil {
		return body, nil
	}

	return s.Decrypter.Decrypt(body)
}
