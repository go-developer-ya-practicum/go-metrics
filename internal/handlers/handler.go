package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/hikjik/go-musthave-devops-tpl.git/internal/storage"
)

type Handler struct {
	Storage *storage.Storage
}

func NewHandler() *Handler {
	return &Handler{
		Storage: storage.NewStorage(),
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		msg := fmt.Sprintf("Unsupported %s method", r.Method)
		http.Error(w, msg, http.StatusNotFound)
		return
	}
	contentType := r.Header.Get("Content-type")
	if contentType != "" && contentType != "text/plain" {
		msg := fmt.Sprintf("Unknown content type: %s", contentType)
		http.Error(w, msg, http.StatusNotFound)
		return
	}
	urlParts := strings.Split(r.URL.Path, "/")
	if len(urlParts) != 5 {
		http.Error(w, "Bad URL", http.StatusNotFound)
		return
	}
	if urlParts[1] != "update" {
		http.Error(w, "Bad URL", http.StatusNotFound)
		return
	}

	metricType := urlParts[2]
	metricName := urlParts[3]
	metricValue := urlParts[4]

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
