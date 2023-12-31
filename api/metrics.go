// metrics.go

package api

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// Define your metrics here
	requestCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "api_requests_total",
			Help: "Total number of API requests.",
		},
		[]string{"method", "endpoint"},
	)
)

// Initialize and register metrics
func init() {
	prometheus.MustRegister(requestCounter)
}

// Update the metrics based on the request
func RecordRequestMetrics(method, endpoint string) {
	requestCounter.WithLabelValues(method, endpoint).Inc()
}

// Add other metric definitions as needed
