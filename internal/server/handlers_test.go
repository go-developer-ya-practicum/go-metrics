package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hikjik/go-metrics/internal/config"
	"github.com/hikjik/go-metrics/internal/metrics"
	"github.com/hikjik/go-metrics/internal/storage"
)

var storageConfig config.StorageConfig

func init() {
	storageConfig = config.StorageConfig{
		StoreFile:     "tmp/storage.json",
		StoreInterval: time.Second * 300,
		Restore:       false,
	}
}

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

	s, err := storage.New(context.Background(), storageConfig)
	require.NoError(t, err)
	router := NewRouter(s, "")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(tt.method, tt.target, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, request)

			response := w.Result()
			assert.Equal(t, tt.want.statusCode, response.StatusCode)

			if response.StatusCode == http.StatusOK {
				body, err := ioutil.ReadAll(response.Body)
				require.NoError(t, err)
				assert.Equal(t, string(body), tt.want.body)
			}
			require.NoError(t, response.Body.Close())
		})
	}
}

func TestGetAllHandler(t *testing.T) {
	t.Run("Get All metrics", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		s, err := storage.New(context.Background(), storageConfig)
		require.NoError(t, err)
		router := NewRouter(s, "")
		router.ServeHTTP(w, request)

		response := w.Result()
		require.NoError(t, response.Body.Close())
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

	s, err := storage.New(context.Background(), storageConfig)
	require.NoError(t, err)
	router := NewRouter(s, "")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, tt.target, strings.NewReader(tt.body))
			request.Header.Add("Content-Type", tt.contentType)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, request)
			response := w.Result()

			body, err := ioutil.ReadAll(response.Body)
			require.NoError(t, err)
			require.Equal(t, tt.want.statusCode, response.StatusCode)
			require.Equal(t, tt.want.contentType, response.Header.Get("Content-Type"))
			if tt.want.body != "" {
				assert.JSONEq(t, string(body), tt.want.body)
			}
			require.NoError(t, response.Body.Close())
		})
	}
}

func BenchmarkServer_PutMetricJSON(b *testing.B) {
	s, err := storage.New(context.Background(), storageConfig)
	require.NoError(b, err)

	srv := httptest.NewServer(NewRouter(s, ""))
	defer srv.Close()

	metric := metrics.NewGauge("SomeMetric", 1.0)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b.StopTimer()

		var buf bytes.Buffer
		err = json.NewEncoder(&buf).Encode(metric)
		require.NoError(b, err)

		req, err := http.NewRequest(http.MethodPost, srv.URL+"/update/", &buf)
		require.NoError(b, err)
		req.Header.Set("Content-Type", "application/json")

		b.StartTimer()

		resp, err := http.DefaultClient.Do(req)
		require.NoError(b, err)
		require.Equal(b, resp.StatusCode, http.StatusOK)
		require.NoError(b, resp.Body.Close())
	}
}

func BenchmarkServer_PutMetricJSONParallel(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		s, err := storage.New(context.Background(), storageConfig)
		require.NoError(b, err)

		srv := httptest.NewServer(NewRouter(s, ""))
		defer srv.Close()

		metric := metrics.NewGauge("SomeMetric", 1.0)

		b.ReportAllocs()
		b.ResetTimer()

		for pb.Next() {
			b.StopTimer()

			var buf bytes.Buffer
			err = json.NewEncoder(&buf).Encode(metric)
			require.NoError(b, err)

			req, err := http.NewRequest(http.MethodPost, srv.URL+"/update/", &buf)
			require.NoError(b, err)
			req.Header.Set("Content-Type", "application/json")

			b.StartTimer()

			resp, err := http.DefaultClient.Do(req)
			require.NoError(b, err)
			require.Equal(b, resp.StatusCode, http.StatusOK)
			require.NoError(b, resp.Body.Close())
		}
	})
}

func BenchmarkServer_GetMetricJSON(b *testing.B) {
	s, err := storage.New(context.Background(), storageConfig)
	require.NoError(b, err)

	srv := httptest.NewServer(NewRouter(s, ""))
	defer srv.Close()

	metric := metrics.NewGauge("SomeMetric", 1.0)
	err = s.Put(context.Background(), metric)
	require.NoError(b, err)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b.StopTimer()

		var buf bytes.Buffer
		err = json.NewEncoder(&buf).Encode(metric)
		require.NoError(b, err)

		req, err := http.NewRequest(http.MethodPost, srv.URL+"/value/", &buf)
		require.NoError(b, err)
		req.Header.Set("Content-Type", "application/json")

		b.StartTimer()

		resp, err := http.DefaultClient.Do(req)
		require.NoError(b, err)
		require.Equal(b, resp.StatusCode, http.StatusOK)
		require.NoError(b, resp.Body.Close())
	}
}

func BenchmarkServer_GetMetricJSONParallel(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		s, err := storage.New(context.Background(), storageConfig)
		require.NoError(b, err)

		srv := httptest.NewServer(NewRouter(s, ""))
		defer srv.Close()

		metric := metrics.NewGauge("SomeMetric", 1.0)
		err = s.Put(context.Background(), metric)
		require.NoError(b, err)

		b.ReportAllocs()
		b.ResetTimer()

		for pb.Next() {
			b.StopTimer()

			var buf bytes.Buffer
			err = json.NewEncoder(&buf).Encode(metric)
			require.NoError(b, err)

			req, err := http.NewRequest(http.MethodPost, srv.URL+"/value/", &buf)
			require.NoError(b, err)
			req.Header.Set("Content-Type", "application/json")

			b.StartTimer()

			resp, err := http.DefaultClient.Do(req)
			require.NoError(b, err)
			require.Equal(b, resp.StatusCode, http.StatusOK)
			require.NoError(b, resp.Body.Close())
		}
	})
}
