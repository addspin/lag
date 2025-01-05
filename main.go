package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	fmt.Println("Hello, World!")
	startMetricsCollection()
	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(":2112", nil))
}

// Define Prometheus metrics
var (
	groupLagMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "api_group_lag",
		Help: "Group lag value from test.ru/api endpoint",
	})
)

func init() {
	// Register metrics with Prometheus
	prometheus.MustRegister(groupLagMetric)
}

func fetchAndUpdateMetrics() {
	// Make HTTP request to API
	resp, err := http.Get("http://test.ru/api")
	if err != nil {
		log.Printf("Error fetching metrics: %v", err)
		return
	}
	defer resp.Body.Close()

	// Parse response
	var data struct {
		GroupLag float64 `json:"group_lag"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		log.Printf("Error parsing response: %v", err)
		return
	}

	// Update Prometheus metric
	groupLagMetric.Set(data.GroupLag)
}

// Start periodic metrics collection
func startMetricsCollection() {
	ticker := time.NewTicker(15 * time.Second)
	go func() {
		for range ticker.C {
			fetchAndUpdateMetrics()
		}
	}()
}
