# RSK Event Listener - Quick Start Integration Guide

## 🚀 Post-Testing Integration Steps

After all tests pass, follow these steps to set up full integration with Gmail credentials, contract addresses, and live RSK events.

## Prerequisites ✅

- [ ] All unit tests passing
- [ ] Gmail account with 2FA enabled
- [ ] Gmail App Password generated
- [ ] Contract addresses to monitor identified
- [ ] RSK network access (mainnet or testnet)

## Step 1: Run Full Integration Setup

```bash
# Make the integration script executable
chmod +x integration_setup.sh

# Run the full integration setup
./integration_setup.sh
```

This script will:
- Set up Gmail credentials
- Configure contract addresses
- Set RSK network settings
- Configure storage and API
- Test all connections
- Build production binary

## Step 2: Manual Configuration (Optional)

If you prefer manual configuration:

### Gmail Setup
```bash
# Add to .env file
GMAIL_USER=your-email@gmail.com
GMAIL_PASSWORD=your-app-password
NOTIFICATION_EMAIL=recipient@example.com
EMAIL_ENABLED=true
```

### Contract Addresses
```bash
# Popular RSK contracts
TARGET_CONTRACTS=0x542fDA317318eBF1d3DEAf76E0b632741A7e677d,0xEFc78fc7d48b64958315949279Ba181c2114ABBd
MONITOR_ENABLED=true
POLL_INTERVAL=15s
```

### RSK Network
```bash
# For mainnet
RSK_ENDPOINT=https://public-node.rsk.co
CHAIN_ID=30

# For testnet
RSK_ENDPOINT=https://public-node.testnet.rsk.co
CHAIN_ID=31
```

## Step 3: Test Individual Components

### Test Gmail Integration
```bash
# Test email notifications
./test_gmail_email.sh

# Expected output:
# ✅ Gmail credentials validated
# ✅ Test email sent successfully
# ✅ Email notifications working
```

### Test RSK Connectivity
```bash
# Create a test config first
cat > config/test.yaml << EOF
app:
  name: "rsk-event-listener"
  environment: "test"
rsk:
  node_url: "https://public-node.rsk.co"
  network_id: 30
  request_timeout: "30s"
storage:
  type: "sqlite"
  connection_string: "./data/test_events.db"
notifications:
  enabled: false
server:
  port: 8081
logging:
  level: "info"
  format: "text"
  output: "stdout"
EOF

# Test blockchain connection
go run cmd/listener/main.go test --config config/test.yaml

# Expected output:
# ✅ RSK connection successful
# ✅ Storage connection successful
# ✅ All connectivity tests passed
```

### Test Storage
```bash
# Test database operations first
# Make sure you have a config file set up
go run cmd/listener/storage_test_main.go

# Expected output:
# ✅ Database connection successful
# ✅ Tables created
# ✅ Test events stored
```

## Available Commands

Your RSK Event Listener uses a command-based CLI structure:

```bash
# Show all available commands
go run cmd/listener/main.go --help

# Available commands:
#   completion  Generate autocompletion script
#   config      Configuration management commands  
#   help        Help about any command
#   test        Test connectivity and configuration
#   version     Print the version number

# Test connectivity (requires config file)
go run cmd/listener/main.go test --config config/production.yaml

# Validate configuration
go run cmd/listener/main.go config validate --config config/production.yaml

# Show version
go run cmd/listener/main.go version

# Run the main listener (requires config file)
go run cmd/listener/main.go --config config/production.yaml
```

## Step 4: Start Full Integration
```bash
# Start with environment variables
source .env
go run cmd/listener/main.go
```

### Production Mode
```bash
# Use the built binary
./bin/rsk-event-listener --config config/production.yaml
```

### Docker Mode
```bash
# Build and run with Docker
docker build -t rsk-event-listener .
docker run -d --name rsk-listener --env-file .env rsk-event-listener
```

## Step 5: Monitor and Verify

### Check Health Status
```bash
# Health check
curl http://localhost:8081/api/v1/health

# Expected response:
{
  "status": "healthy",
  "timestamp": "2025-01-15T10:30:00Z",
  "checks": {
    "rsk_connectivity": "ok",
    "storage": "ok",
    "email": "ok"
  }
}
```

### View Recent Events
```bash
# Get recent events
curl http://localhost:8081/api/v1/events

# Expected response:
{
  "events": [
    {
      "id": "evt_123",
      "contract": "0x542fDA317318eBF1d3DEAf76E0b632741A7e677d",
      "event": "Transfer",
      "block_number": 1234567,
      "timestamp": "2025-01-15T10:30:00Z"
    }
  ],
  "total": 1,
  "page": 1
}
```

### Check Metrics
```bash
# Prometheus metrics
curl http://localhost:9090/metrics

# Look for these metrics:
# rsk_events_processed_total
# rsk_blocks_processed_total
# rsk_notifications_sent_total
```

## Step 6: Production Deployment

### Using Systemd
```bash
# Copy service file
sudo cp rsk-event-listener.service /etc/systemd/system/

# Create service user
sudo useradd -r -s /bin/false rsk-listener

# Create directories
sudo mkdir -p /opt/rsk-event-listener/{bin,config,data,logs}
sudo chown -R rsk-listener:rsk-listener /opt/rsk-event-listener

# Copy files
sudo cp bin/rsk-event-listener /opt/rsk-event-listener/bin/
sudo cp config/production.yaml /opt/rsk-event-listener/config/
sudo cp .env /opt/rsk-event-listener/

# Enable and start service
sudo systemctl daemon-reload
sudo systemctl enable rsk-event-listener
sudo systemctl start rsk-event-listener

# Check status
sudo systemctl status rsk-event-listener
```

### Using Docker Compose
```yaml
version: '3.8'
services:
  rsk-listener:
    build: .
    container_name: rsk-event-listener
    restart: unless-stopped
    env_file: .env
    ports:
      - "8081:8081"
      - "9090:9090"
    volumes:
      - ./data:/app/data
      - ./logs:/app/logs
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8081/api/v1/health"]
      interval: 30s
      timeout: 10s
      retries: 3
```

## Step 7: Monitoring and Alerts

### Log Monitoring
```bash
# View logs in real-time
sudo journalctl -u rsk-event-listener -f

# Or for Docker
docker logs -f rsk-event-listener
```

### Set Up Alerts
```bash
# Check if events are being processed
EVENTS_COUNT=$(curl -s http://localhost:8081/api/v1/events | jq '.total')
if [ "$EVENTS_COUNT" -eq 0 ]; then
    echo "⚠️  No events processed recently"
fi

# Check if notifications are working
if ! curl -s http://localhost:8081/api/v1/health | jq -r '.checks.email' | grep -q "ok"; then
    echo "⚠️  Email notifications not working"
fi
```

## Troubleshooting 🔧

### Common Issues

1. **Gmail Authentication Failed**
   ```bash
   # Check credentials
   echo $GMAIL_USER
   echo $GMAIL_PASSWORD | wc -c  # Should be 16 chars
   
   # Test manually
   go run test/mock/email_test_main.go
   ```

2. **RSK Connection Failed**
   ```bash
   # Test network connectivity manually
   curl -X POST -H "Content-Type: application/json" \
     --data '{"method":"eth_blockNumber","params":[],"id":1,"jsonrpc":"2.0"}' \
     https://public-node.rsk.co
   
   # Test with the application
   go run cmd/listener/main.go test --config config/production.yaml
   ```

3. **No Events Detected**
   ```bash
   # Check if contracts are active
   # Verify contract addresses
   # Check start block setting
   # Monitor for new blocks
   ```

4. **High Memory Usage**
   ```bash
   # Monitor memory usage
   ps aux | grep rsk-event-listener
   
   # Adjust batch size in config
   # Increase GC frequency
   ```

## Success Indicators ✅

- [ ] Health endpoint returns 200 OK
- [ ] Email notifications received
- [ ] Events appear in API responses
- [ ] Metrics show increasing counters
- [ ] Logs show "Event processed" messages
- [ ] No error messages in logs

## Next Steps 🎯

1. **Monitor for 24 hours** to ensure stability
2. **Set up backup** for event database
3. **Configure log rotation** to prevent disk space issues
4. **Set up monitoring alerts** for production
5. **Scale horizontally** if needed for high-volume contracts

## Support 📞

- Check logs: `sudo journalctl -u rsk-event-listener`
- Test components individually using provided scripts
- Monitor system resources (CPU, memory, disk)
- Verify network connectivity to RSK nodes
- Check Gmail security settings if emails fail

---

**🎉 Congratulations!** Your RSK Event Listener is now fully integrated and monitoring live blockchain events with email notifications!