# Terraform Provider for Dotfiles

[![License: MPL-2.0](https://img.shields.io/badge/License-MPL--2.0-yellow.svg)](https://opensource.org/licenses/MPL-2.0)
[![Go Version](https://img.shields.io/badge/Go-1.25.1-blue.svg)](https://golang.org/)
[![Terraform](https://img.shields.io/badge/Terraform-Provider-orange.svg)](https://terraform.io/)

A comprehensive Terraform provider for managing dotfiles in a declarative, cross-platform manner. This provider enables you to version control, deploy, and manage your development environment configuration files using Infrastructure as Code principles.

## Features

### 🚀 Core Capabilities
- **Cross-platform support**: Works on macOS, Linux, and Windows
- **Multiple deployment strategies**: Symlink, copy, or template-based deployment
- **Automatic backups**: Built-in backup system with configurable retention policies
- **Template engine**: Support for Go templates and Handlebars
- **Git integration**: Clone and manage dotfiles from Git repositories
- **Application detection**: Smart detection of installed applications
- **Permission management**: Fine-grained file and directory permissions
- **Dry-run mode**: Preview changes before applying them

### 📦 Resources
- **`dotfiles_repository`**: Manage dotfiles repositories (local or Git-based)
- **`dotfiles_file`**: Deploy individual configuration files
- **`dotfiles_symlink`**: Create symbolic links to configuration directories
- **`dotfiles_directory`**: Manage directory structures and their contents
- **`dotfiles_application`**: Application-specific configuration management

### 📊 Data Sources
- **`dotfiles_system`**: Get system information (platform, architecture, paths)
- **`dotfiles_file_info`**: Inspect file properties and metadata

## Installation

### Using Terraform Registry (Recommended)

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

### Manual Installation

1. Download the latest release from the [releases page](https://github.com/jamesainslie/terraform-provider-dotfiles/releases)
2. Place the binary in your Terraform plugins directory
3. Configure the provider in your Terraform files

## Quick Start

### Basic Configuration

```hcl
terraform {
  required_providers {
    dotfiles = {
      source = "jamesainslie/dotfiles"
    }
  }
}

provider "dotfiles" {
  dotfiles_root    = "~/dotfiles"
  backup_enabled   = true
  backup_directory = "~/.dotfiles-backups"
  strategy         = "symlink"
  dry_run          = false
}

# Get system information
data "dotfiles_system" "current" {}

# Create a repository
resource "dotfiles_repository" "main" {
  name                   = "personal-dotfiles"
  source_path            = "~/dotfiles"
  description            = "Personal development environment dotfiles"
  default_backup_enabled = true
  default_file_mode      = "0644"
  default_dir_mode       = "0755"
}

# Deploy a configuration file
resource "dotfiles_file" "gitconfig" {
  repository  = dotfiles_repository.main.id
  name        = "git-config"
  source_path = "git/gitconfig"
  target_path = "~/.gitconfig"
  is_template = false
  file_mode   = "0644"
}

# Create a symbolic link
resource "dotfiles_symlink" "fish_config" {
  repository     = dotfiles_repository.main.id
  name           = "fish-configuration"
  source_path    = "fish"
  target_path    = "~/.config/fish"
  force_update   = false
  create_parents = true
}
```

### Template-based Configuration

```hcl
# Process a template file with variables
resource "dotfiles_file" "gitconfig_template" {
  repository  = dotfiles_repository.main.id
  name        = "git-configuration"
  source_path = "templates/gitconfig.template"
  target_path = "~/.gitconfig"
  is_template = true
  file_mode   = "0644"

  template_vars = {
    user_name  = "John Doe"
    user_email = "john@example.com"
    editor     = "vim"
    gpg_key    = "ABC123DEF456"
  }
}
```

### Git Repository Integration

```hcl
resource "dotfiles_repository" "remote" {
  name        = "remote-dotfiles"
  source_path = "https://github.com/username/dotfiles.git"
  description = "Remote dotfiles repository"
  
  git_branch              = "main"
  git_personal_access_token = var.github_token
  git_update_interval     = "24h"
}
```

## Provider Configuration

### Required Configuration

```hcl
provider "dotfiles" {
  # Basic configuration
  dotfiles_root    = "~/dotfiles"           # Root directory for dotfiles
  backup_enabled   = true                   # Enable automatic backups
  backup_directory = "~/.dotfiles-backups"  # Backup storage location
  strategy         = "symlink"              # Default deployment strategy
}
```

### Advanced Configuration

```hcl
provider "dotfiles" {
  # Basic settings
  dotfiles_root    = "~/dotfiles"
  backup_enabled   = true
  backup_directory = "~/.dotfiles-backups"
  strategy         = "symlink"
  
  # Advanced settings
  conflict_resolution = "backup"      # How to handle conflicts
  dry_run            = false          # Preview mode
  auto_detect_platform = true         # Auto-detect target platform
  target_platform    = "auto"         # Target platform override
  template_engine    = "go"           # Template engine
  log_level          = "info"         # Logging level
  
  # Enhanced backup strategy
  backup_strategy {
    enabled         = true
    directory       = "~/.dotfiles-backups"
    compression     = true
    incremental     = true
    max_backups     = 10
    retention_policy = "30d"
  }
  
  # Recovery configuration
  recovery {
    create_restore_scripts = true
    validate_backups      = true
    backup_index         = true
    test_recovery        = false
  }
}
```

## Deployment Strategies

### 1. Symlink (Default)
Creates symbolic links from your dotfiles repository to the target locations.

```hcl
resource "dotfiles_symlink" "config" {
  repository     = dotfiles_repository.main.id
  name           = "app-config"
  source_path    = "apps/myapp"
  target_path    = "~/.config/myapp"
  create_parents = true
}
```

### 2. Copy
Copies files from the repository to target locations.

```hcl
resource "dotfiles_file" "config" {
  repository  = dotfiles_repository.main.id
  name        = "app-config"
  source_path = "apps/myapp/config.json"
  target_path = "~/.config/myapp/config.json"
  file_mode   = "0644"
}
```

### 3. Template
Processes template files with variables before deployment.

```hcl
resource "dotfiles_file" "config_template" {
  repository  = dotfiles_repository.main.id
  name        = "app-config-template"
  source_path = "templates/myapp.conf.template"
  target_path = "~/.config/myapp/myapp.conf"
  is_template = true
  
  template_vars = {
    username = "john"
    theme    = "dark"
  }
}
```

## Application Detection

The provider can automatically detect installed applications and conditionally deploy configurations:

```hcl
resource "dotfiles_application" "vscode" {
  repository           = dotfiles_repository.main.id
  application          = "vscode"
  source_path          = "vscode"
  detect_installation  = true
  skip_if_not_installed = true
  warn_if_not_installed = true
  
  detection_methods {
    method = "command"
    command = "code --version"
  }
  
  detection_methods {
    method = "path"
    path   = "/Applications/Visual Studio Code.app"
  }
}
```

## Examples

Check out the [examples directory](./examples/) for comprehensive usage examples:

- **[Basic Setup](./examples/basic-setup/)**: Simple dotfiles management
- **[Complete Environment](./examples/complete-environment/)**: Full development environment setup
- **[GitHub Repository](./examples/github-repository/)**: Remote repository integration

## Development

### Prerequisites

- Go 1.25.1 or later
- Terraform 1.0 or later
- Git

### Building from Source

```bash
# Clone the repository
git clone https://github.com/jamesainslie/terraform-provider-dotfiles.git
cd terraform-provider-dotfiles

# Build the provider
make build

# Run tests
make test

# Generate documentation
make docs
```

### Testing

```bash
# Run unit tests
go test ./...

# Run integration tests
make test-integration

# Run acceptance tests
make test-acceptance
```

## Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

### Development Workflow

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Ensure all tests pass
6. Submit a pull request

## License

This project is licensed under the Mozilla Public License 2.0. See the [LICENSE](LICENSE) file for details.

## Support

- **Documentation**: [Provider Documentation](https://registry.terraform.io/providers/jamesainslie/dotfiles/latest/docs)
- **Issues**: [GitHub Issues](https://github.com/jamesainslie/terraform-provider-dotfiles/issues)
- **Discussions**: [GitHub Discussions](https://github.com/jamesainslie/terraform-provider-dotfiles/discussions)

## Roadmap

- [ ] Enhanced template engine support (Handlebars, Jinja2)
- [ ] Cloud storage integration (S3, GCS, Azure Blob)
- [ ] Configuration validation and linting
- [ ] Multi-environment support
- [ ] Plugin system for custom strategies
- [ ] Web UI for configuration management

## Changelog

See [CHANGELOG.md](CHANGELOG.md) for a detailed list of changes and version history.

---

**Made with ❤️ for the developer community**
