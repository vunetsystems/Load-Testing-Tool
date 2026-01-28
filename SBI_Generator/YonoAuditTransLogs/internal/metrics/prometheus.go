package metrics

import (
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	messagesGenerated = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kafka_messages_generated_total",
			Help: "Total number of messages generated",
		},
		[]string{"type"},
	)

	messagesSent = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "kafka_messages_sent_total",
			Help: "Total number of messages sent to Kafka",
		},
	)

	messagesFailed = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "kafka_messages_failed_total",
			Help: "Total number of messages that failed",
		},
	)

	currentEPS = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "kafka_current_eps",
			Help: "Current events per second",
		},
	)

	messageLatency = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "kafka_message_generation_duration_seconds",
			Help:    "Message generation latency in seconds",
			Buckets: prometheus.DefBuckets,
		},
	)
)

func init() {
	// Register metrics
	prometheus.MustRegister(messagesGenerated)
	prometheus.MustRegister(messagesSent)
	prometheus.MustRegister(messagesFailed)
	prometheus.MustRegister(currentEPS)
	prometheus.MustRegister(messageLatency)
}

// PrometheusExporter exports metrics to Prometheus
type PrometheusExporter struct {
	collector *Collector
	port      int
	server    *http.Server
}

// NewPrometheusExporter creates a new Prometheus exporter
func NewPrometheusExporter(collector *Collector, port int) *PrometheusExporter {
	return &PrometheusExporter{
		collector: collector,
		port:      port,
	}
}

// Start starts the Prometheus HTTP server
func (pe *PrometheusExporter) Start() error {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	
	pe.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", pe.port),
		Handler: mux,
	}
	
	go func() {
		if err := pe.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Prometheus server error: %v\n", err)
		}
	}()
	
	return nil
}

// UpdateMetrics updates Prometheus metrics from collector
func (pe *PrometheusExporter) UpdateMetrics() {
	stats := pe.collector.GetStats()
	
	messagesGenerated.WithLabelValues("error").Add(float64(stats.GeneratedError))
	messagesGenerated.WithLabelValues("trans").Add(float64(stats.GeneratedTrans))
	messagesGenerated.WithLabelValues("both").Add(float64(stats.GeneratedBoth))
	
	messagesSent.Add(float64(stats.Sent))
	messagesFailed.Add(float64(stats.Failed))
	currentEPS.Set(stats.CurrentEPS)
}

// Stop stops the Prometheus HTTP server
func (pe *PrometheusExporter) Stop() error {
	if pe.server != nil {
		return pe.server.Close()
	}
	return nil
}
