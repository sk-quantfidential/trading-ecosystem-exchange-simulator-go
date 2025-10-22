package config

import (
	"context"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"github.com/quantfidential/trading-ecosystem/exchange-data-adapter-go/pkg/adapters"
	"github.com/sirupsen/logrus"
)

type Config struct {
	// Service Identity
	ServiceName             string
	ServiceInstanceName     string // Instance identifier (e.g., "exchange-OKX")
	ServiceVersion          string
	Environment             string // Deployment environment (development, staging, production)

	// Network
	HTTPPort                int
	GRPCPort                int

	// Configuration
	LogLevel                string
	PostgresURL             string
	RedisURL                string
	ConfigurationServiceURL string
	RequestTimeout          time.Duration
	CacheTTL                time.Duration
	HealthCheckInterval     time.Duration

	// Data Adapter
	dataAdapter adapters.DataAdapter
}

func Load() *Config {
	// Try to load .env file (ignore errors if not found)
	_ = godotenv.Load()

	cfg := &Config{
		ServiceName:             getEnv("SERVICE_NAME", "exchange-simulator"),
		ServiceInstanceName:     getEnv("SERVICE_INSTANCE_NAME", ""),
		ServiceVersion:          getEnv("SERVICE_VERSION", "1.0.0"),
		Environment:             getEnv("ENVIRONMENT", "development"),
		HTTPPort:                getEnvAsInt("HTTP_PORT", 8082),
		GRPCPort:                getEnvAsInt("GRPC_PORT", 9092),
		LogLevel:                getEnv("LOG_LEVEL", "info"),
		PostgresURL:             getEnv("POSTGRES_URL", ""),
		RedisURL:                getEnv("REDIS_URL", "redis://localhost:6379"),
		ConfigurationServiceURL: getEnv("CONFIG_SERVICE_URL", "http://localhost:8090"),
		RequestTimeout:          getEnvAsDuration("REQUEST_TIMEOUT", 5*time.Second),
		CacheTTL:                getEnvAsDuration("CACHE_TTL", 5*time.Minute),
		HealthCheckInterval:     getEnvAsDuration("HEALTH_CHECK_INTERVAL", 30*time.Second),
	}

	// Backward compatibility: Default ServiceInstanceName to ServiceName
	if cfg.ServiceInstanceName == "" {
		cfg.ServiceInstanceName = cfg.ServiceName
	}

	return cfg
}

func (c *Config) InitializeDataAdapter(ctx context.Context, logger *logrus.Logger) error {
	adapter, err := adapters.NewExchangeDataAdapterFromEnv(logger)
	if err != nil {
		logger.WithError(err).Warn("Failed to create data adapter, will use stub mode")
		return err
	}

	if err := adapter.Connect(ctx); err != nil {
		logger.WithError(err).Warn("Failed to connect data adapter, will use stub mode")
		return err
	}

	c.dataAdapter = adapter
	logger.Info("Data adapter initialized successfully")
	return nil
}

func (c *Config) GetDataAdapter() adapters.DataAdapter {
	return c.dataAdapter
}

func (c *Config) DisconnectDataAdapter(ctx context.Context) error {
	if c.dataAdapter != nil {
		return c.dataAdapter.Disconnect(ctx)
	}
	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}