package handlers

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandlerStore(t *testing.T) {
	tests := []struct {
		name        string
		request     string
		method      string
		contentType string
		statusCode  int
	}{
		{
			name:        "Bad URL parts count",
			request:     "/update/PollCount/1",
			method:      http.MethodPost,
			contentType: "text/plain",
			statusCode:  http.StatusNotFound,
		},
		{
			name:        "Bad URL update",
			request:     "/set/counter/PollCount/1",
			method:      http.MethodPost,
			contentType: "text/plain",
			statusCode:  http.StatusNotFound,
		},
		{
			name:        "Unknown metric type",
			request:     "/update/metric/PollCount/1",
			method:      http.MethodPost,
			contentType: "text/plain",
			statusCode:  http.StatusNotImplemented,
		},
		{
			name:        "Gauge metric invalid value",
			request:     "/update/gauge/RandomValue/none",
			method:      http.MethodPost,
			contentType: "text/plain",
			statusCode:  http.StatusBadRequest,
		},
		{
			name:        "Counter metric invalid value",
			request:     "/update/counter/PollCount/none",
			method:      http.MethodPost,
			contentType: "text/plain",
			statusCode:  http.StatusBadRequest,
		},
		{
			name:        "Gauge metric ok",
			request:     "/update/gauge/RandomValue/1.0",
			method:      http.MethodPost,
			contentType: "text/plain",
			statusCode:  http.StatusOK,
		},
		{
			name:        "Counter metric ok",
			request:     "/update/counter/PollCount/1",
			method:      http.MethodPost,
			contentType: "text/plain",
			statusCode:  http.StatusOK,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(tt.method, tt.request, nil)
			request.Header.Set("Content-Type", tt.contentType)
			w := httptest.NewRecorder()
			h := NewHandler()
			h.ServeHTTP(w, request)
			response := w.Result()
			defer response.Body.Close()

			assert.Equal(t, tt.statusCode, response.StatusCode)
		})
	}
}
