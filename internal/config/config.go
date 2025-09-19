package config

import (
	"os"
	"strconv"
)

type Config struct {
	HTTPPort int
	GRPCPort int
	LogLevel string
	RedisURL string
}

func Load() *Config {
	return &Config{
		HTTPPort: getEnvAsInt("HTTP_PORT", 8081),
		GRPCPort: getEnvAsInt("GRPC_PORT", 9091),
		LogLevel: getEnv("LOG_LEVEL", "info"),
		RedisURL: getEnv("REDIS_URL", "redis://localhost:6379"),
	}
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