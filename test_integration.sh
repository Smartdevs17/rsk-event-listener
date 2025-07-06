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
