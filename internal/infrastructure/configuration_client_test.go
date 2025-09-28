//go:build unit

package infrastructure

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/quantfidential/trading-ecosystem/exchange-simulator-go/internal/config"
)

func TestConfigurationClient_GetConfiguration(t *testing.T) {
	t.Run("successfully_fetches_configuration", func(t *testing.T) {
		// Setup mock server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/v1/configuration/test-key" {
				t.Errorf("Expected path /api/v1/configuration/test-key, got %s", r.URL.Path)
			}

			response := ConfigurationResponse{
				Success: true,
				Data: []ConfigurationValue{
					{
						Key:         "test-key",
						Value:       "test-value",
						Environment: "test",
						Service:     "exchange-simulator",
						UpdatedAt:   time.Now(),
					},
				},
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		// Setup client
		cfg := &config.Config{
			ServiceName: "exchange-simulator",
		}
		logger := logrus.New()
		logger.SetLevel(logrus.ErrorLevel)

		client := NewConfigurationClient(cfg, logger)
		client.baseURL = server.URL

		ctx := context.Background()

		// Test
		result, err := client.GetConfiguration(ctx, "test-key")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if result.Key != "test-key" {
			t.Errorf("Expected key 'test-key', got %s", result.Key)
		}

		if result.Value != "test-value" {
			t.Errorf("Expected value 'test-value', got %v", result.Value)
		}

		// Verify metrics
		metrics := client.GetMetrics()
		if metrics.RequestCount != 1 {
			t.Errorf("Expected 1 request, got %d", metrics.RequestCount)
		}

		if !metrics.IsConnected {
			t.Error("Expected client to be connected")
		}
	})

	t.Run("uses_cache_for_subsequent_requests", func(t *testing.T) {
		requestCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestCount++
			response := ConfigurationResponse{
				Success: true,
				Data: []ConfigurationValue{
					{
						Key:         "cached-key",
						Value:       "cached-value",
						Environment: "test",
						Service:     "exchange-simulator",
						UpdatedAt:   time.Now(),
					},
				},
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		cfg := &config.Config{ServiceName: "exchange-simulator"}
		logger := logrus.New()
		logger.SetLevel(logrus.ErrorLevel)

		client := NewConfigurationClient(cfg, logger)
		client.baseURL = server.URL

		ctx := context.Background()

		// First request - should hit server
		_, err := client.GetConfiguration(ctx, "cached-key")
		if err != nil {
			t.Fatalf("First request failed: %v", err)
		}

		// Second request - should use cache
		_, err = client.GetConfiguration(ctx, "cached-key")
		if err != nil {
			t.Fatalf("Second request failed: %v", err)
		}

		if requestCount != 1 {
			t.Errorf("Expected 1 server request, got %d", requestCount)
		}

		metrics := client.GetMetrics()
		if metrics.CacheHits != 1 {
			t.Errorf("Expected 1 cache hit, got %d", metrics.CacheHits)
		}

		if metrics.CacheMisses != 1 {
			t.Errorf("Expected 1 cache miss, got %d", metrics.CacheMisses)
		}
	})

	t.Run("handles_service_errors_gracefully", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		cfg := &config.Config{ServiceName: "exchange-simulator"}
		logger := logrus.New()
		logger.SetLevel(logrus.ErrorLevel)

		client := NewConfigurationClient(cfg, logger)
		client.baseURL = server.URL

		ctx := context.Background()

		_, err := client.GetConfiguration(ctx, "error-key")
		if err == nil {
			t.Error("Expected error for server error response")
		}
	})
}

func TestConfigurationClient_SetConfiguration(t *testing.T) {
	t.Run("successfully_sets_configuration", func(t *testing.T) {
		var receivedConfig ConfigurationValue
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "POST" {
				t.Errorf("Expected POST method, got %s", r.Method)
			}

			if r.URL.Path != "/api/v1/configuration" {
				t.Errorf("Expected path /api/v1/configuration, got %s", r.URL.Path)
			}

			// Decode the request body
			err := json.NewDecoder(r.Body).Decode(&receivedConfig)
			if err != nil {
				t.Errorf("Failed to decode request body: %v", err)
			}

			w.WriteHeader(http.StatusCreated)
		}))
		defer server.Close()

		cfg := &config.Config{ServiceName: "exchange-simulator"}
		logger := logrus.New()
		logger.SetLevel(logrus.ErrorLevel)

		client := NewConfigurationClient(cfg, logger)
		client.baseURL = server.URL

		ctx := context.Background()

		err := client.SetConfiguration(ctx, "new-key", "new-value", "test")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Verify the received configuration
		if receivedConfig.Key != "new-key" {
			t.Errorf("Expected key 'new-key', got %s", receivedConfig.Key)
		}

		if receivedConfig.Value != "new-value" {
			t.Errorf("Expected value 'new-value', got %v", receivedConfig.Value)
		}

		if receivedConfig.Environment != "test" {
			t.Errorf("Expected environment 'test', got %s", receivedConfig.Environment)
		}

		if receivedConfig.Service != "exchange-simulator" {
			t.Errorf("Expected service 'exchange-simulator', got %s", receivedConfig.Service)
		}
	})

	t.Run("invalidates_cache_after_set", func(t *testing.T) {
		getRequestCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "GET" {
				getRequestCount++
				response := ConfigurationResponse{
					Success: true,
					Data: []ConfigurationValue{
						{
							Key:         "update-key",
							Value:       "updated-value",
							Environment: "test",
							Service:     "exchange-simulator",
							UpdatedAt:   time.Now(),
						},
					},
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response)
			} else if r.Method == "POST" {
				w.WriteHeader(http.StatusOK)
			}
		}))
		defer server.Close()

		cfg := &config.Config{ServiceName: "exchange-simulator"}
		logger := logrus.New()
		logger.SetLevel(logrus.ErrorLevel)

		client := NewConfigurationClient(cfg, logger)
		client.baseURL = server.URL

		ctx := context.Background()

		// First get - populates cache
		_, err := client.GetConfiguration(ctx, "update-key")
		if err != nil {
			t.Fatalf("First get failed: %v", err)
		}

		// Set configuration - should invalidate cache
		err = client.SetConfiguration(ctx, "update-key", "new-updated-value", "test")
		if err != nil {
			t.Fatalf("Set configuration failed: %v", err)
		}

		// Second get - should hit server again due to cache invalidation
		_, err = client.GetConfiguration(ctx, "update-key")
		if err != nil {
			t.Fatalf("Second get failed: %v", err)
		}

		if getRequestCount != 2 {
			t.Errorf("Expected 2 GET requests, got %d", getRequestCount)
		}
	})
}

func TestConfigurationClient_Metrics(t *testing.T) {
	t.Run("tracks_comprehensive_metrics", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response := ConfigurationResponse{
				Success: true,
				Data: []ConfigurationValue{
					{
						Key:         "metrics-key",
						Value:       "metrics-value",
						Environment: "test",
						Service:     "exchange-simulator",
						UpdatedAt:   time.Now(),
					},
				},
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		cfg := &config.Config{ServiceName: "exchange-simulator"}
		logger := logrus.New()
		logger.SetLevel(logrus.ErrorLevel)

		client := NewConfigurationClient(cfg, logger)
		client.baseURL = server.URL

		ctx := context.Background()

		// Make some requests
		_, _ = client.GetConfiguration(ctx, "metrics-key")
		_, _ = client.GetConfiguration(ctx, "metrics-key") // Cache hit

		metrics := client.GetMetrics()

		if metrics.RequestCount != 2 {
			t.Errorf("Expected 2 requests, got %d", metrics.RequestCount)
		}

		if metrics.CacheHits != 1 {
			t.Errorf("Expected 1 cache hit, got %d", metrics.CacheHits)
		}

		if metrics.CacheMisses != 1 {
			t.Errorf("Expected 1 cache miss, got %d", metrics.CacheMisses)
		}

		if !metrics.IsConnected {
			t.Error("Expected connected status")
		}

		if metrics.ResponseTimeMs < 0 {
			t.Errorf("Expected non-negative response time, got %d", metrics.ResponseTimeMs)
		}
	})
}

func TestConfigurationClient_HealthCheck(t *testing.T) {
	t.Run("reports_healthy_when_connected", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response := ConfigurationResponse{Success: true, Data: []ConfigurationValue{}}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		cfg := &config.Config{ServiceName: "exchange-simulator"}
		logger := logrus.New()
		logger.SetLevel(logrus.ErrorLevel)

		client := NewConfigurationClient(cfg, logger)
		client.baseURL = server.URL

		// Initially not healthy
		if client.IsHealthy() {
			t.Error("Expected client to be unhealthy initially")
		}

		// Make request to establish connection
		ctx := context.Background()
		_, _ = client.GetConfiguration(ctx, "health-key")

		// Should now be healthy
		if !client.IsHealthy() {
			t.Error("Expected client to be healthy after successful request")
		}
	})
}

func TestConfigurationClient_CacheExpiration(t *testing.T) {
	t.Run("cache_expires_after_ttl", func(t *testing.T) {
		requestCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestCount++
			response := ConfigurationResponse{
				Success: true,
				Data: []ConfigurationValue{
					{
						Key:         "expiry-key",
						Value:       "expiry-value",
						Environment: "test",
						Service:     "exchange-simulator",
						UpdatedAt:   time.Now(),
					},
				},
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		cfg := &config.Config{ServiceName: "exchange-simulator"}
		logger := logrus.New()
		logger.SetLevel(logrus.ErrorLevel)

		client := NewConfigurationClient(cfg, logger)
		client.baseURL = server.URL
		client.cacheTTL = 100 * time.Millisecond // Short TTL for testing

		ctx := context.Background()

		// First request
		_, err := client.GetConfiguration(ctx, "expiry-key")
		if err != nil {
			t.Fatalf("First request failed: %v", err)
		}

		// Wait for cache to expire
		time.Sleep(150 * time.Millisecond)

		// Second request - should hit server again
		_, err = client.GetConfiguration(ctx, "expiry-key")
		if err != nil {
			t.Fatalf("Second request failed: %v", err)
		}

		if requestCount != 2 {
			t.Errorf("Expected 2 server requests due to cache expiration, got %d", requestCount)
		}
	})
}