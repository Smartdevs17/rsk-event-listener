#!/bin/bash

# 5. Install any additional dependencies
go mod tidy

# 6. Test compilation
echo "Testing compilation..."
go build ./internal/monitor/

# 7. Run monitor tests
echo "Running event monitor tests..."
go run cmd/listener/monitor_test_main.go

# 8. Run integration tests
echo "Running integration tests..."
go test -v ./test/integration/ -run TestEventMonitor

echo "Step 4 event monitor implementation complete!"