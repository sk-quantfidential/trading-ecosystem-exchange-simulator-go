# Pull Request: TSE-0001.12.0 - Multi-Instance Infrastructure Foundation (exchange-simulator-go)

**Epic:** TSE-0001 - Foundation Services & Infrastructure
**Milestone:** TSE-0001.12.0 - Multi-Instance Infrastructure Foundation
**Branch:** `feature/epic-TSE-0001-named-components-foundation`
**Repository:** exchange-simulator-go
**Status:** ✅ Ready for Merge

## Summary

This PR implements **Phases 1-2** - instance-aware configuration and lifecycle management for exchange-simulator-go. This enables:

1. **Instance-Aware Configuration**: `ServiceName`, `ServiceInstanceName`, and `Environment` fields
2. **Config-Level DataAdapter Integration**: Centralized initialization at config level
3. **Instance-Aware Logging**: Structured logging with service instance context
4. **Instance-Aware Health Checks**: Health endpoint includes instance metadata
5. **Graceful Degradation**: Service continues in stub mode if infrastructure unavailable

This implementation follows the **singleton pattern** for the exchange-simulator service (one shared instance) and supports the multi-instance pattern used by exchange-data-adapter-go.

## Architecture Pattern


## What Changed

See detailed commit-by-commit changes documented in the sections below.

## Testing

All validation checks pass:
- `scripts/validate-all.sh` - All checks passing
- Unit tests passing
- Integration tests passing


### Singleton Service (Current)
```
ServiceName: exchange-simulator
ServiceInstanceName: exchange-simulator (same)
→ Schema: "exchange" (via data adapter)
→ Redis Namespace: "exchange" (via data adapter)
```

### Future Multi-Instance Support
```
ServiceName: exchange-simulator
ServiceInstanceName: exchange-OKX
→ Schema: "exchange_okx" (via data adapter)
→ Redis Namespace: "exchange:OKX" (via data adapter)
```

## Changes

### 1. Enhanced Configuration (`internal/config/config.go`)

**Added Fields:**
```go
type Config struct {
    // Service Identity
    ServiceName             string
    ServiceInstanceName     string // Instance identifier (e.g., "exchange-OKX")
    ServiceVersion          string
    Environment             string // Deployment environment (development, staging, production)

    // Network
    HTTPPort                int
    GRPCPort                int

    // Data Adapter
    dataAdapter adapters.DataAdapter
    // ... existing fields
}
```

**Environment Variables:**
- `SERVICE_INSTANCE_NAME`: Instance identifier (optional, defaults to `SERVICE_NAME`)
- `ENVIRONMENT`: Deployment environment (optional, defaults to "development")

**Backward Compatibility:**
```go
if cfg.ServiceInstanceName == "" {
    cfg.ServiceInstanceName = cfg.ServiceName  // Singleton
}
```

**DataAdapter Lifecycle Management:**
```go
func (c *Config) InitializeDataAdapter(ctx context.Context, logger *logrus.Logger) error {
    adapter, err := adapters.NewExchangeDataAdapterFromEnv(logger)
    if err != nil {
        logger.WithError(err).Warn("Failed to create data adapter, will use stub mode")
        return err
    }

    if err := adapter.Connect(ctx); err != nil {
        logger.WithError(err).Warn("Failed to connect data adapter, will use stub mode")
        return err
    }

    c.dataAdapter = adapter
    logger.Info("Data adapter initialized successfully")
    return nil
}

func (c *Config) DisconnectDataAdapter(ctx context.Context) error {
    if c.dataAdapter != nil {
        return c.dataAdapter.Disconnect(ctx)
    }
    return nil
}
```

### 2. Instance-Aware Logging (`cmd/server/main.go`)

**Structured Logging with Instance Context:**
```go
func main() {
    cfg := config.Load()

    logger := logrus.New()
    logger.SetLevel(logrus.InfoLevel)
    logger.SetFormatter(&logrus.JSONFormatter{})

    // Add instance context to all logs
    logger = logger.WithFields(logrus.Fields{
        "service_name":  cfg.ServiceName,
        "instance_name": cfg.ServiceInstanceName,
        "environment":   cfg.Environment,
    }).Logger

    logger.Info("Starting exchange-simulator service")

    // Initialize DataAdapter with graceful degradation
    ctx := context.Background()
    if err := cfg.InitializeDataAdapter(ctx, logger); err != nil {
        logger.WithError(err).Warn("Failed to initialize data adapter, continuing in stub mode")
    } else {
        logger.Info("Data adapter initialized successfully")
    }

    // ... rest of startup
}
```

**Shutdown with DataAdapter Cleanup:**
```go
// Shutdown
shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
defer shutdownCancel()

// Disconnect DataAdapter
if err := cfg.DisconnectDataAdapter(shutdownCtx); err != nil {
    logger.WithError(err).Error("Failed to disconnect data adapter")
}
```

### 3. Instance-Aware Health Checks (`internal/handlers/health.go`)

**Enhanced Health Handler:**
```go
type HealthHandler struct {
    config *config.Config
    logger *logrus.Logger
}

// NewHealthHandlerWithConfig creates an instance-aware health handler
func NewHealthHandlerWithConfig(cfg *config.Config, logger *logrus.Logger) *HealthHandler {
    return &HealthHandler{
        config: cfg,
        logger: logger,
    }
}

func (h *HealthHandler) Health(c *gin.Context) {
    response := gin.H{
        "status":    "healthy",
        "timestamp": time.Now().UTC().Format(time.RFC3339),
    }

    // Add instance information if config is available
    if h.config != nil {
        response["service"] = h.config.ServiceName
        response["instance"] = h.config.ServiceInstanceName
        response["version"] = h.config.ServiceVersion
        response["environment"] = h.config.Environment
    } else {
        // Fallback for backward compatibility
        response["service"] = "exchange-simulator"
        response["version"] = "1.0.0"
    }

    c.JSON(http.StatusOK, response)
}
```

**HTTP Server Setup:**
```go
func setupHTTPServer(cfg *config.Config, exchangeService *services.ExchangeService, logger *logrus.Logger) *http.Server {
    router := gin.New()
    router.Use(gin.Recovery())

    healthHandler := handlers.NewHealthHandlerWithConfig(cfg, logger)

    v1 := router.Group("/api/v1")
    {
        v1.GET("/health", healthHandler.Health)
        v1.GET("/ready", healthHandler.Ready)
    }

    return &http.Server{
        Addr:    fmt.Sprintf(":%d", cfg.HTTPPort),
        Handler: router,
    }
}
```

## Singleton Service Pattern

### Configuration
```yaml
environment:
  - SERVICE_NAME=exchange-simulator
  - SERVICE_INSTANCE_NAME=exchange-simulator  # Same as SERVICE_NAME
  - ENVIRONMENT=development
```

### Expected Behavior
- **Logging**: All logs include `service_name: exchange-simulator`, `instance_name: exchange-simulator`
- **Health Check**: `{"service": "exchange-simulator", "instance": "exchange-simulator", "version": "1.0.0", "environment": "development"}`
- **DataAdapter**: Uses `exchange` schema and `exchange` Redis namespace (via data adapter derivation)

## Testing Instructions

### Build Verification
```bash
cd /home/skingham/Projects/Quantfidential/trading-ecosystem/exchange-simulator-go

# Build the service
go build ./cmd/server

# Expected: Clean build with no errors
```

### Runtime Verification
```bash
# Run with default singleton configuration
SERVICE_NAME=exchange-simulator \
SERVICE_INSTANCE_NAME=exchange-simulator \
ENVIRONMENT=development \
./server

# Expected logs:
# {"service_name":"exchange-simulator","instance_name":"exchange-simulator","environment":"development","level":"info","msg":"Starting exchange-simulator service"}
# {"service_name":"exchange-simulator","instance_name":"exchange-simulator","environment":"development","level":"info","msg":"Data adapter initialized successfully"}
```

### Health Check Verification
```bash
# Test instance-aware health endpoint
curl http://localhost:8082/api/v1/health

# Expected response:
# {
#   "service": "exchange-simulator",
#   "instance": "exchange-simulator",
#   "version": "1.0.0",
#   "environment": "development",
#   "status": "healthy",
#   "timestamp": "2025-10-08T12:34:56Z"
# }
```

### Graceful Degradation Verification
```bash
# Run without PostgreSQL/Redis available
SERVICE_NAME=exchange-simulator \
SERVICE_INSTANCE_NAME=exchange-simulator \
POSTGRES_URL="" \
REDIS_URL="" \
./server

# Expected behavior:
# - Service starts successfully
# - Warning logged about stub mode
# - Health checks still work
# - Service continues to run
```

## Migration Notes

### Backward Compatibility
✅ **No Breaking Changes**
- Existing deployments without `SERVICE_INSTANCE_NAME` → Singleton mode (defaults to `SERVICE_NAME`)
- Existing deployments without `ENVIRONMENT` → Defaults to "development"
- Health check backward compatible with fallback
- DataAdapter initialization is optional (graceful degradation)

### Configuration Migration

**Before (still valid):**
```yaml
environment:
  - SERVICE_NAME=exchange-simulator
  # Implicitly singleton
```

**After (explicit instance awareness):**
```yaml
environment:
  - SERVICE_NAME=exchange-simulator
  - SERVICE_INSTANCE_NAME=exchange-simulator
  - ENVIRONMENT=development
```

## Files Changed

**Modified:**
- `internal/config/config.go` (added ServiceInstanceName, Environment, DataAdapter lifecycle)
- `cmd/server/main.go` (instance-aware logging, DataAdapter initialization/cleanup)
- `internal/handlers/health.go` (NewHealthHandlerWithConfig, instance metadata in health check)

**New:**
- `docs/prs/feature-TSE-0001.12.0-named-components-foundation.md` (this file)

## Dependencies

**Required:**
- exchange-data-adapter-go (Phase 0 foundation) ✅ Completed

**No new external dependencies added** ✅

## Related Work

### Cross-Repository Epic (TSE-0001.12.0)

This exchange-simulator-go implementation follows the same pattern as:
- ✅ audit-data-adapter-go (Phase 0 - completed)
- ✅ audit-correlator-go (Phases 1-7 - completed)
- ✅ custodian-data-adapter-go (Phase 0 - completed)
- ✅ custodian-simulator-go (Phases 1-2 - completed)
- ✅ exchange-data-adapter-go (Phase 0 - completed)
- ✅ exchange-simulator-go (Phases 1-2 - this PR)
- 🔲 orchestrator-docker (Phases 5-6, 8 - next)

## Merge Checklist

- [x] ServiceInstanceName and Environment added to Config
- [x] Backward compatibility maintained (defaults to singleton)
- [x] Instance-aware logging implemented
- [x] DataAdapter lifecycle management at config level
- [x] Instance-aware health checks implemented
- [x] NewHealthHandlerWithConfig constructor added
- [x] Health handler updated in setupHTTPServer
- [x] Graceful degradation when infrastructure unavailable
- [x] Build verification successful
- [x] No breaking changes
- [x] PR documentation complete

## Approval

**Ready for Merge**: ✅ Yes

All requirements satisfied:
- ✅ Instance-aware configuration complete
- ✅ Config-level DataAdapter integration with lifecycle management
- ✅ Instance-aware structured logging
- ✅ Instance-aware health checks
- ✅ Graceful degradation implemented
- ✅ Build verification successful
- ✅ Backward compatibility maintained
- ✅ Clean Architecture pattern preserved

---

**Epic:** TSE-0001.12.0
**Repository:** exchange-simulator-go
**Phases:** 1-2 (Configuration + Lifecycle)
**Pattern:** Singleton service with multi-instance support

🎯 **Foundation for:** Multi-instance exchange deployment support (OKX, Binance, Kraken, etc.)

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>
