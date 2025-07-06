#!/bin/bash

# Create production.yaml from your existing .env configuration
# This uses your current settings and creates the YAML config file

set -e

echo "🔧 Creating production.yaml from your .env settings..."

# Load your current .env
if [ ! -f .env ]; then
    echo "❌ .env file not found"
    exit 1
fi

source .env

# Create config directory
mkdir -p config

# Create production.yaml with your current settings
cat > config/production.yaml << EOF
# Production Configuration
# Generated from your .env settings

app:
  name: "rsk-event-listener"
  version: "1.0.0"
  environment: "${RSK_LISTENER_APP_ENVIRONMENT:-production}"
  debug: ${RSK_LISTENER_APP_DEBUG:-false}

rsk:
  node_url: "${RSK_LISTENER_RSK_NODE_URL}"
  network_id: ${RSK_LISTENER_RSK_NETWORK_ID}
  request_timeout: "${RSK_CONNECTION_TIMEOUT:-30s}"
  retry_attempts: ${RSK_RETRY_ATTEMPTS:-3}
  retry_delay: "${RSK_RETRY_DELAY:-5s}"
  max_connections: ${RSK_MAX_CONNECTIONS:-10}

storage:
  type: "${RSK_LISTENER_STORAGE_TYPE}"
  connection_string: "${DATABASE_URL}"
  max_connections: ${DB_MAX_CONNECTIONS:-25}
  max_idle_time: "${DB_MAX_IDLE_TIME:-15m}"

monitor:
  poll_interval: "${RSK_LISTENER_MONITOR_POLL_INTERVAL}"
  batch_size: ${MONITOR_BATCH_SIZE:-100}
  confirmation_blocks: ${MONITOR_CONFIRMATION_BLOCKS:-12}
  start_block: ${RSK_LISTENER_MONITOR_START_BLOCK}
  enable_websocket: false
  max_reorg_depth: ${MONITOR_MAX_REORG_DEPTH:-64}

processor:
  workers: ${PROCESSOR_WORKERS:-4}
  queue_size: ${PROCESSOR_QUEUE_SIZE:-1000}
  process_timeout: "${PROCESSOR_TIMEOUT:-30s}"
  retry_attempts: ${PROCESSOR_RETRY_ATTEMPTS:-3}
  retry_delay: "5s"
  enable_async: ${PROCESSOR_ENABLE_ASYNC:-true}

notifications:
  enabled: ${RSK_LISTENER_NOTIFICATIONS_ENABLED}
  queue_size: ${NOTIFICATION_QUEUE_SIZE:-100}
  workers: 2
  retry_delay: "${NOTIFICATION_RETRY_DELAY:-10s}"
  max_retries: ${NOTIFICATION_RETRY_ATTEMPTS:-3}
  default_channel: "log"
  max_concurrent_notifications: ${MAX_CONCURRENT_NOTIFICATIONS:-5}
  notification_timeout: "${NOTIFICATION_TIMEOUT:-30s}"
  retry_attempts: 3
  enable_email_notifications: true
  enable_webhook_notifications: ${WEBHOOK_ENABLED:-false}

server:
  port: ${RSK_LISTENER_SERVER_PORT}
  host: "${RSK_LISTENER_SERVER_HOST}"
  read_timeout: "${SERVER_READ_TIMEOUT:-10s}"
  write_timeout: "${SERVER_WRITE_TIMEOUT:-10s}"
  enable_metrics: true
  enable_health: ${SERVER_ENABLE_HEALTH:-true}

logging:
  level: "${RSK_LISTENER_LOGGING_LEVEL}"
  format: "${RSK_LISTENER_LOGGING_FORMAT}"
  output: "stdout"
  file: ""
EOF

echo "✅ config/production.yaml created successfully!"

# Test the configuration
echo ""
echo "🧪 Testing the configuration..."

if go run cmd/listener/main.go config validate --config config/production.yaml > /dev/null 2>&1; then
    echo "✅ Configuration validation passed!"
else
    echo "❌ Configuration validation failed"
    echo "Running validation to see errors:"
    go run cmd/listener/main.go config validate --config config/production.yaml
    exit 1
fi

# Test connectivity
echo ""
echo "🌐 Testing connectivity..."
if go run cmd/listener/main.go test --config config/production.yaml > /dev/null 2>&1; then
    echo "✅ Connectivity test passed!"
else
    echo "⚠️  Connectivity test had issues (check network/endpoints)"
fi

echo ""
echo "🎯 Your production.yaml is ready!"
echo ""
echo "Configuration Summary:"
echo "  📡 RSK Network: ${RSK_LISTENER_RSK_NETWORK_ID} (${RSK_LISTENER_RSK_NODE_URL})"
echo "  🚀 Server Port: ${RSK_LISTENER_SERVER_PORT}"
echo "  💾 Database: ${DATABASE_URL}"
echo "  📧 Gmail: ${GMAIL_USER}"
echo "  🏗️  Contracts: ${TARGET_CONTRACTS}"
echo ""
echo "Ready to run:"
echo "  go run cmd/listener/main.go --config config/production.yaml"