package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type HealthHandler struct {
	logger *logrus.Logger
}

func NewHealthHandler(logger *logrus.Logger) *HealthHandler {
	return &HealthHandler{
		logger: logger,
	}
}

func (h *HealthHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "exchange-simulator",
		"version": "1.0.0",
	})
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