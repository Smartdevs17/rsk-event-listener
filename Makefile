APP_NAME = rsk-event-listener
BIN_DIR = bin
SRC = cmd/listener/main.go
CONFIG = config/production.yaml

.PHONY: all build run test lint docker docker-run clean fmt

all: build

build:
    go build -o $(BIN_DIR)/$(APP_NAME) $(SRC)

run: build
    ./$(BIN_DIR)/$(APP_NAME) --config $(CONFIG)

test:
    go test ./...

lint:
    golangci-lint run

fmt:
    go fmt ./...

docker:
    docker build -t $(APP_NAME):latest .

docker-run: docker
    docker-compose up --build

clean:
    rm -rf $(BIN_DIR)/*
    rm -rf data/*.db data/*.db-shm data/*.db-wal
    docker-compose down -v

integration-setup:
    bash integration_setup.sh

create-production-config:
    bash create_production_yaml.sh

logs:
    tail -f logs/*.log

help:
    @echo "Usage: make [target]"
    @echo ""
    @echo "Targets:"
    @echo "  build                Build the Go binary"
    @echo "  run                  Run the application"
    @echo "  test                 Run all tests"
    @echo "  lint                 Run golangci-lint"
    @echo "  fmt                  Format Go code"
    @echo "  docker               Build Docker image"
    @echo "  docker-run           Build and run with docker-compose"
    @echo "  clean                Remove binaries and databases"
    @echo "  integration-setup    Run integration setup script"
    @echo "  create-production-config  Generate production config"
    @echo "  logs                 Tail application logs"
    @echo "  help                 Show this help message"