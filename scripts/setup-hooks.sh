#!/bin/bash
# Setup script for terraform-provider-dotfiles development environment
# This script installs and configures pre-commit hooks and development tools

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_header() {
    echo -e "${BLUE}=== $1 ===${NC}"
}

# Check if running from the repository root
if [[ ! -f "go.mod" ]] || [[ ! -d ".git" ]]; then
    print_error "This script must be run from the repository root directory"
    exit 1
fi

print_header "Setting up terraform-provider-dotfiles development environment"

# Check dependencies
print_status "Checking dependencies..."

# Check if Python is available (for pre-commit)
if ! command -v python3 &> /dev/null && ! command -v python &> /dev/null; then
    print_error "Python is required for pre-commit. Please install Python 3.x"
    exit 1
fi

# Check if Go is available
if ! command -v go &> /dev/null; then
    print_error "Go is required. Please install Go 1.23+ from https://golang.org"
    exit 1
fi

# Check Go version
GO_VERSION=$(go version | grep -oE 'go[0-9]+\.[0-9]+' | cut -c3-)
REQUIRED_VERSION="1.23"
if [[ $(echo "$GO_VERSION $REQUIRED_VERSION" | tr " " "\n" | sort -V | head -n1) != "$REQUIRED_VERSION" ]]; then
    print_error "Go version $REQUIRED_VERSION or higher is required. Current version: $GO_VERSION"
    exit 1
fi

print_status " Dependencies check passed"

# Install pre-commit
print_header "Installing pre-commit"
if command -v pre-commit &> /dev/null; then
    print_status "pre-commit is already installed"
else
    print_status "Installing pre-commit..."
    if command -v pip3 &> /dev/null; then
        pip3 install --user pre-commit
    elif command -v pip &> /dev/null; then
        pip install --user pre-commit
    elif command -v brew &> /dev/null; then
        brew install pre-commit
    else
        print_error "Could not install pre-commit. Please install it manually:"
        print_error "  pip install pre-commit  # or  brew install pre-commit"
        exit 1
    fi
fi

# Install development tools
print_header "Installing Go development tools"
print_status "Installing tools via make..."
make tools

# Install pre-commit hooks
print_header "Installing pre-commit hooks"
pre-commit install --install-hooks
pre-commit install --hook-type commit-msg

# Run initial quality check
print_header "Running initial quality check"
print_status "This may take a few minutes on first run..."

# Format code first
print_status "Formatting code..."
make fmt

# Run quality checks
if make quality; then
    print_status " Initial quality check passed"
else
    print_warning "Quality check found issues. Run 'make quality' to see details."
fi

# Setup completion
print_header "Setup Complete!"
echo -e "${GREEN}Development environment is ready!${NC}"
echo ""
echo "Available commands:"
echo "  make help          - Show all available make targets"
echo "  make dev           - Run full development setup"
echo "  make quality       - Run comprehensive quality checks"
echo "  make pre-commit    - Run fast pre-commit checks"
echo "  make test          - Run unit tests"
echo "  make lint          - Run linting"
echo ""
echo "Pre-commit hooks are now active and will run automatically on:"
echo "  - git commit (formatting, linting, fast tests)"
echo "  - git push (comprehensive tests)"
echo ""
echo -e "${BLUE}Happy coding! ${NC}"
