//go:build integration

package internal

import (
	"context"
	"testing"
	"time"

	"github.com/quantfidential/trading-ecosystem/exchange-simulator-go/internal/config"
	"github.com/quantfidential/trading-ecosystem/exchange-simulator-go/internal/infrastructure"
)

// TestInterServiceCommunication_RedPhase defines the expected behaviors for inter-service communication
// These tests will fail initially and drive our implementation (TDD Red-Green-Refactor)
func TestInterServiceCommunication_CustodianIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Run("can_communicate_with_custodian_simulator", func(t *testing.T) {
		t.Parallel()

		cfg := &config.Config{
			ServiceName:             "exchange-simulator",
			ServiceVersion:          "1.0.0",
			RedisURL:                "redis://localhost:6379",
			ConfigurationServiceURL: "http://localhost:8090",
			RequestTimeout:          5 * time.Second,
			CacheTTL:               5 * time.Minute,
			HealthCheckInterval:     30 * time.Second,
			GRPCPort:               9093,
			HTTPPort:               8083,
		}

		clientManager := NewInterServiceClientManager(cfg)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := clientManager.Initialize(ctx)
		if err != nil {
			t.Skip("Inter-service infrastructure not available for test")
		}
		defer clientManager.Cleanup(ctx)

		// Get custodian simulator client
		custodianClient, err := clientManager.GetCustodianSimulatorClient(ctx)
		if err != nil {
			t.Errorf("Failed to get custodian simulator client: %v", err)
			return
		}

		// Test health check
		health, err := custodianClient.HealthCheck(ctx)
		if err != nil {
			t.Errorf("Custodian simulator health check failed: %v", err)
		}

		if health.Status != "healthy" {
			t.Errorf("Expected healthy status, got %s", health.Status)
		}
	})
}

func TestInterServiceCommunication_AuditIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Run("can_communicate_with_audit_correlator", func(t *testing.T) {
		t.Parallel()

		cfg := &config.Config{
			ServiceName:             "exchange-simulator",
			ServiceVersion:          "1.0.0",
			RedisURL:                "redis://localhost:6379",
			ConfigurationServiceURL: "http://localhost:8090",
			RequestTimeout:          5 * time.Second,
			CacheTTL:               5 * time.Minute,
			HealthCheckInterval:     30 * time.Second,
			GRPCPort:               9093,
			HTTPPort:               8083,
		}

		clientManager := NewInterServiceClientManager(cfg)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := clientManager.Initialize(ctx)
		if err != nil {
			t.Skip("Inter-service infrastructure not available for test")
		}
		defer clientManager.Cleanup(ctx)

		// Get audit correlator client
		auditClient, err := clientManager.GetAuditCorrelatorClient(ctx)
		if err != nil {
			t.Errorf("Failed to get audit correlator client: %v", err)
			return
		}

		// Test health check
		health, err := auditClient.HealthCheck(ctx)
		if err != nil {
			t.Errorf("Audit correlator health check failed: %v", err)
		}

		if health.Status != "healthy" {
			t.Errorf("Expected healthy status, got %s", health.Status)
		}
	})
}

func TestInterServiceCommunication_ServiceDiscovery(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Run("discovers_services_dynamically", func(t *testing.T) {
		t.Parallel()

		cfg := &config.Config{
			ServiceName:             "exchange-simulator",
			ServiceVersion:          "1.0.0",
			RedisURL:                "redis://localhost:6379",
			ConfigurationServiceURL: "http://localhost:8090",
			RequestTimeout:          5 * time.Second,
			CacheTTL:               5 * time.Minute,
			HealthCheckInterval:     30 * time.Second,
			GRPCPort:               9093,
			HTTPPort:               8083,
		}

		clientManager := NewInterServiceClientManager(cfg)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := clientManager.Initialize(ctx)
		if err != nil {
			t.Skip("Service discovery not available for test")
		}
		defer clientManager.Cleanup(ctx)

		// Discover available services
		services, err := clientManager.DiscoverServices(ctx)
		if err != nil {
			t.Errorf("Service discovery failed: %v", err)
		}

		// Should find at least one service (potentially ourselves)
		if len(services) == 0 {
			t.Log("No services discovered - this might be expected in test environment")
		}

		// Verify service info structure
		for _, service := range services {
			if service.Name == "" {
				t.Error("Service name should not be empty")
			}
			if service.GRPCPort == 0 {
				t.Error("Service gRPC port should be set")
			}
		}
	})
}

func TestInterServiceCommunication_ConnectionPooling(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Run("reuses_connections_efficiently", func(t *testing.T) {
		t.Parallel()

		cfg := &config.Config{
			ServiceName:             "exchange-simulator",
			ServiceVersion:          "1.0.0",
			RedisURL:                "redis://localhost:6379",
			ConfigurationServiceURL: "http://localhost:8090",
			RequestTimeout:          5 * time.Second,
			CacheTTL:               5 * time.Minute,
			HealthCheckInterval:     30 * time.Second,
			GRPCPort:               9093,
			HTTPPort:               8083,
		}

		clientManager := NewInterServiceClientManager(cfg)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := clientManager.Initialize(ctx)
		if err != nil {
			t.Skip("Inter-service infrastructure not available for test")
		}
		defer clientManager.Cleanup(ctx)

		// Get same client multiple times - should reuse connections
		client1, err := clientManager.GetCustodianSimulatorClient(ctx)
		if err != nil {
			t.Skip("Custodian simulator not available for connection pooling test")
		}

		client2, err := clientManager.GetCustodianSimulatorClient(ctx)
		if err != nil {
			t.Errorf("Failed to get second client instance: %v", err)
		}

		// Verify connection statistics
		stats := clientManager.GetConnectionStats()
		if stats.ActiveConnections == 0 {
			t.Error("Expected active connections for connection pooling")
		}

		_, _ = client1, client2 // Use clients to avoid unused variable warnings
	})
}

func TestInterServiceCommunication_ErrorHandling(t *testing.T) {
	t.Run("handles_service_unavailable_gracefully", func(t *testing.T) {
		t.Parallel()

		cfg := &config.Config{
			ServiceName:             "exchange-simulator",
			ServiceVersion:          "1.0.0",
			RedisURL:                "redis://localhost:6379",
			ConfigurationServiceURL: "http://localhost:8090",
			RequestTimeout:          1 * time.Second,
			CacheTTL:               5 * time.Minute,
			HealthCheckInterval:     30 * time.Second,
			GRPCPort:               9093,
			HTTPPort:               8083,
		}

		clientManager := NewInterServiceClientManager(cfg)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := clientManager.Initialize(ctx)
		if err != nil {
			t.Skip("Inter-service infrastructure not available for test")
		}
		defer clientManager.Cleanup(ctx)

		// Try to get a client for a non-existent service
		_, err = clientManager.GetClientByName(ctx, "non-existent-service")
		if err == nil {
			t.Error("Expected error when getting non-existent service client")
		}

		// Verify error type
		if !IsServiceUnavailableError(err) {
			t.Errorf("Expected ServiceUnavailableError, got %T", err)
		}
	})
}

// InterServiceClientManager interface that needs to be implemented
type InterServiceClientManager interface {
	Initialize(ctx context.Context) error
	Cleanup(ctx context.Context) error
	GetCustodianSimulatorClient(ctx context.Context) (infrastructure.CustodianSimulatorClientInterface, error)
	GetAuditCorrelatorClient(ctx context.Context) (infrastructure.AuditCorrelatorClientInterface, error)
	GetClientByName(ctx context.Context, serviceName string) (infrastructure.ServiceClientInterface, error)
	DiscoverServices(ctx context.Context) ([]infrastructure.ServiceInfo, error)
	GetConnectionStats() infrastructure.ConnectionStats
}

// Error handling
func IsServiceUnavailableError(err error) bool {
	return infrastructure.IsServiceUnavailableError(err)
}

// Constructor function that creates a new inter-service client manager
func NewInterServiceClientManager(cfg *config.Config) InterServiceClientManager {
	return infrastructure.NewInterServiceClientManager(cfg)
}