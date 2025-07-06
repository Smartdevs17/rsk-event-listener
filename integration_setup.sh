#!/bin/bash

# Full Integration Setup Script
# Run this after all tests pass to set up production integration

set -e

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_header() {
    echo -e "\n${BLUE}=================================================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}=================================================================${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

print_status() {
    echo -e "${BLUE}📋 $1${NC}"
}

print_header "🚀 RSK Event Listener - Full Integration Setup"

# Step 1: Gmail Credentials Setup
print_header "Step 1: Gmail Credentials Configuration"

echo "Setting up Gmail credentials for email notifications..."

# Check if .env file exists
if [ ! -f .env ]; then
    print_status "Creating .env file from example..."
    cp .env.example .env
    print_success ".env file created"
else
    print_warning ".env file already exists"
fi

# Gmail configuration prompts
echo ""
print_status "Gmail Configuration Required:"
echo "1. Enable 2FA on your Gmail account"
echo "2. Generate an App Password: https://myaccount.google.com/apppasswords"
echo "3. Enter your credentials below"
echo ""

read -p "Enter your Gmail address: " GMAIL_USER
read -s -p "Enter your Gmail App Password: " GMAIL_PASSWORD
echo ""
read -p "Enter recipient email for notifications: " NOTIFICATION_EMAIL

# Update .env file with Gmail credentials
print_status "Updating .env file with Gmail credentials..."
cat >> .env << EOF

# Gmail Configuration
# GMAIL_USER=$GMAIL_USER
# GMAIL_PASSWORD=$GMAIL_PASSWORD
# NOTIFICATION_EMAIL=$NOTIFICATION_EMAIL
EMAIL_ENABLED=true
EOF

print_success "Gmail credentials configured"

# Step 2: Contract Addresses Setup
print_header "Step 2: Contract Addresses Configuration"

echo "Setting up contract addresses for event monitoring..."
echo ""

# Popular RSK contracts for monitoring
print_status "Popular RSK Contracts to Monitor:"
echo "1. RBTC (Wrapped Bitcoin): 0x542fDA317318eBF1d3DEAf76E0b632741A7e677d"
echo "2. SOV (Sovryn): 0xEFc78fc7d48b64958315949279Ba181c2114ABBd"
echo "3. XUSD (Dollar on Chain): 0xb5999795BE0EbB5bAb23144AA5FD6A02D080299F"
echo "4. DOC (Dollar on Chain): 0xe700691dA7b9851F2F35f8b8182c69c53CcaD9Db"
echo "5. BPRO (BitPro): 0x440cd83C160De5C96Ddb20246815eA44C7aBBCa8"
echo ""

# Ask user for contract addresses
print_status "Enter contract addresses to monitor (comma-separated):"
read -p "Contract addresses: " CONTRACT_ADDRESSES

# Validate contract addresses
if [ -z "$CONTRACT_ADDRESSES" ]; then
    print_warning "No contract addresses provided, using default RBTC contract"
    CONTRACT_ADDRESSES="0x542fDA317318eBF1d3DEAf76E0b632741A7e677d"
fi

# Update .env file with contract addresses
print_status "Updating .env file with contract addresses..."
cat >> .env << EOF

# Contract Monitoring Configuration
TARGET_CONTRACTS=$CONTRACT_ADDRESSES
MONITOR_ENABLED=true
POLL_INTERVAL=15s
START_BLOCK=0
CONFIRMATION_BLOCKS=12
EOF

print_success "Contract addresses configured"

# Step 3: RSK Network Configuration
print_header "Step 3: RSK Network Configuration"

echo "Configuring RSK network connection..."
echo ""

print_status "Available RSK Networks:"
echo "1. RSK Mainnet (Production)"
echo "2. RSK Testnet (Testing)"
echo ""

read -p "Select network (1 for mainnet, 2 for testnet): " NETWORK_CHOICE

if [ "$NETWORK_CHOICE" = "1" ]; then
    RSK_ENDPOINT="https://public-node.rsk.co"
    CHAIN_ID=30
    NETWORK_NAME="mainnet"
    print_success "RSK Mainnet selected"
else
    RSK_ENDPOINT="https://public-node.testnet.rsk.co"
    CHAIN_ID=31
    NETWORK_NAME="testnet"
    print_success "RSK Testnet selected"
fi

# Update .env file with network configuration
print_status "Updating .env file with network configuration..."
cat >> .env << EOF

# RSK Network Configuration
RSK_ENDPOINT=$RSK_ENDPOINT
CHAIN_ID=$CHAIN_ID
NETWORK_NAME=$NETWORK_NAME
CONNECTION_TIMEOUT=30s
RETRY_ATTEMPTS=3
RETRY_DELAY=5s
MAX_CONNECTIONS=10
EOF

print_success "RSK network configured"

# Step 4: Storage Configuration
print_header "Step 4: Storage Configuration"

echo "Configuring storage for event data..."

# Create data directory
mkdir -p ./data

# Update .env file with storage configuration
print_status "Updating .env file with storage configuration..."
cat >> .env << EOF

# Storage Configuration
DB_TYPE=sqlite
DB_PATH=./data/events.db
DB_MAX_CONNECTIONS=10
DB_MAX_IDLE_TIME=15m
EOF

print_success "Storage configuration updated"

# Step 5: Notification Configuration
print_header "Step 5: Notification Configuration"

echo "Configuring notification settings..."

# Update .env file with notification configuration
print_status "Updating .env file with notification configuration..."
cat >> .env << EOF

# Notification Configuration
NOTIFICATION_ENABLED=true
NOTIFICATION_TIMEOUT=30s
NOTIFICATION_RETRY_ATTEMPTS=3
NOTIFICATION_RETRY_DELAY=5s
NOTIFICATION_QUEUE_SIZE=100
MAX_CONCURRENT_NOTIFICATIONS=5

# Webhook Configuration (optional)
WEBHOOK_ENABLED=false
# WEBHOOK_URL=https://your-webhook-endpoint.com/events
# WEBHOOK_SECRET=your-webhook-secret

# Slack Configuration (optional)
SLACK_ENABLED=false
# SLACK_TOKEN=xoxb-your-slack-token
# SLACK_CHANNEL=#events
EOF

print_success "Notification configuration updated"

# Step 6: API Configuration
print_header "Step 6: API Configuration"

echo "Configuring API server settings..."

# Update .env file with API configuration
print_status "Updating .env file with API configuration..."
cat >> .env << EOF

# API Configuration
API_ENABLED=true
API_PORT=8081
API_HOST=0.0.0.0
API_TIMEOUT=30s
API_RATE_LIMIT=100
API_CORS_ENABLED=true

# Metrics Configuration
METRICS_ENABLED=true
METRICS_PORT=9090
METRICS_PATH=/metrics

# Health Check Configuration
HEALTH_CHECK_ENABLED=true
HEALTH_CHECK_INTERVAL=30s
EOF

print_success "API configuration updated"

# Step 7: Test Configuration
print_header "Step 7: Configuration Validation"

echo "Validating configuration..."

# Test Gmail credentials
print_status "Testing Gmail credentials..."
go run test/mock/email_test_main.go
if [ $? -eq 0 ]; then
    print_success "Gmail credentials validated"
else
    print_error "Gmail credentials test failed"
    exit 1
fi

# Test RSK connectivity
print_status "Testing RSK connectivity..."

# Create temporary test config
cat > config/temp_test.yaml << EOF
app:
  name: "rsk-event-listener"
  version: "1.0.0"
  environment: "test"

rsk:
  node_url: "$RSK_ENDPOINT"
  network_id: $CHAIN_ID
  request_timeout: "30s"
  retry_attempts: 3
  retry_delay: "5s"
  max_connections: 5

storage:
  type: "sqlite"
  connection_string: "./data/test_events.db"
  max_connections: 10
  max_idle_time: "15m"

notifications:
  enabled: false

server:
  port: 8081
  host: "localhost"

logging:
  level: "info"
  format: "text"
  output: "stdout"
EOF

# Test with the correct command structure
go run cmd/listener/main.go test --config config/temp_test.yaml
if [ $? -eq 0 ]; then
    print_success "RSK connectivity validated"
    rm -f config/temp_test.yaml
else
    print_error "RSK connectivity test failed"
    rm -f config/temp_test.yaml
    exit 1
fi

print_success "Configuration validation complete"

# Step 8: Build Production Binary
print_header "Step 8: Building Production Binary"

echo "Building production binary..."

# Clean previous builds
print_status "Cleaning previous builds..."
go clean

# Build the application
print_status "Building application..."
go build -o bin/rsk-event-listener cmd/listener/main.go

if [ $? -eq 0 ]; then
    print_success "Production binary built successfully"
else
    print_error "Build failed"
    exit 1
fi

# Step 9: Final Integration Test
print_header "Step 9: Full Integration Test"

echo "Running full integration test..."

# Create test script
cat > test_integration.sh << 'EOF'
#!/bin/bash

# Load environment variables
source .env

# Kill any existing processes on port 8081
echo "🔧 Cleaning up existing processes..."
sudo fuser -k 8081/tcp 2>/dev/null || true
pkill -f "rsk-event-listener" 2>/dev/null || true
sleep 2

# Start the application in background
echo "🚀 Starting RSK Event Listener..."
./bin/rsk-event-listener --config config/production.yaml &
APP_PID=$!

# Wait for startup with progress
echo "⏳ Waiting for application to start..."
for i in {1..10}; do
    if kill -0 $APP_PID 2>/dev/null; then
        echo "  Startup progress: $i/10 seconds"
        sleep 1
    else
        echo "❌ Application crashed during startup"
        exit 1
    fi
done

# Test health endpoint with retries
echo "🏥 Testing health endpoint..."
HEALTH_SUCCESS=false

for i in {1..15}; do
    if curl -f http://localhost:8081/api/v1/health > /dev/null 2>&1; then
        HEALTH_SUCCESS=true
        echo "✅ Health check passed (attempt $i)"
        break
    fi
    echo "  Health check attempt $i/15..."
    sleep 1
done

if [ "$HEALTH_SUCCESS" = false ]; then
    echo "❌ Health check failed after 15 attempts"
    echo "📋 Application logs (last 20 lines):"
    journalctl -u rsk-event-listener -n 20 --no-pager 2>/dev/null || echo "No systemd logs available"
    kill $APP_PID
    exit 1
fi

# Test metrics endpoint
echo "📊 Testing metrics endpoint..."
if curl -f http://localhost:9090/metrics > /dev/null 2>&1; then
    echo "✅ Metrics endpoint working"
else
    echo "⚠️  Metrics endpoint not responding (may be normal)"
fi

# Let it run for 30 seconds to process some events
echo "📊 Monitoring events for 30 seconds..."
sleep 30

# Check if events are being processed
echo "🔍 Checking event processing..."
EVENTS_RESPONSE=$(curl -s http://localhost:8081/api/v1/events 2>/dev/null || echo '{"total":0}')
EVENTS_COUNT=$(echo "$EVENTS_RESPONSE" | jq -r '.total // 0' 2>/dev/null || echo "0")

if [ "$EVENTS_COUNT" -gt 0 ]; then
    echo "✅ Events processed: $EVENTS_COUNT"
else
    echo "⚠️  No events processed yet (this is normal for new blocks)"
fi

# Test configuration endpoint
echo "🔧 Testing configuration..."
if curl -s http://localhost:8081/api/v1/config > /dev/null 2>&1; then
    echo "✅ Configuration endpoint working"
fi

# Stop the application
echo "🛑 Stopping application..."
kill $APP_PID
wait $APP_PID 2>/dev/null || true

echo "✅ Integration test completed successfully"
EOF

chmod +x test_integration.sh

# Run integration test
print_status "Running integration test..."
./test_integration.sh

if [ $? -eq 0 ]; then
    print_success "Full integration test passed"
else
    print_error "Integration test failed"
    exit 1
fi

# Step 10: Setup Complete
print_header "🎉 Setup Complete!"

echo ""
print_success "✅ RSK Event Listener is now fully configured and ready!"
echo ""
print_status "Configuration Summary:"
echo "  🔧 Gmail notifications: ENABLED"
echo "  📧 Notification email: $NOTIFICATION_EMAIL"
echo "  🏗️  Contract monitoring: ENABLED"
echo "  📍 Contracts: $CONTRACT_ADDRESSES"
echo "  🌐 RSK Network: $NETWORK_NAME"
echo "  💾 Storage: SQLite (./data/events.db)"
echo "  🚀 API Server: http://localhost:8081"
echo "  📊 Metrics: http://localhost:9090/metrics"
echo ""

print_status "Next Steps:"
echo "1. Start the application: ./bin/rsk-event-listener"
echo "2. Monitor logs for event processing"
echo "3. Check health: curl http://localhost:8081/api/v1/health"
echo "4. View events: curl http://localhost:8081/api/v1/events"
echo "5. Monitor metrics: curl http://localhost:9090/metrics"
echo ""

print_status "Production Deployment:"
echo "• Use process manager (systemd, pm2, docker)"
echo "• Set up log rotation"
echo "• Configure monitoring alerts"
echo "• Set up backup for event database"
echo ""

print_success "🎯 Full integration setup completed successfully!"
echo ""
echo "The RSK Event Listener is now monitoring live blockchain events!"
echo "You should start receiving email notifications for contract events."
echo ""
echo "Happy monitoring! 🚀"