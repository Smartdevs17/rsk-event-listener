APP_NAME = rsk-event-listener
BIN_DIR = bin
SRC = cmd/listener/main.go
CONFIG = config/production.yaml

# Colors for output
RED=\033[0;31m
GREEN=\033[0;32m
YELLOW=\033[1;33m
BLUE=\033[0;34m
BOLD=\033[1m
NC=\033[0m # No Color

.PHONY: all build run test test-metrics lint fmt docker docker-run monitoring-stack monitoring-down clean integration-setup create-production-config logs help

all: build

# Build commands
build: ## Build the Go binary
	@echo "$(BLUE)📦 Building $(APP_NAME)...$(NC)"
	@go build -o $(BIN_DIR)/$(APP_NAME) $(SRC)
	@echo "$(GREEN)✅ Build complete: $(BIN_DIR)/$(APP_NAME)$(NC)"

# Run commands
run: build ## Run the application with production config
	@echo "$(BLUE)🚀 Starting $(APP_NAME)...$(NC)"
	@./$(BIN_DIR)/$(APP_NAME) --config $(CONFIG)

# Monitoring stack commands
monitoring-stack: ## Start complete monitoring stack (RSK + Prometheus + Grafana)
	@echo "$(BOLD)🚀 Starting complete monitoring stack...$(NC)"
	@echo "$(YELLOW)📋 Setting up directories and config...$(NC)"
	@cp .env.example .env 2>/dev/null || true
	@mkdir -p monitoring/grafana/{dashboards,provisioning/{dashboards,datasources}}
	@mkdir -p data logs
	@echo "$(YELLOW)🐳 Starting Docker containers...$(NC)"
	@docker-compose -f docker-compose.monitoring.yml up -d
	@echo "$(YELLOW)⏳ Waiting for services to start...$(NC)"
	@sleep 15
	@echo ""
	@echo "$(GREEN)✅ Monitoring stack started successfully!$(NC)"
	@echo ""
	@echo "$(BOLD)🌐 Access your dashboards:$(NC)"
	@echo "   $(BLUE)📊 Grafana Dashboard: http://localhost:3000$(NC) $(YELLOW)(admin/admin123)$(NC)"
	@echo "   $(BLUE)📈 Prometheus Metrics: http://localhost:9090$(NC)"
	@echo "   $(BLUE)🔧 RSK Event Listener API: http://localhost:8081$(NC)"
	@echo "   $(BLUE)❤️  Health Check: http://localhost:8081/api/v1/health$(NC)"
	@echo ""
	@echo "$(GREEN)💡 Quick test: curl http://localhost:8081/metrics | grep rsk_$(NC)"

monitoring-down: ## Stop monitoring stack
	@echo "$(YELLOW)🛑 Stopping monitoring stack...$(NC)"
	@docker-compose -f docker-compose.monitoring.yml down
	@echo "$(GREEN)✅ Monitoring stack stopped$(NC)"

monitoring-logs: ## View monitoring stack logs
	@docker-compose -f docker-compose.monitoring.yml logs -f

# Test commands
test: ## Run all unit and integration tests
	@echo "$(BLUE)🧪 Running tests...$(NC)"
	@go test ./...
	@echo "$(GREEN)✅ Tests completed$(NC)"

test-metrics: ## Run metrics integration tests
	@echo "$(BLUE)🧪 Running metrics integration tests...$(NC)"
	@chmod +x scripts/test_metrics_integration.sh
	@./scripts/test_metrics_integration.sh

test-verbose: ## Run tests with verbose output
	@echo "$(BLUE)🧪 Running tests with verbose output...$(NC)"
	@go test -v ./...

test-coverage: ## Run tests with coverage report
	@echo "$(BLUE)🧪 Running tests with coverage...$(NC)"
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)✅ Coverage report generated: coverage.html$(NC)"

# Code quality commands
lint: ## Run golangci-lint
	@echo "$(BLUE)🔍 Running linter...$(NC)"
	@golangci-lint run
	@echo "$(GREEN)✅ Linting completed$(NC)"

fmt: ## Format Go code
	@echo "$(BLUE)📝 Formatting code...$(NC)"
	@go fmt ./...
	@echo "$(GREEN)✅ Code formatted$(NC)"

# Docker commands
docker: ## Build Docker image
	@echo "$(BLUE)🐳 Building Docker image...$(NC)"
	@docker build -t $(APP_NAME):latest .
	@echo "$(GREEN)✅ Docker image built: $(APP_NAME):latest$(NC)"

docker-run: docker ## Build and run with docker-compose
	@echo "$(BLUE)🐳 Starting with Docker Compose...$(NC)"
	@docker-compose up --build

docker-clean: ## Clean Docker images and containers
	@echo "$(YELLOW)🧹 Cleaning Docker resources...$(NC)"
	@docker-compose down -v
	@docker rmi $(APP_NAME):latest 2>/dev/null || true
	@echo "$(GREEN)✅ Docker cleanup completed$(NC)"

# Setup and configuration commands
integration-setup: ## Run integration setup script
	@echo "$(BLUE)⚙️  Running integration setup...$(NC)"
	@bash integration_setup.sh

create-production-config: ## Generate production config
	@echo "$(BLUE)⚙️  Creating production configuration...$(NC)"
	@bash create_production_yaml.sh

deps: ## Download and verify dependencies
	@echo "$(BLUE)📦 Downloading dependencies...$(NC)"
	@go mod download
	@go mod verify
	@echo "$(GREEN)✅ Dependencies updated$(NC)"

# Utility commands
clean: ## Remove binaries, databases, and Docker resources
	@echo "$(YELLOW)🧹 Cleaning up...$(NC)"
	@rm -rf $(BIN_DIR)/*
	@rm -rf data/*.db data/*.db-shm data/*.db-wal
	@rm -f coverage.out coverage.html
	@docker-compose down -v 2>/dev/null || true
	@docker-compose -f docker-compose.monitoring.yml down 2>/dev/null || true
	@echo "$(GREEN)✅ Cleanup completed$(NC)"

logs: ## Tail application logs
	@echo "$(BLUE)📄 Tailing logs...$(NC)"
	@tail -f logs/*.log 2>/dev/null || echo "$(YELLOW)⚠️  No log files found in logs/ directory$(NC)"

status: ## Show application and monitoring status
	@echo "$(BOLD)📊 System Status$(NC)"
	@echo "$(BLUE)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo "$(YELLOW)🔧 Application Status:$(NC)"
	@curl -s http://localhost:8081/api/v1/health 2>/dev/null | jq '.' || echo "$(RED)❌ Application not running$(NC)"
	@echo ""
	@echo "$(YELLOW)🐳 Docker Services:$(NC)"
	@docker-compose -f docker-compose.monitoring.yml ps 2>/dev/null || echo "$(RED)❌ Monitoring stack not running$(NC)"
	@echo ""
	@echo "$(YELLOW)📈 Quick Metrics Check:$(NC)"
	@curl -s http://localhost:8081/metrics 2>/dev/null | grep -c "rsk_" | xargs -I {} echo "   {} RSK metrics available" || echo "$(RED)❌ Metrics not available$(NC)"

install-tools: ## Install development tools
	@echo "$(BLUE)🛠️  Installing development tools...$(NC)"
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install github.com/swaggo/swag/cmd/swag@latest
	@which jq > /dev/null || (echo "$(YELLOW)⚠️  Installing jq...$(NC)" && brew install jq 2>/dev/null || sudo apt-get install jq -y 2>/dev/null || echo "$(RED)❌ Please install jq manually$(NC)")
	@echo "$(GREEN)✅ Development tools installed$(NC)"

# Development workflow commands
dev: ## Start development environment (with monitoring)
	@echo "$(BOLD)🚀 Starting development environment...$(NC)"
	@make monitoring-stack
	@echo "$(GREEN)✅ Development environment ready!$(NC)"
	@echo "$(BLUE)💡 Run 'make status' to check everything is working$(NC)"

quick-test: ## Quick test of core functionality
	@echo "$(BLUE)⚡ Running quick functionality test...$(NC)"
	@make build
	@timeout 10s ./$(BIN_DIR)/$(APP_NAME) --config $(CONFIG) > /dev/null 2>&1 & 
	@sleep 3
	@curl -s http://localhost:8081/api/v1/health > /dev/null && echo "$(GREEN)✅ Application starts successfully$(NC)" || echo "$(RED)❌ Application failed to start$(NC)"
	@pkill -f $(APP_NAME) 2>/dev/null || true

# Help command
help: ## Show this help message
	@echo "$(BOLD)$(APP_NAME) - Makefile Help$(NC)"
	@echo "$(BLUE)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@echo "$(BOLD)📋 Available Commands:$(NC)"
	@echo ""
	@echo "$(YELLOW)🚀 Quick Start:$(NC)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | grep -E '(monitoring-stack|dev|quick-test)' | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-20s$(NC) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(YELLOW)🛠️  Build & Run:$(NC)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | grep -E '(build|run|docker)' | grep -v monitoring | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-20s$(NC) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(YELLOW)📊 Monitoring:$(NC)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | grep monitoring | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-20s$(NC) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(YELLOW)🧪 Testing:$(NC)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | grep test | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-20s$(NC) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(YELLOW)🔧 Development:$(NC)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | grep -E '(lint|fmt|deps|install-tools)' | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-20s$(NC) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(YELLOW)⚙️  Setup & Utils:$(NC)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | grep -E '(setup|create|clean|logs|status)' | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-20s$(NC) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(BOLD)💡 Quick Examples:$(NC)"
	@echo "  $(BLUE)make dev$(NC)                    # Start full development environment"
	@echo "  $(BLUE)make monitoring-stack$(NC)       # Start monitoring (Grafana + Prometheus)"
	@echo "  $(BLUE)make test-metrics$(NC)           # Test metrics integration"
	@echo "  $(BLUE)make status$(NC)                 # Check system status"
	@echo ""