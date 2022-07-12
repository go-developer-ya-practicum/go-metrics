package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"

	log "github.com/sirupsen/logrus"

	"github.com/hikjik/go-metrics/internal/metrics"
	"github.com/hikjik/go-metrics/internal/storage"
)

func ExampleServer_PutMetricJSON() {
	s, err := storage.New(context.Background(), storageConfig)
	if err != nil {
		log.Fatal("Failed to create storage")
	}
	router := NewRouter(s, "")

	srv := httptest.NewServer(router)
	defer srv.Close()

	metric := metrics.NewGauge("SomeMetric", 1.0)
	var buf bytes.Buffer
	if err = json.NewEncoder(&buf).Encode(metric); err != nil {
		log.Fatal("Failed to encode metric")
	}

	req, err := http.NewRequest(http.MethodPost, srv.URL+"/update/", &buf)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	if err = resp.Body.Close(); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Put request JSON: %s\n", resp.Status)

	// Output: Put request JSON: 200 OK
}

func ExampleServer_PutMetric() {
	s, err := storage.New(context.Background(), storageConfig)
	if err != nil {
		log.Fatal("Failed to create storage")
	}
	router := NewRouter(s, "")

	srv := httptest.NewServer(router)
	defer srv.Close()

	metric := metrics.NewGauge("SomeMetric", 1.0)

	url := fmt.Sprintf("%s/update/gauge/%s/%f", srv.URL, metric.ID, *metric.Value)
	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		log.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	if err = resp.Body.Close(); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Put request url-params: %s\n", resp.Status)

	// Output: Put request url-params: 200 OK
}

func ExampleServer_GetMetricJSON() {
	s, err := storage.New(context.Background(), storageConfig)
	if err != nil {
		log.Fatal(err)
	}
	router := NewRouter(s, "")

	srv := httptest.NewServer(router)
	defer srv.Close()

	metric := metrics.NewGauge("SomeMetric", 1.0)
	if err = s.Put(context.Background(), metric); err != nil {
		log.Fatal(err)
	}
	var buf bytes.Buffer
	if err = json.NewEncoder(&buf).Encode(metric); err != nil {
		log.Fatal("Failed to encode metric")
	}

	req, err := http.NewRequest(http.MethodPost, srv.URL+"/value/", &buf)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Get request JSON: %s\n", resp.Status)

	var m metrics.Metric
	if err = json.NewDecoder(resp.Body).Decode(&m); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Metric: %s %.2f\n", m.MType, *m.Value)

	if err = resp.Body.Close(); err != nil {
		log.Fatal(err)
	}

	// Output Get request JSON: 200 OK
	// Metric: gauge 1.00
}