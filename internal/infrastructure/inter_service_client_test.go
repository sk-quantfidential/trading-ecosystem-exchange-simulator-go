//go:build unit

package infrastructure

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/quantfidential/trading-ecosystem/exchange-simulator-go/internal/config"
)

func TestInterServiceClientManager_Creation(t *testing.T) {
	t.Run("creates_manager_successfully", func(t *testing.T) {
		cfg := &config.Config{
			ServiceName: "exchange-simulator",
		}

		logger := logrus.New()
		logger.SetLevel(logrus.ErrorLevel)

		manager := NewInterServiceClientManager(cfg, logger,
			&ServiceDiscoveryClient{},
			&ConfigurationClient{})

		// Verify manager is initialized
		if manager == nil {
			t.Fatal("Expected manager to be created")
		}

		if manager.config.ServiceName != "exchange-simulator" {
			t.Errorf("Expected service name 'exchange-simulator', got %s", manager.config.ServiceName)
		}

		metrics := manager.GetMetrics()
		if metrics.ActiveConnections != 0 {
			t.Errorf("Expected 0 active connections, got %d", metrics.ActiveConnections)
		}

		if manager.connections == nil {
			t.Error("Expected connections map to be initialized")
		}

		if manager.clients == nil {
			t.Error("Expected clients map to be initialized")
		}
	})
}

func TestInterServiceClientManager_ClientStorage(t *testing.T) {
	t.Run("stores_and_retrieves_clients", func(t *testing.T) {
		cfg := &config.Config{
			ServiceName: "exchange-simulator",
		}

		logger := logrus.New()
		logger.SetLevel(logrus.ErrorLevel)

		manager := NewInterServiceClientManager(cfg, logger,
			&ServiceDiscoveryClient{},
			&ConfigurationClient{})

		// Test that no client exists initially
		_, exists := manager.getClient("test-service")
		if exists {
			t.Error("Expected no client to exist initially")
		}

		// Store a mock client
		mockClient := "mock-client-data"
		manager.setClient("test-service", mockClient)

		// Retrieve the client
		retrievedClient, exists := manager.getClient("test-service")
		if !exists {
			t.Error("Expected client to exist after setting")
		}

		if retrievedClient != mockClient {
			t.Error("Expected to retrieve the same client that was stored")
		}
	})
}

func TestInterServiceClientManager_Metrics(t *testing.T) {
	t.Run("tracks_comprehensive_metrics", func(t *testing.T) {
		cfg := &config.Config{
			ServiceName: "exchange-simulator",
		}

		logger := logrus.New()
		logger.SetLevel(logrus.ErrorLevel)

		manager := NewInterServiceClientManager(cfg, logger,
			&ServiceDiscoveryClient{},
			&ConfigurationClient{})

		// Test initial metrics
		metrics := manager.GetMetrics()
		if metrics.ActiveConnections != 0 {
			t.Errorf("Expected 0 active connections, got %d", metrics.ActiveConnections)
		}

		if metrics.TotalConnections != 0 {
			t.Errorf("Expected 0 total connections, got %d", metrics.TotalConnections)
		}

		if metrics.ServiceCallCount != 0 {
			t.Errorf("Expected 0 service calls, got %d", metrics.ServiceCallCount)
		}

		// Test metric increments
		manager.incrementTotalConnection()
		manager.incrementServiceCall()
		manager.incrementServiceCallError()
		manager.incrementFailedConnection()
		manager.updateActiveConnections(2)

		updatedMetrics := manager.GetMetrics()
		if updatedMetrics.TotalConnections != 1 {
			t.Errorf("Expected 1 total connection, got %d", updatedMetrics.TotalConnections)
		}

		if updatedMetrics.ServiceCallCount != 1 {
			t.Errorf("Expected 1 service call, got %d", updatedMetrics.ServiceCallCount)
		}

		if updatedMetrics.ServiceCallErrors != 1 {
			t.Errorf("Expected 1 service call error, got %d", updatedMetrics.ServiceCallErrors)
		}

		if updatedMetrics.FailedConnections != 1 {
			t.Errorf("Expected 1 failed connection, got %d", updatedMetrics.FailedConnections)
		}

		if updatedMetrics.ActiveConnections != 2 {
			t.Errorf("Expected 2 active connections, got %d", updatedMetrics.ActiveConnections)
		}
	})

	t.Run("tracks_connection_attempts", func(t *testing.T) {
		cfg := &config.Config{
			ServiceName: "exchange-simulator",
		}

		logger := logrus.New()
		logger.SetLevel(logrus.ErrorLevel)

		manager := NewInterServiceClientManager(cfg, logger,
			&ServiceDiscoveryClient{},
			&ConfigurationClient{})

		initialMetrics := manager.GetMetrics()
		if !initialMetrics.LastConnectionAttempt.IsZero() {
			t.Error("Expected initial connection attempt time to be zero")
		}

		manager.incrementConnectionAttempt()

		updatedMetrics := manager.GetMetrics()
		if updatedMetrics.LastConnectionAttempt.IsZero() {
			t.Error("Expected connection attempt time to be updated")
		}
	})
}

func TestInterServiceClientManager_Close(t *testing.T) {
	t.Run("closes_successfully_with_no_connections", func(t *testing.T) {
		cfg := &config.Config{
			ServiceName: "exchange-simulator",
		}

		logger := logrus.New()
		logger.SetLevel(logrus.ErrorLevel)

		manager := NewInterServiceClientManager(cfg, logger,
			&ServiceDiscoveryClient{},
			&ConfigurationClient{})

		err := manager.Close()
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Verify connections and clients are cleared
		if len(manager.connections) != 0 {
			t.Errorf("Expected 0 connections, got %d", len(manager.connections))
		}

		if len(manager.clients) != 0 {
			t.Errorf("Expected 0 clients, got %d", len(manager.clients))
		}

		metrics := manager.GetMetrics()
		if metrics.ActiveConnections != 0 {
			t.Errorf("Expected 0 active connections after close, got %d", metrics.ActiveConnections)
		}
	})
}

func TestAuditCorrelatorClient_SubmitAuditEvent(t *testing.T) {
	t.Run("submits_audit_event", func(t *testing.T) {
		logger := logrus.New()
		logger.SetLevel(logrus.ErrorLevel)

		client := &auditCorrelatorClientImpl{
			conn:   nil, // We're not testing the gRPC connection here
			logger: logger,
		}

		ctx := context.Background()
		event := map[string]interface{}{
			"type":      "trade",
			"amount":    1000.0,
			"timestamp": time.Now(),
		}

		err := client.SubmitAuditEvent(ctx, event)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})
}

func TestCustodianSimulatorClient_ProcessSettlement(t *testing.T) {
	t.Run("processes_settlement", func(t *testing.T) {
		logger := logrus.New()
		logger.SetLevel(logrus.ErrorLevel)

		client := &custodianSimulatorClientImpl{
			conn:   nil, // We're not testing the gRPC connection here
			logger: logger,
		}

		ctx := context.Background()
		settlement := map[string]interface{}{
			"id":           "settlement-123",
			"amount":       5000.0,
			"currency":     "USD",
			"counterparty": "BANK-A",
		}

		err := client.ProcessSettlement(ctx, settlement)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})
}

func TestServiceUnavailableError(t *testing.T) {
	t.Run("creates_service_unavailable_error", func(t *testing.T) {
		err := &ServiceUnavailableError{
			ServiceName: "test-service",
			Message:     "connection failed",
		}

		expectedMessage := "service test-service unavailable: connection failed"
		if err.Error() != expectedMessage {
			t.Errorf("Expected error message '%s', got '%s'", expectedMessage, err.Error())
		}
	})
}

func TestInterServiceClientManager_ContextCancellation(t *testing.T) {
	t.Run("handles_context_cancellation", func(t *testing.T) {
		cfg := &config.Config{
			ServiceName: "exchange-simulator",
		}

		logger := logrus.New()
		logger.SetLevel(logrus.ErrorLevel)

		manager := NewInterServiceClientManager(cfg, logger,
			&ServiceDiscoveryClient{},
			&ConfigurationClient{})

		// Verify context is created
		if manager.ctx == nil {
			t.Error("Expected context to be created")
		}

		if manager.cancel == nil {
			t.Error("Expected cancel function to be created")
		}

		// Test cancellation doesn't cause issues
		manager.cancel()

		// Should still be able to close normally
		err := manager.Close()
		if err != nil {
			t.Errorf("Expected no error after context cancellation, got %v", err)
		}
	})
}