//go:build unit

package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/quantfidential/trading-ecosystem/exchange-simulator-go/internal/handlers"
	"github.com/quantfidential/trading-ecosystem/exchange-simulator-go/internal/infrastructure/observability"
)

// TestMetricsHandler defines the expected behaviors for metrics endpoint using Clean Architecture
// Following BDD Given/When/Then pattern
func TestMetricsHandler_Metrics(t *testing.T) {
	t.Run("exposes_prometheus_metrics_through_port", func(t *testing.T) {
		// Given: A Prometheus metrics adapter (implements MetricsPort)
		constantLabels := map[string]string{
			"service":  "exchange-simulator",
			"instance": "exchange-simulator",
			"version":  "1.0.0",
		}
		metricsPort := observability.NewPrometheusMetricsAdapter(constantLabels)

		// And: A metrics handler using the port
		metricsHandler := handlers.NewMetricsHandler(metricsPort)

		// And: A test HTTP server
		gin.SetMode(gin.TestMode)
		router := gin.New()
		router.GET("/metrics", metricsHandler.Metrics)

		// When: A GET request is made to /metrics
		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Then: The response status should be 200 OK
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 OK, got %d", w.Code)
		}

		// And: The response should contain Prometheus metrics format
		body := w.Body.String()
		expectedMetrics := []string{
			"# HELP go_goroutines",
			"# TYPE go_goroutines gauge",
			"# HELP go_info",
			"# TYPE go_info gauge",
			"go_info{version=",
		}

		for _, metric := range expectedMetrics {
			if !strings.Contains(body, metric) {
				t.Errorf("Expected response to contain '%s', but it was not found", metric)
			}
		}
	})

	t.Run("returns_text_plain_content_type", func(t *testing.T) {
		// Given: A Prometheus metrics adapter
		constantLabels := map[string]string{
			"service":  "exchange-simulator",
			"instance": "exchange-simulator",
			"version":  "1.0.0",
		}
		metricsPort := observability.NewPrometheusMetricsAdapter(constantLabels)

		// And: A metrics handler using the port
		metricsHandler := handlers.NewMetricsHandler(metricsPort)

		// And: A test HTTP server
		gin.SetMode(gin.TestMode)
		router := gin.New()
		router.GET("/metrics", metricsHandler.Metrics)

		// When: A GET request is made to /metrics
		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Then: The Content-Type should be text/plain
		contentType := w.Header().Get("Content-Type")
		if !strings.Contains(contentType, "text/plain") {
			t.Errorf("Expected Content-Type to contain 'text/plain', got '%s'", contentType)
		}
	})

	t.Run("includes_standard_go_runtime_metrics", func(t *testing.T) {
		// Given: A Prometheus metrics adapter
		constantLabels := map[string]string{
			"service":  "exchange-simulator",
			"instance": "exchange-simulator",
			"version":  "1.0.0",
		}
		metricsPort := observability.NewPrometheusMetricsAdapter(constantLabels)

		// And: A metrics handler using the port
		metricsHandler := handlers.NewMetricsHandler(metricsPort)

		// And: A test HTTP server
		gin.SetMode(gin.TestMode)
		router := gin.New()
		router.GET("/metrics", metricsHandler.Metrics)

		// When: A GET request is made to /metrics
		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Then: The response should include standard Go runtime metrics
		body := w.Body.String()
		expectedRuntimeMetrics := []string{
			"go_gc_duration_seconds",
			"go_goroutines",
			"go_memstats_alloc_bytes",
			"go_threads",
			"process_cpu_seconds_total",
			"process_resident_memory_bytes",
		}

		for _, metric := range expectedRuntimeMetrics {
			if !strings.Contains(body, metric) {
				t.Errorf("Expected response to contain runtime metric '%s', but it was not found", metric)
			}
		}
	})

	t.Run("metrics_are_parseable_by_prometheus", func(t *testing.T) {
		// Given: A Prometheus metrics adapter
		constantLabels := map[string]string{
			"service":  "exchange-simulator",
			"instance": "exchange-simulator",
			"version":  "1.0.0",
		}
		metricsPort := observability.NewPrometheusMetricsAdapter(constantLabels)

		// And: A metrics handler using the port
		metricsHandler := handlers.NewMetricsHandler(metricsPort)

		// And: A test HTTP server
		gin.SetMode(gin.TestMode)
		router := gin.New()
		router.GET("/metrics", metricsHandler.Metrics)

		// When: A GET request is made to /metrics
		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Then: The response should follow Prometheus text format
		body := w.Body.String()
		lines := strings.Split(body, "\n")

		foundHelp := false
		foundType := false
		foundMetric := false

		for _, line := range lines {
			if strings.HasPrefix(line, "# HELP") {
				foundHelp = true
			}
			if strings.HasPrefix(line, "# TYPE") {
				foundType = true
			}
			// Metric lines don't start with #
			if len(line) > 0 && !strings.HasPrefix(line, "#") && strings.Contains(line, " ") {
				foundMetric = true
			}
		}

		if !foundHelp {
			t.Error("Expected to find '# HELP' lines in metrics output")
		}
		if !foundType {
			t.Error("Expected to find '# TYPE' lines in metrics output")
		}
		if !foundMetric {
			t.Error("Expected to find metric value lines in metrics output")
		}
	})
}

// TestMetricsHandler_Integration verifies metrics endpoint integration
func TestMetricsHandler_Integration(t *testing.T) {
	t.Run("metrics_endpoint_works_in_full_router", func(t *testing.T) {
		// Given: A Prometheus metrics adapter with constant labels
		constantLabels := map[string]string{
			"service":  "exchange-simulator",
			"instance": "exchange-simulator",
			"version":  "1.0.0",
		}
		metricsPort := observability.NewPrometheusMetricsAdapter(constantLabels)

		// And: A full router setup similar to main.go
		gin.SetMode(gin.TestMode)
		router := gin.New()
		router.Use(gin.Recovery())

		metricsHandler := handlers.NewMetricsHandler(metricsPort)
		router.GET("/metrics", metricsHandler.Metrics)

		// When: Multiple requests are made to /metrics
		for i := 0; i < 3; i++ {
			req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Then: Each request should succeed
			if w.Code != http.StatusOK {
				t.Errorf("Request %d: Expected status 200 OK, got %d", i+1, w.Code)
			}

			// And: Each response should contain valid metrics
			body := w.Body.String()
			if !strings.Contains(body, "go_goroutines") {
				t.Errorf("Request %d: Expected response to contain metrics", i+1)
			}
		}
	})
}
