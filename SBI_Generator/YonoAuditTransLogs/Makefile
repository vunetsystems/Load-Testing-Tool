.PHONY: build test run clean install-deps

# Build the application
build:
	go build -o bin/kafka-message-generator cmd/generator/main.go

# Run tests
test:
	go test -v -race -coverprofile=coverage.out ./...

# Run with default config
run:
	go run cmd/generator/main.go --config config/config.yaml

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f coverage.out

# Install dependencies
install-deps:
	go mod download
	go mod tidy

# Build for production
build-prod:
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/kafka-message-generator cmd/generator/main.go

# Run with custom config
run-custom:
	go run cmd/generator/main.go --config $(CONFIG)

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	golangci-lint run

# View coverage
coverage:
	go tool cover -html=coverage.out
