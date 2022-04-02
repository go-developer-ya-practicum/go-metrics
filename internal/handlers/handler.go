package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/hikjik/go-musthave-devops-tpl.git/internal/storage"
)

type Handler struct {
	*chi.Mux
	Storage *storage.Storage
}

func NewHandler() *Handler {
	h := &Handler{
		Mux:     chi.NewMux(),
		Storage: storage.NewStorage(),
	}
	h.Post("/update/{metricType}/{metricName}/{metricValue}", h.PutMetric())
	h.Get("/value/{metricType}/{metricName}", h.GetMetric())
	return h
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
