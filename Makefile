.PHONY: all build test clean fmt lint deps install coverage help ci-test

# Default target
all: test build

# Build the application
build:
	@echo "Building GoScry..."
	go build -v -o bin/goscry ./cmd/goscry

# Cross-platform builds
build-all: build-linux build-darwin build-windows

build-linux:
	@echo "Building for Linux..."
	GOOS=linux GOARCH=amd64 go build -v -o bin/goscry-linux-amd64 ./cmd/goscry

build-darwin:
	@echo "Building for macOS..."
	GOOS=darwin GOARCH=amd64 go build -v -o bin/goscry-darwin-amd64 ./cmd/goscry

build-windows:
	@echo "Building for Windows..."
	GOOS=windows GOARCH=amd64 go build -v -o bin/goscry-windows-amd64.exe ./cmd/goscry

# Install dependencies
deps:
	@echo "Installing dependencies..."
	go mod download

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Run tests with coverage
coverage:
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Run CI tests (used in GitHub Actions)
ci-test:
	@echo "Running CI tests..."
	go test -race -coverprofile=coverage.out -covermode=atomic ./...

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Run linter
lint:
	@echo "Linting code..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not found, installing..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run ./...

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	rm -f coverage.out coverage.html

# Install the application
install: build
	@echo "Installing GoScry..."
	cp bin/goscry $(GOPATH)/bin/

# Help
help:
	@echo "Available targets:"
	@echo "  all        - Run tests and build the application (default)"
	@echo "  build      - Build the application"
	@echo "  build-all  - Build for Linux, macOS, and Windows"
	@echo "  deps       - Install dependencies"
	@echo "  test       - Run tests"
	@echo "  ci-test    - Run tests for CI (with race detection)"
	@echo "  coverage   - Run tests with coverage"
	@echo "  fmt        - Format code"
	@echo "  lint       - Run linter"
	@echo "  clean      - Clean build artifacts"
	@echo "  install    - Install the application"
	@echo "  help       - Show this help message"