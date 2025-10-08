package observability

import (
	"net/http"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/quantfidential/trading-ecosystem/exchange-simulator-go/internal/domain/ports"
)

// Compile-time check that PrometheusMetricsAdapter implements MetricsPort
var _ ports.MetricsPort = (*PrometheusMetricsAdapter)(nil)

// PrometheusMetricsAdapter implements the MetricsPort using Prometheus client library
// This adapter can be swapped with OpenTelemetry in the future without changing domain logic
type PrometheusMetricsAdapter struct {
	registry *prometheus.Registry

	// Metric collectors (lazy-initialized)
	counters   map[string]*prometheus.CounterVec
	histograms map[string]*prometheus.HistogramVec
	gauges     map[string]*prometheus.GaugeVec

	// Mutex for thread-safe lazy initialization
	mu sync.RWMutex

	// Constant labels applied to all metrics
	constantLabels map[string]string
}

// NewPrometheusMetricsAdapter creates a new Prometheus metrics adapter
// constantLabels: labels applied to all metrics (service, instance, version)
func NewPrometheusMetricsAdapter(constantLabels map[string]string) *PrometheusMetricsAdapter {
	registry := prometheus.NewRegistry()

	// Register default Go runtime metrics
	registry.MustRegister(prometheus.NewGoCollector())
	registry.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))

	return &PrometheusMetricsAdapter{
		registry:       registry,
		counters:       make(map[string]*prometheus.CounterVec),
		histograms:     make(map[string]*prometheus.HistogramVec),
		gauges:         make(map[string]*prometheus.GaugeVec),
		constantLabels: constantLabels,
	}
}

// IncCounter increments a counter metric
func (a *PrometheusMetricsAdapter) IncCounter(name string, labels map[string]string) {
	counter := a.getOrCreateCounter(name, labels)
	counter.With(prometheus.Labels(labels)).Inc()
}

// ObserveHistogram records a value in a histogram metric
func (a *PrometheusMetricsAdapter) ObserveHistogram(name string, value float64, labels map[string]string) {
	histogram := a.getOrCreateHistogram(name, labels)
	histogram.With(prometheus.Labels(labels)).Observe(value)
}

// SetGauge sets a gauge metric to a specific value
func (a *PrometheusMetricsAdapter) SetGauge(name string, value float64, labels map[string]string) {
	gauge := a.getOrCreateGauge(name, labels)
	gauge.With(prometheus.Labels(labels)).Set(value)
}

// GetHTTPHandler returns the Prometheus HTTP handler for /metrics endpoint
func (a *PrometheusMetricsAdapter) GetHTTPHandler() http.Handler {
	return promhttp.HandlerFor(a.registry, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	})
}

// getOrCreateCounter gets or creates a counter metric (thread-safe lazy initialization)
func (a *PrometheusMetricsAdapter) getOrCreateCounter(name string, labels map[string]string) *prometheus.CounterVec {
	// Fast path: read lock
	a.mu.RLock()
	counter, exists := a.counters[name]
	a.mu.RUnlock()

	if exists {
		return counter
	}

	// Slow path: write lock and create
	a.mu.Lock()
	defer a.mu.Unlock()

	// Double-check after acquiring write lock
	if counter, exists := a.counters[name]; exists {
		return counter
	}

	// Extract label names from the provided labels
	labelNames := a.extractLabelNames(labels)

	// Create new counter
	counter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name:        name,
			Help:        name, // TODO: Add proper help text
			ConstLabels: prometheus.Labels(a.constantLabels),
		},
		labelNames,
	)

	a.registry.MustRegister(counter)
	a.counters[name] = counter

	return counter
}

// getOrCreateHistogram gets or creates a histogram metric (thread-safe lazy initialization)
func (a *PrometheusMetricsAdapter) getOrCreateHistogram(name string, labels map[string]string) *prometheus.HistogramVec {
	// Fast path: read lock
	a.mu.RLock()
	histogram, exists := a.histograms[name]
	a.mu.RUnlock()

	if exists {
		return histogram
	}

	// Slow path: write lock and create
	a.mu.Lock()
	defer a.mu.Unlock()

	// Double-check after acquiring write lock
	if histogram, exists := a.histograms[name]; exists {
		return histogram
	}

	// Extract label names from the provided labels
	labelNames := a.extractLabelNames(labels)

	// Create new histogram with sensible buckets for request duration
	// Buckets: 5ms, 10ms, 25ms, 50ms, 100ms, 250ms, 500ms, 1s, 2.5s, 5s, 10s
	histogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:        name,
			Help:        name, // TODO: Add proper help text
			ConstLabels: prometheus.Labels(a.constantLabels),
			Buckets:     []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
		labelNames,
	)

	a.registry.MustRegister(histogram)
	a.histograms[name] = histogram

	return histogram
}

// getOrCreateGauge gets or creates a gauge metric (thread-safe lazy initialization)
func (a *PrometheusMetricsAdapter) getOrCreateGauge(name string, labels map[string]string) *prometheus.GaugeVec {
	// Fast path: read lock
	a.mu.RLock()
	gauge, exists := a.gauges[name]
	a.mu.RUnlock()

	if exists {
		return gauge
	}

	// Slow path: write lock and create
	a.mu.Lock()
	defer a.mu.Unlock()

	// Double-check after acquiring write lock
	if gauge, exists := a.gauges[name]; exists {
		return gauge
	}

	// Extract label names from the provided labels
	labelNames := a.extractLabelNames(labels)

	// Create new gauge
	gauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name:        name,
			Help:        name, // TODO: Add proper help text
			ConstLabels: prometheus.Labels(a.constantLabels),
		},
		labelNames,
	)

	a.registry.MustRegister(gauge)
	a.gauges[name] = gauge

	return gauge
}

// extractLabelNames extracts label names from a labels map
// Excludes constant labels (service, instance, version)
func (a *PrometheusMetricsAdapter) extractLabelNames(labels map[string]string) []string {
	labelNames := make([]string, 0, len(labels))

	for key := range labels {
		// Skip constant labels (they're already in ConstLabels)
		if key == "service" || key == "instance" || key == "version" {
			continue
		}
		labelNames = append(labelNames, key)
	}

	return labelNames
}
