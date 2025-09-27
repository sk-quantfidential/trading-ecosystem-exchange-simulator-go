package grpc

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/quantfidential/trading-ecosystem/exchange-simulator-go/internal/config"
	"github.com/quantfidential/trading-ecosystem/exchange-simulator-go/internal/services"
)

type ExchangeGRPCServer struct {
	config          *config.Config
	exchangeService *services.ExchangeService
	logger          *logrus.Logger

	// Server management
	grpcServer   *grpc.Server
	healthServer *health.Server
	listener     net.Listener

	// Metrics and monitoring
	startTime         time.Time
	connectionCount   int64
	requestCount      int64
	lastRequestTime   time.Time
	metricsLock       sync.RWMutex

	// Lifecycle management
	isRunning bool
	stopChan  chan struct{}
	wg        sync.WaitGroup
}

type ExchangeServerMetrics struct {
	StartTime         time.Time `json:"start_time"`
	UptimeSeconds     int64     `json:"uptime_seconds"`
	ConnectionCount   int64     `json:"connection_count"`
	RequestCount      int64     `json:"request_count"`
	LastRequestTime   time.Time `json:"last_request_time"`
	IsRunning         bool      `json:"is_running"`
}

func NewExchangeGRPCServer(cfg *config.Config, exchangeService *services.ExchangeService, logger *logrus.Logger) *ExchangeGRPCServer {
	return &ExchangeGRPCServer{
		config:          cfg,
		exchangeService: exchangeService,
		logger:          logger,
		startTime:       time.Now(),
		stopChan:        make(chan struct{}),
	}
}

func (s *ExchangeGRPCServer) Start(ctx context.Context) error {
	// Create listener
	address := fmt.Sprintf(":%d", s.config.GRPCPort)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", address, err)
	}
	s.listener = listener

	// Create gRPC server with enhanced options
	s.grpcServer = grpc.NewServer(
		grpc.UnaryInterceptor(s.unaryInterceptor),
	)

	// Setup health service
	s.healthServer = health.NewServer()
	grpc_health_v1.RegisterHealthServer(s.grpcServer, s.healthServer)

	// Set initial health status
	s.healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	s.healthServer.SetServingStatus("exchange-simulator", grpc_health_v1.HealthCheckResponse_SERVING)

	s.isRunning = true
	s.logger.WithFields(logrus.Fields{
		"service": s.config.ServiceName,
		"version": s.config.ServiceVersion,
		"port":    s.config.GRPCPort,
	}).Info("Exchange gRPC server initialized")

	// Start server in goroutine
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.logger.WithField("address", address).Info("Starting exchange gRPC server")

		if err := s.grpcServer.Serve(listener); err != nil {
			s.logger.WithError(err).Error("gRPC server error")
		}
	}()

	return nil
}

func (s *ExchangeGRPCServer) Stop(ctx context.Context) error {
	if !s.isRunning {
		return nil
	}

	s.logger.Info("Gracefully stopping exchange gRPC server")
	s.isRunning = false

	// Update health status
	if s.healthServer != nil {
		s.healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
		s.healthServer.SetServingStatus("exchange-simulator", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	}

	// Signal shutdown
	close(s.stopChan)

	// Graceful stop with timeout
	done := make(chan struct{})
	go func() {
		if s.grpcServer != nil {
			s.grpcServer.GracefulStop()
		}
		close(done)
	}()

	select {
	case <-done:
		s.logger.Info("Exchange gRPC server stopped")
	case <-ctx.Done():
		s.logger.Warn("Force stopping exchange gRPC server due to timeout")
		if s.grpcServer != nil {
			s.grpcServer.Stop()
		}
	}

	// Wait for all goroutines
	s.wg.Wait()
	return nil
}

func (s *ExchangeGRPCServer) GetMetrics() ExchangeServerMetrics {
	s.metricsLock.RLock()
	defer s.metricsLock.RUnlock()

	return ExchangeServerMetrics{
		StartTime:         s.startTime,
		UptimeSeconds:     int64(time.Since(s.startTime).Seconds()),
		ConnectionCount:   s.connectionCount,
		RequestCount:      s.requestCount,
		LastRequestTime:   s.lastRequestTime,
		IsRunning:         s.isRunning,
	}
}

func (s *ExchangeGRPCServer) GetHealthStatus() grpc_health_v1.HealthCheckResponse_ServingStatus {
	if s.healthServer == nil {
		return grpc_health_v1.HealthCheckResponse_NOT_SERVING
	}

	// Simple health check - can be enhanced with actual service checks
	if s.isRunning {
		return grpc_health_v1.HealthCheckResponse_SERVING
	}
	return grpc_health_v1.HealthCheckResponse_NOT_SERVING
}

// Unary interceptor for metrics and logging
func (s *ExchangeGRPCServer) unaryInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	start := time.Now()

	// Update metrics
	s.metricsLock.Lock()
	s.requestCount++
	s.lastRequestTime = start
	s.metricsLock.Unlock()

	// Log request
	s.logger.WithFields(logrus.Fields{
		"method":    info.FullMethod,
		"timestamp": start,
	}).Debug("gRPC request received")

	// Handle request
	resp, err := handler(ctx, req)

	// Log response
	duration := time.Since(start)
	logFields := logrus.Fields{
		"method":   info.FullMethod,
		"duration": duration,
		"success":  err == nil,
	}

	if err != nil {
		logFields["error"] = err.Error()
		s.logger.WithFields(logFields).Warn("gRPC request failed")
	} else {
		s.logger.WithFields(logFields).Debug("gRPC request completed")
	}

	return resp, err
}

// IsRunning returns the current running status
func (s *ExchangeGRPCServer) IsRunning() bool {
	s.metricsLock.RLock()
	defer s.metricsLock.RUnlock()
	return s.isRunning
}

// GetAddress returns the server address
func (s *ExchangeGRPCServer) GetAddress() string {
	if s.listener != nil {
		return s.listener.Addr().String()
	}
	return fmt.Sprintf(":%d", s.config.GRPCPort)
}