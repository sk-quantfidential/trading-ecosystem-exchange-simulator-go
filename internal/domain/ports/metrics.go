package ports

import "net/http"

// MetricsPort defines the interface for observability metrics collection
// This port abstracts the metrics implementation (Prometheus, OpenTelemetry, etc.)
// following Clean Architecture principles: domain doesn't depend on infrastructure
type MetricsPort interface {
	// RED Pattern - Request Layer Metrics

	// IncCounter increments a counter metric
	// name: metric name (e.g., "requests_total")
	// labels: key-value pairs (e.g., {"method": "GET", "route": "/api/v1/health", "code": "200"})
	IncCounter(name string, labels map[string]string)

	// ObserveHistogram records a value in a histogram metric
	// name: metric name (e.g., "request_duration_seconds")
	// value: observed value (e.g., 0.123 for 123ms)
	// labels: key-value pairs
	ObserveHistogram(name string, value float64, labels map[string]string)

	// SetGauge sets a gauge metric to a specific value
	// name: metric name (e.g., "service_dependency_ready")
	// value: gauge value
	// labels: key-value pairs
	SetGauge(name string, value float64, labels map[string]string)

	// HTTP Serving

	// GetHTTPHandler returns an http.Handler that serves the metrics endpoint
	// This handler will be mounted at /metrics
	GetHTTPHandler() http.Handler
}

// MetricsLabels defines standard labels used across all metrics
// These should have LOW CARDINALITY to avoid metric explosion
type MetricsLabels struct {
	// Constant labels (set once at startup)
	Service  string // service name (e.g., "exchange-simulator")
	Instance string // instance identifier (e.g., "exchange-simulator" or "exchange-OKX")
	Version  string // version (git SHA or semver, e.g., "1.0.0" or "abc123")

	// Request labels (per-request, but limited cardinality)
	Method string // HTTP method (GET, POST, etc.) - low cardinality
	Route  string // Route pattern (e.g., "/api/v1/health") - low cardinality
	Code   string // HTTP status code (200, 404, 500) - low cardinality
}

// ToMap converts MetricsLabels to a map for use with metrics methods
// Only includes non-empty labels
func (l *MetricsLabels) ToMap() map[string]string {
	labels := make(map[string]string)

	if l.Service != "" {
		labels["service"] = l.Service
	}
	if l.Instance != "" {
		labels["instance"] = l.Instance
	}
	if l.Version != "" {
		labels["version"] = l.Version
	}
	if l.Method != "" {
		labels["method"] = l.Method
	}
	if l.Route != "" {
		labels["route"] = l.Route
	}
	if l.Code != "" {
		labels["code"] = l.Code
	}

	return labels
}

// ConstantLabels returns only the constant labels (service, instance, version)
func (l *MetricsLabels) ConstantLabels() map[string]string {
	return map[string]string{
		"service":  l.Service,
		"instance": l.Instance,
		"version":  l.Version,
	}
}
