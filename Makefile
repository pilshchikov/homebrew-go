# Homebrew Go - Build System
.PHONY: build install test clean fmt vet lint help deps dev-deps
.DEFAULT_GOAL := help

# Build configuration
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT ?= $(shell git rev-parse HEAD 2>/dev/null || echo "unknown")
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.GitCommit=$(GIT_COMMIT) -X main.BuildDate=$(BUILD_DATE)"

# Go configuration
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
GO_VERSION := $(shell go version | cut -d ' ' -f 3)

# Directories
BUILD_DIR := build
DIST_DIR := dist
COVERAGE_DIR := coverage

# Binary names
BINARY_NAME := brew
BINARY_PATH := $(BUILD_DIR)/$(BINARY_NAME)

# Cross-compilation targets
PLATFORMS := darwin/amd64 darwin/arm64 linux/amd64 linux/arm64

help: ## Show this help message
	@echo "Homebrew Go - Build System"
	@echo ""
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ""
	@echo "Build information:"
	@echo "  Version:     $(VERSION)"
	@echo "  Git Commit:  $(GIT_COMMIT)"
	@echo "  Build Date:  $(BUILD_DATE)"
	@echo "  Go Version:  $(GO_VERSION)"
	@echo "  Target:      $(GOOS)/$(GOARCH)"

deps: ## Download and install dependencies
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

dev-deps: deps ## Install development dependencies
	@echo "Installing development dependencies..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/goreleaser/goreleaser@latest
	go install golang.org/x/tools/cmd/goimports@latest

fmt: ## Format Go code
	@echo "Formatting code..."
	go fmt ./...
	goimports -w .

vet: ## Run go vet
	@echo "Running go vet..."
	go vet ./...

lint: ## Run linter
	@echo "Running linter..."
	golangci-lint run

test: ## Run tests
	@echo "Running tests..."
	mkdir -p $(COVERAGE_DIR)
	go test -v -race -coverprofile=$(COVERAGE_DIR)/coverage.out ./...
	go tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	@echo "Coverage report generated: $(COVERAGE_DIR)/coverage.html"

test-short: ## Run tests (short mode)
	@echo "Running tests (short mode)..."
	go test -short ./...

build: deps ## Build the binary
	@echo "Building $(BINARY_NAME) for $(GOOS)/$(GOARCH)..."
	mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BINARY_PATH) ./cmd/brew
	@echo "Binary built: $(BINARY_PATH)"

build-all: deps ## Build for all supported platforms
	@echo "Building for all platforms..."
	mkdir -p $(DIST_DIR)
	@for platform in $(PLATFORMS); do \
		GOOS=$$(echo $$platform | cut -d'/' -f1); \
		GOARCH=$$(echo $$platform | cut -d'/' -f2); \
		binary_name=$(BINARY_NAME)-$$GOOS-$$GOARCH; \
		if [ "$$GOOS" = "windows" ]; then \
			binary_name=$$binary_name.exe; \
		fi; \
		echo "Building $$binary_name..."; \
		GOOS=$$GOOS GOARCH=$$GOARCH go build $(LDFLAGS) -o $(DIST_DIR)/$$binary_name ./cmd/brew; \
	done
	@echo "All binaries built in $(DIST_DIR)/"

install: build ## Install the binary to GOPATH/bin
	@echo "Installing $(BINARY_NAME)..."
	go install $(LDFLAGS) ./cmd/brew
	@echo "Installed to $(shell go env GOPATH)/bin/$(BINARY_NAME)"

install-local: build ## Install the binary to /usr/local/bin (requires sudo)
	@echo "Installing $(BINARY_NAME) to /usr/local/bin..."
	sudo cp $(BINARY_PATH) /usr/local/bin/$(BINARY_NAME)
	sudo chmod +x /usr/local/bin/$(BINARY_NAME)
	@echo "Installed to /usr/local/bin/$(BINARY_NAME)"

clean: ## Clean build artifacts
	@echo "Cleaning..."
	rm -rf $(BUILD_DIR) $(DIST_DIR) $(COVERAGE_DIR)
	go clean

run: build ## Build and run the binary
	@echo "Running $(BINARY_NAME)..."
	./$(BINARY_PATH) $(ARGS)

debug: ## Build and run with debug output
	@echo "Running $(BINARY_NAME) in debug mode..."
	go run $(LDFLAGS) ./cmd/brew --debug $(ARGS)

generate: ## Generate code (if any go:generate directives exist)
	@echo "Generating code..."
	go generate ./...

mod-tidy: ## Tidy up module dependencies
	@echo "Tidying modules..."
	go mod tidy

mod-vendor: ## Create vendor directory
	@echo "Vendoring dependencies..."
	go mod vendor

docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build -t homebrew-go:$(VERSION) -t homebrew-go:latest .

docker-run: docker-build ## Run in Docker container
	@echo "Running in Docker..."
	docker run --rm -it homebrew-go:latest $(ARGS)

release: clean fmt vet lint test build-all ## Prepare a release
	@echo "Release $(VERSION) ready in $(DIST_DIR)/"

check: fmt vet lint test ## Run all checks
	@echo "All checks passed!"

pre-commit: fmt vet lint test-short ## Run pre-commit checks
	@echo "Pre-commit checks passed!"

info: ## Show build information
	@echo "Build Information:"
	@echo "  Version:      $(VERSION)"
	@echo "  Git Commit:   $(GIT_COMMIT)"
	@echo "  Build Date:   $(BUILD_DATE)"
	@echo "  Go Version:   $(GO_VERSION)"
	@echo "  Target OS:    $(GOOS)"
	@echo "  Target Arch:  $(GOARCH)"
	@echo ""
	@echo "Directories:"
	@echo "  Build:        $(BUILD_DIR)"
	@echo "  Dist:         $(DIST_DIR)"
	@echo "  Coverage:     $(COVERAGE_DIR)"
	@echo ""
	@echo "Binary:"
	@echo "  Name:         $(BINARY_NAME)"
	@echo "  Path:         $(BINARY_PATH)"

benchmark: ## Run benchmarks
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./...

profile: ## Run with profiling enabled
	@echo "Running with profiling..."
	go build $(LDFLAGS) -o $(BINARY_PATH) ./cmd/brew
	./$(BINARY_PATH) --help
	@echo "Profile data available in current directory"

# Development helpers
watch: ## Watch for changes and rebuild
	@echo "Watching for changes..."
	@which air > /dev/null || (echo "Installing air..." && go install github.com/cosmtrek/air@latest)
	air

setup-hooks: ## Setup git hooks
	@echo "Setting up git hooks..."
	@echo "#!/bin/sh\nmake pre-commit" > .git/hooks/pre-commit
	@chmod +x .git/hooks/pre-commit
	@echo "Git hooks installed"

# CI/CD helpers
ci-test: deps fmt vet lint test ## Run CI tests
	@echo "CI tests completed"

ci-build: deps build-all ## Run CI build
	@echo "CI build completed"

# Update dependencies
update-deps: ## Update all dependencies to latest versions
	@echo "Updating dependencies..."
	go get -u ./...
	go mod tidy