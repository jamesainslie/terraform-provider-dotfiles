# Makefile for terraform-provider-dotfiles

# This Makefile provides standard targets for building, testing, and maintaining
# the Terraform provider.

# Variables
GO := go
GOLANGCI_LINT := golangci-lint
BINARY_NAME := terraform-provider-dotfiles
TOOLS_DIR := tools
DOCS_VERSION := v0.22.0  # Match the version in tools/go.mod for terraform-plugin-docs
VERSION := $(shell git describe --tags --always --dirty)
COMMIT_HASH := $(shell git rev-parse HEAD)

# Go toolchain and build flags
GOTOOLCHAIN := go1.25.1
GOPROXY := https://proxy.golang.org,direct
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT_HASH)"
GOFLAGS_BUILD := -gcflags=all=-lang=go1.25
GOFLAGS_TEST := -gcflags=all=-lang=go1.24

# Export environment variables for all targets
export GOTOOLCHAIN
export GOPROXY

# Default target
.PHONY: default
default: fmt lint install generate

# Help
.PHONY: help
help: ## Show this help
	@egrep -h '\\s##\\s' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# Build
.PHONY: build
build: ## Build the provider binary
	@echo "Building $(BINARY_NAME)..."
	GOFLAGS="$(GOFLAGS_BUILD)" $(GO) build $(LDFLAGS) -v -o $(BINARY_NAME) ./

# Install
.PHONY: install
install: build ## Install the provider binary
	@echo "Installing provider binary..."
	GOFLAGS="$(GOFLAGS_BUILD)" $(GO) install $(LDFLAGS) -v ./...

# Test
.PHONY: test
test: ## Run unit tests
	@echo "Running unit tests..."
	GOFLAGS="$(GOFLAGS_TEST)" $(GO) test -v -cover -timeout=120s -parallel=10 -vet=off ./...

# Test with coverage report
.PHONY: test-coverage
test-coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	GOFLAGS="$(GOFLAGS_TEST)" $(GO) test -v -coverprofile=coverage.out -covermode=atomic -timeout=120s -parallel=10 -vet=off ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Test with race detector
.PHONY: test-race
test-race: ## Run tests with race detector
	@echo "Running tests with race detector..."
	GOFLAGS="$(GOFLAGS_TEST)" $(GO) test -race -v -vet=off ./...

# Acceptance tests (requires TF_ACC=1 env var)
.PHONY: testacc
testacc: ## Run acceptance tests
	@echo "Running acceptance tests..."
	TF_ACC=1 GOFLAGS="$(GOFLAGS_TEST)" $(GO) test -v -cover -timeout 120m -vet=off ./...

# Lint
.PHONY: lint
lint: ## Run golangci-lint
	@echo "Running golangci-lint..."
	GOFLAGS="$(GOFLAGS_BUILD)" $(GOLANGCI_LINT) run

# Static analysis
.PHONY: staticcheck
staticcheck: ## Run staticcheck static analysis
	@echo "Running staticcheck..."
	@command -v staticcheck >/dev/null 2>&1 || { echo "staticcheck not found. Install with: go install honnef.co/go/tools/cmd/staticcheck@latest"; exit 1; }
	GOFLAGS="$(GOFLAGS_BUILD)" staticcheck ./...

# Security checks
.PHONY: security
security: ## Run security vulnerability checks
	@echo "Running security checks..."
	@command -v govulncheck >/dev/null 2>&1 || { echo "govulncheck not found. Install with: go install golang.org/x/vuln/cmd/govulncheck@latest"; exit 1; }
	GOFLAGS="" govulncheck ./...

# Go vet
.PHONY: vet
vet: ## Run go vet
	@echo "Running go vet..."
	GOFLAGS="" $(GO) vet ./internal/... ./cmd/... .

# Format
.PHONY: fmt
fmt: ## Format Go code
	@echo "Formatting Go code..."
	gofmt -s -w -e .
	@command -v goimports >/dev/null 2>&1 && goimports -w . || echo "goimports not found, skipping import formatting"

# Generate
.PHONY: generate
generate: ## Generate code and documentation
	@echo "Generating code and documentation..."
	cd $(TOOLS_DIR) && $(GO) generate ./...
	$(GO) mod tidy

# Module management
.PHONY: mod
mod: ## Tidy and verify go modules
	@echo "Tidying go modules..."
	$(GO) mod tidy
	$(GO) mod verify

# Install tools
.PHONY: tools
tools: ## Install development tools from tools/go.mod
	@echo "Installing development tools..."
	cd $(TOOLS_DIR) && $(GO) install github.com/hashicorp/terraform-plugin-docs@${DOCS_VERSION}
	cd $(TOOLS_DIR) && $(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	$(GO) install honnef.co/go/tools/cmd/staticcheck@latest
	$(GO) install golang.org/x/tools/cmd/goimports@latest
	$(GO) install golang.org/x/vuln/cmd/govulncheck@latest

# Cross-platform builds
.PHONY: build-all
build-all: ## Build for all supported platforms
	@echo "Building for all platforms..."
	mkdir -p dist/
	GOFLAGS="$(GOFLAGS_BUILD)" GOOS=darwin GOARCH=amd64 $(GO) build $(LDFLAGS) -o dist/$(BINARY_NAME)_darwin_amd64 ./
	GOFLAGS="$(GOFLAGS_BUILD)" GOOS=darwin GOARCH=arm64 $(GO) build $(LDFLAGS) -o dist/$(BINARY_NAME)_darwin_arm64 ./
	GOFLAGS="$(GOFLAGS_BUILD)" GOOS=linux GOARCH=amd64 $(GO) build $(LDFLAGS) -o dist/$(BINARY_NAME)_linux_amd64 ./
	GOFLAGS="$(GOFLAGS_BUILD)" GOOS=linux GOARCH=arm64 $(GO) build $(LDFLAGS) -o dist/$(BINARY_NAME)_linux_arm64 ./
	GOFLAGS="$(GOFLAGS_BUILD)" GOOS=windows GOARCH=amd64 $(GO) build $(LDFLAGS) -o dist/$(BINARY_NAME)_windows_amd64.exe ./

# Quality checks workflow
.PHONY: quality
quality: ## Run comprehensive quality checks
	@echo "=== COMPREHENSIVE QUALITY CHECK ==="
	@$(MAKE) lint
	@echo " golangci-lint passed"
	@echo "2. Running staticcheck..."
	@$(MAKE) staticcheck
	@echo " staticcheck passed"
	@echo "3. Running security checks..."
	@$(MAKE) security
	@echo " security checks passed"
	@$(MAKE) test
	@echo " All tests passed"
	@echo "5. Building project..."
	@GOFLAGS="$(GOFLAGS_BUILD)" $(GO) build $(LDFLAGS) ./...
	@echo " Build successful"
	@echo ""
	@echo " ALL QUALITY CHECKS PASSED! "

# Pre-commit checks
.PHONY: pre-commit
pre-commit: ## Run pre-commit checks
	@echo "Running pre-commit checks..."
	@$(MAKE) fmt mod lint test

# Dev setup
.PHONY: dev
dev: tools generate fmt lint test ## Setup development environment (install tools, generate, fmt, lint, test)

# Clean
.PHONY: clean
clean: ## Clean build artifacts and caches
	@echo "Cleaning build artifacts..."
	$(GO) clean -cache -testcache -modcache
	rm -f $(BINARY_NAME) coverage.out coverage.html
	rm -rf dist/

# Build migrate-config tool
.PHONY: build-migrate
build-migrate: ## Build the migrate-config binary
	@echo "Building migrate-config tool..."
	GOFLAGS="$(GOFLAGS_BUILD)" $(GO) build -o bin/migrate-config ./cmd/migrate-config

# Benchmarks
.PHONY: benchmark
benchmark: ## Run benchmark tests
	@echo "Running benchmark tests..."
	GOFLAGS="$(GOFLAGS_TEST)" $(GO) test -bench=. -benchmem ./...

# CI (what CI runs)
.PHONY: ci
ci: mod fmt lint staticcheck test test-race ## Run comprehensive CI checks

# Pre-release quality gate
.PHONY: pre-release
pre-release: ## Run comprehensive pre-release quality gate
	@echo "Running pre-release quality gate..."
	@echo "=================================="
	@echo "1. Checking Go modules..."
	@$(MAKE) mod
	@echo "Go modules OK"
	@echo "2. Running static analysis..."
	@$(MAKE) lint
	@echo "Linting passed"
	@echo "3. Running security checks..."
	@$(MAKE) security  
	@echo "Security checks passed"
	@echo "4. Running unit tests with race detection..."
	@$(MAKE) test-race
	@echo "Unit tests passed"
	@echo "5. Building binaries..."
	@$(MAKE) build
	@echo "Build successful"
	@echo "6. Running test coverage analysis..."
	@$(MAKE) test-coverage
	@echo "Coverage analysis complete"
	@echo ""
	@echo "All quality gates passed! Ready for release."
	@echo "=================================="
