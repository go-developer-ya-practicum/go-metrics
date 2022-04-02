package handlers

import (
	"bytes"
	"encoding/json"
	"github.com/hikjik/go-musthave-devops-tpl.git/internal/types"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHandler_PutMetric(t *testing.T) {
	tests := []struct {
		name       string
		target     string
		statusCode int
	}{
		{
			name:       "Counter metric ok",
			target:     "/update/counter/TestCounter/1",
			statusCode: http.StatusOK,
		},
		{
			name:       "Counter metric invalid value",
			target:     "/update/counter/TestCounter/none",
			statusCode: http.StatusBadRequest,
		},
		{
			name:       "Counter without metric name",
			target:     "/update/counter/",
			statusCode: http.StatusNotFound,
		},
		{
			name:       "Gauge metric ok",
			target:     "/update/gauge/TestGauge/1.0",
			statusCode: http.StatusOK,
		},
		{
			name:       "Gauge metric invalid value",
			target:     "/update/gauge/TestGauge/none",
			statusCode: http.StatusBadRequest,
		},
		{
			name:       "Gauge without metric name",
			target:     "/update/gauge/",
			statusCode: http.StatusNotFound,
		},
		{
			name:       "Unknown metric type",
			target:     "/update/unknown/TestCounter/1",
			statusCode: http.StatusNotImplemented,
		},
		{
			name:       "Unknown method",
			target:     "/unknown/counter/TestCounter/1",
			statusCode: http.StatusNotFound,
		},
	}
	for _, tt := range tests {
		h := NewHandler()

		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, tt.target, nil)
			w := httptest.NewRecorder()

			h.ServeHTTP(w, request)

			response := w.Result()
			defer response.Body.Close()
			assert.Equal(t, tt.statusCode, response.StatusCode)
		})
	}
}

func TestHandler_PutGetMetric(t *testing.T) {
	type Request struct {
		target     string
		statusCode int
	}
	tests := []struct {
		name        string
		postRequest *Request
		getRequest  *Request
		want        string
	}{
		{
			name: "Gauge unknown metric name",
			getRequest: &Request{
				target:     "/value/gauge/TestGauge",
				statusCode: http.StatusNotFound,
			},
		},
		{
			name: "Counter unknown metric name",
			getRequest: &Request{
				target:     "/value/counter/TestGauge",
				statusCode: http.StatusNotFound,
			},
		},
		{
			name: "Get gauge metric ok",
			postRequest: &Request{
				target:     "/update/gauge/TestGauge/123.45",
				statusCode: http.StatusOK,
			},
			getRequest: &Request{
				target:     "/value/gauge/TestGauge",
				statusCode: http.StatusOK,
			},
			want: "123.45",
		},
		{
			name: "Get counter metric ok",
			postRequest: &Request{
				target:     "/update/counter/TestCounter/123",
				statusCode: http.StatusOK,
			},
			getRequest: &Request{
				target:     "/value/counter/TestCounter",
				statusCode: http.StatusOK,
			},
			want: "123",
		},
		{
			name: "Unknown metric type",
			getRequest: &Request{
				target:     "/value/unknown/TestGauge",
				statusCode: http.StatusNotImplemented,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler()

			if tt.postRequest != nil {
				request := httptest.NewRequest(http.MethodPost, tt.postRequest.target, nil)
				w := httptest.NewRecorder()

				h.ServeHTTP(w, request)

				response := w.Result()
				defer response.Body.Close()
				assert.Equal(t, tt.postRequest.statusCode, response.StatusCode)
			}

			request := httptest.NewRequest(http.MethodGet, tt.getRequest.target, nil)
			w := httptest.NewRecorder()

			h.ServeHTTP(w, request)

			response := w.Result()
			defer response.Body.Close()
			assert.Equal(t, tt.getRequest.statusCode, response.StatusCode)

			if response.StatusCode == http.StatusOK {
				body, err := ioutil.ReadAll(response.Body)
				if err != nil {
					t.Fatal(err)
				}
				if string(body) != tt.want {
					t.Errorf("Expected body %s, got %s", tt.want, w.Body.String())
				}
			}
		})
	}
}

func TestHandler_GetAllMetric(t *testing.T) {
	t.Run("Get All metrics", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		h := NewHandler()
		h.ServeHTTP(w, request)

		response := w.Result()
		defer response.Body.Close()
		assert.Equal(t, http.StatusOK, response.StatusCode)
	})
}

func TestHandler_PutJSONMetric(t *testing.T) {
	type Metric struct {
		Type  string
		Name  string
		Value interface{}
	}
	tests := []struct {
		name        string
		statusCode  int
		contentType string
		metric      Metric
	}{
		{
			name:        "PutJSON metric bad content type",
			contentType: "text/plain",
			statusCode:  http.StatusBadRequest,
		},
		{
			name:        "PutJSON unknown metric type",
			contentType: "application/json",
			statusCode:  http.StatusNotImplemented,
			metric: Metric{
				Type: "unknown",
			},
		},
		{
			name:        "PutJSON Counter ok",
			contentType: "application/json",
			statusCode:  http.StatusOK,
			metric: Metric{
				Type:  "counter",
				Name:  "TestCounter",
				Value: int64(100),
			},
		},
		{
			name:        "PutJSON Gauge ok",
			contentType: "application/json",
			statusCode:  http.StatusOK,
			metric: Metric{
				Type:  "gauge",
				Name:  "TestGauge",
				Value: 123.45,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler()

			metric := types.Metrics{
				ID:    tt.metric.Name,
				MType: tt.metric.Type,
			}

			switch tt.metric.Type {
			case "gauge":
				value := tt.metric.Value.(float64)
				metric.Value = &value
			case "counter":
				value := tt.metric.Value.(int64)
				metric.Delta = &value
			}

			data, err := json.Marshal(metric)
			if err != nil {
				t.Fatal(err)
			}

			request := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewBuffer(data))
			request.Header.Add("Content-Type", tt.contentType)
			w := httptest.NewRecorder()

			h.ServeHTTP(w, request)

			response := w.Result()
			defer response.Body.Close()
			assert.Equal(t, tt.statusCode, response.StatusCode)
		})
	}
}

func TestHandler_PutJSONGetJSONMetric(t *testing.T) {
	type Metric struct {
		Type  string
		Name  string
		Value interface{}
	}
	type Request struct {
		target     string
		statusCode int
	}
	tests := []struct {
		name        string
		metric      Metric
		postRequest *Request
		getRequest  *Request
	}{
		{
			name: "PutJSON Counter metric ok",
			metric: Metric{
				Type:  "counter",
				Name:  "TestCounter",
				Value: int64(100),
			},
			postRequest: &Request{
				target:     "/update/",
				statusCode: http.StatusOK,
			},
			getRequest: &Request{
				target:     "/value/",
				statusCode: http.StatusOK,
			},
		},
		{
			name: "PutJSON Gauge metric ok",
			metric: Metric{
				Type:  "gauge",
				Name:  "TestGauge",
				Value: 123.45,
			},
			postRequest: &Request{
				target:     "/update/",
				statusCode: http.StatusOK,
			},
			getRequest: &Request{
				target:     "/value/",
				statusCode: http.StatusOK,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler()

			if tt.postRequest != nil {
				metric := types.Metrics{
					ID:    tt.metric.Name,
					MType: tt.metric.Type,
				}

				switch tt.metric.Type {
				case "gauge":
					value := tt.metric.Value.(float64)
					metric.Value = &value
				case "counter":
					value := tt.metric.Value.(int64)
					metric.Delta = &value
				}

				data, err := json.Marshal(metric)
				if err != nil {
					t.Fatal(err)
				}

				request := httptest.NewRequest(http.MethodPost, tt.postRequest.target, bytes.NewBuffer(data))
				request.Header.Add("Content-Type", "application/json")
				w := httptest.NewRecorder()

				h.ServeHTTP(w, request)

				response := w.Result()
				defer response.Body.Close()
				assert.Equal(t, tt.postRequest.statusCode, response.StatusCode)
			}

			metric := types.Metrics{
				ID:    tt.metric.Name,
				MType: tt.metric.Type,
			}
			data, err := json.Marshal(metric)
			if err != nil {
				t.Fatal(err)
			}

			request := httptest.NewRequest(http.MethodPost, tt.getRequest.target, bytes.NewBuffer(data))
			request.Header.Add("Content-Type", "application/json")
			w := httptest.NewRecorder()

			h.ServeHTTP(w, request)

			response := w.Result()
			defer response.Body.Close()
			assert.Equal(t, tt.getRequest.statusCode, response.StatusCode)

			if response.StatusCode == http.StatusOK {
				body, err := ioutil.ReadAll(response.Body)
				if err != nil {
					t.Fatal(err)
				}

				var m types.Metrics
				err = json.Unmarshal(body, &m)
				if err != nil {
					t.Fatal(err)
				}

				assert.Equal(t, tt.metric.Type, m.MType)
				assert.Equal(t, tt.metric.Name, m.ID)

				switch tt.metric.Type {
				case "gauge":
					assert.Equal(t, tt.metric.Value, *m.Value)
				case "counter":
					assert.Equal(t, tt.metric.Value, *m.Delta)
				default:
					assert.False(t, true)
				}
			}
		})
	}
}
