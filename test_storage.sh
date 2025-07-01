#!/bin/bash

echo "Testing compilation..."
go build ./internal/storage/

# 9. Run storage tests
echo "Running storage layer tests..."
go run cmd/listener/storage_test_main.go

# 10. Run integration tests (optional)
echo "Running integration tests..."
go test -v ./test/integration/

# 11. Test with different configurations
echo "Testing with different storage configurations..."

# Test with absolute path
mkdir -p /tmp/rsk-test
RSK_LISTENER_STORAGE_CONNECTION_STRING="/tmp/rsk-test/events.db" go run cmd/listener/storage_test_main.go

# Clean up test files
rm -f ./test_storage.db
rm -rf /tmp/rsk-test

echo "Step 3 storage layer implementation complete!"