# Pull Request: TSE-0001.12.0b - Prometheus Metrics with Clean Architecture (exchange-simulator-go)

**Epic:** TSE-0001 - Foundation Services & Infrastructure
**Milestone:** TSE-0001.12.0b - Prometheus Metrics Foundation
**Branch:** `feature/TSE-0001.12.0-prometheus-metric-client`
**Repository:** exchange-simulator-go
**Status:** ✅ Ready for Merge

## Summary

This PR implements Prometheus metrics collection using Clean Architecture principles, enabling future migration to OpenTelemetry without changing domain logic. This follows the RED pattern (Rate, Errors, Duration) for comprehensive HTTP request monitoring.

### Key Features

1. **Clean Architecture**: Domain layer never depends on infrastructure (Prometheus)
2. **RED Pattern Metrics**: Rate, Errors, Duration for all HTTP requests
3. **Low Cardinality Labels**: Constant labels (service, instance, version) + request labels (method, route, code)
4. **Thread-Safe**: Lazy metric initialization with sync.RWMutex
5. **Comprehensive Tests**: 9 unit tests following BDD pattern

## Architecture Pattern

### Clean Architecture - Ports & Adapters

```
Domain Layer (internal/domain/ports)
↓ depends on
MetricsPort interface (abstraction)
↑ implements
Infrastructure Layer (internal/infrastructure/observability)
PrometheusMetricsAdapter (concrete implementation)
```

**Key Principle**: Domain code depends on `MetricsPort` interface, NOT on Prometheus.
**Future-Proofing**: Swap Prometheus for OpenTelemetry by changing the adapter only.

## Changes

### 1. Domain Layer - MetricsPort Interface (NEW)

**File:** `internal/domain/ports/metrics.go`

```go
package ports

import "net/http"

// MetricsPort defines the interface for observability metrics collection
// This port abstracts the metrics implementation (Prometheus, OpenTelemetry, etc.)
// following Clean Architecture principles: domain doesn't depend on infrastructure
type MetricsPort interface {
	// RED Pattern - Request Layer Metrics

	// IncCounter increments a counter metric
	IncCounter(name string, labels map[string]string)

	// ObserveHistogram records a value in a histogram metric
	ObserveHistogram(name string, value float64, labels map[string]string)

	// SetGauge sets a gauge metric to a specific value
	SetGauge(name string, value float64, labels map[string]string)

	// HTTP Serving
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

func (l *MetricsLabels) ToMap() map[string]string
func (l *MetricsLabels) ConstantLabels() map[string]string
```

**Purpose**: Domain-level abstraction for metrics. No Prometheus dependencies.

### 2. Infrastructure Layer - Prometheus Adapter (NEW)

**File:** `internal/infrastructure/observability/prometheus_adapter.go`

```go
package observability

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/quantfidential/trading-ecosystem/exchange-simulator-go/internal/domain/ports"
)

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

func NewPrometheusMetricsAdapter(constantLabels map[string]string) *PrometheusMetricsAdapter
```

**Key Features**:
- Separate Prometheus registry (isolated from default)
- Lazy metric initialization (thread-safe with double-checked locking)
- Constant labels applied to all metrics (service, instance, version)
- Go runtime metrics automatically registered
- Histogram buckets: 5ms to 10s

### 3. RED Metrics Middleware (NEW)

**File:** `internal/infrastructure/observability/middleware.go`

```go
package observability

// REDMetricsMiddleware creates Gin middleware for RED pattern metrics
// RED: Rate (requests_total), Errors (request_errors_total), Duration (request_duration_seconds)
func REDMetricsMiddleware(metricsPort ports.MetricsPort) gin.HandlerFunc
```

**Metrics Collected**:
1. **Rate**: `http_requests_total` (counter) - Total number of requests
2. **Errors**: `http_request_errors_total` (counter) - Requests with 4xx/5xx status codes
3. **Duration**: `http_request_duration_seconds` (histogram) - Request latency

**Labels** (low cardinality):
- `method`: HTTP method (GET, POST, etc.)
- `route`: Route pattern (e.g., `/api/v1/health`, NOT full path)
- `code`: HTTP status code (200, 404, 500)

**Low Cardinality Pattern**:
- ✅ Route pattern: `/api/v1/orders/:id` (low cardinality)
- ❌ Full path: `/api/v1/orders/123456` (high cardinality - AVOIDED)

### 4. Metrics Handler (NEW)

**File:** `internal/handlers/metrics.go`

```go
package handlers

// MetricsHandler handles observability metrics endpoint
// Uses Clean Architecture: depends on MetricsPort (interface), not concrete implementation
type MetricsHandler struct {
	metricsPort ports.MetricsPort
}

func NewMetricsHandler(metricsPort ports.MetricsPort) *MetricsHandler

// Metrics serves the metrics endpoint
func (h *MetricsHandler) Metrics(c *gin.Context)
```

**Purpose**: Serves `/metrics` endpoint via the MetricsPort interface.

### 5. Configuration Enhancement

**File:** `internal/config/config.go`

```go
type Config struct {
	// ... existing fields

	// Metrics
	metricsPort ports.MetricsPort
}

func (c *Config) SetMetricsPort(metricsPort ports.MetricsPort)
func (c *Config) GetMetricsPort() ports.MetricsPort
```

### 6. Main Server Integration

**File:** `cmd/server/main.go`

```go
func main() {
	cfg := config.Load()
	logger := logrus.New()

	// Initialize Prometheus Metrics Adapter
	constantLabels := (&ports.MetricsLabels{
		Service:  cfg.ServiceName,
		Instance: cfg.ServiceInstanceName,
		Version:  cfg.ServiceVersion,
	}).ConstantLabels()
	metricsPort := observability.NewPrometheusMetricsAdapter(constantLabels)
	cfg.SetMetricsPort(metricsPort)

	// ... rest of startup
}

func setupHTTPServer(cfg *config.Config, ...) *http.Server {
	router := gin.New()
	router.Use(gin.Recovery())

	// Add RED metrics middleware for all routes
	metricsPort := cfg.GetMetricsPort()
	if metricsPort != nil {
		router.Use(observability.REDMetricsMiddleware(metricsPort))
		router.Use(observability.HealthMetricsMiddleware(metricsPort, "exchange-simulator"))
	}

	metricsHandler := handlers.NewMetricsHandler(metricsPort)

	v1 := router.Group("/api/v1")
	{
		v1.GET("/health", healthHandler.Health)
		v1.GET("/ready", healthHandler.Ready)
	}

	// Metrics endpoint (outside v1 group, at root level)
	router.GET("/metrics", metricsHandler.Metrics)

	return &http.Server{...}
}
```

## Test Coverage (9 Tests)

### Metrics Handler Tests (5 tests)

**File:** `internal/handlers/metrics_test.go`

```
TestMetricsHandler_Metrics:
✅ exposes_prometheus_metrics_through_port
✅ returns_text_plain_content_type
✅ includes_standard_go_runtime_metrics
✅ metrics_are_parseable_by_prometheus

TestMetricsHandler_Integration:
✅ metrics_endpoint_works_in_full_router
```

### Middleware Tests (4 tests)

**File:** `internal/infrastructure/observability/middleware_test.go`

```
TestREDMetricsMiddleware:
✅ instruments_successful_requests_with_RED_metrics
✅ instruments_error_requests_with_error_counter
✅ uses_route_pattern_not_full_path_for_low_cardinality
✅ handles_unknown_routes_gracefully
```

**All tests follow BDD Given/When/Then pattern.**

## Metrics Exposed

### Standard Go Runtime Metrics

```
# Go Metrics
go_gc_duration_seconds
go_goroutines
go_memstats_alloc_bytes
go_threads
process_cpu_seconds_total
process_resident_memory_bytes
```

### RED Pattern Metrics

```
# Rate - Total requests
http_requests_total{service="exchange-simulator",instance="exchange-simulator",version="1.0.0",method="GET",route="/api/v1/health",code="200"}

# Duration - Request latency histogram
http_request_duration_seconds{service="exchange-simulator",instance="exchange-simulator",version="1.0.0",method="GET",route="/api/v1/health",code="200"}
http_request_duration_seconds_bucket{le="0.005"}  # 5ms
http_request_duration_seconds_bucket{le="0.01"}   # 10ms
http_request_duration_seconds_bucket{le="0.1"}    # 100ms
http_request_duration_seconds_bucket{le="1"}      # 1s
http_request_duration_seconds_bucket{le="10"}     # 10s

# Errors - Failed requests (4xx, 5xx)
http_request_errors_total{service="exchange-simulator",instance="exchange-simulator",version="1.0.0",method="GET",route="/unknown",code="404"}
```

### Dependency Metrics

```
# Service dependencies readiness
service_dependency_ready{service="exchange-simulator",instance="exchange-simulator",version="1.0.0",dependency="exchange-simulator"} 1
```

## Testing Instructions

### Run Unit Tests

```bash
cd /home/skingham/Projects/Quantfidential/trading-ecosystem/exchange-simulator-go

# Run all tests with unit tag
go test -v -tags=unit ./internal/handlers/... ./internal/infrastructure/observability/...

# Expected: 9/9 tests passing
```

### Verify Metrics Endpoint

```bash
# Start the service
./server

# Query metrics endpoint
curl http://localhost:8082/metrics

# Expected output:
# # HELP http_requests_total ...
# # TYPE http_requests_total counter
# http_requests_total{...} 123
# # HELP http_request_duration_seconds ...
# # TYPE http_request_duration_seconds histogram
# http_request_duration_seconds_bucket{le="0.005",...} 100
# ...
# go_goroutines 42
# process_cpu_seconds_total 1.23
```

### Verify RED Metrics

```bash
# Make a request to generate metrics
curl http://localhost:8082/api/v1/health

# Query metrics
curl http://localhost:8082/metrics | grep http_requests_total

# Expected: Counter incremented with labels
# http_requests_total{service="exchange-simulator",instance="exchange-simulator",version="1.0.0",method="GET",route="/api/v1/health",code="200"} 1
```

### Verify Low Cardinality

```bash
# Make requests with different IDs
curl http://localhost:8082/api/v1/orders/123
curl http://localhost:8082/api/v1/orders/456
curl http://localhost:8082/api/v1/orders/789

# Query metrics
curl http://localhost:8082/metrics | grep route=

# Expected: Single route pattern (low cardinality)
# route="/api/v1/orders/:id"

# NOT expected: Multiple full paths (high cardinality)
# route="/api/v1/orders/123" ← WRONG
# route="/api/v1/orders/456" ← WRONG
```

## Migration Notes

### No Breaking Changes

✅ **Backward Compatible**
- Metrics are opt-in (initialize metricsPort to enable)
- Service continues to work without metrics
- No changes to existing endpoints or handlers

### Future Migration to OpenTelemetry

**Current (Prometheus)**:
```go
import "github.com/quantfidential/trading-ecosystem/exchange-simulator-go/internal/infrastructure/observability"

metricsPort := observability.NewPrometheusMetricsAdapter(constantLabels)
```

**Future (OpenTelemetry)**:
```go
import "github.com/quantfidential/trading-ecosystem/exchange-simulator-go/internal/infrastructure/observability"

metricsPort := observability.NewOpenTelemetryMetricsAdapter(constantLabels)
```

**No changes required**:
- ✅ Domain code (ports, middleware, handlers) - unchanged
- ✅ main.go - single line change
- ✅ Tests - unchanged (mock MetricsPort)

## Files Changed

**Modified:**
- `internal/config/config.go` (added metricsPort field and methods)
- `cmd/server/main.go` (Prometheus initialization, middleware setup)

**New:**
- `internal/domain/ports/metrics.go` (MetricsPort interface, MetricsLabels)
- `internal/infrastructure/observability/prometheus_adapter.go` (Prometheus implementation)
- `internal/infrastructure/observability/middleware.go` (RED metrics middleware)
- `internal/handlers/metrics.go` (Metrics endpoint handler)
- `internal/handlers/metrics_test.go` (5 handler tests)
- `internal/infrastructure/observability/middleware_test.go` (4 middleware tests)
- `docs/prs/feature-TSE-0001.12.0-prometheus-metric-client.md` (this file)

## Dependencies

**Added:**
- `github.com/prometheus/client_golang` v1.23.2
- `github.com/prometheus/client_model` v0.6.2
- `github.com/prometheus/common` v0.66.1
- `github.com/prometheus/procfs` v0.16.1

## Related Work

### Cross-Repository Epic (TSE-0001.12.0)

This exchange-simulator-go implementation follows the same pattern as:
- ✅ audit-correlator-go (Prometheus metrics - completed)
- ✅ custodian-simulator-go (Prometheus metrics - completed)
- ✅ exchange-simulator-go (Prometheus metrics - this PR)
- 🔲 orchestrator-docker (Prometheus config - next)

## Merge Checklist

- [x] MetricsPort interface created (domain layer)
- [x] PrometheusMetricsAdapter implemented (infrastructure layer)
- [x] RED metrics middleware implemented
- [x] Metrics handler implemented
- [x] Config integration (SetMetricsPort/GetMetricsPort)
- [x] Main server integration (initialization + middleware)
- [x] 9 unit tests passing (5 handler + 4 middleware)
- [x] All tests follow BDD Given/When/Then pattern
- [x] Low cardinality labels (route pattern, not full path)
- [x] Thread-safe lazy initialization
- [x] Clean Architecture preserved (domain → interface ← infrastructure)
- [x] No breaking changes
- [x] PR documentation complete

## Approval

**Ready for Merge**: ✅ Yes

All requirements satisfied:
- ✅ Clean Architecture with ports/adapters pattern
- ✅ RED pattern metrics (Rate, Errors, Duration)
- ✅ Low cardinality labels (constant + request)
- ✅ Thread-safe metric collection
- ✅ 9/9 unit tests passing
- ✅ Future-proof for OpenTelemetry migration
- ✅ No breaking changes
- ✅ Comprehensive test coverage

---

**Epic:** TSE-0001.12.0b
**Repository:** exchange-simulator-go
**Test Coverage:** 9/9 tests passing
**Pattern:** Clean Architecture with Prometheus metrics

🎯 **Foundation for:** Grafana dashboards, alerting, SRE monitoring

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>
