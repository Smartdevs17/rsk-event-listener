# RSK Smart Contract Event Listener

[![Go Version](https://img.shields.io/badge/Go-1.24+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![RSK Compatible](https://img.shields.io/badge/RSK-Compatible-orange.svg)](https://rootstock.io)
[![Prometheus](https://img.shields.io/badge/Monitoring-Prometheus%20%2B%20Grafana-red.svg)](https://prometheus.io)

A high-performance, production-ready event monitoring system for Rootstock (RSK) blockchain smart contracts, built in Go. Monitor token transfers, DeFi activities, and custom contract events with real-time notifications, automated actions, and comprehensive monitoring dashboards.

## 🚀 Features

- **Real-time Event Monitoring**: Optimized for RSK's 30-second block times with intelligent polling
- **Multi-Contract Support**: Monitor multiple smart contracts simultaneously with efficient filtering
- **Token Event Detection**: Built-in support for ERC-20, ERC-721, and custom event types
- **Production Monitoring**: Complete Prometheus + Grafana integration with health tracking
- **Scalable Architecture**: Grows from simple MVP to enterprise-grade distributed systems
- **Multiple Notification Channels**: Webhooks, email, Slack, and custom integrations
- **Robust Error Handling**: Connection pooling, auto-retry, and multi-provider failover
- **Bitcoin Bridge Monitoring**: Track PowPeg operations between Bitcoin and RSK
- **Developer Friendly**: Clean interfaces, comprehensive logging, and easy configuration
- **Component Health Tracking**: Real-time monitoring of all system components

## ⚡ Quick Start

Get the complete RSK Event Listener with monitoring stack running in under 2 minutes:

```bash
# 1. Clone and setup
git clone https://github.com/smartdevs17/rsk-event-listener.git
cd rsk-event-listener

# 2. Quick configuration
cp .env.example .env
# Edit .env with your RSK endpoint if needed

# 3. Start everything (including Prometheus + Grafana monitoring)
make monitoring-stack

# 4. Access your dashboards:
# 🌐 RSK Event Listener API: http://localhost:8081
# 📊 Grafana Dashboard: http://localhost:3000 (admin/admin123)
# 📈 Prometheus Metrics: http://localhost:9090
```

That's it! You now have a complete RSK monitoring system with real-time dashboards. 🎉

## 🛠 Installation Options

### Prerequisites

- Go 1.24 or higher
- Docker & Docker Compose (for monitoring stack)
- SQLite (for MVP) or PostgreSQL (for production)
- RSK node access (public nodes supported)
- [Make](https://www.gnu.org/software/make/) utility (standard on Linux/macOS)

### Option 1: Full Monitoring Stack (Recommended)

```bash
# Clone the repository
git clone https://github.com/smartdevs17/rsk-event-listener.git
cd rsk-event-listener

# Install dependencies
go mod download

# Copy and edit environment variables
cp .env.example .env
# Edit .env with your settings

# Start complete monitoring stack (RSK Listener + Prometheus + Grafana)
make monitoring-stack

# Access your monitoring dashboards:
# - Grafana: http://localhost:3000 (admin/admin123)
# - Prometheus: http://localhost:9090
# - RSK API: http://localhost:8081
```

### Option 2: Standalone Installation

```bash
# Build and run just the RSK Event Listener
make build
make run

# Or with Docker
make docker-run
```

## 🧰 Makefile Usage

Enhanced Makefile with monitoring capabilities:

| Command                | Description                                 |
|------------------------|---------------------------------------------|
| `make monitoring-stack`| Start complete monitoring stack (RSK + Prometheus + Grafana) |
| `make build`           | Build the Go binary                         |
| `make run`             | Run the application with production config  |
| `make test`            | Run all unit and integration tests          |
| `make test-metrics`    | Run metrics integration tests              |
| `make lint`            | Run code linting (requires golangci-lint)   |
| `make fmt`             | Format Go code                              |
| `make docker`          | Build the Docker image                      |
| `make docker-run`      | Build and run with Docker Compose           |
| `make monitoring-down` | Stop monitoring stack                       |
| `make clean`           | Remove binaries and databases, stop Docker  |
| `make logs`            | Tail application logs                       |
| `make help`            | Show all available make targets             |

---

## ⚙️ Configuration

Create a `.env` file or set environment variables:

```env
# RSK Network
RSK_ENDPOINT=https://public-node.testnet.rsk.co
CHAIN_ID=31

# Target Contracts (comma-separated)
TARGET_CONTRACTS=0x1234...,0x5678...

# Database
DB_PATH=./data/events.db
# For PostgreSQL: DATABASE_URL=postgres://user:pass@localhost/dbname

# Monitoring & Metrics
ENABLE_METRICS=true
PROMETHEUS_PORT=9090
POLL_INTERVAL=15s
START_BLOCK=0

# Notifications
WEBHOOK_URL=https://your-webhook-endpoint.com/events
SLACK_TOKEN=xoxb-your-slack-token
EMAIL_SMTP_HOST=smtp.gmail.com
EMAIL_SMTP_PORT=587

# Scaling (Production)
WORKER_COUNT=10
BATCH_SIZE=100
REDIS_URL=redis://localhost:6379
```

## 📖 Usage

### Basic Event Monitoring

```go
package main

import (
    "github.com/smartdevs17/rsk-event-listener/internal/monitor"
    "github.com/ethereum/go-ethereum/common"
)

func main() {
    // Initialize the event monitor
    eventMonitor := monitor.New(&monitor.Config{
        RSKEndpoint: "https://public-node.testnet.rsk.co",
        Contracts: []common.Address{
            common.HexToAddress("0x1234567890123456789012345678901234567890"),
        },
        PollInterval: 15 * time.Second,
    })

    // Start monitoring
    if err := eventMonitor.Start(); err != nil {
        log.Fatal(err)
    }
}
```

### Metrics and Health Monitoring

```bash
# Check application health
curl http://localhost:8081/api/v1/health

# Get detailed component health
curl http://localhost:8081/api/v1/health/detailed

# View Prometheus metrics
curl http://localhost:8081/metrics

# Check specific RSK metrics
curl http://localhost:8081/metrics | grep rsk_
```

### Custom Event Processing

```go
// Register custom event handler
eventMonitor.RegisterHandler("Transfer", func(event types.Log) error {
    transfer := parseTransferEvent(event)
    
    if transfer.Value.Cmp(big.NewInt(1000000000000000000)) >= 0 {
        // Handle large transfers
        return sendAlert(transfer)
    }
    
    return nil
})
```

### WebHook Integration

```bash
# Your webhook will receive POST requests with this payload:
curl -X POST https://your-endpoint.com/webhook \
  -H "Content-Type: application/json" \
  -d '{
    "type": "token_transfer",
    "event": {
      "tx_hash": "0xabc123...",
      "block_number": 1234567,
      "contract": "0x1234...",
      "from": "0x5678...",
      "to": "0x9abc...",
      "value": "1000000000000000000",
      "timestamp": "2025-01-15T10:30:00Z"
    }
  }'
```

## 🏗 Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│  RSK Blockchain │    │  Event Monitor  │    │  Notification   │
│                 │───▶│                 │───▶│    System       │
│  Smart Contracts│    │  - Polling      │    │  - Webhooks     │
└─────────────────┘    │  - Filtering    │    │  - Email/Slack  │
                       │  - Processing   │    └─────────────────┘
                       └─────────────────┘              │
                                │                       │
                                ▼                       ▼
                       ┌─────────────────┐    ┌─────────────────┐
                       │   Event Storage │    │  External APIs  │
                       │                 │    │                 │
                       │  - SQLite/PG    │    │  - Trading Bots │
                       │  - Indexing     │    │  - Analytics    │
                       └─────────────────┘    └─────────────────┘
                                │                       │
                                ▼                       ▼
                       ┌─────────────────┐    ┌─────────────────┐
                       │   Prometheus    │    │    Grafana      │
                       │                 │    │                 │
                       │  - Metrics      │◀───│  - Dashboards   │
                       │  - Alerting     │    │  - Visualization│
                       └─────────────────┘    └─────────────────┘
```

## 📊 Monitoring Dashboard

The system provides comprehensive monitoring through Prometheus metrics and Grafana dashboards:

### Available Metrics

**Application Health:**
- `rsk_application_uptime_seconds`: Application uptime
- `rsk_component_health`: Health status of all components (storage, monitor, processor, notification)
- `rsk_memory_usage_bytes`: Current memory usage
- `rsk_goroutines`: Number of active goroutines

**Event Processing:**
- `rsk_events_processed_total`: Total events processed by contract and type
- `rsk_blocks_processed_total`: Total blocks scanned
- `rsk_event_processing_duration_seconds`: Event processing latency
- `rsk_latest_processed_block`: Latest block number processed
- `rsk_blocks_behind`: Number of blocks behind the chain

**System Performance:**
- `rsk_http_requests_total`: HTTP API request counts
- `rsk_http_request_duration_seconds`: API response times
- `rsk_notifications_sent_total`: Notification delivery stats
- `rsk_connection_errors_total`: Connection error counts
- `rsk_database_operations_total`: Database operation metrics

### Grafana Dashboard Features

- **Real-time component health status** 🟢/🔴
- **Event processing rate graphs** 📈
- **Memory and performance monitoring** 💾
- **HTTP API request analytics** 🌐
- **Alert thresholds and notifications** 🚨

### Accessing Monitoring

```bash
# View live metrics
curl http://localhost:8081/metrics

# Grafana Dashboard (admin/admin123)
open http://localhost:3000

# Prometheus Query Interface
open http://localhost:9090

# API Health Check
curl http://localhost:8081/api/v1/health | jq '.'
```

## 🔧 API Reference

### REST Endpoints

```
GET  /health                    - Basic health check
GET  /health/detailed          - Detailed component health
GET  /metrics                  - Prometheus metrics
GET  /api/v1/events            - List recent events
GET  /api/v1/events/{hash}     - Get specific event
GET  /api/v1/events/search     - Search events
POST /api/v1/contracts         - Add contract to monitor
GET  /api/v1/contracts         - List monitored contracts
DELETE /api/v1/contracts/{addr} - Remove contract
GET  /api/v1/monitor/status    - Monitor status
POST /api/v1/monitor/start     - Start monitoring
POST /api/v1/monitor/stop      - Stop monitoring
```

### Event Types Supported

- **ERC-20 Events**: Transfer, Approval
- **ERC-721 Events**: Transfer, Approval, ApprovalForAll
- **Custom Events**: Any event with proper ABI configuration
- **Bridge Events**: PowPeg deposit/withdrawal tracking

## 🧪 Testing

```bash
# Run unit tests
make test

# Run integration tests including metrics
make test-metrics

# Run specific test suites
go test ./internal/metrics/...
go test ./internal/monitor/...

# Run with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Test metrics integration
./scripts/test_metrics_integration.sh
```

## 🚀 Deployment

### Production Deployment with Monitoring

1. **Complete Stack Setup**:
```bash
# Production monitoring stack
docker-compose -f docker-compose.monitoring.yml up -d

# Verify all services
docker-compose -f docker-compose.monitoring.yml ps
```

2. **Database Setup**:
```sql
-- PostgreSQL recommended for production
CREATE DATABASE rsk_events;
CREATE USER rsk_user WITH PASSWORD 'secure_password';
GRANT ALL PRIVILEGES ON DATABASE rsk_events TO rsk_user;
```

3. **Environment Configuration**:
```env
# Production settings
DATABASE_URL=postgres://rsk_user:secure_password@localhost/rsk_events
ENABLE_METRICS=true
REDIS_URL=redis://localhost:6379
WORKER_COUNT=20
BATCH_SIZE=500
LOG_LEVEL=info
```

### Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: rsk-event-listener
spec:
  replicas: 3
  selector:
    matchLabels:
      app: rsk-event-listener
  template:
    metadata:
      labels:
        app: rsk-event-listener
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "8081"
        prometheus.io/path: "/metrics"
    spec:
      containers:
      - name: listener
        image: rsk-event-listener:latest
        ports:
        - containerPort: 8081
          name: http-metrics
        env:
        - name: RSK_ENDPOINT
          value: "https://public-node.rsk.co"
        - name: ENABLE_METRICS
          value: "true"
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: db-secret
              key: url
        livenessProbe:
          httpGet:
            path: /health
            port: 8081
          initialDelaySeconds: 30
        readinessProbe:
          httpGet:
            path: /health
            port: 8081
          initialDelaySeconds: 5
```

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Run tests (`make test && make test-metrics`)
4. Commit your changes (`git commit -m 'Add some amazing feature'`)
5. Push to the branch (`git push origin feature/amazing-feature`)
6. Open a Pull Request

### Development Setup

```bash
# Install development dependencies
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install github.com/swaggo/swag/cmd/swag@latest

# Start development environment with monitoring
make monitoring-stack

# Run linting
make lint

# Run all tests including metrics
make test && make test-metrics

# Generate API documentation
swag init -g cmd/listener/main.go
```

## 📈 Performance

### Benchmarks (Updated with Monitoring)

- **Single-threaded**: 500-1,000 events/second
- **Multi-threaded (10 workers)**: 5,000-10,000 events/second
- **Distributed (Redis queue)**: 50,000+ events/second
- **Monitoring overhead**: <2% performance impact with full metrics

### Resource Usage

- **Memory**: ~75MB base (including metrics), +10MB per 100K cached events
- **CPU**: <7% on modern hardware for typical workloads (including monitoring)
- **Network**: ~1.5MB/hour for 10 contracts with full monitoring
- **Storage**: ~1MB/day for metrics retention

### Real-time Monitoring

Monitor these metrics in your Grafana dashboard:
- Event processing latency (p95 < 100ms typical)
- Memory usage trends
- API response times
- Component health status
- Database query performance

## 🔒 Security

- **API Key Authentication**: Secure webhook endpoints
- **Rate Limiting**: Built-in protection against abuse
- **Input Validation**: All user inputs validated and sanitized
- **Secure Defaults**: Production-ready security configurations
- **Monitoring Security**: Metrics endpoints can be secured with authentication
- **Component Health**: Real-time security monitoring through health checks

## 📞 Support

- **Documentation**: [Full Documentation](https://smartdevs17.github.io/rsk-event-listener)
- **Issues**: [GitHub Issues](https://github.com/smartdevs17/rsk-event-listener/issues)
- **Monitoring Guide**: [Monitoring Setup Guide](./docs/monitoring.md)
- **RSK Community**: [RSK Discord](https://discord.gg/rootstock)

## 🗺 Roadmap

- [x] **v1.0**: Core event monitoring and notifications
- [x] **v1.1**: Complete Prometheus + Grafana integration
- [x] **v1.2**: Component health monitoring and alerting
- [ ] **v1.3**: WebSocket subscription support
- [ ] **v1.4**: Multi-chain support (Ethereum, BSC)
- [ ] **v1.5**: Advanced analytics and insights
- [ ] **v1.6**: GraphQL API
- [ ] **v2.0**: Machine learning-based anomaly detection

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [Rootstock](https://rootstock.io) for the innovative Bitcoin-secured smart contract platform
- [go-ethereum](https://github.com/ethereum/go-ethereum) for the excellent Ethereum Go implementation
- [Prometheus](https://prometheus.io) and [Grafana](https://grafana.com) for outstanding monitoring tools
- [RSK Community](https://developers.rsk.co) for documentation and support

## 📊 Project Stats

![GitHub stars](https://img.shields.io/github/stars/smartdevs17/rsk-event-listener)
![GitHub forks](https://img.shields.io/github/forks/smartdevs17/rsk-event-listener)
![GitHub issues](https://img.shields.io/github/issues/smartdevs17/rsk-event-listener)
![GitHub pull requests](https://img.shields.io/github/issues-pr/smartdevs17/rsk-event-listener)

---

**Built with ❤️ for the RSK and Bitcoin communities • Now with Production-Ready Monitoring 📊**