package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"

	"github.com/quantfidential/trading-ecosystem/exchange-simulator-go/internal/config"
)

// RedisClient interface for mocking
type RedisClient interface {
	Ping(ctx context.Context) *redis.StatusCmd
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
	Get(ctx context.Context, key string) *redis.StringCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
	Keys(ctx context.Context, pattern string) *redis.StringSliceCmd
	Close() error
}

type ServiceInfo struct {
	ServiceName string            `json:"service_name"`
	Host        string            `json:"host"`
	GRPCPort    int               `json:"grpc_port"`
	HTTPPort    int               `json:"http_port"`
	Version     string            `json:"version"`
	Environment string            `json:"environment"`
	Status      string            `json:"status"`
	LastSeen    time.Time         `json:"last_seen"`
	Metadata    map[string]string `json:"metadata"`
}

type ServiceDiscoveryMetrics struct {
	RegisteredServices   int       `json:"registered_services"`
	HealthyServices      int       `json:"healthy_services"`
	LastHeartbeatTime    time.Time `json:"last_heartbeat_time"`
	LastDiscoveryTime    time.Time `json:"last_discovery_time"`
	HeartbeatCount       int64     `json:"heartbeat_count"`
	DiscoveryCount       int64     `json:"discovery_count"`
	IsConnected          bool      `json:"is_connected"`
	ServiceLookupCount   int64     `json:"service_lookup_count"`
	ServiceLookupErrors  int64     `json:"service_lookup_errors"`
}

type ServiceDiscoveryClient struct {
	config         *config.Config
	logger         *logrus.Logger
	redisClient    RedisClient
	serviceInfo    ServiceInfo
	heartbeatTicker *time.Ticker
	ctx            context.Context
	cancel         context.CancelFunc
	metrics        ServiceDiscoveryMetrics
	metricsMutex   sync.RWMutex
	isRunning      bool
	runningMutex   sync.RWMutex
}

const (
	serviceKeyPrefix     = "services:"
	heartbeatInterval    = 30 * time.Second
	serviceTimeout       = 90 * time.Second
	discoveryKeyPattern  = "services:*"
)

func NewServiceDiscoveryClient(cfg *config.Config, logger *logrus.Logger) *ServiceDiscoveryClient {
	ctx, cancel := context.WithCancel(context.Background())

	opt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		logger.WithError(err).Error("Failed to parse Redis URL, using defaults")
		opt = &redis.Options{
			Addr: "localhost:6379",
		}
	}

	redisClient := redis.NewClient(opt)

	serviceInfo := ServiceInfo{
		ServiceName: cfg.ServiceName,
		Host:        "localhost", // This could be made configurable
		GRPCPort:    cfg.GRPCPort,
		HTTPPort:    cfg.HTTPPort,
		Version:     cfg.ServiceVersion,
		Environment: "development", // This could be made configurable
		Status:      "healthy",
		LastSeen:    time.Now(),
		Metadata: map[string]string{
			"type":        "exchange-simulator",
			"deployment":  "local",
			"instance_id": fmt.Sprintf("%s-%d", cfg.ServiceName, time.Now().Unix()),
		},
	}

	return &ServiceDiscoveryClient{
		config:      cfg,
		logger:      logger,
		redisClient: redisClient,
		serviceInfo: serviceInfo,
		ctx:         ctx,
		cancel:      cancel,
		metrics: ServiceDiscoveryMetrics{
			IsConnected: false,
		},
	}
}

func (s *ServiceDiscoveryClient) Start() error {
	s.runningMutex.Lock()
	defer s.runningMutex.Unlock()

	if s.isRunning {
		return fmt.Errorf("service discovery already running")
	}

	// Test Redis connection
	err := s.redisClient.Ping(s.ctx).Err()
	if err != nil {
		s.updateConnectionStatus(false)
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}

	s.updateConnectionStatus(true)

	// Register service
	err = s.registerService()
	if err != nil {
		return fmt.Errorf("failed to register service: %w", err)
	}

	// Start heartbeat
	s.heartbeatTicker = time.NewTicker(heartbeatInterval)
	go s.heartbeatLoop()

	s.isRunning = true

	s.logger.WithFields(logrus.Fields{
		"service":     s.serviceInfo.ServiceName,
		"grpc_port":   s.serviceInfo.GRPCPort,
		"http_port":   s.serviceInfo.HTTPPort,
		"environment": s.serviceInfo.Environment,
	}).Info("Service discovery started")

	return nil
}

func (s *ServiceDiscoveryClient) Stop() error {
	s.runningMutex.Lock()
	defer s.runningMutex.Unlock()

	if !s.isRunning {
		return nil
	}

	s.logger.Info("Stopping service discovery")

	// Stop heartbeat
	if s.heartbeatTicker != nil {
		s.heartbeatTicker.Stop()
	}

	// Unregister service
	err := s.unregisterService()
	if err != nil {
		s.logger.WithError(err).Error("Failed to unregister service")
	}

	// Cancel context
	s.cancel()

	// Close Redis connection
	if s.redisClient != nil {
		s.redisClient.Close()
	}

	s.isRunning = false

	s.logger.Info("Service discovery stopped")
	return nil
}

func (s *ServiceDiscoveryClient) DiscoverServices(serviceName string) ([]ServiceInfo, error) {
	s.incrementDiscoveryCount()

	pattern := discoveryKeyPattern
	if serviceName != "" {
		pattern = fmt.Sprintf("services:%s:*", serviceName)
	}

	keys, err := s.redisClient.Keys(s.ctx, pattern).Result()
	if err != nil {
		s.incrementLookupError()
		return nil, fmt.Errorf("failed to discover services: %w", err)
	}

	if len(keys) == 0 {
		s.incrementLookupCount() // Still count as a lookup even if no results
		return []ServiceInfo{}, nil
	}

	services := make([]ServiceInfo, 0, len(keys))

	for _, key := range keys {
		serviceData, err := s.redisClient.Get(s.ctx, key).Result()
		if err != nil {
			s.logger.WithError(err).WithField("key", key).Warn("Failed to get service data")
			continue
		}

		var serviceInfo ServiceInfo
		if err := json.Unmarshal([]byte(serviceData), &serviceInfo); err != nil {
			s.logger.WithError(err).WithField("key", key).Warn("Failed to unmarshal service data")
			continue
		}

		// Check if service is still healthy (not timed out)
		if time.Since(serviceInfo.LastSeen) < serviceTimeout {
			services = append(services, serviceInfo)
		}
	}

	s.incrementLookupCount()

	s.logger.WithFields(logrus.Fields{
		"pattern":        pattern,
		"keys_found":     len(keys),
		"healthy_services": len(services),
	}).Debug("Service discovery completed")

	return services, nil
}

func (s *ServiceDiscoveryClient) GetServiceEndpoint(serviceName string) (string, error) {
	services, err := s.DiscoverServices(serviceName)
	if err != nil {
		s.incrementLookupError()
		return "", err
	}

	if len(services) == 0 {
		s.incrementLookupError()
		return "", fmt.Errorf("no healthy instances of service %s found", serviceName)
	}

	// For simplicity, return the first healthy service
	// In production, you might want load balancing logic here
	service := services[0]
	endpoint := fmt.Sprintf("%s:%d", service.Host, service.GRPCPort)

	s.incrementLookupCount()

	s.logger.WithFields(logrus.Fields{
		"service":  serviceName,
		"endpoint": endpoint,
		"version":  service.Version,
	}).Debug("Service endpoint resolved")

	return endpoint, nil
}

func (s *ServiceDiscoveryClient) GetMetrics() ServiceDiscoveryMetrics {
	s.metricsMutex.RLock()
	defer s.metricsMutex.RUnlock()
	return s.metrics
}

func (s *ServiceDiscoveryClient) IsRunning() bool {
	s.runningMutex.RLock()
	defer s.runningMutex.RUnlock()
	return s.isRunning
}

func (s *ServiceDiscoveryClient) registerService() error {
	key := s.getServiceKey()

	s.serviceInfo.LastSeen = time.Now()

	data, err := json.Marshal(s.serviceInfo)
	if err != nil {
		return fmt.Errorf("failed to marshal service info: %w", err)
	}

	err = s.redisClient.Set(s.ctx, key, data, serviceTimeout).Err()
	if err != nil {
		return fmt.Errorf("failed to register service in Redis: %w", err)
	}

	s.logger.WithField("key", key).Info("Service registered")
	return nil
}

func (s *ServiceDiscoveryClient) unregisterService() error {
	key := s.getServiceKey()

	err := s.redisClient.Del(s.ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to unregister service: %w", err)
	}

	s.logger.WithField("key", key).Info("Service unregistered")
	return nil
}

func (s *ServiceDiscoveryClient) heartbeatLoop() {
	for {
		select {
		case <-s.heartbeatTicker.C:
			err := s.registerService() // Re-register to update LastSeen
			if err != nil {
				s.logger.WithError(err).Error("Heartbeat failed")
				s.updateConnectionStatus(false)
			} else {
				s.updateConnectionStatus(true)
				s.incrementHeartbeatCount()
			}

		case <-s.ctx.Done():
			return
		}
	}
}

func (s *ServiceDiscoveryClient) getServiceKey() string {
	return fmt.Sprintf("services:%s:%s:%d",
		s.serviceInfo.ServiceName,
		s.serviceInfo.Host,
		s.serviceInfo.GRPCPort)
}

func (s *ServiceDiscoveryClient) updateConnectionStatus(connected bool) {
	s.metricsMutex.Lock()
	defer s.metricsMutex.Unlock()
	s.metrics.IsConnected = connected
}

func (s *ServiceDiscoveryClient) incrementHeartbeatCount() {
	s.metricsMutex.Lock()
	defer s.metricsMutex.Unlock()
	s.metrics.HeartbeatCount++
	s.metrics.LastHeartbeatTime = time.Now()
}

func (s *ServiceDiscoveryClient) incrementDiscoveryCount() {
	s.metricsMutex.Lock()
	defer s.metricsMutex.Unlock()
	s.metrics.DiscoveryCount++
	s.metrics.LastDiscoveryTime = time.Now()
}

func (s *ServiceDiscoveryClient) incrementLookupCount() {
	s.metricsMutex.Lock()
	defer s.metricsMutex.Unlock()
	s.metrics.ServiceLookupCount++
}

func (s *ServiceDiscoveryClient) incrementLookupError() {
	s.metricsMutex.Lock()
	defer s.metricsMutex.Unlock()
	s.metrics.ServiceLookupErrors++
}