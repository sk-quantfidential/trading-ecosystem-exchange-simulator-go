//go:build unit

package observability_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/quantfidential/trading-ecosystem/exchange-simulator-go/internal/infrastructure/observability"
)

// TestREDMetricsMiddleware verifies RED pattern metrics instrumentation
// Following BDD Given/When/Then pattern
func TestREDMetricsMiddleware(t *testing.T) {
	t.Run("instruments_successful_requests_with_RED_metrics", func(t *testing.T) {
		// Given: A Prometheus metrics adapter
		constantLabels := map[string]string{
			"service":  "exchange-simulator",
			"instance": "exchange-simulator",
			"version":  "1.0.0",
		}
		metricsPort := observability.NewPrometheusMetricsAdapter(constantLabels)

		// And: A Gin router with RED metrics middleware
		gin.SetMode(gin.TestMode)
		router := gin.New()
		router.Use(observability.REDMetricsMiddleware(metricsPort))

		// And: A test endpoint
		router.GET("/api/v1/health", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "healthy"})
		})

		// When: A successful request is made
		req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Then: The request should succeed
		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		// And: Metrics should be recorded
		// Get metrics output
		metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		metricsW := httptest.NewRecorder()
		metricsPort.GetHTTPHandler().ServeHTTP(metricsW, metricsReq)
		metricsOutput := metricsW.Body.String()

		// RED Metric 1: http_requests_total should be incremented
		if !strings.Contains(metricsOutput, "http_requests_total") {
			t.Error("Expected http_requests_total metric to be present")
		}

		// RED Metric 2: http_request_duration_seconds should be observed
		if !strings.Contains(metricsOutput, "http_request_duration_seconds") {
			t.Error("Expected http_request_duration_seconds metric to be present")
		}

		// Verify labels are included (method, route, code)
		if !strings.Contains(metricsOutput, `method="GET"`) {
			t.Error("Expected method label in metrics")
		}
		if !strings.Contains(metricsOutput, `route="/api/v1/health"`) {
			t.Error("Expected route label in metrics")
		}
		if !strings.Contains(metricsOutput, `code="200"`) {
			t.Error("Expected code label in metrics")
		}

		// Verify constant labels are included
		if !strings.Contains(metricsOutput, `service="exchange-simulator"`) {
			t.Error("Expected service constant label in metrics")
		}
		if !strings.Contains(metricsOutput, `instance="exchange-simulator"`) {
			t.Error("Expected instance constant label in metrics")
		}
		if !strings.Contains(metricsOutput, `version="1.0.0"`) {
			t.Error("Expected version constant label in metrics")
		}
	})

	t.Run("instruments_error_requests_with_error_counter", func(t *testing.T) {
		// Given: A Prometheus metrics adapter
		constantLabels := map[string]string{
			"service":  "exchange-simulator",
			"instance": "exchange-simulator",
			"version":  "1.0.0",
		}
		metricsPort := observability.NewPrometheusMetricsAdapter(constantLabels)

		// And: A Gin router with RED metrics middleware
		gin.SetMode(gin.TestMode)
		router := gin.New()
		router.Use(observability.REDMetricsMiddleware(metricsPort))

		// And: A test endpoint that returns 404
		// (no route defined, so Gin returns 404)

		// When: A request to non-existent endpoint is made
		req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Then: The request should return 404
		if w.Code != 404 {
			t.Errorf("Expected status 404, got %d", w.Code)
		}

		// And: Error metrics should be recorded
		metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		metricsW := httptest.NewRecorder()
		metricsPort.GetHTTPHandler().ServeHTTP(metricsW, metricsReq)
		metricsOutput := metricsW.Body.String()

		// RED Metric 3: http_request_errors_total should be incremented
		if !strings.Contains(metricsOutput, "http_request_errors_total") {
			t.Error("Expected http_request_errors_total metric to be present for 404")
		}

		// Verify error code label
		if !strings.Contains(metricsOutput, `code="404"`) {
			t.Error("Expected code=404 label in error metrics")
		}
	})

	t.Run("uses_route_pattern_not_full_path_for_low_cardinality", func(t *testing.T) {
		// Given: A Prometheus metrics adapter
		constantLabels := map[string]string{
			"service":  "exchange-simulator",
			"instance": "exchange-simulator",
			"version":  "1.0.0",
		}
		metricsPort := observability.NewPrometheusMetricsAdapter(constantLabels)

		// And: A Gin router with RED metrics middleware
		gin.SetMode(gin.TestMode)
		router := gin.New()
		router.Use(observability.REDMetricsMiddleware(metricsPort))

		// And: A parameterized route
		router.GET("/api/v1/orders/:id", func(c *gin.Context) {
			c.JSON(200, gin.H{"id": c.Param("id")})
		})

		// When: Multiple requests with different IDs are made
		ids := []string{"123", "456", "789"}
		for _, id := range ids {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/orders/"+id, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != 200 {
				t.Errorf("Expected status 200 for ID %s, got %d", id, w.Code)
			}
		}

		// Then: Metrics should use route pattern, not full path
		metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		metricsW := httptest.NewRecorder()
		metricsPort.GetHTTPHandler().ServeHTTP(metricsW, metricsReq)
		metricsOutput := metricsW.Body.String()

		// Should have route pattern label (low cardinality)
		if !strings.Contains(metricsOutput, `route="/api/v1/orders/:id"`) {
			t.Error("Expected route pattern /api/v1/orders/:id in metrics (low cardinality)")
		}

		// Should NOT have full paths (high cardinality)
		if strings.Contains(metricsOutput, `route="/api/v1/orders/123"`) {
			t.Error("Metrics should not contain full path /api/v1/orders/123 (high cardinality)")
		}
	})

	t.Run("handles_unknown_routes_gracefully", func(t *testing.T) {
		// Given: A Prometheus metrics adapter
		constantLabels := map[string]string{
			"service":  "exchange-simulator",
			"instance": "exchange-simulator",
			"version":  "1.0.0",
		}
		metricsPort := observability.NewPrometheusMetricsAdapter(constantLabels)

		// And: A Gin router with RED metrics middleware
		gin.SetMode(gin.TestMode)
		router := gin.New()
		router.Use(observability.REDMetricsMiddleware(metricsPort))

		// When: A request to unknown route is made
		req := httptest.NewRequest(http.MethodGet, "/unknown", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Then: Metrics should use "unknown" for empty route
		metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		metricsW := httptest.NewRecorder()
		metricsPort.GetHTTPHandler().ServeHTTP(metricsW, metricsReq)
		metricsOutput := metricsW.Body.String()

		if !strings.Contains(metricsOutput, `route="unknown"`) {
			t.Error("Expected route=unknown for unmatched routes")
		}
	})
}
