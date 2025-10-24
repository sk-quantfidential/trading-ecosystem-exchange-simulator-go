package observability

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/quantfidential/trading-ecosystem/exchange-simulator-go/internal/domain/ports"
)

// REDMetricsMiddleware creates Gin middleware for RED pattern metrics
// RED: Rate (requests_total), Errors (request_errors_total), Duration (request_duration_seconds)
//
// This middleware instruments all HTTP requests with:
// - requests_total: Total number of requests (counter)
// - request_duration_seconds: Request duration (histogram)
// - request_errors_total: Total number of errors (counter, 4xx/5xx)
//
// Labels: method, route, code (low cardinality)
func REDMetricsMiddleware(metricsPort ports.MetricsPort) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Record start time
		start := time.Now()

		// Process request
		c.Next()

		// Calculate duration
		duration := time.Since(start).Seconds()

		// Extract labels (low cardinality)
		labels := map[string]string{
			"method": c.Request.Method,
			"route":  c.FullPath(), // Route pattern, not full path (avoids high cardinality)
			"code":   strconv.Itoa(c.Writer.Status()),
		}

		// If route is empty (404), use special marker
		if labels["route"] == "" {
			labels["route"] = "unknown"
		}

		// RED Metric 1: Rate - Total requests
		metricsPort.IncCounter("http_requests_total", labels)

		// RED Metric 2: Duration - Request duration histogram
		metricsPort.ObserveHistogram("http_request_duration_seconds", duration, labels)

		// RED Metric 3: Errors - Error counter (4xx, 5xx)
		if c.Writer.Status() >= 400 {
			metricsPort.IncCounter("http_request_errors_total", labels)
		}
	}
}

// HealthMetricsMiddleware tracks health check metrics specifically
// Sets a gauge for dependency readiness (can be used for custom readiness checks)
func HealthMetricsMiddleware(metricsPort ports.MetricsPort, dependencyName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// If this is a readiness check, track dependency status
		if c.Request.URL.Path == "/api/v1/ready" {
			ready := float64(0)
			if c.Writer.Status() == 200 {
				ready = 1
			}

			labels := map[string]string{
				"dependency": dependencyName,
			}

			metricsPort.SetGauge("service_dependency_ready", ready, labels)
		}
	}
}
