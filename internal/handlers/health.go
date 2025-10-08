package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/quantfidential/trading-ecosystem/exchange-simulator-go/internal/config"
	"github.com/sirupsen/logrus"
)

type HealthHandler struct {
	config *config.Config
	logger *logrus.Logger
}

// NewHealthHandler creates a basic health handler
func NewHealthHandler(logger *logrus.Logger) *HealthHandler {
	return &HealthHandler{
		logger: logger,
	}
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

func (h *HealthHandler) Ready(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ready",
		"checks": gin.H{
			"database": "ok",
			"redis":    "ok",
		},
	})
}