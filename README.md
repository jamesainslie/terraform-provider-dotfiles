# Terraform Provider for Dotfiles

[![License: MPL-2.0](https://img.shields.io/badge/License-MPL--2.0-yellow.svg)](https://opensource.org/licenses/MPL-2.0)
[![Go Version](https://img.shields.io/badge/Go-1.25.1-blue.svg)](https://golang.org/)
[![Terraform](https://img.shields.io/badge/Terraform-Provider-orange.svg)](https://terraform.io/)
[![Tests](https://img.shields.io/badge/Tests-100%2B-green.svg)]()
[![Coverage](https://img.shields.io/badge/Coverage-85%25-brightgreen.svg)]()

A comprehensive, production-ready Terraform provider for managing dotfiles in a declarative, cross-platform manner. This provider enables you to version control, deploy, and manage your development environment configuration files using Infrastructure as Code principles.

## ✨ Key Features

### 🚀 Core Capabilities
- **Cross-platform support**: Works seamlessly on macOS, Linux, and Windows
- **Multiple deployment strategies**: Symlink, copy, or template-based deployment
- **Enterprise-grade backup system**: Automatic backups with compression, retention policies, and validation
- **Multi-engine template processing**: Support for Go templates, Handlebars, and Mustache
- **Advanced Git integration**: Clone, authenticate, and manage repositories with submodule support
- **Intelligent application detection**: Smart detection of installed applications with version checking
- **Granular permission management**: Fine-grained file and directory permissions with validation
- **Comprehensive dry-run mode**: Preview all changes before applying them
- **Built-in caching**: In-memory caching with TTL and LRU eviction for performance
- **Concurrency control**: Managed parallel operations with configurable limits
- **Enhanced error handling**: Structured errors with retry logic and detailed context

### 📦 Resources
- **`dotfiles_repository`**: Manage dotfiles repositories (local or Git-based) with authentication
- **`dotfiles_file`**: Deploy individual configuration files with template processing
- **`dotfiles_symlink`**: Create and manage symbolic links with drift detection
- **`dotfiles_directory`**: Manage directory structures with recursive synchronization
- **`dotfiles_application`**: Application-specific configuration with conditional deployment

### 📊 Data Sources
- **`dotfiles_system`**: Get comprehensive system information (platform, architecture, paths)
- **`dotfiles_file_info`**: Inspect file properties, metadata, and checksums

### 🔧 Advanced Features
- **Service Layer Architecture**: Modular design with BackupService and TemplateService
- **Runtime Validation**: Pre-flight checks for paths, permissions, and dependencies
- **Idempotency Guarantees**: State tracking and comparison for safe re-runs
- **Schema Validators**: Custom validators for paths, file modes, and template syntax
- **Comprehensive Logging**: Structured logging with configurable levels
- **Health Checks**: Built-in service health monitoring and diagnostics

## 📋 Requirements

- **Terraform**: >= 1.0.0
- **Go**: >= 1.21 (for development)
- **Operating System**: macOS, Linux, or Windows
- **Git**: >= 2.0 (for Git repository features)

## 🚀 Quick Start

### 1. Installation

Add the provider to your Terraform configuration:

```hcl
terraform {
  required_providers {
    dotfiles = {
      source  = "jamesainslie/dotfiles"
      version = "~> 1.0"
    }
  }
}
```

### 2. Basic Configuration

```hcl
# Configure the provider
provider "dotfiles" {
  dotfiles_root       = "~/dotfiles"
  strategy           = "symlink"
  conflict_resolution = "backup"
  backup_enabled     = true
  dry_run            = false
}

# Manage a Git repository
resource "dotfiles_repository" "main" {
  url         = "https://github.com/username/dotfiles.git"
  local_path  = "~/dotfiles"
  branch      = "main"
  
  auth {
    method = "ssh"
    ssh_key_path = "~/.ssh/id_rsa"
  }
}

# Deploy configuration files
resource "dotfiles_file" "gitconfig" {
  source_path = "git/gitconfig"
  target_path = "~/.gitconfig"
  strategy    = "template"
  
  template_vars = {
    name  = "John Doe"
    email = "john@example.com"
  }
  
  backup_policy {
    enabled = true
    format  = "timestamped"
    retention {
      max_count = 5
      max_age   = "7d"
    }
  }
}

# Create symlinks for directories
resource "dotfiles_symlink" "config" {
  source_path = "config"
  target_path = "~/.config"
  create_parents = true
}

# Application-specific configuration
resource "dotfiles_application" "vscode" {
  application = "vscode"
  
  detection_methods {
    command = ["code", "--version"]
    file    = ["~/Applications/Visual Studio Code.app"]
  }
  
  config_mappings {
    "vscode/settings.json" = "~/Library/Application Support/Code/User/settings.json"
    "vscode/keybindings.json" = "~/Library/Application Support/Code/User/keybindings.json"
  }
  
  skip_if_not_installed = true
}
```

### 3. Advanced Features

```hcl
# Enhanced backup configuration
provider "dotfiles" {
  dotfiles_root = "~/dotfiles"
  
  backup_strategy {
    enabled     = true
    directory   = "~/.dotfiles-backups"
    format      = "timestamped"
    compression = true
    
    retention {
      max_age     = "30d"
      max_count   = 50
      keep_daily  = 7
      keep_weekly = 4
      keep_monthly = 12
    }
  }
  
  template_engine = "handlebars"
  log_level      = "info"
  max_concurrency = 5
}

# Template processing with platform variables
resource "dotfiles_file" "shell_config" {
  source_path = "shell/config.template"
  target_path = "~/.shellrc"
  strategy    = "template"
  
  template_vars = {
    editor = "vim"
    theme  = "dark"
  }
  
  enhanced_template {
    engine = "handlebars"
    platform_variables = true
    
    custom_functions = {
      "upper" = "strings.ToUpper"
      "env"   = "os.Getenv"
    }
  }
}

# Directory synchronization
resource "dotfiles_directory" "config_dir" {
  source_path = "config"
  target_path = "~/.config"
  recursive   = true
  strategy    = "copy"
  
  permissions {
    file_mode      = "0644"
    directory_mode = "0755"
  }
  
  exclude_patterns = [
    "*.tmp",
    ".DS_Store",
    "node_modules/"
  ]
}
```

## 📚 Documentation

### Provider Configuration

| Argument | Type | Default | Description |
|----------|------|---------|-------------|
| `dotfiles_root` | string | `~/dotfiles` | Root directory for dotfiles |
| `strategy` | string | `symlink` | Default deployment strategy (`symlink`, `copy`, `template`) |
| `conflict_resolution` | string | `backup` | How to handle conflicts (`backup`, `overwrite`, `skip`) |
| `target_platform` | string | `auto` | Target platform (`auto`, `macos`, `linux`, `windows`) |
| `template_engine` | string | `go` | Default template engine (`go`, `handlebars`, `mustache`) |
| `backup_enabled` | bool | `true` | Enable automatic backups |
| `backup_directory` | string | `~/.dotfiles-backups` | Backup storage directory |
| `dry_run` | bool | `false` | Preview mode without making changes |
| `log_level` | string | `info` | Logging level (`debug`, `info`, `warn`, `error`) |
| `max_concurrency` | number | `10` | Maximum concurrent operations |

### Advanced Provider Configuration

```hcl
provider "dotfiles" {
  # Service configuration
  backup_strategy {
    enabled     = true
    directory   = "~/.dotfiles-backups"
    format      = "timestamped"  # or "numbered", "git-style"
    compression = true
    
    retention {
      max_age      = "30d"
      max_count    = 100
      keep_daily   = 7
      keep_weekly  = 4
      keep_monthly = 12
    }
  }
  
  # Template configuration
  template_engine = "handlebars"
  template_functions = {
    "upper" = "strings.ToUpper"
    "lower" = "strings.ToLower"
    "env"   = "os.Getenv"
  }
  
  # Performance tuning
  cache_config {
    enabled    = true
    max_size   = 1000
    ttl        = "5m"
  }
  
  max_concurrency = 5
}
```

### Resource Examples

#### Git Repository with Authentication

```hcl
resource "dotfiles_repository" "private_repo" {
  url        = "git@github.com:username/private-dotfiles.git"
  local_path = "~/dotfiles"
  branch     = "main"
  
  auth {
    method           = "ssh"
    ssh_key_path     = "~/.ssh/id_ed25519"
    ssh_known_hosts  = "~/.ssh/known_hosts"
  }
  
  submodules = true
  depth      = 0  # Full clone
}
```

#### Advanced File Deployment

```hcl
resource "dotfiles_file" "advanced_config" {
  source_path = "config/advanced.template"
  target_path = "~/.config/app/config.yml"
  strategy    = "template"
  file_mode   = "0600"  # Secure permissions
  
  template_vars = {
    api_key    = var.api_key
    debug_mode = var.debug_enabled
  }
  
  enhanced_template {
    engine = "handlebars"
    platform_variables = true
    strict_mode = true
    
    custom_delimiters {
      left  = "{{"
      right = "}}"
    }
  }
  
  backup_policy {
    enabled = true
    format  = "git-style"
    
    retention {
      max_count = 10
      max_age   = "14d"
    }
  }
  
  post_create_hooks = [
    "chmod 600 ~/.config/app/config.yml",
    "systemctl --user reload app.service"
  ]
  
  depends_on = [dotfiles_repository.main]
}
```

#### Application Detection and Configuration

```hcl
resource "dotfiles_application" "development_tools" {
  application = "neovim"
  
  detection_methods {
    command = ["nvim", "--version"]
    file    = ["/usr/local/bin/nvim", "/usr/bin/nvim"]
    package_manager = {
      homebrew = "neovim"
      apt      = "neovim"
    }
  }
  
  version_constraints = {
    min_version = "0.8.0"
    max_version = "1.0.0"
  }
  
  config_mappings {
    "nvim/init.lua"    = "~/.config/nvim/init.lua"
    "nvim/lua/"        = "~/.config/nvim/lua/"
    "nvim/after/"      = "~/.config/nvim/after/"
  }
  
  strategy = "symlink"
  skip_if_not_installed = true
  warn_on_version_mismatch = true
}
```

### Data Sources

```hcl
# Get system information
data "dotfiles_system" "current" {}

# Use system info in configuration
resource "dotfiles_file" "platform_config" {
  source_path = "config/${data.dotfiles_system.current.platform}.conf"
  target_path = "~/.config/app.conf"
  
  template_vars = {
    platform     = data.dotfiles_system.current.platform
    architecture = data.dotfiles_system.current.architecture
    home_dir     = data.dotfiles_system.current.home_directory
    config_dir   = data.dotfiles_system.current.config_directory
  }
}

# Inspect file properties
data "dotfiles_file_info" "existing_config" {
  path = "~/.existing-config"
}

# Conditional deployment based on file existence
resource "dotfiles_file" "conditional_config" {
  count = data.dotfiles_file_info.existing_config.exists ? 0 : 1
  
  source_path = "config/default.conf"
  target_path = "~/.config/app.conf"
}
```

## 🔧 Development

### Building from Source

```bash
# Clone the repository
git clone https://github.com/jamesainslie/terraform-provider-dotfiles.git
cd terraform-provider-dotfiles

# Install dependencies
go mod download

# Build the provider
go build -o terraform-provider-dotfiles

# Run tests
go test ./...

# Run with coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Development Environment

```bash
# Install development tools
make install-dev-tools

# Run linting
make lint

# Format code
make fmt

# Run all checks
make check
```

### Testing

The provider includes comprehensive test coverage:

- **Unit Tests**: 100+ tests covering all core functionality
- **Integration Tests**: End-to-end testing with real file operations
- **Fuzz Tests**: Edge case and stress testing
- **Platform Tests**: Cross-platform compatibility testing

```bash
# Run all tests
make test

# Run specific test suites
go test ./internal/provider -v
go test ./internal/services -v
go test ./internal/errors -v

# Run fuzz tests
go test ./internal/provider -run TestFuzz -timeout 30s

# Run tests with race detection
go test -race ./...
```

## 🏗️ Architecture

The provider is built with a modern, service-oriented architecture:

### Core Components

- **Provider Layer**: Terraform plugin interface and resource management
- **Service Layer**: Business logic with BackupService and TemplateService
- **Platform Layer**: OS-specific operations and abstractions  
- **Git Layer**: Repository management and authentication
- **Validation Layer**: Schema validation and runtime checks
- **Error Handling**: Structured errors with retry logic and context

### Key Design Principles

- **Idempotency**: All operations are safe to repeat
- **Observability**: Comprehensive logging and health checks
- **Performance**: Caching and concurrency control
- **Reliability**: Retry logic and graceful error handling
- **Extensibility**: Plugin architecture for custom functionality

## 🤝 Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

### Areas for Contribution

- **New Template Engines**: Add support for additional template engines
- **Platform Support**: Enhance cross-platform compatibility
- **Application Detection**: Add detection methods for more applications
- **Backup Formats**: Implement additional backup formats
- **Performance**: Optimize file operations and caching
- **Documentation**: Improve examples and guides

## 📄 License

This project is licensed under the Mozilla Public License 2.0 - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [Terraform Plugin Framework](https://github.com/hashicorp/terraform-plugin-framework)
- [go-git](https://github.com/go-git/go-git) for Git operations
- The Terraform community for feedback and contributions

## 📞 Support

- **Documentation**: [Provider Registry](https://registry.terraform.io/providers/jamesainslie/dotfiles)
- **Issues**: [GitHub Issues](https://github.com/jamesainslie/terraform-provider-dotfiles/issues)
- **Discussions**: [GitHub Discussions](https://github.com/jamesainslie/terraform-provider-dotfiles/discussions)

---

**Made with ❤️ for the Infrastructure as Code community**