# Exchange Simulator

A high-performance crypto exchange simulator built in Go that provides realistic order matching, account management, and chaos engineering capabilities for testing trading systems.

## 🎯 Overview

The Exchange Simulator replicates the behavior of major crypto exchanges (Binance, Coinbase Pro, etc.) with realistic latency, slippage, and liquidity constraints. It serves as a critical component in the trading ecosystem simulation, enabling comprehensive testing of trading strategies and risk management systems.

### Key Features
- **Realistic Order Matching**: FIFO price-time priority with configurable latency
- **Multi-Asset Support**: BTC/USD, USDT/BTC, USDT/ETH, ETH/USD, BTC/ETH trading pairs
- **Account Management**: Sub-account isolation with real-time balance tracking
- **Chaos Engineering**: Controllable failure injection for resilience testing
- **Production APIs**: Standard REST and gRPC interfaces matching real exchanges
- **Comprehensive Observability**: OpenTelemetry tracing and Prometheus metrics

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────┐
│                Exchange Simulator                       │
├─────────────────────────────────────────────────────────┤
│  gRPC Services          │  REST APIs                    │
│  ├─Trading Service      │  ├─Account Management         │
│  ├─Account Service      │  ├─Market Data                │
│  └─Market Data Service  │  └─Chaos Engineering          │
├─────────────────────────────────────────────────────────┤
│  Core Engine                                            │
│  ├─Order Matching Engine (FIFO Price-Time Priority)    │
│  ├─Account Manager (Sub-account isolation)             │
│  ├─Market Data Publisher (Real-time price feeds)       │
│  └─Chaos Controller (Failure injection)                │
├─────────────────────────────────────────────────────────┤
│  Data Layer                                             │
│  ├─Order Books (In-memory with persistence)            │
│  ├─Account Balances (Redis-backed)                     │
│  └─Trade History (PostgreSQL)                          │
└─────────────────────────────────────────────────────────┘
```

## 🚀 Quick Start

### Prerequisites
- Go 1.21+
- Docker and Docker Compose
- Protocol Buffers compiler

### Development Setup
```bash
# Clone the repository
git clone <repo-url>
cd exchange-simulator

# Install dependencies
go mod download

# Generate protobuf files
make generate-proto

# Run tests
make test

# Start development server
make run-dev
```

### Docker Deployment
```bash
# Build container
docker build -t exchange-simulator .

# Run with docker-compose (recommended)
docker-compose up exchange-simulator

# Verify health
curl http://localhost:8080/health
```

## 📡 API Reference

### gRPC Services

#### Trading Service
```protobuf
service TradingService {
  rpc PlaceOrder(PlaceOrderRequest) returns (PlaceOrderResponse);
  rpc CancelOrder(CancelOrderRequest) returns (CancelOrderResponse);
  rpc GetOrder(GetOrderRequest) returns (Order);
  rpc ListOrders(ListOrdersRequest) returns (ListOrdersResponse);
}
```

#### Account Service  
```protobuf
service AccountService {
  rpc GetBalance(GetBalanceRequest) returns (Balance);
  rpc GetPositions(GetPositionsRequest) returns (PositionsResponse);
  rpc CreateSubAccount(CreateSubAccountRequest) returns (SubAccount);
}
```

### REST Endpoints

#### Production APIs (Risk Monitor Accessible)
```
GET    /api/v1/accounts/{account_id}/balances
GET    /api/v1/accounts/{account_id}/positions  
GET    /api/v1/orderbook/{symbol}
GET    /api/v1/trades/{symbol}/recent
POST   /api/v1/orders
DELETE /api/v1/orders/{order_id}
```

#### Chaos Engineering APIs (Audit Only)
```
POST   /chaos/inject-latency
POST   /chaos/reject-orders  
POST   /chaos/simulate-downtime
POST   /chaos/manipulate-spreads
GET    /chaos/status
DELETE /chaos/clear-all
```

#### State Inspection APIs (Development/Audit)
```
GET    /debug/orderbooks
GET    /debug/accounts
GET    /debug/trade-history
GET    /metrics (Prometheus format)
```

## 🎮 Order Matching Engine

### Supported Order Types
- **Market Orders**: Immediate execution at best available price
- **Limit Orders**: Execute only at specified price or better

### Matching Algorithm
```
1. Price Priority: Best prices matched first
2. Time Priority: Earlier orders at same price matched first  
3. Partial Fills: Large orders filled incrementally
4. Realistic Latency: Configurable processing delays (1-50ms)
```

### Slippage Simulation
- **Market Impact**: Large orders move prices realistically
- **Liquidity Constraints**: Order book depth affects execution
- **Price Improvement**: Occasional better fills for market orders

## 💰 Account Management

### Sub-Account Architecture
```
Master Account: trading-firm-001
├── Sub-Account: strategy-arbitrage  
│   ├── BTC: 1.5 BTC
│   ├── USD: 50,000 USD
│   └── USDT: 25,000 USDT
├── Sub-Account: strategy-momentum
│   ├── ETH: 100 ETH  
│   └── USD: 75,000 USD
└── Sub-Account: risk-reserve
    └── USD: 100,000 USD
```

### Balance Management
- **Real-time Updates**: Balances updated immediately on trades
- **Margin Calculations**: Available balance considers open orders
- **Multi-Asset Support**: Native support for crypto and fiat assets
- **Precision Handling**: Proper decimal precision for all assets

## 🎭 Chaos Engineering

### Failure Injection Capabilities

#### Latency Injection
```bash
# Add 500ms delay to all order operations
curl -X POST localhost:8080/chaos/inject-latency \
  -d '{"operation": "place_order", "delay_ms": 500, "duration_s": 300}'
```

#### Order Rejection
```bash  
# Reject 20% of orders for next 5 minutes
curl -X POST localhost:8080/chaos/reject-orders \
  -d '{"rejection_rate": 0.2, "duration_s": 300, "reason": "insufficient_liquidity"}'
```

#### Exchange Downtime
```bash
# Simulate exchange offline for 30 seconds
curl -X POST localhost:8080/chaos/simulate-downtime \
  -d '{"duration_s": 30, "error_message": "exchange_maintenance"}'
```

#### Spread Manipulation
```bash
# Artificially widen spreads by 2x
curl -X POST localhost:8080/chaos/manipulate-spreads \
  -d '{"multiplier": 2.0, "symbols": ["BTC/USD", "ETH/USD"], "duration_s": 600}'
```

## 📊 Monitoring & Observability

### Prometheus Metrics
```
# Order processing metrics
exchange_orders_total{type="market|limit", status="filled|rejected|cancelled"}
exchange_order_latency_seconds{operation="place|cancel"}
exchange_volume_24h{symbol="BTC/USD"}

# Account metrics  
exchange_account_balance{account_id, asset}
exchange_active_orders{account_id, symbol}

# System health
exchange_uptime_seconds
exchange_chaos_active{type="latency|rejection|downtime"}
```

### OpenTelemetry Tracing
- **Request Correlation**: All operations traced with correlation IDs
- **Cross-Service**: Traces span calls to market data and shared storage
- **Performance**: Detailed timing for order matching and balance updates

### Structured Logging
```json
{
  "timestamp": "2025-09-16T14:23:45Z",
  "level": "info",
  "service": "exchange-simulator", 
  "correlation_id": "req-abc123",
  "event": "order_placed",
  "account_id": "strategy-arbitrage",
  "symbol": "BTC/USD",
  "order_type": "limit",
  "size": "0.1",
  "price": "45000.00"
}
```

## 🧪 Testing

### Unit Tests
```bash
# Run all unit tests
make test

# Run with coverage  
make test-coverage

# Run specific test suite
go test ./internal/matching -v
```

### Integration Tests
```bash
# Test with real Redis/PostgreSQL
make test-integration

# Test chaos injection
make test-chaos
```

### Load Testing
```bash
# Simulate high-frequency trading load
make load-test

# Custom scenario
go run cmd/load-test/main.go -orders-per-second=1000 -duration=60s
```

## ⚙️ Configuration

### Environment Variables
```bash
# Core settings
EXCHANGE_PORT=8080
EXCHANGE_GRPC_PORT=50051
EXCHANGE_LOG_LEVEL=info

# Dependencies
REDIS_URL=redis://localhost:6379
POSTGRES_URL=postgres://user:pass@localhost/exchange

# Simulation parameters
MATCHING_LATENCY_MS=10
MAX_ORDER_SIZE=1000000
ENABLE_SLIPPAGE=true

# Chaos engineering
CHAOS_ENABLED=true
CHAOS_DEFAULT_DURATION=300
```

### Configuration File (config.yaml)
```yaml
exchange:
  name: "simulator-1"
  trading_pairs:
    - symbol: "BTC/USD"
      base_asset: "BTC"
      quote_asset: "USD"
      min_size: "0.0001"
      max_size: "100"
      tick_size: "0.01"
    
matching_engine:
  latency_ms: 10
  enable_slippage: true
  max_spread_bps: 10

accounts:
  require_kyc: false
  max_sub_accounts: 100
  margin_enabled: false
```

## 🐳 Docker Configuration

### Dockerfile
```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o exchange-simulator cmd/server/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /app/exchange-simulator /usr/local/bin/
EXPOSE 8080 50051
CMD ["exchange-simulator"]
```

### Health Checks
```yaml
healthcheck:
  test: ["CMD", "grpc_health_probe", "-addr=:50051"]
  interval: 30s
  timeout: 10s
  retries: 3
  start_period: 40s
```

## 🔒 Security Considerations

### API Security
- **Rate Limiting**: Configurable per-account order rate limits
- **Input Validation**: Strict validation of all order parameters
- **Audit Logging**: All account operations logged for compliance

### Chaos API Protection
- **Network Isolation**: Chaos APIs only accessible on internal network
- **Authentication**: API key required for chaos operations
- **Safety Limits**: Maximum chaos duration and impact constraints

## 🚀 Performance

### Benchmarks
- **Order Throughput**: >10,000 orders/second (single instance)
- **Order Latency**: <10ms p99 under normal load
- **Memory Usage**: <100MB baseline, <500MB under load
- **Startup Time**: <5 seconds with empty order books

### Scaling Considerations
- **Horizontal Scaling**: Multiple instances with symbol sharding
- **Order Book Persistence**: Optional persistence for disaster recovery
- **Cache Strategy**: Redis for hot data, PostgreSQL for cold storage

## 🤝 Contributing

### Development Workflow
1. Create feature branch from `main`
2. Implement changes with tests
3. Run full test suite: `make test-all`
4. Update documentation if needed
5. Submit pull request with description

### Code Standards
- **Go formatting**: Use `gofmt` and `golint`
- **Test coverage**: Minimum 80% coverage for new code
- **Documentation**: All public functions must have godoc comments
- **Error handling**: Proper error wrapping and logging

## 📚 References

- **Trading System Design**: [Link to architecture docs]
- **Protobuf Schemas**: [Link to protobuf repository]
- **API Documentation**: [Link to OpenAPI specs]
- **Chaos Engineering Guide**: [Link to chaos testing docs]

---

**Status**: 🚧 Development Phase  
**Maintainer**: [Your team]  
**Last Updated**: September 2025
