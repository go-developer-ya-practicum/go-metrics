package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/rs/zerolog/log"

	"github.com/hikjik/go-metrics/internal/metrics"
)

func ExampleServer_PutMetricJSON() {
	s, err := NewTempStorage()
	if err != nil {
		log.Fatal().Msg("Failed to create storage")
	}
	router := NewRouter(s, nil, nil)

	srv := httptest.NewServer(router)

	metric := metrics.NewGauge("SomeMetric", 1.0)
	var buf bytes.Buffer
	if err = json.NewEncoder(&buf).Encode(metric); err != nil {
		log.Fatal().Msg("Failed to encode metric")
	}

	req, err := http.NewRequest(http.MethodPost, srv.URL+"/update/", &buf)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create request")
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal().Err(err)
	}
	if err = resp.Body.Close(); err != nil {
		log.Fatal().Err(err)
	}
	fmt.Printf("Put request JSON: %s\n", resp.Status)
	srv.Close()

	// Output: Put request JSON: 200 OK
}

func ExampleServer_PutMetric() {
	s, err := NewTempStorage()
	if err != nil {
		log.Fatal().Msg("Failed to create storage")
	}
	router := NewRouter(s, nil, nil)

	srv := httptest.NewServer(router)

	metric := metrics.NewGauge("SomeMetric", 1.0)

	url := fmt.Sprintf("%s/update/gauge/%s/%f", srv.URL, metric.ID, *metric.Value)
	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		log.Fatal().Err(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal().Err(err)
	}
	if err = resp.Body.Close(); err != nil {
		log.Fatal().Err(err)
	}
	fmt.Printf("Put request url-params: %s\n", resp.Status)
	srv.Close()

	// Output: Put request url-params: 200 OK
}

func ExampleServer_GetMetricJSON() {
	s, err := NewTempStorage()
	if err != nil {
		log.Fatal().Msg("Failed to create storage")
	}
	router := NewRouter(s, nil, nil)

	srv := httptest.NewServer(router)

	metric := metrics.NewGauge("SomeMetric", 1.0)
	if err = s.Put(context.Background(), metric); err != nil {
		log.Fatal().Err(err)
	}
	var buf bytes.Buffer
	if err = json.NewEncoder(&buf).Encode(metric); err != nil {
		log.Fatal().Msg("Failed to encode metric")
	}

	req, err := http.NewRequest(http.MethodPost, srv.URL+"/value/", &buf)
	if err != nil {
		log.Fatal().Err(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal().Err(err)
	}
	fmt.Printf("Get request JSON: %s\n", resp.Status)

	var m metrics.Metric
	if err = json.NewDecoder(resp.Body).Decode(&m); err != nil {
		log.Fatal().Err(err)
	}
	fmt.Printf("Metric: %s %.2f\n", m.MType, *m.Value)

	if err = resp.Body.Close(); err != nil {
		log.Fatal().Err(err)
	}
	srv.Close()

	// Output Get request JSON: 200 OK
	// Metric: gauge 1.00
}
