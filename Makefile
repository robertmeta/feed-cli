.PHONY: all build test test-unit test-model test-store test-feed test-coverage lint clean install tidy help

# Variables
BINARY_NAME=feed-cli
BUILD_DIR=bin
MAIN_PATH=./cmd/feed-cli
COVERAGE_FILE=coverage.out

# Default target
all: test build

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "Binary built: $(BUILD_DIR)/$(BINARY_NAME)"

# Run all tests
test:
	@echo "Running all tests..."
	@go test -v ./...

# Run unit tests only (skip integration tests)
test-unit:
	@echo "Running unit tests..."
	@go test -short -v ./model/... ./store/... ./feed/...

# Run model tests
test-model:
	@echo "Running model tests..."
	@go test -v ./model

# Run store tests
test-store:
	@echo "Running store tests..."
	@go test -v ./store

# Run feed tests
test-feed:
	@echo "Running feed tests..."
	@go test -v ./feed

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test -coverprofile=$(COVERAGE_FILE) ./...
	@echo "Coverage report generated: $(COVERAGE_FILE)"
	@go tool cover -func=$(COVERAGE_FILE)

# View coverage in browser
coverage-html: test-coverage
	@go tool cover -html=$(COVERAGE_FILE)

# Run linter (requires golangci-lint)
lint:
	@echo "Running linter..."
	@golangci-lint run || echo "golangci-lint not installed. Run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR) $(COVERAGE_FILE)
	@echo "Clean complete"

# Install local binary to $GOPATH/bin
install:
	@echo "Installing local $(BINARY_NAME) to $$GOPATH/bin..."
	@go install $(MAIN_PATH)
	@echo "Installed successfully"
	@which feed-cli
	@feed-cli --version

# Tidy dependencies
tidy:
	@echo "Tidying dependencies..."
	@go mod tidy
	@echo "Dependencies tidied"

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	@go mod download
	@echo "Dependencies downloaded"

# Run the binary
run: build
	@$(BUILD_DIR)/$(BINARY_NAME)

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Vet code
vet:
	@echo "Vetting code..."
	@go vet ./...

# Check for common issues
check: fmt vet test
	@echo "All checks passed!"

# Install from GitHub (test public install)
install-from-github:
	@echo "Installing from GitHub..."
	@go install github.com/robertmeta/feed-cli/cmd/feed-cli@latest
	@echo "Installed! Testing..."
	@which feed-cli
	@feed-cli --version

# Help
help:
	@echo "feed-cli Makefile targets:"
	@echo "  make build               - Build the binary"
	@echo "  make test                - Run all tests"
	@echo "  make test-unit           - Run unit tests only"
	@echo "  make test-model          - Run model tests"
	@echo "  make test-store          - Run store tests"
	@echo "  make test-feed           - Run feed tests"
	@echo "  make test-coverage       - Run tests with coverage"
	@echo "  make coverage-html       - View coverage in browser"
	@echo "  make lint                - Run linter"
	@echo "  make clean               - Clean build artifacts"
	@echo "  make install             - Install to GOPATH/bin (local)"
	@echo "  make install-from-github - Install from GitHub (test public install)"
	@echo "  make tidy                - Tidy dependencies"
	@echo "  make deps                - Download dependencies"
	@echo "  make fmt                 - Format code"
	@echo "  make vet                 - Vet code"
	@echo "  make check               - Format, vet, and test"
	@echo "  make help                - Show this help message"
