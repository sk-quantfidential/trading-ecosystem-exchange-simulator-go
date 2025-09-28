package infrastructure

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/quantfidential/trading-ecosystem/exchange-simulator-go/internal/config"
)

type ConfigurationValue struct {
	Key         string      `json:"key"`
	Value       interface{} `json:"value"`
	Environment string      `json:"environment"`
	Service     string      `json:"service"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

type ConfigurationResponse struct {
	Success bool                 `json:"success"`
	Data    []ConfigurationValue `json:"data"`
	Error   string               `json:"error,omitempty"`
}

type ConfigurationClientMetrics struct {
	RequestCount     int64     `json:"request_count"`
	CacheHits        int64     `json:"cache_hits"`
	CacheMisses      int64     `json:"cache_misses"`
	LastRequestTime  time.Time `json:"last_request_time"`
	LastCacheUpdate  time.Time `json:"last_cache_update"`
	IsConnected      bool      `json:"is_connected"`
	ResponseTimeMs   int64     `json:"response_time_ms"`
}

type configCacheEntry struct {
	value     ConfigurationValue
	expiresAt time.Time
}

type ConfigurationClient struct {
	config         *config.Config
	logger         *logrus.Logger
	httpClient     *http.Client
	baseURL        string
	cache          map[string]configCacheEntry
	cacheTTL       time.Duration
	cacheMutex     sync.RWMutex
	metrics        ConfigurationClientMetrics
	metricsMutex   sync.RWMutex
	isInitialized  bool
}

func NewConfigurationClient(cfg *config.Config, logger *logrus.Logger) *ConfigurationClient {
	return &ConfigurationClient{
		config: cfg,
		logger: logger,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		baseURL:  "http://configuration-service:8080",
		cache:    make(map[string]configCacheEntry),
		cacheTTL: 5 * time.Minute,
		metrics: ConfigurationClientMetrics{
			IsConnected: false,
		},
		isInitialized: true,
	}
}

func (c *ConfigurationClient) GetConfiguration(ctx context.Context, key string) (*ConfigurationValue, error) {
	start := time.Now()
	defer func() {
		c.updateMetrics(time.Since(start))
	}()

	// Check cache first
	if cachedValue, found := c.getCachedValue(key); found {
		c.incrementCacheHit()
		c.logger.WithField("key", key).Debug("Configuration cache hit")
		return &cachedValue, nil
	}

	c.incrementCacheMiss()

	// Fetch from service
	url := fmt.Sprintf("%s/api/v1/configuration/%s", c.baseURL, key)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Service-Name", c.config.ServiceName)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.setConnectionStatus(false)
		return nil, fmt.Errorf("failed to fetch configuration: %w", err)
	}
	defer resp.Body.Close()

	c.setConnectionStatus(true)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("configuration service returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var configResp ConfigurationResponse
	if err := json.Unmarshal(body, &configResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if !configResp.Success {
		return nil, fmt.Errorf("configuration service error: %s", configResp.Error)
	}

	if len(configResp.Data) == 0 {
		return nil, fmt.Errorf("configuration key not found: %s", key)
	}

	configValue := configResp.Data[0]

	// Cache the result
	c.cacheValue(key, configValue)

	c.logger.WithFields(logrus.Fields{
		"key":         key,
		"environment": configValue.Environment,
		"service":     configValue.Service,
	}).Debug("Configuration fetched successfully")

	return &configValue, nil
}

func (c *ConfigurationClient) SetConfiguration(ctx context.Context, key string, value interface{}, environment string) error {
	start := time.Now()
	defer func() {
		c.updateMetrics(time.Since(start))
	}()

	configValue := ConfigurationValue{
		Key:         key,
		Value:       value,
		Environment: environment,
		Service:     c.config.ServiceName,
		UpdatedAt:   time.Now(),
	}

	payload, err := json.Marshal(configValue)
	if err != nil {
		return fmt.Errorf("failed to marshal configuration: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/configuration", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Service-Name", c.config.ServiceName)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.setConnectionStatus(false)
		return fmt.Errorf("failed to set configuration: %w", err)
	}
	defer resp.Body.Close()

	c.setConnectionStatus(true)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("configuration service returned status %d", resp.StatusCode)
	}

	// Invalidate cache for this key
	c.invalidateCache(key)

	c.logger.WithFields(logrus.Fields{
		"key":         key,
		"environment": environment,
		"service":     c.config.ServiceName,
	}).Info("Configuration set successfully")

	return nil
}

func (c *ConfigurationClient) GetMetrics() ConfigurationClientMetrics {
	c.metricsMutex.RLock()
	defer c.metricsMutex.RUnlock()
	return c.metrics
}

func (c *ConfigurationClient) IsHealthy() bool {
	c.metricsMutex.RLock()
	defer c.metricsMutex.RUnlock()
	return c.metrics.IsConnected
}

func (c *ConfigurationClient) getCachedValue(key string) (ConfigurationValue, bool) {
	c.cacheMutex.RLock()
	defer c.cacheMutex.RUnlock()

	entry, exists := c.cache[key]
	if !exists {
		return ConfigurationValue{}, false
	}

	if time.Now().After(entry.expiresAt) {
		// Cache expired, remove it
		c.cacheMutex.RUnlock()
		c.cacheMutex.Lock()
		delete(c.cache, key)
		c.cacheMutex.Unlock()
		c.cacheMutex.RLock()
		return ConfigurationValue{}, false
	}

	return entry.value, true
}

func (c *ConfigurationClient) cacheValue(key string, value ConfigurationValue) {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()

	c.cache[key] = configCacheEntry{
		value:     value,
		expiresAt: time.Now().Add(c.cacheTTL),
	}

	c.metricsMutex.Lock()
	c.metrics.LastCacheUpdate = time.Now()
	c.metricsMutex.Unlock()
}

func (c *ConfigurationClient) invalidateCache(key string) {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	delete(c.cache, key)
}

func (c *ConfigurationClient) incrementCacheHit() {
	c.metricsMutex.Lock()
	defer c.metricsMutex.Unlock()
	c.metrics.CacheHits++
}

func (c *ConfigurationClient) incrementCacheMiss() {
	c.metricsMutex.Lock()
	defer c.metricsMutex.Unlock()
	c.metrics.CacheMisses++
}

func (c *ConfigurationClient) setConnectionStatus(connected bool) {
	c.metricsMutex.Lock()
	defer c.metricsMutex.Unlock()
	c.metrics.IsConnected = connected
}

func (c *ConfigurationClient) updateMetrics(duration time.Duration) {
	c.metricsMutex.Lock()
	defer c.metricsMutex.Unlock()

	c.metrics.RequestCount++
	c.metrics.LastRequestTime = time.Now()
	c.metrics.ResponseTimeMs = duration.Milliseconds()
}