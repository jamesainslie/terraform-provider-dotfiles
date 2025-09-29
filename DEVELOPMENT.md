# Development Guide

This guide explains how to develop, test, and contribute to the terraform-provider-dotfiles project using our comprehensive CI/CD system.

## Local Development Setup

### Prerequisites

- Go 1.23+ (preferably Go 1.25.1)
- Git
- Python 3.x (for pre-commit hooks)
- Make

### Quick Start

```bash
# Clone the repository
git clone https://github.com/jamesainslie/terraform-provider-dotfiles.git
cd terraform-provider-dotfiles

# Run the automated setup script
./scripts/setup-hooks.sh

# Or manually set up the environment
make dev
```

### Manual Setup

If you prefer to set up manually:

```bash
# Install development tools
make tools

# Install pre-commit hooks
pip install pre-commit
pre-commit install --install-hooks
pre-commit install --hook-type commit-msg

# Run initial quality check
make quality
```

## Testing & Quality

### Make Targets

Our enhanced Makefile provides comprehensive quality control:

```bash
# Quick development cycle
make fmt lint test              # Fast formatting, linting, and tests

# Comprehensive quality checks
make quality                    # Run all quality gates
make pre-commit                # Fast pre-commit checks
make ci                        # Full CI simulation

# Specific checks
make lint                      # golangci-lint (comprehensive)
make lint-fast                 # golangci-lint (changed files only)
make staticcheck              # Static analysis
make security                 # Security/vulnerability scanning
make vet                      # Go vet
make vuln-scan               # Detailed vulnerability scan

# Testing
make test                     # Unit tests
make test-fast               # Fast unit tests (no race detection)
make test-race               # Tests with race detection
make test-coverage           # Tests with coverage report
make testacc                 # Acceptance tests (TF_ACC=1)
make benchmark              # Performance benchmarks

# Building
make build                   # Single binary
make build-all              # Multi-platform binaries
make build-migrate          # Migration tool

# Utilities  
make deps                   # Check/update dependencies
make license-check         # Check dependency licenses
make bench-compare         # Compare benchmark performance
make clean                 # Clean artifacts
```

### Pre-commit Hooks

Pre-commit hooks automatically run on every commit:

- **Go formatting** (gofmt, goimports)
- **Linting** (golangci-lint on changed files)
- **Security scanning** (gosec)
- **Module tidying** (go mod tidy)
- **Terraform validation** (for examples)
- **Conventional commit validation**

### Testing Strategy

We use a multi-layered testing approach:

1. **Unit Tests**: Fast, isolated tests (`make test`)
2. **Integration Tests**: Cross-component testing
3. **Acceptance Tests**: Full Terraform provider tests (`TF_ACC=1`)
4. **Platform Tests**: OS-specific functionality
5. **Race Detection**: Concurrent safety (`make test-race`)
6. **Benchmarks**: Performance monitoring (`make benchmark`)

## CI/CD Pipeline

### GitHub Workflows

#### Tests Workflow (`.github/workflows/test.yml`)

Runs on every PR and push:

- **Quality Gate**: Comprehensive linting, security, and static analysis
- **Code Generation Check**: Ensures generated docs are up-to-date
- **Multi-dimensional Matrix Testing**:
  - **OS**: ubuntu-latest, macos-latest, windows-latest
  - **Go versions**: 1.23.x, 1.24.x, 1.25.x
  - **Terraform versions**: 1.0.*, 1.2.*, 1.4.*, 1.9.*
- **Coverage Reporting**: Uploaded to Codecov
- **Performance Benchmarks**: Tracked over time

#### Security Workflow (`.github/workflows/security.yml`)

Daily security scanning:

- **Vulnerability Scanning**: govulncheck for Go vulnerabilities
- **Static Security Analysis**: gosec security scanner
- **CodeQL Analysis**: GitHub's semantic analysis
- **License Compliance**: Automated license checking
- **SARIF Reporting**: Integrated with GitHub Security tab

#### Release Workflow (`.github/workflows/release.yml`)

Triggered by version tags:

- **Comprehensive Quality Gate**: All tests must pass
- **Multi-platform Builds**: All supported OS/arch combinations
- **GPG Signing**: Cryptographic signing of releases
- **Automated Changelog**: Generated from conventional commits
- **Terraform Registry**: Automated publishing (when configured)

#### Quality Metrics (`.github/workflows/metrics.yml`)

Daily quality tracking:

- **Code Metrics**: Lines of code, complexity, dependencies
- **Test Coverage**: Historical coverage tracking  
- **Technical Debt**: TODO/FIXME/HACK comment tracking
- **Performance**: Benchmark result archiving

### Dependency Management

Automated via Dependabot (`.github/dependabot.yml`):

- **Go Modules**: Weekly updates for main and tools
- **GitHub Actions**: Weekly security and feature updates
- **Terraform**: Weekly updates for examples
- **Grouped Updates**: Related dependencies updated together
- **Security Prioritization**: Security updates get priority

## Commit Guidelines

We use [Conventional Commits](https://conventionalcommits.org/) for consistent, automated changelog generation:

### Format

```
<type>(<scope>): <description>

[optional body]

[optional footer(s)]
```

### Types

- `feat`: New features
- `fix`: Bug fixes
- `docs`: Documentation updates
- `style`: Code style (no logic changes)
- `refactor`: Code restructuring (no behavior changes)
- `test`: Test additions/improvements
- `chore`: Maintenance tasks
- `perf`: Performance improvements
- `ci`: CI/CD changes
- `build`: Build system changes

### Examples

```bash
feat(file): add template variable substitution support
fix(platform): resolve Windows path handling edge case
docs(readme): update installation instructions
refactor(provider): extract common validation logic
test(integration): add cross-platform file operations tests
```

### Breaking Changes

```bash
feat(config)!: migrate to new configuration schema

BREAKING CHANGE: Configuration format has changed from v1 to v2.
Run the migration tool: make build-migrate && ./bin/migrate-config
```

## Release Process

### Creating Releases

1. **Ensure Quality**: All CI checks pass
2. **Update Documentation**: Run `make generate` if needed  
3. **Create Tag**: Follow semantic versioning

   ```bash
   git tag v1.2.3
   git push origin v1.2.3
   ```

4. **Automated Release**: GitHub Actions handles the rest

### Release Artifacts

Our releases include:

- **Multi-platform Binaries**: All supported OS/architecture combinations
- **Checksums**: SHA256 verification files
- **GPG Signatures**: Cryptographic verification
- **Documentation**: Auto-generated from source
- **Changelog**: Generated from conventional commits

### GoReleaser Configuration

See `.goreleaser.yml` for complete build configuration:

- **Cross-compilation**: 12+ platform combinations
- **Archive Generation**: ZIP files with metadata
- **Signing**: GPG signature verification
- **Registry Integration**: Terraform Registry publishing
- **Custom Metadata**: Version embedding in binaries

## Troubleshooting

### Common Issues

#### Pre-commit Hooks Failing

```bash
# Update pre-commit hooks
pre-commit autoupdate

# Run hooks manually
pre-commit run --all-files

# Skip hooks for emergency commits (use sparingly)
git commit --no-verify
```

#### Linting Issues

```bash
# Auto-fix most issues
make fmt

# Check specific linter rules
golangci-lint run --enable-all

# Run only fast linters
make lint-fast
```

#### Test Failures

```bash
# Run tests with verbose output
go test -v ./...

# Run specific test
go test -v ./internal/provider/ -run TestSpecificFunction

# Run acceptance tests
TF_ACC=1 go test -v ./internal/provider/
```

#### Security Scan Failures

```bash
# Run local security scan
make security

# Check specific vulnerability
govulncheck ./...

# Review security report
gosec ./...
```

### Development Tips

1. **Use Fast Feedback Loops**: `make pre-commit` for quick checks
2. **Run Full Quality**: `make quality` before pushing
3. **Monitor Coverage**: `make test-coverage` and review `coverage.html`
4. **Check Dependencies**: `make deps` to see available updates
5. **Benchmark Performance**: `make bench-compare` for performance tracking

## Quality Standards

### Code Quality Targets

- **Test Coverage**: > 80%
- **Cyclomatic Complexity**: < 12 per function
- **Function Length**: < 100 lines
- **Linting**: Zero golangci-lint issues
- **Security**: Zero high/critical security issues
- **Dependencies**: Regular updates, no known vulnerabilities

### CI/CD Success Criteria

- **All Tests Pass**: Unit, integration, and acceptance tests
- **Quality Gates**: Linting, security, and complexity checks
- **Multi-platform**: Successful builds on Linux, macOS, Windows
- **Performance**: No significant benchmark regressions
- **Documentation**: Auto-generated docs are current

## Contributing

1. **Fork & Branch**: Create feature branches from `main`
2. **Develop**: Use the local development setup
3. **Test**: Run `make quality` before pushing
4. **Commit**: Follow conventional commit format
5. **PR**: Submit pull request with clear description
6. **Review**: Address feedback from maintainers
7. **Merge**: Maintainers handle merging

### PR Checklist

- [ ] Tests pass locally (`make quality`)
- [ ] Pre-commit hooks installed and passing
- [ ] Conventional commit format used
- [ ] Documentation updated (if needed)
- [ ] Acceptance tests pass (if applicable)
- [ ] No breaking changes (or properly documented)

---

This comprehensive CI/CD system ensures high code quality, security, and reliability. The automated processes handle most of the heavy lifting, allowing you to focus on writing great code!

For questions or issues, please open a GitHub issue or contact the maintainers.
