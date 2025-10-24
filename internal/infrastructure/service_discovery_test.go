//go:build unit

package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"

	"github.com/quantfidential/trading-ecosystem/exchange-simulator-go/internal/config"
)

// Mock Redis client for testing
type mockRedisClient struct {
	data      map[string]string
	pingError error
	setError  error
	getError  error
	delError  error
	keysError error
}

func newMockRedisClient() *mockRedisClient {
	return &mockRedisClient{
		data: make(map[string]string),
	}
}

func (m *mockRedisClient) Ping(ctx context.Context) *redis.StatusCmd {
	cmd := redis.NewStatusCmd(ctx, "ping")
	if m.pingError != nil {
		cmd.SetErr(m.pingError)
	} else {
		cmd.SetVal("PONG")
	}
	return cmd
}

func (m *mockRedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	cmd := redis.NewStatusCmd(ctx, "set", key, value)
	if m.setError != nil {
		cmd.SetErr(m.setError)
	} else {
		// Handle different value types
		switch v := value.(type) {
		case string:
			m.data[key] = v
		case []byte:
			m.data[key] = string(v)
		default:
			m.data[key] = fmt.Sprintf("%v", v)
		}
		cmd.SetVal("OK")
	}
	return cmd
}

func (m *mockRedisClient) Get(ctx context.Context, key string) *redis.StringCmd {
	cmd := redis.NewStringCmd(ctx, "get", key)
	if m.getError != nil {
		cmd.SetErr(m.getError)
	} else if value, exists := m.data[key]; exists {
		cmd.SetVal(value)
	} else {
		cmd.SetErr(redis.Nil)
	}
	return cmd
}

func (m *mockRedisClient) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	cmd := redis.NewIntCmd(ctx, "del")
	if m.delError != nil {
		cmd.SetErr(m.delError)
	} else {
		deleted := int64(0)
		for _, key := range keys {
			if _, exists := m.data[key]; exists {
				delete(m.data, key)
				deleted++
			}
		}
		cmd.SetVal(deleted)
	}
	return cmd
}

func (m *mockRedisClient) Keys(ctx context.Context, pattern string) *redis.StringSliceCmd {
	cmd := redis.NewStringSliceCmd(ctx, "keys", pattern)
	if m.keysError != nil {
		cmd.SetErr(m.keysError)
	} else {
		var keys []string
		for key := range m.data {
			// Simple pattern matching for testing
			if pattern == "services:*" || pattern == discoveryKeyPattern {
				if len(key) > 9 && key[:9] == "services:" {
					keys = append(keys, key)
				}
			} else if len(pattern) > 2 && pattern[len(pattern)-1] == '*' {
				// Handle patterns like "services:test-service:*" or "services:target-service:*"
				prefix := pattern[:len(pattern)-1] // Remove the "*"
				if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
					keys = append(keys, key)
				}
			}
		}
		cmd.SetVal(keys)
	}
	return cmd
}

func (m *mockRedisClient) Close() error {
	return nil
}

func TestServiceDiscoveryClient_Start(t *testing.T) {
	t.Run("successfully_starts_and_registers_service", func(t *testing.T) {
		cfg := &config.Config{
			ServiceName:    "test-service",
			ServiceVersion: "1.0.0",
			GRPCPort:       50051,
			HTTPPort:       8080,
			RedisURL:       "redis://localhost:6379",
		}

		logger := logrus.New()
		logger.SetLevel(logrus.ErrorLevel)

		client := NewServiceDiscoveryClient(cfg, logger)

		// Replace with mock Redis client
		mockRedis := newMockRedisClient()
		client.redisClient = mockRedis

		err := client.Start()
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		defer client.Stop()

		if !client.IsRunning() {
			t.Error("Expected service discovery to be running")
		}

		// Verify service was registered in Redis
		if len(mockRedis.data) == 0 {
			t.Error("Expected service to be registered in Redis")
		}

		// Check if the service key exists
		serviceKey := ""
		for key := range mockRedis.data {
			if len(key) > 9 && key[:9] == "services:" {
				serviceKey = key
				break
			}
		}

		if serviceKey == "" {
			t.Error("Expected service key to be found in Redis")
		}

		// Verify service info
		var serviceInfo ServiceInfo
		err = json.Unmarshal([]byte(mockRedis.data[serviceKey]), &serviceInfo)
		if err != nil {
			t.Fatalf("Failed to unmarshal service info: %v", err)
		}

		if serviceInfo.ServiceName != "test-service" {
			t.Errorf("Expected service name 'test-service', got %s", serviceInfo.ServiceName)
		}

		if serviceInfo.GRPCPort != 50051 {
			t.Errorf("Expected gRPC port 50051, got %d", serviceInfo.GRPCPort)
		}

		metrics := client.GetMetrics()
		if !metrics.IsConnected {
			t.Error("Expected metrics to show connected status")
		}
	})

	t.Run("fails_when_redis_unavailable", func(t *testing.T) {
		cfg := &config.Config{
			ServiceName:    "test-service",
			ServiceVersion: "1.0.0",
			RedisURL:       "redis://localhost:6379",
		}

		logger := logrus.New()
		logger.SetLevel(logrus.ErrorLevel)

		client := NewServiceDiscoveryClient(cfg, logger)

		// Set mock Redis to fail ping
		mockRedis := newMockRedisClient()
		mockRedis.pingError = redis.ErrClosed
		client.redisClient = mockRedis

		err := client.Start()
		if err == nil {
			t.Error("Expected error when Redis is unavailable")
		}

		if client.IsRunning() {
			t.Error("Expected service discovery not to be running when Redis fails")
		}
	})
}

func TestServiceDiscoveryClient_Stop(t *testing.T) {
	t.Run("successfully_stops_and_unregisters_service", func(t *testing.T) {
		cfg := &config.Config{
			ServiceName:    "test-service",
			ServiceVersion: "1.0.0",
			GRPCPort:       50051,
			HTTPPort:       8080,
			RedisURL:       "redis://localhost:6379",
		}

		logger := logrus.New()
		logger.SetLevel(logrus.ErrorLevel)

		client := NewServiceDiscoveryClient(cfg, logger)

		mockRedis := newMockRedisClient()
		client.redisClient = mockRedis

		// Start first
		err := client.Start()
		if err != nil {
			t.Fatalf("Failed to start: %v", err)
		}

		if len(mockRedis.data) == 0 {
			t.Error("Expected service to be registered")
		}

		// Stop
		err = client.Stop()
		if err != nil {
			t.Fatalf("Expected no error stopping, got %v", err)
		}

		if client.IsRunning() {
			t.Error("Expected service discovery to be stopped")
		}

		// Verify service was unregistered
		if len(mockRedis.data) != 0 {
			t.Error("Expected service to be unregistered from Redis")
		}
	})
}

func TestServiceDiscoveryClient_DiscoverServices(t *testing.T) {
	t.Run("discovers_all_services", func(t *testing.T) {
		cfg := &config.Config{
			ServiceName: "test-service",
			RedisURL:    "redis://localhost:6379",
		}

		logger := logrus.New()
		logger.SetLevel(logrus.ErrorLevel)

		client := NewServiceDiscoveryClient(cfg, logger)

		mockRedis := newMockRedisClient()
		client.redisClient = mockRedis

		// Add test services to mock Redis
		service1 := ServiceInfo{
			ServiceName: "service1",
			Host:        "localhost",
			GRPCPort:    9001,
			HTTPPort:    8001,
			Version:     "1.0.0",
			Status:      "healthy",
			LastSeen:    time.Now(),
		}

		service2 := ServiceInfo{
			ServiceName: "service2",
			Host:        "localhost",
			GRPCPort:    9002,
			HTTPPort:    8002,
			Version:     "1.0.0",
			Status:      "healthy",
			LastSeen:    time.Now(),
		}

		service1Data, _ := json.Marshal(service1)
		service2Data, _ := json.Marshal(service2)

		mockRedis.data["services:service1:localhost:9001"] = string(service1Data)
		mockRedis.data["services:service2:localhost:9002"] = string(service2Data)

		services, err := client.DiscoverServices("")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(services) != 2 {
			t.Errorf("Expected 2 services, got %d", len(services))
		}

		// Verify services
		serviceNames := make(map[string]bool)
		for _, service := range services {
			serviceNames[service.ServiceName] = true
		}

		if !serviceNames["service1"] {
			t.Error("Expected to find service1")
		}

		if !serviceNames["service2"] {
			t.Error("Expected to find service2")
		}

		metrics := client.GetMetrics()
		if metrics.DiscoveryCount == 0 {
			t.Error("Expected discovery count to be incremented")
		}
	})

	t.Run("discovers_specific_service", func(t *testing.T) {
		cfg := &config.Config{
			ServiceName: "test-service",
			RedisURL:    "redis://localhost:6379",
		}

		logger := logrus.New()
		logger.SetLevel(logrus.ErrorLevel)

		client := NewServiceDiscoveryClient(cfg, logger)

		mockRedis := newMockRedisClient()
		client.redisClient = mockRedis

		// Add test service
		service := ServiceInfo{
			ServiceName: "test-service",
			Host:        "localhost",
			GRPCPort:    9000,
			HTTPPort:    8000,
			Version:     "1.0.0",
			Status:      "healthy",
			LastSeen:    time.Now(),
		}

		serviceData, _ := json.Marshal(service)
		mockRedis.data["services:test-service:localhost:9000"] = string(serviceData)

		services, err := client.DiscoverServices("test-service")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(services) != 1 {
			t.Errorf("Expected 1 service, got %d", len(services))
		}

		if services[0].ServiceName != "test-service" {
			t.Errorf("Expected service name 'test-service', got %s", services[0].ServiceName)
		}
	})

	t.Run("filters_out_stale_services", func(t *testing.T) {
		cfg := &config.Config{
			ServiceName: "test-service",
			RedisURL:    "redis://localhost:6379",
		}

		logger := logrus.New()
		logger.SetLevel(logrus.ErrorLevel)

		client := NewServiceDiscoveryClient(cfg, logger)

		mockRedis := newMockRedisClient()
		client.redisClient = mockRedis

		// Add healthy service
		healthyService := ServiceInfo{
			ServiceName: "healthy-service",
			Host:        "localhost",
			GRPCPort:    9001,
			Status:      "healthy",
			LastSeen:    time.Now(),
		}

		// Add stale service
		staleService := ServiceInfo{
			ServiceName: "stale-service",
			Host:        "localhost",
			GRPCPort:    9002,
			Status:      "healthy",
			LastSeen:    time.Now().Add(-2 * time.Hour), // Very old
		}

		healthyData, _ := json.Marshal(healthyService)
		staleData, _ := json.Marshal(staleService)

		mockRedis.data["services:healthy-service:localhost:9001"] = string(healthyData)
		mockRedis.data["services:stale-service:localhost:9002"] = string(staleData)

		services, err := client.DiscoverServices("")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(services) != 1 {
			t.Errorf("Expected 1 healthy service, got %d", len(services))
		}

		if services[0].ServiceName != "healthy-service" {
			t.Errorf("Expected healthy-service, got %s", services[0].ServiceName)
		}
	})
}

func TestServiceDiscoveryClient_GetServiceEndpoint(t *testing.T) {
	t.Run("returns_endpoint_for_healthy_service", func(t *testing.T) {
		cfg := &config.Config{
			ServiceName: "test-service",
			RedisURL:    "redis://localhost:6379",
		}

		logger := logrus.New()
		logger.SetLevel(logrus.ErrorLevel)

		client := NewServiceDiscoveryClient(cfg, logger)

		mockRedis := newMockRedisClient()
		client.redisClient = mockRedis

		// Add test service
		service := ServiceInfo{
			ServiceName: "target-service",
			Host:        "service-host",
			GRPCPort:    50051,
			Status:      "healthy",
			LastSeen:    time.Now(),
		}

		serviceData, _ := json.Marshal(service)
		serviceKey := "services:target-service:service-host:50051"
		mockRedis.data[serviceKey] = string(serviceData)

		endpoint, err := client.GetServiceEndpoint("target-service")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		expectedEndpoint := "service-host:50051"
		if endpoint != expectedEndpoint {
			t.Errorf("Expected endpoint '%s', got '%s'", expectedEndpoint, endpoint)
		}
	})

	t.Run("returns_error_when_service_not_found", func(t *testing.T) {
		cfg := &config.Config{
			ServiceName: "test-service",
			RedisURL:    "redis://localhost:6379",
		}

		logger := logrus.New()
		logger.SetLevel(logrus.ErrorLevel)

		client := NewServiceDiscoveryClient(cfg, logger)

		mockRedis := newMockRedisClient()
		client.redisClient = mockRedis

		_, err := client.GetServiceEndpoint("nonexistent-service")
		if err == nil {
			t.Error("Expected error when service not found")
		}
	})
}

func TestServiceDiscoveryClient_Metrics(t *testing.T) {
	t.Run("tracks_comprehensive_metrics", func(t *testing.T) {
		cfg := &config.Config{
			ServiceName: "test-service",
			RedisURL:    "redis://localhost:6379",
		}

		logger := logrus.New()
		logger.SetLevel(logrus.ErrorLevel)

		client := NewServiceDiscoveryClient(cfg, logger)

		mockRedis := newMockRedisClient()
		client.redisClient = mockRedis

		// Perform some operations
		_, _ = client.DiscoverServices("")
		_, _ = client.GetServiceEndpoint("some-service")

		metrics := client.GetMetrics()

		if metrics.DiscoveryCount == 0 {
			t.Error("Expected discovery count to be incremented")
		}

		if metrics.ServiceLookupCount == 0 {
			t.Error("Expected service lookup count to be incremented")
		}

		if metrics.ServiceLookupErrors == 0 {
			t.Error("Expected service lookup errors to be incremented for missing service")
		}
	})
}
