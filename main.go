package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/viper"
)

var stateValues = map[string]float64{
	"UNKNOWN":              0,
	"PREPARING_REBALANCE":  1,
	"COMPLETING_REBALANCE": 2,
	"STABLE":               3,
	"DEAD":                 4,
	"EMPTY":                5,
}

type AppMetrics struct {
	pollingInterval     int
	consumerGroupLag    *prometheus.GaugeVec
	consumerGroupHealth *prometheus.GaugeVec
}

type Response struct {
	ConsumerGroups []ConsumerGroup `json:"consumerGroups"`
}

type ConsumerGroup struct {
	GroupID     string  `json:"groupId"`
	State       string  `json:"state"`
	ConsumerLag float64 `json:"consumerLag"`
}

func NewAppMetrics(pollingInterval int) *AppMetrics {
	metrics := &AppMetrics{
		pollingInterval: pollingInterval,
		consumerGroupLag: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "kafka_consumergroup_lag",
				Help: "Consumer group lag",
			},
			[]string{"consumergroup"},
		),
		consumerGroupHealth: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "kafka_consumergroup_health",
				Help: "Consumer group health state",
			},
			[]string{"consumergroup"},
		),
	}

	prometheus.MustRegister(metrics.consumerGroupLag)
	prometheus.MustRegister(metrics.consumerGroupHealth)

	return metrics
}

func (a *AppMetrics) fetch() {
	resp, err := http.Get(viper.GetString("path"))
	if err != nil {
		log.Printf("Error fetching data: %v", err)
		return
	}
	defer resp.Body.Close()

	var response Response
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		log.Printf("Error parsing JSON: %v", err)
		return
	}

	// log.Printf("Received groups: %d", len(response.ConsumerGroups))

	for _, group := range response.ConsumerGroups {
		if viper.GetBool("verbose") {
			log.Printf("Group: %s, Lag: %f", group.GroupID, group.ConsumerLag)
			log.Printf("Group: %s, State: %s", group.GroupID, group.State)
			a.consumerGroupLag.WithLabelValues(group.GroupID).Set(group.ConsumerLag)
			a.consumerGroupHealth.WithLabelValues(group.GroupID).Set(convertStateToValue(group.State))
		} else {
			a.consumerGroupLag.WithLabelValues(group.GroupID).Set(group.ConsumerLag)
			a.consumerGroupHealth.WithLabelValues(group.GroupID).Set(convertStateToValue(group.State))
		}
	}
}

func (a *AppMetrics) RunMetricsLoop() {
	for {
		a.fetch()
		time.Sleep(time.Duration(a.pollingInterval) * time.Second)
	}
}

func convertStateToValue(state string) float64 {
	if val, ok := stateValues[state]; ok {
		return val
	}
	return 0
}

func main() {
	log.Printf("Starting exporter...")

	viper.SetConfigFile("config.yaml")
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file: %v", err)
	}

	pollingInterval := viper.GetInt("pollingIntervalSeconds")
	exporterPort := viper.GetInt("exporterPort")

	log.Printf("Polling interval: %d seconds", pollingInterval)
	log.Printf("Exporter port: %d", exporterPort)

	metrics := NewAppMetrics(pollingInterval)

	http.Handle("/metrics", promhttp.Handler())

	go metrics.RunMetricsLoop()

	log.Printf("Exporter is running on port: %d", exporterPort)
	if err := http.ListenAndServe(":"+strconv.Itoa(exporterPort), nil); err != nil {
		log.Fatal("Error starting server:", err)
	}
}
