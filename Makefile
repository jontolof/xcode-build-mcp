# Xcode Build MCP Server - Build Automation

.PHONY: build test clean lint fmt vet install run dev help

# Variables
BINARY_NAME=xcode-build-mcp
GO_VERSION=$(shell go version | cut -d' ' -f3)
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo 'unknown')

# Build flags
LDFLAGS=-ldflags "-X main.version=$(GIT_COMMIT) -X main.buildTime=$(BUILD_TIME)"

# Default target
all: lint test build

# Build the server binary
build:
	@echo "Building $(BINARY_NAME)..."
	@go build $(LDFLAGS) -o bin/$(BINARY_NAME) cmd/server/main.go
	@echo "Built bin/$(BINARY_NAME)"

# Run tests
test:
	@echo "Running tests..."
	@go test -race -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test -race -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run linter
lint:
	@echo "Running linter..."
	@go vet ./...
	@test -z "$$(gofmt -l .)" || (echo "Code not formatted, run 'make fmt'" && exit 1)

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Run go vet
vet:
	@echo "Running go vet..."
	@go vet ./...

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -f coverage.out coverage.html
	@go clean

# Install dependencies
deps:
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy

# Run the server in development mode
dev:
	@echo "Running in development mode..."
	@MCP_LOG_LEVEL=debug go run cmd/server/main.go stdio

# Run the built server
run: build
	@echo "Running $(BINARY_NAME)..."
	@./bin/$(BINARY_NAME) stdio

# Install server to GOPATH/bin
install:
	@echo "Installing $(BINARY_NAME)..."
	@go install $(LDFLAGS) ./cmd/server

# Run benchmarks
bench:
	@echo "Running benchmarks..."
	@go test -bench=. -benchmem ./...

# Check for security vulnerabilities
security:
	@echo "Running security checks..."
	@go list -json -deps ./... | nancy sleuth

# Generate documentation
docs:
	@echo "Generating documentation..."
	@go doc -all > docs/API.md

# Help
help:
	@echo "Available targets:"
	@echo "  build        - Build the server binary"
	@echo "  test         - Run all tests"
	@echo "  test-coverage- Run tests with coverage report"
	@echo "  lint         - Run linter and format check"
	@echo "  fmt          - Format all Go code"
	@echo "  vet          - Run go vet"
	@echo "  clean        - Clean build artifacts"
	@echo "  deps         - Install/update dependencies"
	@echo "  dev          - Run server in development mode"
	@echo "  run          - Build and run server"
	@echo "  install      - Install server to GOPATH/bin"
	@echo "  bench        - Run benchmarks"
	@echo "  security     - Run security vulnerability checks"
	@echo "  docs         - Generate documentation"
	@echo "  help         - Show this help"