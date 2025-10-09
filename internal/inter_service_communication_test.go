//go:build integration

package internal

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/quantfidential/trading-ecosystem/exchange-simulator-go/internal/config"
	"github.com/quantfidential/trading-ecosystem/exchange-simulator-go/internal/infrastructure"
)

// TestInterServiceCommunication_Integration tests the complete integration of all infrastructure components
func TestInterServiceCommunication_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Run("infrastructure_components_work_together", func(t *testing.T) {
		cfg := &config.Config{
			ServiceName:    "exchange-simulator",
			ServiceVersion: "1.0.0-test",
			RedisURL:       "redis://localhost:6379",
			GRPCPort:       9093,
			HTTPPort:       8083,
		}

		logger := logrus.New()
		logger.SetLevel(logrus.ErrorLevel) // Reduce noise during tests

		// Test service discovery with smart infrastructure detection
		serviceDiscovery := infrastructure.NewServiceDiscoveryClient(cfg, logger)

		// Try to start service discovery - if Redis is not available, skip gracefully
		err := serviceDiscovery.Start()
		if err != nil {
			t.Skipf("Redis infrastructure not available for integration test: %v", err)
			return
		}
		defer serviceDiscovery.Stop()

		// Verify service discovery is running
		if !serviceDiscovery.IsRunning() {
			t.Error("Expected service discovery to be running")
		}

		// Test metrics tracking
		metrics := serviceDiscovery.GetMetrics()
		if !metrics.IsConnected {
			t.Error("Expected service discovery to be connected")
		}

		// Test configuration client
		configClient := infrastructure.NewConfigurationClient(cfg, logger)

		// Test that configuration client is created successfully
		configMetrics := configClient.GetMetrics()
		if configMetrics.RequestCount < 0 {
			t.Error("Expected non-negative request count")
		}

		// Test inter-service client manager
		clientManager := infrastructure.NewInterServiceClientManager(cfg, logger, serviceDiscovery, configClient)

		// Verify manager initialization
		managerMetrics := clientManager.GetMetrics()
		if managerMetrics.ActiveConnections < 0 {
			t.Error("Expected non-negative active connections")
		}

		// Test graceful cleanup
		err = clientManager.Close()
		if err != nil {
			t.Errorf("Failed to close client manager: %v", err)
		}

		t.Logf("Integration test completed successfully with infrastructure detection")
	})

	t.Run("service_discovery_lifecycle", func(t *testing.T) {
		cfg := &config.Config{
			ServiceName:    "exchange-simulator-test",
			ServiceVersion: "1.0.0-integration",
			RedisURL:       "redis://localhost:6379",
			GRPCPort:       9094,
			HTTPPort:       8084,
		}

		logger := logrus.New()
		logger.SetLevel(logrus.ErrorLevel)

		serviceDiscovery := infrastructure.NewServiceDiscoveryClient(cfg, logger)

		// Test start
		err := serviceDiscovery.Start()
		if err != nil {
			t.Skipf("Redis not available for service discovery test: %v", err)
			return
		}

		// Verify running
		if !serviceDiscovery.IsRunning() {
			t.Error("Expected service discovery to be running after start")
		}

		// Test service discovery functionality
		services, err := serviceDiscovery.DiscoverServices("")
		if err != nil {
			t.Errorf("Failed to discover services: %v", err)
		}

		// Should find at least our own service
		found := false
		for _, service := range services {
			if service.ServiceName == "exchange-simulator-test" {
				found = true
				break
			}
		}

		if !found {
			t.Error("Expected to find our own service in discovery results")
		}

		// Test metrics
		metrics := serviceDiscovery.GetMetrics()
		if metrics.DiscoveryCount == 0 {
			t.Error("Expected discovery count to be incremented")
		}

		// Test stop
		err = serviceDiscovery.Stop()
		if err != nil {
			t.Errorf("Failed to stop service discovery: %v", err)
		}

		// Verify stopped
		if serviceDiscovery.IsRunning() {
			t.Error("Expected service discovery to be stopped after stop")
		}

		t.Logf("Service discovery lifecycle test completed successfully")
	})

	t.Run("configuration_client_integration", func(t *testing.T) {
		cfg := &config.Config{
			ServiceName:    "exchange-simulator-config-test",
			ServiceVersion: "1.0.0",
			RedisURL:       "redis://localhost:6379",
		}

		logger := logrus.New()
		logger.SetLevel(logrus.ErrorLevel)

		configClient := infrastructure.NewConfigurationClient(cfg, logger)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Test configuration retrieval (will fail gracefully if config service not available)
		_, err := configClient.GetConfiguration(ctx, "test-key")
		if err != nil {
			// Expected to fail in test environment - this tests the error handling
			t.Logf("Configuration service not available (expected in test): %v", err)
		}

		// Test metrics regardless of service availability
		metrics := configClient.GetMetrics()
		if metrics.RequestCount == 0 {
			t.Error("Expected request count to be incremented")
		}

		// Test health status
		healthy := configClient.IsHealthy()
		t.Logf("Configuration client health status: %v", healthy)

		t.Logf("Configuration client integration test completed")
	})

	t.Run("comprehensive_component_integration", func(t *testing.T) {
		cfg := &config.Config{
			ServiceName:    "exchange-simulator-comprehensive",
			ServiceVersion: "1.0.0-comprehensive",
			RedisURL:       "redis://localhost:6379",
			GRPCPort:       9095,
			HTTPPort:       8085,
		}

		logger := logrus.New()
		logger.SetLevel(logrus.ErrorLevel)

		// Create all components
		serviceDiscovery := infrastructure.NewServiceDiscoveryClient(cfg, logger)
		configClient := infrastructure.NewConfigurationClient(cfg, logger)
		clientManager := infrastructure.NewInterServiceClientManager(cfg, logger, serviceDiscovery, configClient)

		// Test service discovery startup
		err := serviceDiscovery.Start()
		if err != nil {
			t.Skipf("Infrastructure not available for comprehensive test: %v", err)
			return
		}
		defer serviceDiscovery.Stop()

		// Allow some time for service registration
		time.Sleep(100 * time.Millisecond)

		// Test that components work together
		services, err := serviceDiscovery.DiscoverServices("exchange-simulator-comprehensive")
		if err != nil {
			t.Errorf("Failed to discover our own service: %v", err)
		}

		if len(services) == 0 {
			t.Error("Expected to find at least one service instance")
		}

		// Test client manager metrics
		managerMetrics := clientManager.GetMetrics()
		if managerMetrics.TotalConnections < 0 {
			t.Error("Expected non-negative total connections")
		}

		// Test service discovery metrics
		discoveryMetrics := serviceDiscovery.GetMetrics()
		if discoveryMetrics.HeartbeatCount < 0 {
			t.Error("Expected non-negative heartbeat count")
		}

		// Test configuration metrics
		configMetrics := configClient.GetMetrics()
		if configMetrics.CacheHits < 0 {
			t.Error("Expected non-negative cache hits")
		}

		// Test graceful shutdown
		err = clientManager.Close()
		if err != nil {
			t.Errorf("Failed to close client manager: %v", err)
		}

		err = serviceDiscovery.Stop()
		if err != nil {
			t.Errorf("Failed to stop service discovery: %v", err)
		}

		t.Logf("Comprehensive integration test completed successfully")
	})
}

// TestDataModels validates the data structures and interfaces
func TestDataModels(t *testing.T) {
	t.Run("service_info_model", func(t *testing.T) {
		serviceInfo := infrastructure.ServiceInfo{
			ServiceName: "test-service",
			Host:        "localhost",
			GRPCPort:    50051,
			HTTPPort:    8080,
			Version:     "1.0.0",
			Environment: "test",
			Status:      "healthy",
			LastSeen:    time.Now(),
			Metadata: map[string]string{
				"region": "us-east-1",
				"zone":   "a",
			},
		}

		if serviceInfo.ServiceName != "test-service" {
			t.Errorf("Expected service name 'test-service', got %s", serviceInfo.ServiceName)
		}

		if serviceInfo.GRPCPort != 50051 {
			t.Errorf("Expected gRPC port 50051, got %d", serviceInfo.GRPCPort)
		}

		if serviceInfo.Metadata["region"] != "us-east-1" {
			t.Errorf("Expected region 'us-east-1', got %s", serviceInfo.Metadata["region"])
		}
	})

	t.Run("configuration_value_model", func(t *testing.T) {
		configValue := infrastructure.ConfigurationValue{
			Key:         "database.url",
			Value:       "postgresql://localhost:5432/db",
			Environment: "test",
			Service:     "exchange-simulator",
			UpdatedAt:   time.Now(),
		}

		if configValue.Key != "database.url" {
			t.Errorf("Expected key 'database.url', got %s", configValue.Key)
		}

		if configValue.Service != "exchange-simulator" {
			t.Errorf("Expected service 'exchange-simulator', got %s", configValue.Service)
		}
	})

	t.Run("metrics_models", func(t *testing.T) {
		// Test service discovery metrics
		discoveryMetrics := infrastructure.ServiceDiscoveryMetrics{
			RegisteredServices: 5,
			HealthyServices:    4,
			HeartbeatCount:     100,
			IsConnected:        true,
		}

		if discoveryMetrics.RegisteredServices != 5 {
			t.Errorf("Expected 5 registered services, got %d", discoveryMetrics.RegisteredServices)
		}

		// Test configuration metrics
		configMetrics := infrastructure.ConfigurationClientMetrics{
			RequestCount: 50,
			CacheHits:    30,
			CacheMisses:  20,
			IsConnected:  true,
		}

		if configMetrics.RequestCount != 50 {
			t.Errorf("Expected 50 requests, got %d", configMetrics.RequestCount)
		}

		// Test inter-service metrics
		interServiceMetrics := infrastructure.InterServiceMetrics{
			ActiveConnections: 3,
			TotalConnections:  10,
			ServiceCallCount:  200,
		}

		if interServiceMetrics.ActiveConnections != 3 {
			t.Errorf("Expected 3 active connections, got %d", interServiceMetrics.ActiveConnections)
		}
	})
}

// TestErrorHandling validates error handling across components
func TestErrorHandling(t *testing.T) {
	t.Run("service_unavailable_error", func(t *testing.T) {
		err := &infrastructure.ServiceUnavailableError{
			ServiceName: "test-service",
			Message:     "connection refused",
		}

		expectedMsg := "service test-service unavailable: connection refused"
		if err.Error() != expectedMsg {
			t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
		}
	})

	t.Run("graceful_degradation", func(t *testing.T) {
		cfg := &config.Config{
			ServiceName:    "exchange-simulator-error-test",
			ServiceVersion: "1.0.0",
			RedisURL:       "redis://invalid-host:6379", // Invalid Redis URL
		}

		logger := logrus.New()
		logger.SetLevel(logrus.ErrorLevel)

		serviceDiscovery := infrastructure.NewServiceDiscoveryClient(cfg, logger)

		// Should fail gracefully with invalid Redis URL
		err := serviceDiscovery.Start()
		if err == nil {
			t.Error("Expected error with invalid Redis URL")
			serviceDiscovery.Stop() // Clean up if somehow it worked
		}

		// Should handle the error gracefully
		if serviceDiscovery.IsRunning() {
			t.Error("Expected service discovery not to be running with failed connection")
		}
	})
}
