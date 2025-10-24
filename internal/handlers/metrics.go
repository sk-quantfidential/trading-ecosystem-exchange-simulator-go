package handlers

import (
	"github.com/gin-gonic/gin"

	"github.com/quantfidential/trading-ecosystem/exchange-simulator-go/internal/domain/ports"
)

// MetricsHandler handles observability metrics endpoint
// Uses Clean Architecture: depends on MetricsPort (interface), not concrete implementation
// This allows swapping Prometheus for OpenTelemetry without changing this handler
type MetricsHandler struct {
	metricsPort ports.MetricsPort
}

// NewMetricsHandler creates a new metrics handler
// metricsPort: abstraction for metrics collection (Prometheus, OpenTelemetry, etc.)
func NewMetricsHandler(metricsPort ports.MetricsPort) *MetricsHandler {
	return &MetricsHandler{
		metricsPort: metricsPort,
	}
}

// Metrics serves the metrics endpoint
// The actual format (Prometheus text, OpenMetrics, etc.) is determined by the adapter
func (h *MetricsHandler) Metrics(c *gin.Context) {
	handler := h.metricsPort.GetHTTPHandler()
	handler.ServeHTTP(c.Writer, c.Request)
}
