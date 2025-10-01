//go:build integration

package tests

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/quantfidential/trading-ecosystem/exchange-data-adapter-go/pkg/adapters"
)

// TestDataAdapterSmoke performs basic smoke tests for the DataAdapter integration
func TestDataAdapterSmoke(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Setup: Use orchestrator credentials
	os.Setenv("POSTGRES_URL", "postgres://exchange_adapter:exchange-adapter-db-pass@localhost:5432/trading_ecosystem?sslmode=disable")
	os.Setenv("REDIS_URL", "redis://exchange-adapter:exchange-pass@localhost:6379/0")
	defer os.Unsetenv("POSTGRES_URL")
	defer os.Unsetenv("REDIS_URL")

	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	t.Run("adapter_initialization", func(t *testing.T) {
		// Given: DataAdapter factory
		adapter, err := adapters.NewExchangeDataAdapterFromEnv(logger)
		if err != nil {
			t.Skipf("DataAdapter creation failed (infrastructure not available): %v", err)
			return
		}

		// When: Connecting to infrastructure
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err = adapter.Connect(ctx)
		if err != nil {
			t.Skipf("DataAdapter connection failed (infrastructure not available): %v", err)
			return
		}
		defer adapter.Disconnect(ctx)

		// Then: Repositories should be accessible
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

		t.Log("✓ DataAdapter initialized successfully")
	})

	t.Run("account_repository_basic_crud", func(t *testing.T) {
		// Account repository requires UUID generation enhancement - deferred to future epic
		t.Skip("Account repository requires UUID generation enhancement - deferred to future epic")
	})

	t.Run("order_repository_basic_crud", func(t *testing.T) {
		// Requires account creation - deferred to future epic
		t.Skip("Order repository tests require UUID generation enhancement - deferred to future epic")
	})

	t.Run("balance_repository_basic_crud", func(t *testing.T) {
		// Requires account creation - deferred to future epic
		t.Skip("Balance repository tests require UUID generation enhancement - deferred to future epic")
	})

	t.Run("service_discovery_smoke", func(t *testing.T) {
		// Requires Redis ACL permissions (keys, scan) for exchange-adapter user
		t.Skip("Service discovery requires Redis ACL enhancement - deferred to future epic")
	})

	t.Run("cache_repository_smoke", func(t *testing.T) {
		// Given: Connected DataAdapter
		adapter, err := adapters.NewExchangeDataAdapterFromEnv(logger)
		if err != nil {
			t.Skipf("DataAdapter not available: %v", err)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := adapter.Connect(ctx); err != nil {
			t.Skipf("Infrastructure not available: %v", err)
			return
		}
		defer adapter.Disconnect(ctx)

		// When: Setting a cache value
		cacheRepo := adapter.CacheRepository()
		testKey := "exchange:smoke-test:" + time.Now().Format("20060102150405")
		testValue := "test-value-123"

		err = cacheRepo.Set(ctx, testKey, testValue, 1*time.Minute)
		if err != nil {
			t.Fatalf("Failed to set cache value: %v", err)
		}
		defer cacheRepo.Delete(ctx, testKey)

		// Then: Should be able to retrieve it
		retrieved, err := cacheRepo.Get(ctx, testKey)
		if err != nil {
			t.Fatalf("Failed to get cache value: %v", err)
		}

		if retrieved != testValue {
			t.Errorf("Expected value '%s', got '%s'", testValue, retrieved)
		}

		t.Logf("✓ Cache smoke test passed (key: %s)", testKey)
	})
}
