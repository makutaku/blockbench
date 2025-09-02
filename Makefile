# Makefile for Blockbench

# Version information
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT := $(shell git rev-parse HEAD 2>/dev/null || echo "unknown")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Build settings
BINARY_NAME := blockbench
MAIN_PACKAGE := ./cmd/blockbench
BUILD_DIR := ./bin

# Linker flags to inject version information
LDFLAGS := -ldflags "\
	-X github.com/makutaku/blockbench/internal/version.Version=$(VERSION) \
	-X github.com/makutaku/blockbench/internal/version.GitCommit=$(GIT_COMMIT) \
	-X github.com/makutaku/blockbench/internal/version.BuildDate=$(BUILD_DATE)"

# Default target
.PHONY: all
all: clean build

# Build the binary
.PHONY: build
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PACKAGE)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Build for development (without version injection)
.PHONY: build-dev
build-dev:
	@echo "Building $(BINARY_NAME) for development..."
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PACKAGE)
	@echo "Development build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Install to GOPATH/bin
.PHONY: install
install:
	@echo "Installing $(BINARY_NAME)..."
	go install $(LDFLAGS) $(MAIN_PACKAGE)
	@echo "Install complete"

# Build for multiple platforms
.PHONY: build-all
build-all: clean
	@echo "Building for multiple platforms..."
	@mkdir -p $(BUILD_DIR)
	
	# Linux AMD64
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PACKAGE)
	
	# Linux ARM64
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(MAIN_PACKAGE)
	
	# macOS AMD64
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_PACKAGE)
	
	# macOS ARM64
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(MAIN_PACKAGE)
	
	# Windows AMD64
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PACKAGE)
	
	@echo "Cross-platform build complete"
	@ls -la $(BUILD_DIR)/

# Run tests
.PHONY: test
test:
	@echo "Running tests..."
	go test -v ./...

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out -covermode=atomic ./...
	@if [ -f coverage.out ]; then \
		go tool cover -html=coverage.out -o coverage.html; \
		echo "Coverage report generated: coverage.html"; \
	else \
		echo "No test coverage data generated (no tests found)"; \
		touch coverage.out; \
	fi

# Lint the code
.PHONY: lint
lint:
	@echo "Running linter..."
	@command -v golangci-lint >/dev/null 2>&1 || { \
		echo "golangci-lint not found. Installing..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	}
	golangci-lint run

# Format the code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Vet the code
.PHONY: vet
vet:
	@echo "Vetting code..."
	go vet ./...

# Tidy module dependencies
.PHONY: tidy
tidy:
	@echo "Tidying module dependencies..."
	go mod tidy

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@rm -f $(BINARY_NAME)
	@rm -f coverage.out coverage.html
	@echo "Clean complete"

# Run all quality checks
.PHONY: check
check: fmt vet tidy test lint
	@echo "All quality checks passed"

# Show version information that would be injected
.PHONY: version-info
version-info:
	@echo "Version: $(VERSION)"
	@echo "Git Commit: $(GIT_COMMIT)"
	@echo "Build Date: $(BUILD_DATE)"

# Help target
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  all          - Clean and build"
	@echo "  build        - Build the binary with version injection"
	@echo "  build-dev    - Build for development (no version injection)"
	@echo "  build-all    - Cross-compile for multiple platforms"
	@echo "  install      - Install to GOPATH/bin"
	@echo "  test         - Run tests"
	@echo "  test-coverage- Run tests with coverage report"
	@echo "  lint         - Run linter"
	@echo "  fmt          - Format code"
	@echo "  vet          - Vet code"
	@echo "  tidy         - Tidy module dependencies"
	@echo "  clean        - Clean build artifacts"
	@echo "  check        - Run all quality checks"
	@echo "  version-info - Show version information"
	@echo "  help         - Show this help"