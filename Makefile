# Makefile for terraform-provider-dotfiles

# This Makefile provides standard targets for building, testing, and maintaining
# the Terraform provider.

GO := go
GOLANGCI_LINT := golangci-lint
BINARY_NAME := terraform-provider-dotfiles
TOOLS_DIR := tools
DOCS_VERSION := v0.22.0  # Match the version in tools/go.mod for terraform-plugin-docs

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
	$(GO) build -v ./...

# Install
.PHONY: install
install: build ## Install the provider binary
	$(GO) install -v ./...

# Test
.PHONY: test
test: ## Run unit tests
	GOTOOLCHAIN=go1.25.1 $(GO) test -v -cover -timeout=120s -parallel=10 -gcflags=all=-lang=go1.24 -vet=off ./...

# Test with race detector
.PHONY: test-race
test-race: ## Run tests with race detector
	GOTOOLCHAIN=go1.25.1 $(GO) test -race -v -gcflags=all=-lang=go1.24 -vet=off ./...

# Acceptance tests (requires TF_ACC=1 env var)
.PHONY: testacc
testacc: ## Run acceptance tests
	TF_ACC=1 GOTOOLCHAIN=go1.25.1 $(GO) test -v -cover -timeout 120m -gcflags=all=-lang=go1.24 -vet=off ./...

# Lint
.PHONY: lint
lint: ## Run golangci-lint
	$(GOLANGCI_LINT) run

# Format
.PHONY: fmt
fmt: ## Format Go code
	gofmt -s -w -e .

# Generate
.PHONY: generate
generate: ## Generate code and documentation
	cd $(TOOLS_DIR) && $(GO) generate ./...
	$(GO) mod tidy

# Install tools
.PHONY: tools
tools: ## Install development tools from tools/go.mod
	cd $(TOOLS_DIR) && $(GO) install github.com/hashicorp/terraform-plugin-docs@${DOCS_VERSION}
	cd $(TOOLS_DIR) && $(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Dev setup
.PHONY: dev
dev: tools generate fmt lint test ## Setup development environment (install tools, generate, fmt, lint, test)

# Clean
.PHONY: clean
clean: ## Clean build artifacts and caches
	$(GO) clean -cache -testcache -modcache
	rm -f $(BINARY_NAME) coverage.out coverage.html

# Build migrate-config tool
.PHONY: build-migrate
build-migrate: ## Build the migrate-config binary
	$(GO) build -o bin/migrate-config ./cmd/migrate-config

# CI (what CI runs)
.PHONY: ci
ci: fmt lint test test-race ## Run checks for CI
