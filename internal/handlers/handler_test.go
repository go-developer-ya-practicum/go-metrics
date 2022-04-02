package handlers

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHandler_PutMetric(t *testing.T) {
	tests := []struct {
		name       string
		request    string
		statusCode int
	}{
		{
			name:       "Counter metric ok",
			request:    "/update/counter/TestCounter/1",
			statusCode: http.StatusOK,
		},
		{
			name:       "Counter metric invalid value",
			request:    "/update/counter/TestCounter/none",
			statusCode: http.StatusBadRequest,
		},
		{
			name:       "Counter without metric name",
			request:    "/update/counter/",
			statusCode: http.StatusNotFound,
		},
		{
			name:       "Gauge metric ok",
			request:    "/update/gauge/TestGauge/1.0",
			statusCode: http.StatusOK,
		},
		{
			name:       "Gauge metric invalid value",
			request:    "/update/gauge/TestGauge/none",
			statusCode: http.StatusBadRequest,
		},
		{
			name:       "Gauge without metric name",
			request:    "/update/gauge/",
			statusCode: http.StatusNotFound,
		},
		{
			name:       "Unknown metric type",
			request:    "/update/unknown/TestCounter/1",
			statusCode: http.StatusNotImplemented,
		},
		{
			name:       "Unknown method",
			request:    "/unknown/counter/TestCounter/1",
			statusCode: http.StatusNotFound,
		},
	}
	for _, tt := range tests {
		h := NewHandler()

		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, tt.request, nil)
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
