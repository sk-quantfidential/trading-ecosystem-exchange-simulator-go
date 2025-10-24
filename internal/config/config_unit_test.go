//go:build unit || !integration

package config

import (
	"context"
	"os"
	"testing"
)

func TestConfig_GetDataAdapter(t *testing.T) {
	t.Run("returns_nil_when_not_initialized", func(t *testing.T) {
		// Given: A fresh config
		cfg := &Config{}

		// When: Getting DataAdapter before initialization
		adapter := cfg.GetDataAdapter()

		// Then: Should return nil
		if adapter != nil {
			t.Error("Expected GetDataAdapter to return nil when not initialized")
		}
	})
}

func TestConfig_DisconnectDataAdapter(t *testing.T) {
	t.Run("handles_nil_adapter_gracefully", func(t *testing.T) {
		// Given: A config without initialized adapter
		cfg := &Config{}

		// When: Disconnecting
		ctx := context.Background()
		err := cfg.DisconnectDataAdapter(ctx)

		// Then: Should not error
		if err != nil {
			t.Errorf("Expected no error when disconnecting nil adapter, got: %v", err)
		}
	})
}

func TestConfig_Load(t *testing.T) {
	t.Run("loads_default_values", func(t *testing.T) {
		// Given: Clean environment
		origPort := os.Getenv("HTTP_PORT")
		os.Unsetenv("HTTP_PORT")
		defer func() {
			if origPort != "" {
				os.Setenv("HTTP_PORT", origPort)
			}
		}()

		// When: Loading config
		cfg := Load()

		// Then: Should have default values
		if cfg.HTTPPort != 8080 {
			t.Errorf("Expected HTTPPort 8080, got %d", cfg.HTTPPort)
		}
		if cfg.GRPCPort != 50051 {
			t.Errorf("Expected GRPCPort 50051, got %d", cfg.GRPCPort)
		}
		if cfg.ServiceName != "exchange-simulator" {
			t.Errorf("Expected ServiceName 'exchange-simulator', got %s", cfg.ServiceName)
		}
	})

	t.Run("loads_environment_overrides", func(t *testing.T) {
		// Given: Custom environment values
		os.Setenv("HTTP_PORT", "9000")
		os.Setenv("SERVICE_NAME", "test-service")
		defer os.Unsetenv("HTTP_PORT")
		defer os.Unsetenv("SERVICE_NAME")

		// When: Loading config
		cfg := Load()

		// Then: Should use environment values
		if cfg.HTTPPort != 9000 {
			t.Errorf("Expected HTTPPort 9000, got %d", cfg.HTTPPort)
		}
		if cfg.ServiceName != "test-service" {
			t.Errorf("Expected ServiceName 'test-service', got %s", cfg.ServiceName)
		}
	})
}
