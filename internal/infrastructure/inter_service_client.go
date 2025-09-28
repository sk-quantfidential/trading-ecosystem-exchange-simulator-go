package infrastructure

import (
	"context"
	"fmt"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
	"github.com/sirupsen/logrus"

	"github.com/quantfidential/trading-ecosystem/exchange-simulator-go/internal/config"
)

// ServiceUnavailableError represents an error when a service is not available
type ServiceUnavailableError struct {
	ServiceName string
	Message     string
}

func (e *ServiceUnavailableError) Error() string {
	return fmt.Sprintf("service %s unavailable: %s", e.ServiceName, e.Message)
}

// InterServiceClientManager manages gRPC clients for inter-service communication
type InterServiceClientManager struct {
	config              *config.Config
	logger              *logrus.Logger
	serviceDiscovery    *ServiceDiscoveryClient
	configurationClient *ConfigurationClient
	connections         map[string]*grpc.ClientConn
	clients             map[string]interface{}
	connectionMutex     sync.RWMutex
	clientMutex         sync.RWMutex
	metrics             InterServiceMetrics
	metricsMutex        sync.RWMutex
	ctx                 context.Context
	cancel              context.CancelFunc
}

type InterServiceMetrics struct {
	ActiveConnections     int       `json:"active_connections"`
	TotalConnections      int64     `json:"total_connections"`
	FailedConnections     int64     `json:"failed_connections"`
	LastConnectionAttempt time.Time `json:"last_connection_attempt"`
	ServiceCallCount      int64     `json:"service_call_count"`
	ServiceCallErrors     int64     `json:"service_call_errors"`
	CircuitBreakerTrips   int64     `json:"circuit_breaker_trips"`
}

// AuditCorrelatorClient interface for audit-correlator service
type AuditCorrelatorClient interface {
	HealthCheck(ctx context.Context) error
	SubmitAuditEvent(ctx context.Context, event interface{}) error
}

// CustodianSimulatorClient interface for custodian-simulator service
type CustodianSimulatorClient interface {
	HealthCheck(ctx context.Context) error
	ProcessSettlement(ctx context.Context, settlement interface{}) error
}

type auditCorrelatorClientImpl struct {
	conn         *grpc.ClientConn
	healthClient grpc_health_v1.HealthClient
	logger       *logrus.Logger
}

type custodianSimulatorClientImpl struct {
	conn         *grpc.ClientConn
	healthClient grpc_health_v1.HealthClient
	logger       *logrus.Logger
}

func NewInterServiceClientManager(
	cfg *config.Config,
	logger *logrus.Logger,
	serviceDiscovery *ServiceDiscoveryClient,
	configurationClient *ConfigurationClient,
) *InterServiceClientManager {
	ctx, cancel := context.WithCancel(context.Background())

	return &InterServiceClientManager{
		config:              cfg,
		logger:              logger,
		serviceDiscovery:    serviceDiscovery,
		configurationClient: configurationClient,
		connections:         make(map[string]*grpc.ClientConn),
		clients:             make(map[string]interface{}),
		ctx:                 ctx,
		cancel:              cancel,
		metrics: InterServiceMetrics{
			ActiveConnections: 0,
		},
	}
}

func (m *InterServiceClientManager) GetAuditCorrelatorClient() (AuditCorrelatorClient, error) {
	serviceName := "audit-correlator"

	client, exists := m.getClient(serviceName)
	if exists {
		if auditClient, ok := client.(AuditCorrelatorClient); ok {
			return auditClient, nil
		}
	}

	// Create new client
	conn, err := m.getOrCreateConnection(serviceName)
	if err != nil {
		return nil, &ServiceUnavailableError{
			ServiceName: serviceName,
			Message:     err.Error(),
		}
	}

	auditClient := &auditCorrelatorClientImpl{
		conn:         conn,
		healthClient: grpc_health_v1.NewHealthClient(conn),
		logger:       m.logger,
	}

	m.setClient(serviceName, auditClient)

	m.logger.WithField("service", serviceName).Info("Audit correlator client created")
	return auditClient, nil
}

func (m *InterServiceClientManager) GetCustodianSimulatorClient() (CustodianSimulatorClient, error) {
	serviceName := "custodian-simulator"

	client, exists := m.getClient(serviceName)
	if exists {
		if custodianClient, ok := client.(CustodianSimulatorClient); ok {
			return custodianClient, nil
		}
	}

	// Create new client
	conn, err := m.getOrCreateConnection(serviceName)
	if err != nil {
		return nil, &ServiceUnavailableError{
			ServiceName: serviceName,
			Message:     err.Error(),
		}
	}

	custodianClient := &custodianSimulatorClientImpl{
		conn:         conn,
		healthClient: grpc_health_v1.NewHealthClient(conn),
		logger:       m.logger,
	}

	m.setClient(serviceName, custodianClient)

	m.logger.WithField("service", serviceName).Info("Custodian simulator client created")
	return custodianClient, nil
}

func (m *InterServiceClientManager) GetMetrics() InterServiceMetrics {
	m.metricsMutex.RLock()
	defer m.metricsMutex.RUnlock()
	return m.metrics
}

func (m *InterServiceClientManager) Close() error {
	m.logger.Info("Closing inter-service client manager")

	// Cancel context
	m.cancel()

	// Close all connections
	m.connectionMutex.Lock()
	defer m.connectionMutex.Unlock()

	for serviceName, conn := range m.connections {
		if conn != nil {
			err := conn.Close()
			if err != nil {
				m.logger.WithError(err).WithField("service", serviceName).Error("Failed to close connection")
			} else {
				m.logger.WithField("service", serviceName).Debug("Connection closed")
			}
		}
	}

	// Clear connections and clients
	m.connections = make(map[string]*grpc.ClientConn)
	m.clients = make(map[string]interface{})

	m.updateActiveConnections(0)

	return nil
}

func (m *InterServiceClientManager) getOrCreateConnection(serviceName string) (*grpc.ClientConn, error) {
	m.connectionMutex.Lock()
	defer m.connectionMutex.Unlock()

	// Check if connection already exists and is ready
	if conn, exists := m.connections[serviceName]; exists {
		state := conn.GetState()
		if state == connectivity.Ready || state == connectivity.Idle {
			return conn, nil
		}
		// Close bad connection
		conn.Close()
		delete(m.connections, serviceName)
	}

	m.incrementConnectionAttempt()

	// Discover service endpoint
	endpoint, err := m.serviceDiscovery.GetServiceEndpoint(serviceName)
	if err != nil {
		m.incrementFailedConnection()
		return nil, fmt.Errorf("failed to discover service %s: %w", serviceName, err)
	}

	// Create new connection with timeout
	ctx, cancel := context.WithTimeout(m.ctx, 10*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, endpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(m.unaryInterceptor),
		grpc.WithBlock(),
	)
	if err != nil {
		m.incrementFailedConnection()
		return nil, fmt.Errorf("failed to connect to %s at %s: %w", serviceName, endpoint, err)
	}

	m.connections[serviceName] = conn
	m.incrementTotalConnection()
	m.updateActiveConnections(len(m.connections))

	m.logger.WithFields(logrus.Fields{
		"service":  serviceName,
		"endpoint": endpoint,
	}).Info("Service connection established")

	return conn, nil
}

func (m *InterServiceClientManager) getClient(serviceName string) (interface{}, bool) {
	m.clientMutex.RLock()
	defer m.clientMutex.RUnlock()
	client, exists := m.clients[serviceName]
	return client, exists
}

func (m *InterServiceClientManager) setClient(serviceName string, client interface{}) {
	m.clientMutex.Lock()
	defer m.clientMutex.Unlock()
	m.clients[serviceName] = client
}

func (m *InterServiceClientManager) unaryInterceptor(
	ctx context.Context,
	method string,
	req, reply interface{},
	cc *grpc.ClientConn,
	invoker grpc.UnaryInvoker,
	opts ...grpc.CallOption,
) error {
	start := time.Now()

	m.incrementServiceCall()

	err := invoker(ctx, method, req, reply, cc, opts...)

	duration := time.Since(start)

	if err != nil {
		m.incrementServiceCallError()
		m.logger.WithFields(logrus.Fields{
			"method":   method,
			"duration": duration,
			"error":    err.Error(),
		}).Warn("Inter-service call failed")
	} else {
		m.logger.WithFields(logrus.Fields{
			"method":   method,
			"duration": duration,
		}).Debug("Inter-service call completed")
	}

	return err
}

func (m *InterServiceClientManager) incrementConnectionAttempt() {
	m.metricsMutex.Lock()
	defer m.metricsMutex.Unlock()
	m.metrics.LastConnectionAttempt = time.Now()
}

func (m *InterServiceClientManager) incrementTotalConnection() {
	m.metricsMutex.Lock()
	defer m.metricsMutex.Unlock()
	m.metrics.TotalConnections++
}

func (m *InterServiceClientManager) incrementFailedConnection() {
	m.metricsMutex.Lock()
	defer m.metricsMutex.Unlock()
	m.metrics.FailedConnections++
}

func (m *InterServiceClientManager) incrementServiceCall() {
	m.metricsMutex.Lock()
	defer m.metricsMutex.Unlock()
	m.metrics.ServiceCallCount++
}

func (m *InterServiceClientManager) incrementServiceCallError() {
	m.metricsMutex.Lock()
	defer m.metricsMutex.Unlock()
	m.metrics.ServiceCallErrors++
}

func (m *InterServiceClientManager) updateActiveConnections(count int) {
	m.metricsMutex.Lock()
	defer m.metricsMutex.Unlock()
	m.metrics.ActiveConnections = count
}

// Implementation of AuditCorrelatorClient interface
func (c *auditCorrelatorClientImpl) HealthCheck(ctx context.Context) error {
	resp, err := c.healthClient.Check(ctx, &grpc_health_v1.HealthCheckRequest{
		Service: "audit-correlator",
	})
	if err != nil {
		return err
	}

	if resp.Status != grpc_health_v1.HealthCheckResponse_SERVING {
		return fmt.Errorf("audit-correlator service not serving")
	}

	return nil
}

func (c *auditCorrelatorClientImpl) SubmitAuditEvent(ctx context.Context, event interface{}) error {
	// In a real implementation, this would call the actual audit service gRPC method
	c.logger.WithField("event", event).Debug("Audit event submitted")
	return nil
}

// Implementation of CustodianSimulatorClient interface
func (c *custodianSimulatorClientImpl) HealthCheck(ctx context.Context) error {
	resp, err := c.healthClient.Check(ctx, &grpc_health_v1.HealthCheckRequest{
		Service: "custodian-simulator",
	})
	if err != nil {
		return err
	}

	if resp.Status != grpc_health_v1.HealthCheckResponse_SERVING {
		return fmt.Errorf("custodian-simulator service not serving")
	}

	return nil
}

func (c *custodianSimulatorClientImpl) ProcessSettlement(ctx context.Context, settlement interface{}) error {
	// In a real implementation, this would call the actual custodian service gRPC method
	c.logger.WithField("settlement", settlement).Debug("Settlement processed")
	return nil
}