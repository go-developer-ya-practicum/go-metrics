package handlers

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPutGetHandler(t *testing.T) {
	type want struct {
		body       string
		statusCode int
	}
	tests := []struct {
		name   string
		target string
		method string
		want   want
	}{
		{
			name:   "Put gauge metric ok",
			target: "/update/gauge/TestGauge/123.45",
			method: http.MethodPost,
			want:   want{statusCode: http.StatusOK},
		},
		{
			name:   "Get gauge metric ok",
			target: "/value/gauge/TestGauge",
			method: http.MethodGet,
			want:   want{statusCode: http.StatusOK, body: "123.45"},
		},
		{
			name:   "Put counter metric ok",
			target: "/update/counter/TestCounter/123",
			method: http.MethodPost,
			want:   want{statusCode: http.StatusOK},
		},
		{
			name:   "Get counter metric ok",
			target: "/value/counter/TestCounter",
			method: http.MethodGet,
			want:   want{statusCode: http.StatusOK, body: "123"},
		},
		{
			name:   "Get unknown metric type",
			target: "/value/unknown/TestGauge",
			method: http.MethodGet,
			want:   want{statusCode: http.StatusNotImplemented},
		},
		{
			name:   "Get unknown counter metric name",
			target: "/value/counter/unknown",
			method: http.MethodGet,
			want:   want{statusCode: http.StatusNotFound},
		},
		{
			name:   "Get unknown gauge metric name",
			target: "/value/gauge/unknown",
			method: http.MethodGet,
			want:   want{statusCode: http.StatusNotFound},
		},
		{
			name:   "Put counter metric invalid value",
			target: "/update/counter/TestCounter/none",
			method: http.MethodPost,
			want:   want{statusCode: http.StatusBadRequest},
		},
		{
			name:   "Put counter metric without name",
			target: "/update/counter/",
			method: http.MethodPost,
			want:   want{statusCode: http.StatusNotFound},
		},
		{
			name:   "Put gauge metric invalid value",
			target: "/update/gauge/TestGauge/none",
			method: http.MethodPost,
			want:   want{statusCode: http.StatusBadRequest},
		},
		{
			name:   "Put gauge metric without name",
			target: "/update/gauge/",
			method: http.MethodPost,
			want:   want{statusCode: http.StatusNotFound},
		},
		{
			name:   "Put metric unknown metric type",
			target: "/update/unknown/TestCounter/1",
			method: http.MethodPost,
			want:   want{statusCode: http.StatusNotImplemented},
		},
	}

	h := NewHandler()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(tt.method, tt.target, nil)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, request)

			response := w.Result()
			defer response.Body.Close()
			assert.Equal(t, tt.want.statusCode, response.StatusCode)

			if response.StatusCode == http.StatusOK {
				body, err := ioutil.ReadAll(response.Body)
				require.NoError(t, err)
				assert.Equal(t, string(body), tt.want.body)
			}
		})
	}
}

func TestGetAllHandler(t *testing.T) {
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

func TestPutGetJSONHandler(t *testing.T) {
	type want struct {
		body        string
		statusCode  int
		contentType string
	}
	tests := []struct {
		name        string
		target      string
		contentType string
		body        string
		want        want
	}{
		{
			name:        "PutJSON counter metric ok",
			target:      "/update/",
			contentType: "application/json",
			body:        `{"id":"TestCounter","type":"counter","delta":100}`,
			want:        want{statusCode: http.StatusOK},
		},
		{
			name:        "GetJSON counter metric ok",
			target:      "/value/",
			contentType: "application/json",
			body:        `{"id":"TestCounter","type":"counter"}`,
			want: want{
				statusCode:  http.StatusOK,
				contentType: "application/json",
				body:        `{"id":"TestCounter","type":"counter","delta":100}`,
			},
		},
		{
			name:        "PutJSON gauge metric ok",
			target:      "/update/",
			contentType: "application/json",
			body:        `{"id":"TestGauge","type":"gauge","value":123.45}`,
			want:        want{statusCode: http.StatusOK},
		},
		{
			name:        "GetJSON gauge metric ok",
			target:      "/value/",
			contentType: "application/json",
			body:        `{"id":"TestGauge","type":"gauge"}`,
			want: want{
				statusCode:  http.StatusOK,
				contentType: "application/json",
				body:        `{"id":"TestGauge","type":"gauge","value":123.45}`,
			},
		},
		{
			name:        "GetJSON bad content type",
			target:      "/value/",
			contentType: "text/plain",
			want: want{
				statusCode:  http.StatusBadRequest,
				contentType: "application/json",
			},
		},
		{
			name:        "GetJSON invalid body",
			target:      "/value/",
			contentType: "application/json",
			body:        `{"id":`,
			want: want{
				statusCode:  http.StatusBadRequest,
				contentType: "application/json",
			},
		},
		{
			name:        "GetJSON unknown metric type",
			target:      "/value/",
			contentType: "application/json",
			body:        `{"id":"TestGauge","type":"unknown","value":0}`,
			want: want{
				statusCode:  http.StatusNotImplemented,
				contentType: "application/json",
			},
		},
		{
			name:        "PutJSON bad content type",
			target:      "/update/",
			contentType: "text/plain",
			want:        want{statusCode: http.StatusBadRequest},
		},
		{
			name:        "PutJSON invalid body",
			target:      "/update/",
			contentType: "application/json",
			body:        `{"id":`,
			want:        want{statusCode: http.StatusBadRequest},
		},
		{
			name:        "PutJSON unknown metric type",
			target:      "/update/",
			contentType: "application/json",
			body:        `{"id":"TestGauge","type":"unknown","value":0}`,
			want:        want{statusCode: http.StatusNotImplemented},
		},
	}

	h := NewHandler()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, tt.target, strings.NewReader(tt.body))
			request.Header.Add("Content-Type", tt.contentType)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, request)
			response := w.Result()
			defer response.Body.Close()

			body, err := ioutil.ReadAll(response.Body)
			require.NoError(t, err)
			require.Equal(t, tt.want.statusCode, response.StatusCode)
			require.Equal(t, tt.want.contentType, response.Header.Get("Content-Type"))
			if tt.want.body != "" {
				assert.JSONEq(t, string(body), tt.want.body)
			}
		})
	}
}
