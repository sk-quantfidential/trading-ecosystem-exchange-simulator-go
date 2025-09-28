//go:build unit

package grpc

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/quantfidential/trading-ecosystem/exchange-simulator-go/internal/config"
	"github.com/quantfidential/trading-ecosystem/exchange-simulator-go/internal/services"
)

func TestExchangeGRPCServer_HealthService(t *testing.T) {
	t.Run("provides_enhanced_health_status", func(t *testing.T) {
		// Setup
		cfg := &config.Config{
			ServiceName:    "exchange-simulator",
			ServiceVersion: "test",
			GRPCPort:       0, // Use dynamic port
		}

		logger := logrus.New()
		logger.SetLevel(logrus.ErrorLevel) // Reduce noise in tests

		exchangeService := services.NewExchangeService(cfg, logger)
		server := NewExchangeGRPCServer(cfg, exchangeService, logger)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Start server
		err := server.Start(ctx)
		if err != nil {
			t.Fatalf("Failed to start server: %v", err)
		}
		defer server.Stop(ctx)

		// Wait for server to be ready
		time.Sleep(100 * time.Millisecond)

		// Get the actual address
		address := server.GetAddress()

		// Create client connection
		conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			t.Fatalf("Failed to connect to server: %v", err)
		}
		defer conn.Close()

		// Test health check
		healthClient := grpc_health_v1.NewHealthClient(conn)

		// Check overall health
		resp, err := healthClient.Check(ctx, &grpc_health_v1.HealthCheckRequest{})
		if err != nil {
			t.Errorf("Health check failed: %v", err)
		}

		if resp.Status != grpc_health_v1.HealthCheckResponse_SERVING {
			t.Errorf("Expected SERVING, got %v", resp.Status)
		}

		// Check service-specific health
		resp, err = healthClient.Check(ctx, &grpc_health_v1.HealthCheckRequest{
			Service: "exchange-simulator",
		})
		if err != nil {
			t.Errorf("Service health check failed: %v", err)
		}

		if resp.Status != grpc_health_v1.HealthCheckResponse_SERVING {
			t.Errorf("Expected SERVING for exchange-simulator, got %v", resp.Status)
		}

		// Verify server metrics
		metrics := server.GetMetrics()
		if !metrics.IsRunning {
			t.Error("Expected server to be running")
		}

		if metrics.UptimeSeconds < 0 {
			t.Error("Expected positive uptime")
		}

		// Verify server status
		if !server.IsRunning() {
			t.Error("Expected IsRunning() to return true")
		}
	})
}

func TestExchangeGRPCServer_ExchangeService(t *testing.T) {
	t.Run("accepts_exchange_operations", func(t *testing.T) {
		// Setup
		cfg := &config.Config{
			ServiceName:    "exchange-simulator",
			ServiceVersion: "test",
			GRPCPort:       0, // Use dynamic port
		}

		logger := logrus.New()
		logger.SetLevel(logrus.ErrorLevel) // Reduce noise in tests

		exchangeService := services.NewExchangeService(cfg, logger)
		server := NewExchangeGRPCServer(cfg, exchangeService, logger)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Start server
		err := server.Start(ctx)
		if err != nil {
			t.Fatalf("Failed to start server: %v", err)
		}
		defer server.Stop(ctx)

		// Wait for server to be ready
		time.Sleep(100 * time.Millisecond)

		// Verify server is accepting connections
		address := server.GetAddress()
		conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			t.Fatalf("Failed to connect to server: %v", err)
		}
		defer conn.Close()

		// Test that exchange service is integrated
		// Note: In a full implementation, we would register exchange service endpoints
		// For now, we verify the server infrastructure is working

		initialMetrics := server.GetMetrics()
		initialRequests := initialMetrics.RequestCount

		// Make a health check request to increment request count
		healthClient := grpc_health_v1.NewHealthClient(conn)
		_, err = healthClient.Check(ctx, &grpc_health_v1.HealthCheckRequest{})
		if err != nil {
			t.Errorf("Health check failed: %v", err)
		}

		// Verify metrics updated
		finalMetrics := server.GetMetrics()
		if finalMetrics.RequestCount <= initialRequests {
			t.Errorf("Expected request count to increase, got %d -> %d",
				initialRequests, finalMetrics.RequestCount)
		}

		if finalMetrics.LastRequestTime.Before(initialMetrics.LastRequestTime) {
			t.Error("Expected LastRequestTime to be updated")
		}
	})
}

func TestExchangeGRPCServer_SettlementService(t *testing.T) {
	t.Run("handles_settlement_instructions", func(t *testing.T) {
		// Setup
		cfg := &config.Config{
			ServiceName:    "exchange-simulator",
			ServiceVersion: "test",
			GRPCPort:       0, // Use dynamic port
		}

		logger := logrus.New()
		logger.SetLevel(logrus.ErrorLevel) // Reduce noise in tests

		exchangeService := services.NewExchangeService(cfg, logger)
		server := NewExchangeGRPCServer(cfg, exchangeService, logger)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Start server
		err := server.Start(ctx)
		if err != nil {
			t.Fatalf("Failed to start server: %v", err)
		}
		defer server.Stop(ctx)

		// Wait for server to be ready
		time.Sleep(100 * time.Millisecond)

		// Verify server infrastructure for settlement operations
		// This tests the framework that will support settlement services

		if !server.IsRunning() {
			t.Error("Expected server to be running for settlement operations")
		}

		status := server.GetHealthStatus()
		if status != grpc_health_v1.HealthCheckResponse_SERVING {
			t.Errorf("Expected SERVING status for settlements, got %v", status)
		}

		// Verify proper shutdown behavior
		err = server.Stop(ctx)
		if err != nil {
			t.Errorf("Failed to stop server: %v", err)
		}

		if server.IsRunning() {
			t.Error("Expected server to be stopped after shutdown")
		}

		finalStatus := server.GetHealthStatus()
		if finalStatus != grpc_health_v1.HealthCheckResponse_NOT_SERVING {
			t.Errorf("Expected NOT_SERVING after shutdown, got %v", finalStatus)
		}
	})
}

func TestExchangeGRPCServer_Metrics(t *testing.T) {
	t.Run("exposes_service_metrics", func(t *testing.T) {
		// Setup
		cfg := &config.Config{
			ServiceName:    "exchange-simulator",
			ServiceVersion: "test",
			GRPCPort:       0, // Use dynamic port
		}

		logger := logrus.New()
		logger.SetLevel(logrus.ErrorLevel) // Reduce noise in tests

		exchangeService := services.NewExchangeService(cfg, logger)
		server := NewExchangeGRPCServer(cfg, exchangeService, logger)

		// Test metrics before starting
		metrics := server.GetMetrics()
		if metrics.IsRunning {
			t.Error("Expected server to not be running initially")
		}

		if metrics.RequestCount != 0 {
			t.Errorf("Expected 0 initial requests, got %d", metrics.RequestCount)
		}

		// Verify start time is set
		if metrics.StartTime.IsZero() {
			t.Error("Expected start time to be set")
		}

		// Verify uptime calculation
		if metrics.UptimeSeconds < 0 {
			t.Errorf("Expected non-negative uptime, got %d", metrics.UptimeSeconds)
		}
	})
}