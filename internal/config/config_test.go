package config

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

// TestConfig_DataAdapterInitialization tests the DataAdapter initialization in config
func TestConfig_DataAdapterInitialization(t *testing.T) {
	t.Run("data_adapter_graceful_degradation_without_infrastructure", func(t *testing.T) {
		// Given: A config with invalid database URLs
		os.Setenv("POSTGRES_URL", "postgres://invalid:invalid@localhost:9999/invalid?sslmode=disable")
		os.Setenv("REDIS_URL", "redis://invalid@localhost:9999/0")
		defer os.Unsetenv("POSTGRES_URL")
		defer os.Unsetenv("REDIS_URL")

		cfg := Load()
		logger := logrus.New()
		logger.SetLevel(logrus.ErrorLevel) // Reduce noise

		// When: Attempting to initialize DataAdapter
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := cfg.InitializeDataAdapter(ctx, logger)

		// Then: Should fail gracefully (returns error but doesn't panic)
		if err == nil {
			t.Log("DataAdapter initialized (infrastructure available)")
		} else {
			t.Logf("DataAdapter failed gracefully: %v", err)
		}

		// GetDataAdapter should return nil when initialization failed
		adapter := cfg.GetDataAdapter()
		if err != nil && adapter != nil {
			t.Error("Expected GetDataAdapter to return nil when initialization failed")
		}
	})

	t.Run("data_adapter_with_orchestrator_infrastructure", func(t *testing.T) {
		// Given: Config with orchestrator URLs (from docker-compose.yml)
		os.Setenv("POSTGRES_URL", "postgres://exchange_adapter:exchange-adapter-db-pass@localhost:5432/trading_ecosystem?sslmode=disable")
		os.Setenv("REDIS_URL", "redis://exchange-adapter:exchange-pass@localhost:6379/0")
		defer os.Unsetenv("POSTGRES_URL")
		defer os.Unsetenv("REDIS_URL")

		cfg := Load()
		logger := logrus.New()
		logger.SetLevel(logrus.WarnLevel)

		// When: Attempting to initialize DataAdapter
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := cfg.InitializeDataAdapter(ctx, logger)

		// Then: Should connect if orchestrator is running
		if err == nil {
			t.Log("✓ DataAdapter initialized successfully (orchestrator available)")
			adapter := cfg.GetDataAdapter()
			if adapter == nil {
				t.Error("Expected GetDataAdapter to return non-nil when initialization succeeded")
			}

			// Verify repositories are accessible
			if adapter.AccountRepository() == nil {
				t.Error("Expected AccountRepository to be non-nil")
			}
			if adapter.OrderRepository() == nil {
				t.Error("Expected OrderRepository to be non-nil")
			}
			if adapter.TradeRepository() == nil {
				t.Error("Expected TradeRepository to be non-nil")
			}
			if adapter.BalanceRepository() == nil {
				t.Error("Expected BalanceRepository to be non-nil")
			}

			// Cleanup
			cfg.DisconnectDataAdapter(ctx)
		} else {
			t.Skipf("Orchestrator infrastructure not available: %v", err)
		}
	})
}

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
		if cfg.HTTPPort != 8082 {
			t.Errorf("Expected HTTPPort 8082, got %d", cfg.HTTPPort)
		}
		if cfg.GRPCPort != 9092 {
			t.Errorf("Expected GRPCPort 9092, got %d", cfg.GRPCPort)
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
