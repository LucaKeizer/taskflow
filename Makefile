# TaskFlow Makefile
.PHONY: help build test lint clean docker-build docker-run setup deps migration

# Default target
.DEFAULT_GOAL := help

# Variables
APP_NAME := taskflow
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Go build flags
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)"
BUILD_DIR := bin
COVERAGE_DIR := coverage

# Color output
RED := \033[31m
GREEN := \033[32m
YELLOW := \033[33m
BLUE := \033[34m
RESET := \033[0m

help: ## Show this help message
	@echo "$(BLUE)TaskFlow Build System$(RESET)"
	@echo ""
	@echo "$(YELLOW)Available targets:$(RESET)"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  $(GREEN)%-15s$(RESET) %s\n", $$1, $$2}' $(MAKEFILE_LIST)

setup: ## Install development dependencies
	@echo "$(BLUE)Setting up development environment...$(RESET)"
	go mod download
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/swaggo/swag/cmd/swag@latest
	@echo "$(GREEN)Setup complete!$(RESET)"

deps: ## Download and verify dependencies
	@echo "$(BLUE)Downloading dependencies...$(RESET)"
	go mod download
	go mod verify
	go mod tidy

build: test ## Build all binaries
	@echo "$(BLUE)Building binaries...$(RESET)"
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/taskflow-api cmd/server/main.go
	CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/taskflow-worker cmd/worker/main.go
	@echo "$(GREEN)Build complete! Binaries in $(BUILD_DIR)/$(RESET)"

build-race: ## Build with race detection enabled
	@echo "$(BLUE)Building with race detection...$(RESET)"
	@mkdir -p $(BUILD_DIR)
	go build -race $(LDFLAGS) -o $(BUILD_DIR)/taskflow-api-race cmd/server/main.go
	go build -race $(LDFLAGS) -o $(BUILD_DIR)/taskflow-worker-race cmd/worker/main.go

test: ## Run all tests
	@echo "$(BLUE)Running tests...$(RESET)"
	go test -race -v ./...

test-coverage: ## Run tests with coverage report
	@echo "$(BLUE)Running tests with coverage...$(RESET)"
	@mkdir -p $(COVERAGE_DIR)
	go test -race -coverprofile=$(COVERAGE_DIR)/coverage.out ./...
	go tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	@echo "$(GREEN)Coverage report generated: $(COVERAGE_DIR)/coverage.html$(RESET)"

test-integration: ## Run integration tests
	@echo "$(BLUE)Running integration tests...$(RESET)"
	docker-compose -f docker-compose.test.yml up --build --abort-on-container-exit
	docker-compose -f docker-compose.test.yml down

benchmark: ## Run benchmark tests
	@echo "$(BLUE)Running benchmarks...$(RESET)"
	go test -bench=. -benchmem ./...

lint: ## Run linter
	@echo "$(BLUE)Running linter...$(RESET)"
	golangci-lint run

lint-fix: ## Run linter with auto-fix
	@echo "$(BLUE)Running linter with auto-fix...$(RESET)"
	golangci-lint run --fix

fmt: ## Format code
	@echo "$(BLUE)Formatting code...$(RESET)"
	go fmt ./...
	goimports -w -local taskflow .

vet: ## Run go vet
	@echo "$(BLUE)Running go vet...$(RESET)"
	go vet ./...

security: ## Run security checks
	@echo "$(BLUE)Running security checks...$(RESET)"
	go list -json -m all | nancy sleuth

clean: ## Clean build artifacts
	@echo "$(BLUE)Cleaning build artifacts...$(RESET)"
	rm -rf $(BUILD_DIR)
	rm -rf $(COVERAGE_DIR)
	docker system prune -f
	@echo "$(GREEN)Clean complete!$(RESET)"

docker-build: ## Build Docker images
	@echo "$(BLUE)Building Docker images...$(RESET)"
	docker build -t $(APP_NAME)-api:$(VERSION) --target api .
	docker build -t $(APP_NAME)-worker:$(VERSION) --target worker .
	@echo "$(GREEN)Docker images built successfully!$(RESET)"

docker-run: ## Run the application with Docker Compose
	@echo "$(BLUE)Starting application with Docker Compose...$(RESET)"
	docker-compose up -d
	@echo "$(GREEN)Application started! API available at http://localhost:8080$(RESET)"

docker-stop: ## Stop Docker Compose services
	@echo "$(BLUE)Stopping Docker Compose services...$(RESET)"
	docker-compose down

docker-logs: ## View Docker Compose logs
	docker-compose logs -f

migration-up: ## Run database migrations up
	@echo "$(BLUE)Running database migrations...$(RESET)"
	# TODO: Add migration tool command here
	@echo "$(YELLOW)Migration tool not implemented yet$(RESET)"

migration-down: ## Rollback database migrations
	@echo "$(BLUE)Rolling back database migrations...$(RESET)"
	# TODO: Add migration rollback command here
	@echo "$(YELLOW)Migration tool not implemented yet$(RESET)"

load-test: ## Run load tests
	@echo "$(BLUE)Running load tests...$(RESET)"
	go run scripts/load-test.go -jobs=1000 -concurrent=50

dev-setup: ## Set up local development environment
	@echo "$(BLUE)Setting up local development environment...$(RESET)"
	docker-compose up -d redis postgres
	@echo "$(GREEN)Development services started!$(RESET)"
	@echo "$(YELLOW)Run 'make run-api' and 'make run-worker' in separate terminals$(RESET)"

run-api: ## Run API server locally
	@echo "$(BLUE)Starting API server...$(RESET)"
	go run cmd/server/main.go

run-worker: ## Run worker locally
	@echo "$(BLUE)Starting worker...$(RESET)"
	go run cmd/worker/main.go

install: build ## Install binaries to GOPATH
	@echo "$(BLUE)Installing binaries...$(RESET)"
	go install $(LDFLAGS) ./cmd/server
	go install $(LDFLAGS) ./cmd/worker
	@echo "$(GREEN)Binaries installed to GOPATH!$(RESET)"

release: ## Build release binaries for multiple platforms
	@echo "$(BLUE)Building release binaries...$(RESET)"
	@mkdir -p $(BUILD_DIR)/release
	
	# Linux AMD64
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/release/taskflow-api-linux-amd64 cmd/server/main.go
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/release/taskflow-worker-linux-amd64 cmd/worker/main.go
	
	# Linux ARM64
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/release/taskflow-api-linux-arm64 cmd/server/main.go
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/release/taskflow-worker-linux-arm64 cmd/worker/main.go
	
	# macOS AMD64
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/release/taskflow-api-darwin-amd64 cmd/server/main.go
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/release/taskflow-worker-darwin-amd64 cmd/worker/main.go
	
	# macOS ARM64
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/release/taskflow-api-darwin-arm64 cmd/server/main.go
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/release/taskflow-worker-darwin-arm64 cmd/worker/main.go
	
	@echo "$(GREEN)Release binaries built in $(BUILD_DIR)/release/$(RESET)"

check: lint vet test ## Run all checks (lint, vet, test)

ci: deps check build ## Run CI pipeline locally

version: ## Show version information
	@echo "$(BLUE)Version Information:$(RESET)"
	@echo "Version: $(VERSION)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo "Git Commit: $(GIT_COMMIT)"

# Development helpers
watch-api: ## Watch and rebuild API server on changes
	@echo "$(BLUE)Watching API server for changes...$(RESET)"
	find . -name "*.go" | entr -r make run-api

watch-worker: ## Watch and rebuild worker on changes
	@echo "$(BLUE)Watching worker for changes...$(RESET)"
	find . -name "*.go" | entr -r make run-worker