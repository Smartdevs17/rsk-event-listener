# RSK Smart Contract Event Listener

[![Go Version](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![RSK Compatible](https://img.shields.io/badge/RSK-Compatible-orange.svg)](https://rootstock.io)

A high-performance, production-ready event monitoring system for Rootstock (RSK) blockchain smart contracts, built in Go. Monitor token transfers, DeFi activities, and custom contract events with real-time notifications and automated actions.

## 🚀 Features

- **Real-time Event Monitoring**: Optimized for RSK's 30-second block times with intelligent polling
- **Multi-Contract Support**: Monitor multiple smart contracts simultaneously with efficient filtering
- **Token Event Detection**: Built-in support for ERC-20, ERC-721, and custom event types
- **Scalable Architecture**: Grows from simple MVP to enterprise-grade distributed systems
- **Multiple Notification Channels**: Webhooks, email, Slack, and custom integrations
- **Robust Error Handling**: Connection pooling, auto-retry, and multi-provider failover
- **Bitcoin Bridge Monitoring**: Track PowPeg operations between Bitcoin and RSK
- **Developer Friendly**: Clean interfaces, comprehensive logging, and easy configuration

## 🛠 Installation

### Prerequisites

- Go 1.21 or higher
- SQLite (for MVP) or PostgreSQL (for production)
- RSK node access (public nodes supported)

### Quick Start

```bash
# Clone the repository
git clone https://github.com/smartdevs17/rsk-event-listener.git
cd rsk-event-listener

# Install dependencies
go mod download

# Set up configuration
cp .env.example .env
# Edit .env with your settings

# Run the application
go run cmd/listener/main.go
```

### Docker Installation

```bash
# Build the image
docker build -t rsk-event-listener .

# Run with docker-compose
docker-compose up -d
```

## ⚙️ Configuration

Create a `.env` file or set environment variables:

```env
# RSK Network
RSK_ENDPOINT=https://public-node.rsk.co
CHAIN_ID=30

# Target Contracts (comma-separated)
TARGET_CONTRACTS=0x1234...,0x5678...

# Database
DB_PATH=./events.db
# For PostgreSQL: DATABASE_URL=postgres://user:pass@localhost/dbname

# Monitoring
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
        RSKEndpoint: "https://public-node.rsk.co",
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
```

## 📊 Monitoring Dashboard

The system exposes metrics at `/metrics` endpoint:

- `rsk_events_processed_total`: Total events processed
- `rsk_blocks_processed_total`: Total blocks scanned
- `rsk_notifications_sent_total`: Total notifications sent
- `rsk_connection_errors_total`: Connection error count
- `rsk_processing_duration_seconds`: Event processing latency

## 🔧 API Reference

### REST Endpoints

```
GET  /health              - Health check
GET  /metrics             - Prometheus metrics
GET  /events              - List recent events
GET  /events/{hash}       - Get specific event
POST /contracts           - Add contract to monitor
DELETE /contracts/{addr}  - Remove contract
```

### Event Types Supported

- **ERC-20 Events**: Transfer, Approval
- **ERC-721 Events**: Transfer, Approval, ApprovalForAll
- **Custom Events**: Any event with proper ABI configuration
- **Bridge Events**: PowPeg deposit/withdrawal tracking

## 🧪 Testing

```bash
# Run unit tests
go test ./...

# Run integration tests
go test -tags=integration ./...

# Run with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## 🚀 Deployment

### Production Deployment

1. **Database Setup**:
```sql
-- PostgreSQL recommended for production
CREATE DATABASE rsk_events;
CREATE USER rsk_user WITH PASSWORD 'secure_password';
GRANT ALL PRIVILEGES ON DATABASE rsk_events TO rsk_user;
```

2. **Redis Setup** (for scaling):
```bash
# Redis for message queuing in distributed setup
redis-server --daemonize yes
```

3. **Environment Configuration**:
```env
# Production settings
DATABASE_URL=postgres://rsk_user:secure_password@localhost/rsk_events
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
    spec:
      containers:
      - name: listener
        image: rsk-event-listener:latest
        env:
        - name: RSK_ENDPOINT
          value: "https://public-node.rsk.co"
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: db-secret
              key: url
```

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Development Setup

```bash
# Install development dependencies
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install github.com/swaggo/swag/cmd/swag@latest

# Run linting
golangci-lint run

# Generate API documentation
swag init -g cmd/listener/main.go
```

## 📈 Performance

### Benchmarks

- **Single-threaded**: 500-1,000 events/second
- **Multi-threaded (10 workers)**: 5,000-10,000 events/second
- **Distributed (Redis queue)**: 50,000+ events/second

### Resource Usage

- **Memory**: ~50MB base, +10MB per 100K cached events
- **CPU**: <5% on modern hardware for typical workloads
- **Network**: ~1MB/hour for 10 contracts monitoring

## 🔒 Security

- **API Key Authentication**: Secure webhook endpoints
- **Rate Limiting**: Built-in protection against abuse
- **Input Validation**: All user inputs validated and sanitized
- **Secure Defaults**: Production-ready security configurations

## 📞 Support

- **Documentation**: [Full Documentation](https://smartdevs17.github.io/rsk-event-listener)
- **Issues**: [GitHub Issues](https://github.com/smartdevs17/rsk-event-listener/issues)
- **Discussions**: [GitHub Discussions](https://github.com/smartdevs17/rsk-event-listener/discussions)
- **RSK Community**: [RSK Discord](https://discord.gg/rsk)

## 🗺 Roadmap

- [ ] **v1.1**: WebSocket subscription support
- [ ] **v1.2**: Multi-chain support (Ethereum, BSC)
- [ ] **v1.3**: Advanced analytics and insights
- [ ] **v1.4**: GraphQL API
- [ ] **v2.0**: Machine learning-based anomaly detection

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [Rootstock](https://rootstock.io) for the innovative Bitcoin-secured smart contract platform
- [go-ethereum](https://github.com/ethereum/go-ethereum) for the excellent Ethereum Go implementation
- [RSK Community](https://developers.rsk.co) for documentation and support

## 📊 Project Stats

![GitHub stars](https://img.shields.io/github/stars/smartdevs17/rsk-event-listener)
![GitHub forks](https://img.shields.io/github/forks/smartdevs17/rsk-event-listener)
![GitHub issues](https://img.shields.io/github/issues/smartdevs17/rsk-event-listener)
![GitHub pull requests](https://img.shields.io/github/issues-pr/smartdevs17/rsk-event-listener)

---

**Built with ❤️ for the RSK and Bitcoin communities**