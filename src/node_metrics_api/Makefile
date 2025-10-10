# Node Metrics API Makefile

BINARY_NAME=node_metrics_api
BUILD_DIR=build
GO_FILES=$(shell find . -name "*.go" -not -path "./$(BUILD_DIR)/*")

# Default target
.PHONY: all
all: build

# Build the binary
.PHONY: build
build: $(BUILD_DIR)/$(BINARY_NAME)

$(BUILD_DIR)/$(BINARY_NAME): $(GO_FILES)
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) .

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)

# Run the server (requires build first)
.PHONY: run
run: build
	@echo "Starting Node Metrics API server..."
	./$(BUILD_DIR)/$(BINARY_NAME)

# Run tests
.PHONY: test
test:
	@echo "Running tests..."
	go test -v ./...

# Format code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Lint code (requires golangci-lint)
.PHONY: lint
lint:
	@echo "Linting code..."
	golangci-lint run

# Install dependencies
.PHONY: deps
deps:
	@echo "Installing dependencies..."
	go mod tidy
	go mod download

# Cross-compile for Linux (for deployment to nodes)
.PHONY: build-linux
build-linux: $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64

$(BUILD_DIR)/$(BINARY_NAME)-linux-amd64: $(GO_FILES)
	@echo "Cross-compiling for Linux amd64..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 .

# Show help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build       - Build the binary"
	@echo "  clean       - Clean build artifacts"
	@echo "  run         - Build and run the server"
	@echo "  test        - Run tests"
	@echo "  fmt         - Format code"
	@echo "  lint        - Lint code"
	@echo "  deps        - Install dependencies"
	@echo "  build-linux - Cross-compile for Linux"
	@echo "  help        - Show this help"