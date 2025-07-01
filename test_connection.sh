#!/bin/bash

echo "Installing dependencies..."
go mod tidy

echo "Running Connection test..."
go run cmd/listener/connection_test_main.go

echo "Cleaning up..."
# Keep the data directory for future use