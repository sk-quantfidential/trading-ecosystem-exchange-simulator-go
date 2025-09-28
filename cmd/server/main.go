package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/quantfidential/trading-ecosystem/exchange-simulator-go/internal/config"
	"github.com/quantfidential/trading-ecosystem/exchange-simulator-go/internal/handlers"
	"github.com/quantfidential/trading-ecosystem/exchange-simulator-go/internal/services"
)

func main() {
	cfg := config.Load()

	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	logger.SetFormatter(&logrus.JSONFormatter{})

	exchangeService := services.NewExchangeService(cfg, logger)

	grpcServer := setupGRPCServer(cfg, exchangeService, logger)
	httpServer := setupHTTPServer(cfg, exchangeService, logger)

	go func() {
		logger.WithField("port", cfg.GRPCPort).Info("Starting gRPC server")
		if err := startGRPCServer(grpcServer, cfg.GRPCPort); err != nil {
			logger.WithError(err).Fatal("Failed to start gRPC server")
		}
	}()

	go func() {
		logger.WithField("port", cfg.HTTPPort).Info("Starting HTTP server")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.WithError(err).Fatal("Failed to start HTTP server")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down servers...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.WithError(err).Error("HTTP server forced to shutdown")
	}

	grpcServer.GracefulStop()
	logger.Info("Servers shutdown complete")
}

func setupGRPCServer(cfg *config.Config, exchangeService *services.ExchangeService, logger *logrus.Logger) *grpc.Server {
	server := grpc.NewServer()

	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(server, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	return server
}

func setupHTTPServer(cfg *config.Config, exchangeService *services.ExchangeService, logger *logrus.Logger) *http.Server {
	router := gin.New()
	router.Use(gin.Recovery())

	healthHandler := handlers.NewHealthHandler(logger)

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

func startGRPCServer(server *grpc.Server, port int) error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}
	return server.Serve(lis)
}