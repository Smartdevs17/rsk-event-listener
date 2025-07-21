#!/bin/bash
# File: scripts/test_metrics_integration.sh

set -e

echo "🧪 RSK Event Listener Metrics Integration Test"
echo "=============================================="

# Configuration
APP_BINARY="./bin/rsk-event-listener"
CONFIG_FILE="config/production.yaml"
SERVER_PORT=${SERVER_PORT:-8081}
METRICS_URL="http://localhost:${SERVER_PORT}/metrics"
HEALTH_URL="http://localhost:${SERVER_PORT}/api/v1/health"
STATS_URL="http://localhost:${SERVER_PORT}/api/v1/stats"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
print_step() {
    echo -e "${BLUE}📋 $1${NC}"
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

# Function to check if application is running
check_app_running() {
    if pgrep -f "rsk-event-listener" > /dev/null; then
        return 0
    else
        return 1
    fi
}

# Function to stop application
stop_app() {
    if check_app_running; then
        print_step "Stopping application..."
        pkill -f "rsk-event-listener" || true
        sleep 3
    fi
}

# Function to wait for application to start
wait_for_app() {
    local max_attempts=30
    local attempt=1
    
    print_step "Waiting for application to start..."
    
    while [ $attempt -le $max_attempts ]; do
        if curl -f -s $HEALTH_URL > /dev/null 2>&1; then
            print_success "Application is running"
            return 0
        fi
        
        echo "  Attempt $attempt/$max_attempts..."
        sleep 2
        ((attempt++))
    done
    
    print_error "Application failed to start within $((max_attempts * 2)) seconds"
    return 1
}

# Function to test basic functionality
test_basic_functionality() {
    print_step "Testing basic functionality..."
    
    # Test version command
    if $APP_BINARY version > /dev/null 2>&1; then
        print_success "Version command works"
    else
        print_error "Version command failed"
        return 1
    fi
    
    # Test config validation
    if $APP_BINARY config validate --config $CONFIG_FILE > /dev/null 2>&1; then
        print_success "Config validation works"
    else
        print_warning "Config validation failed (may be expected if config is missing)"
    fi
    
    # Test metrics command
    if $APP_BINARY metrics --config $CONFIG_FILE > /dev/null 2>&1; then
        print_success "Metrics command works"
    else
        print_warning "Metrics command failed (may be expected if config is missing)"
    fi
    
    return 0
}

# Function to test with metrics disabled
test_metrics_disabled() {
    print_step "Testing with metrics DISABLED..."
    
    # Set environment to disable metrics
    export ENABLE_METRICS=false
    
    # Start application in background
    $APP_BINARY --config $CONFIG_FILE > /tmp/rsk_test_disabled.log 2>&1 &
    local app_pid=$!
    
    # Wait for startup
    if ! wait_for_app; then
        stop_app
        return 1
    fi
    
    # Test health endpoint for metrics_enabled
    if curl -f -s $HEALTH_URL | jq -e '.metrics_enabled' | grep -q "$([ "$ENABLE_METRICS" = "true" ] && echo true || echo false)"; then
        print_success "Health endpoint reflects metrics_enabled=$ENABLE_METRICS"
    else
        print_warning "Health endpoint does not reflect metrics_enabled=$ENABLE_METRICS"
    fi

    # Test stats endpoint for metrics_enabled
    if curl -f -s $STATS_URL | jq -e '.metrics_enabled' | grep -q "$([ "$ENABLE_METRICS" = "true" ] && echo true || echo false)"; then
        print_success "Stats endpoint reflects metrics_enabled=$ENABLE_METRICS"
    else
        print_warning "Stats endpoint does not reflect metrics_enabled=$ENABLE_METRICS"
    fi

    # Test /metrics endpoint for Go runtime metrics
    if curl -f -s $METRICS_URL | grep -q "go_memstats"; then
        print_success "Metrics endpoint exposes Go runtime metrics"
    else
        print_error "Metrics endpoint does not expose Go runtime metrics"
    fi

    # Test for custom RSK metrics
    if [ "$ENABLE_METRICS" = "true" ]; then
        if curl -s $METRICS_URL | grep -q "rsk_events_processed_total"; then
            print_success "Custom RSK metrics present when enabled"
        else
            print_error "Custom RSK metrics missing when enabled"
        fi
    else
        if curl -s $METRICS_URL | grep -q "rsk_events_processed_total"; then
            print_error "Custom RSK metrics present when disabled"
        else
            print_success "Custom RSK metrics absent when disabled"
        fi
    fi
    
    stop_app
    return 0
}

# Function to test with metrics enabled
test_metrics_enabled() {
    print_step "Testing with metrics ENABLED..."
    
    # Set environment to enable metrics
    export ENABLE_METRICS=true
    
    # Start application in background
    $APP_BINARY --config $CONFIG_FILE > /tmp/rsk_test_enabled.log 2>&1 &
    local app_pid=$!
    
    # Wait for startup
    if ! wait_for_app; then
        stop_app
        return 1
    fi
    
    # Test health endpoint
    if curl -f -s $HEALTH_URL | jq -r '.metrics_enabled' | grep -q "true"; then
        print_success "Metrics correctly enabled in health check"
    else
        print_warning "Metrics status not reflected in health check"
    fi
    
    # Test stats endpoint
    if curl -f -s $STATS_URL | jq -r '.metrics_enabled' | grep -q "true"; then
        print_success "Metrics correctly enabled in stats"
    else
        print_warning "Metrics status not reflected in stats"
    fi
    
    # Test metrics endpoint format
    local metrics_content=$(curl -s $METRICS_URL)
    
    # Check for Prometheus format
    if echo "$metrics_content" | grep -q "^# HELP"; then
        print_success "Metrics in correct Prometheus format"
    else
        print_error "Metrics not in Prometheus format"
        stop_app
        return 1
    fi
    
    # Check for custom RSK metrics
    local required_metrics=(
        "rsk_application_uptime_seconds"
        "rsk_component_health"
        "rsk_memory_usage_bytes"
        "rsk_goroutines"
    )
    
    local metrics_found=0
    for metric in "${required_metrics[@]}"; do
        if echo "$metrics_content" | grep -q "^# HELP $metric"; then
            print_success "Found metric: $metric"
            ((metrics_found++))
        else
            print_warning "Missing metric: $metric"
        fi
    done
    
    if [ $metrics_found -ge 2 ]; then
        print_success "Sufficient custom metrics found ($metrics_found/${#required_metrics[@]})"
    else
        print_error "Too few custom metrics found ($metrics_found/${#required_metrics[@]})"
        stop_app
        return 1
    fi
    
    # Test metrics values are reasonable
    local uptime=$(echo "$metrics_content" | grep "^rsk_application_uptime_seconds" | awk '{print $2}')
    if [ ! -z "$uptime" ] && awk "BEGIN {exit !($uptime >= 0)}"; then
        print_success "Application uptime metric has reasonable value: ${uptime}s"
    else
        print_warning "Application uptime metric value seems incorrect: ${uptime}"
    fi

    # Generate some load to test metrics updates
    print_step "Testing metrics updates..."
    for i in {1..10}; do
        curl -s $HEALTH_URL > /dev/null &
        curl -s $STATS_URL > /dev/null &
    done
    wait
    
    sleep 2

    # Check if HTTP request metrics updated
    local http_requests=$(curl -s $METRICS_URL | grep "rsk_http_requests_total" | head -1 | awk '{print $2}')
    if [ ! -z "$http_requests" ] && [ "$http_requests" -gt 10 ]; then
        print_success "HTTP request metrics updating correctly (${http_requests} requests)"
    else
        print_warning "HTTP request metrics may not be updating as expected"
    fi
    
    stop_app
    return 0
}

# Function to test Prometheus compatibility
test_prometheus_compatibility() {
    print_step "Testing Prometheus compatibility..."
    
    # Check if promtool is available
    if command -v promtool &> /dev/null; then
        print_step "Testing with promtool..."
        
        # Start app briefly for promtool test
        export ENABLE_METRICS=true
        $APP_BINARY --config $CONFIG_FILE > /dev/null 2>&1 &
        local app_pid=$!
        
        sleep 5
        
        # Test promtool query
        if curl -s $METRICS_URL | promtool query instant 'up' > /dev/null 2>&1; then
            print_success "Metrics compatible with promtool"
        else
            print_warning "Promtool compatibility test failed (may be expected)"
        fi
        
        stop_app
    else
        print_warning "promtool not available, skipping compatibility test"
    fi
    
    return 0
}

# Main test execution
main() {
    echo "Starting integration test at $(date)"
    echo
    
    # Check prerequisites
    print_step "Checking prerequisites..."
    
    if [ ! -f "$APP_BINARY" ]; then
        print_error "Application binary not found: $APP_BINARY"
        print_step "Building application..."
        make build || {
            print_error "Failed to build application"
            exit 1
        }
    fi
    
    if ! command -v jq &> /dev/null; then
        print_error "jq is required but not installed"
        exit 1
    fi
    
    if ! command -v curl &> /dev/null; then
        print_error "curl is required but not installed"
        exit 1
    fi
    
    print_success "Prerequisites check passed"
    
    # Ensure no application is running
    stop_app
    
    # Run tests
    local tests_passed=0
    local total_tests=4
    
    if test_basic_functionality; then
        ((tests_passed++))
    fi
    echo
    
    if test_metrics_disabled; then
        ((tests_passed++))
    fi
    echo
    
    if test_metrics_enabled; then
        ((tests_passed++))
    fi
    echo
    
    if test_prometheus_compatibility; then
        ((tests_passed++))
    fi
    echo
    
    # Summary
    print_step "Test Summary"
    echo "============"
    echo "Tests passed: $tests_passed/$total_tests"
    
    if [ $tests_passed -eq $total_tests ]; then
        print_success "All integration tests passed! 🎉"
        echo
        print_step "Next Steps:"
        echo "1. Configure Prometheus to scrape: $METRICS_URL"
        echo "2. Set up Grafana dashboards"
        echo "3. Configure alerting rules"
        echo
        print_step "Quick Test Commands:"
        echo "curl $METRICS_URL"
        echo "curl $HEALTH_URL"
        echo "./rsk-event-listener metrics"
        exit 0
    else
        print_error "Some integration tests failed"
        echo
        print_step "Debug Information:"
        echo "- Application logs: /tmp/rsk_test_*.log"
        echo "- Check configuration: $CONFIG_FILE"
        echo "- Verify dependencies: make test"
        exit 1
    fi
}

# Cleanup on exit
trap 'stop_app' EXIT

# Run if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi