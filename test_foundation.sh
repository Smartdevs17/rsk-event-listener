#!/bin/bash

echo "Creating data directory..."
mkdir -p data

echo "Installing dependencies..."
go mod tidy

echo "Running foundation test..."
go run cmd/listener/main.go

echo "Cleaning up..."
# Keep the data directory for future use