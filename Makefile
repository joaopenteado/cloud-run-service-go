.PHONY: help build test run docker-build docker-run clean

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the application
	@echo "Building application..."
	@go build -o bin/service .
	@echo "Build complete: bin/service"

test: ## Run tests
	@echo "Running tests..."
	@go test -v -cover ./...

test-coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

run: ## Run the application locally
	@echo "Starting service..."
	@go run main.go

docker-build: ## Build Docker image
	@echo "Building Docker image..."
	@docker build -t cloud-run-service-go:latest .
	@echo "Docker image built: cloud-run-service-go:latest"

docker-run: ## Run Docker container locally
	@echo "Starting Docker container..."
	@docker run -p 8080:8080 --rm --name cloud-run-service-go \
		-e ENVIRONMENT=development \
		-e LOG_LEVEL=debug \
		cloud-run-service-go:latest

fmt: ## Format code
	@echo "Formatting code..."
	@go fmt ./...

lint: ## Run linter
	@echo "Running linter..."
	@golangci-lint run || echo "golangci-lint not installed. Install with: curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin"

mod-download: ## Download dependencies
	@echo "Downloading dependencies..."
	@go mod download

mod-tidy: ## Tidy go.mod
	@echo "Tidying go.mod..."
	@go mod tidy

clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -f bin/service
	@rm -f coverage.out coverage.html
	@echo "Clean complete"

dev: ## Run in development mode with auto-reload (requires air)
	@echo "Starting development server..."
	@air || echo "air not installed. Install with: go install github.com/cosmtrek/air@latest"

.DEFAULT_GOAL := help
