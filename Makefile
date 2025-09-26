# Makefile for terraform-provider-dotfiles

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt
GOLINT=golangci-lint

# Binary names
BINARY_NAME=terraform-provider-dotfiles
BINARY_UNIX=$(BINARY_NAME)_unix

# Build targets
.PHONY: all build clean test coverage lint fmt vet deps help

all: test build

## Build the binary
build:
	$(GOBUILD) -o $(BINARY_NAME) -v ./...

## Clean build artifacts
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_UNIX)

## Run tests
test:
	$(GOTEST) -v ./...

## Run tests with coverage
coverage:
	$(GOTEST) -race -coverprofile=coverage.out -covermode=atomic ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

## Run tests with race detection
test-race:
	$(GOTEST) -race -short ./...

## Run benchmarks
bench:
	$(GOTEST) -bench=. -benchmem ./...

## Run linting
lint:
	$(GOLINT) run

## Run linting with auto-fix
lint-fix:
	$(GOLINT) run --fix

## Format code
fmt:
	$(GOFMT) -s -w .
	$(GOCMD) mod tidy

## Run go vet
vet:
	$(GOCMD) vet ./...

## Update dependencies
deps:
	$(GOMOD) download
	$(GOMOD) tidy
	$(GOMOD) verify

## Install development tools
install-tools:
	@echo "Installing development tools..."
	$(GOGET) -u github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "Installing pre-commit..."
	@command -v pre-commit >/dev/null 2>&1 || { echo "Please install pre-commit: pip install pre-commit"; exit 1; }
	pre-commit install

## Run security scan
security:
	@command -v gosec >/dev/null 2>&1 || { echo "Installing gosec..."; $(GOGET) -u github.com/securecodewarrior/gosec/v2/cmd/gosec@latest; }
	gosec ./...

## Generate mocks (if using mockery)
mocks:
	@command -v mockery >/dev/null 2>&1 || { echo "Installing mockery..."; $(GOGET) -u github.com/vektra/mockery/v2@latest; }
	mockery --all --output=./mocks

## Cross compilation
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BINARY_UNIX) -v

## Docker build
docker-build:
	docker build -t $(BINARY_NAME) .

## Run pre-commit hooks
pre-commit:
	pre-commit run --all-files

## Full check (what CI would run)
ci: deps fmt vet lint test-race coverage

## Development setup
dev-setup: install-tools deps
	@echo "Development environment setup complete!"

## Help
help:
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

# Default target
.DEFAULT_GOAL := help
