# Terraform Provider for Dotfiles

[![License: MPL-2.0](https://img.shields.io/badge/License-MPL--2.0-yellow.svg)](https://opensource.org/licenses/MPL-2.0)
[![Go Version](https://img.shields.io/badge/Go-1.25.1-blue.svg)](https://golang.org/)
[![Terraform](https://img.shields.io/badge/Terraform-Provider-orange.svg)](https://terraform.io/)
[![Tests](https://img.shields.io/badge/Tests-100%2B-green.svg)]()
[![Coverage](https://img.shields.io/badge/Coverage-85%25-brightgreen.svg)]()

A focused, production-ready Terraform provider for managing configuration files (dotfiles) in a declarative, cross-platform manner. This provider enables you to version control, deploy, and manage your development environment configuration files using Infrastructure as Code principles.

**🔗 Companion Provider**: For application installation and package management, use [terraform-provider-package](https://github.com/jamesainslie/terraform-provider-package). This separation follows best practices for single-responsibility providers.

## 🏗️ Provider Separation of Concerns

This provider is designed to work alongside `terraform-provider-package` with clear separation of responsibilities:

| Provider | Responsibility | Resources |
|----------|----------------|-----------|
| `terraform-provider-dotfiles` | **Configuration Management** | File operations, Git repositories, templating, backups |
| `terraform-provider-package` | **Application Lifecycle** | Package installation, service management, dependency resolution |

### Example: Complete Setup
```hcl
# Install application using package provider
resource "pkg_package" "vscode" {
  name = "visual-studio-code"
  type = "cask"
}

# Configure application using dotfiles provider
resource "dotfiles_application" "vscode" {
  application_name = "vscode"
  
  config_mappings = {
    "settings.json" = {
      target_path = "~/Library/Application Support/Code/User/settings.json"
      strategy   = "symlink"
    }
  }
  
  depends_on = [pkg_package.vscode]
}
```

## ✨ Key Features

### 🚀 Core Capabilities
- **Cross-platform support**: Works seamlessly on macOS, Linux, and Windows
- **Multiple deployment strategies**: Symlink, copy, or template-based deployment
- **Enterprise-grade backup system**: Automatic backups with compression, retention policies, and validation
- **Multi-engine template processing**: Support for Go templates, Handlebars, and Mustache
- **Advanced Git integration**: Clone, authenticate, and manage repositories with submodule support
- **Configuration file focus**: Dedicated to file operations without application installation concerns
- **Granular permission management**: Fine-grained file and directory permissions with validation
- **Comprehensive dry-run mode**: Preview all changes before applying them
- **Built-in caching**: In-memory caching with TTL and LRU eviction for performance
- **Concurrency control**: Managed parallel operations with configurable limits
- **Enhanced error handling**: Structured errors with retry logic and detailed context

### 🔧 Architectural Improvements
- **Service Layer Architecture**: Modular services for backup, templating, and caching
- **Service Registry**: Centralized service management with health checks
- **Enhanced Git Operations**: Advanced authentication, submodule support, and validation
- **Schema Validators**: Custom validators for paths, file modes, and templates
- **Idempotency Utilities**: Built-in state tracking and comparison
- **Runtime Validation**: Pre-flight checks for directory writability and file existence

## 📦 Installation

### Terraform Registry (Recommended)
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

### Development Build
```bash
git clone https://github.com/jamesainslie/terraform-provider-dotfiles.git
cd terraform-provider-dotfiles
go build -o terraform-provider-dotfiles
```

## 🚀 Quick Start

### 1. Basic Configuration

```hcl
provider "dotfiles" {
  dotfiles_root           = "~/dotfiles"
  backup_enabled          = true
  backup_directory        = "~/.dotfiles-backups"
  strategy               = "symlink"
  conflict_resolution    = "backup"
  template_engine        = "go"
  log_level             = "info"
  dry_run               = false
  
  backup_strategy {
    format          = "timestamped"
    retention_count = 10
    compression     = true
  }
}
```

### 2. Repository Management

```hcl
# Git repository
resource "dotfiles_repository" "main" {
  source_path = "~/dotfiles"
  
  git_config {
    url                    = "https://github.com/user/dotfiles.git"
    branch                = "main"
    personal_access_token = var.github_token
    clone_depth          = 1
    recurse_submodules   = false
  }
}

# Local directory
resource "dotfiles_repository" "local" {
  source_path = "/path/to/local/dotfiles"
}
```

### 3. File Management

```hcl
# Simple file symlink
resource "dotfiles_file" "gitconfig" {
  source_path = "git/gitconfig"
  target_path = "~/.gitconfig"
  file_mode  = "0644"
}

# Template-based configuration
resource "dotfiles_file" "ssh_config" {
  source_path = "ssh/config.template"
  target_path = "~/.ssh/config"
  
  is_template     = true
  template_engine = "go"
  template_vars = {
    hostname = "my-server"
    username = "myuser"
  }
  
  file_mode = "0600"
}
```

### 4. Directory Operations

```hcl
resource "dotfiles_directory" "config" {
  source_path = "config"
  target_path = "~/.config"
  recursive   = true
  
  sync_strategy = "mirror"
  create_parents = true
}
```

### 5. Symlink Management

```hcl
resource "dotfiles_symlink" "fish_config" {
  source_path = "fish"
  target_path = "~/.config/fish"
  create_parents = true
}
```

### 6. Application Configuration Management

**Note**: Application installation should be handled by [terraform-provider-package](https://github.com/jamesainslie/terraform-provider-package). This resource focuses on configuration file management.

```hcl
resource "dotfiles_application" "development_tools" {
  application_name = "neovim"
  
  config_mappings = {
    "nvim/init.lua" = {
      target_path = "~/.config/nvim/init.lua"
      strategy   = "symlink"
    }
    "nvim/lua/" = {
      target_path = "~/.config/nvim/lua/"
      strategy   = "symlink"
    }
  }
}
```

## 📚 Resources

### Core Resources

| Resource | Purpose | Use Case | Strategy Support |
|----------|---------|----------|------------------|
| `dotfiles_repository` | Git/local repository management | Clone and manage dotfiles repositories | N/A |
| `dotfiles_file` | Individual file management | Deploy **single files via copy only** | ❌ No strategy field |
| `dotfiles_symlink` | Symlink creation | Create **single symbolic links** | ❌ Symlinks only |
| `dotfiles_directory` | Directory operations | Sync **entire directories** | ❌ Copy/sync only |
| `dotfiles_application` | Application config mapping | **Multi-file configs with strategy support** | ✅ Full strategy support |

### 🎯 Resource Selection Guide

**Choose the right resource for your use case:**

```hcl
# ✅ Single file copy with templating
resource "dotfiles_file" "gitconfig" {
  source_path = "git/gitconfig" 
  target_path = "~/.gitconfig"
  is_template = true
  # Note: No strategy field - always copies
}

# ✅ Single symlink
resource "dotfiles_symlink" "fish_config" {
  source_path = "fish"
  target_path = "~/.config/fish"
  create_parents = true
}

# ✅ Application configs with multiple strategies
resource "dotfiles_application" "development_tools" {
  application_name = "neovim"
  config_mappings = {
    "nvim/init.lua" = {
      target_path = "~/.config/nvim/init.lua"
      strategy   = "symlink"  # Properly supported
    }
    "nvim/templates/" = {
      target_path = "~/.config/nvim/templates/"
      strategy   = "copy"     # Different strategy
    }
  }
}
```

### ⚠️ Common Mistakes

```hcl
# ❌ DON'T DO THIS - strategy field ignored
resource "dotfiles_file" "wrong_usage" {
  source_path = "config.json"
  target_path = "~/.config/app/config.json"
  strategy   = "symlink"  # THIS IS IGNORED!
}

# ✅ DO THIS INSTEAD
resource "dotfiles_application" "correct_usage" {
  application_name = "myapp"
  config_mappings = {
    "config.json" = {
      target_path = "~/.config/app/config.json"
      strategy   = "symlink"  # This works correctly
    }
  }
}
```

### Data Sources

| Data Source | Purpose | Use Case |
|-------------|---------|----------|
| `dotfiles_system` | System information | Get platform details for conditional logic |
| `dotfiles_file_info` | File metadata | Check file existence and properties |

## 🎯 Advanced Features

### Template Processing

Support for multiple template engines with custom functions:

```hcl
resource "dotfiles_file" "gitconfig" {
  source_path = "git/gitconfig.template"
  target_path = "~/.gitconfig"
  
  is_template     = true
  template_engine = "go"  # or "handlebars", "mustache"
  
  template_vars = {
    user_name  = "John Doe"
    user_email = "john@example.com"
    editor     = "nvim"
  }
  
  platform_template_vars = {
    editor_path = data.dotfiles_system.current.platform == "darwin" ? "/usr/local/bin/nvim" : "/usr/bin/nvim"
  }
}
```

### Backup Management

Comprehensive backup system with multiple formats and retention policies:

```hcl
provider "dotfiles" {
  backup_enabled = true
  
  backup_strategy {
    format          = "timestamped"  # or "versioned", "simple"
    retention_count = 10
    compression     = true
    validation     = true
  }
}
```

### Permission Management

Fine-grained permission control:

```hcl
resource "dotfiles_file" "ssh_key" {
  source_path = "ssh/id_ed25519"
  target_path = "~/.ssh/id_ed25519"
  file_mode  = "0600"  # Secure private key permissions
}

resource "dotfiles_directory" "ssh_dir" {
  source_path = "ssh"
  target_path = "~/.ssh"
  directory_mode = "0700"  # Secure directory permissions
}
```

### Template Path Variables

Application resources support template variables in target paths:

```hcl
resource "dotfiles_application" "vscode" {
  application_name = "vscode"
  
  config_mappings = {
    "settings.json" = {
      target_path = "{{.app_support_dir}}/Code/User/settings.json"
      strategy   = "symlink"
    }
    "snippets/" = {
      target_path = "{{.config_dir}}/Code/User/snippets/"
      strategy   = "copy"
    }
  }
}
```

Available template variables:
- `{{.home_dir}}` - User home directory
- `{{.config_dir}}` - ~/.config directory
- `{{.app_support_dir}}` - ~/Library/Application Support (macOS)
- `{{.application}}` - Application name

## 🔧 Configuration

### Provider Configuration

```hcl
provider "dotfiles" {
  # Required
  dotfiles_root = "~/dotfiles"
  
  # Backup Configuration
  backup_enabled    = true
  backup_directory  = "~/.dotfiles-backups"
  
  # Deployment Strategy
  strategy            = "symlink"  # or "copy", "template"
  conflict_resolution = "backup"   # or "overwrite", "skip"
  
  # Template Engine
  template_engine = "go"  # or "handlebars", "mustache"
  
  # Behavior
  dry_run   = false
  log_level = "info"  # debug, info, warn, error
  
  # Enhanced Backup Strategy
  backup_strategy {
    format          = "timestamped"
    retention_count = 10
    compression     = true
    validation     = true
  }
}
```

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `DOTFILES_ROOT` | Root directory for dotfiles | `~/dotfiles` |
| `DOTFILES_BACKUP_DIR` | Backup directory | `~/.dotfiles-backups` |
| `DOTFILES_DRY_RUN` | Enable dry-run mode | `false` |
| `DOTFILES_LOG_LEVEL` | Logging level | `info` |
| `DOTFILES_GIT_TOKEN` | Git personal access token | - |

## 🔍 Data Sources

### System Information

```hcl
data "dotfiles_system" "current" {}

output "platform_info" {
  value = {
    platform      = data.dotfiles_system.current.platform
    architecture  = data.dotfiles_system.current.architecture
    home_dir     = data.dotfiles_system.current.home_directory
    config_dir   = data.dotfiles_system.current.config_directory
  }
}
```

### File Information

```hcl
data "dotfiles_file_info" "check_config" {
  file_path = "~/.vimrc"
}

output "file_exists" {
  value = data.dotfiles_file_info.check_config.exists
}
```

## 🧪 Testing

### Unit Tests
```bash
go test ./internal/... -v
```

### Integration Tests
```bash
go test ./internal/provider -v -run TestIntegration
```

### Acceptance Tests
```bash
TF_ACC=1 go test ./internal/provider -v -run TestAcc
```

### Coverage Report
```bash
go test ./internal/... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## 🛠️ Development

### Prerequisites
- Go 1.25.1+
- Terraform 1.5+
- Git 2.30+

### Building
```bash
git clone https://github.com/jamesainslie/terraform-provider-dotfiles.git
cd terraform-provider-dotfiles
go build -o terraform-provider-dotfiles
```

### Development Override
```hcl
# ~/.terraformrc
provider_installation {
  dev_overrides {
    "jamesainslie/dotfiles" = "/path/to/terraform-provider-dotfiles"
  }
  direct {}
}
```

### Code Quality
```bash
# Run linting
golangci-lint run

# Format code
go fmt ./...

# Run all checks
make ci
```

## 📖 Examples

### Complete Development Environment

```hcl
terraform {
  required_providers {
    pkg = {
      source = "jamesainslie/package"
    }
    dotfiles = {
      source = "jamesainslie/dotfiles"
    }
  }
}

# Install applications
resource "pkg_package" "development_tools" {
  for_each = {
    "neovim" = { name = "neovim", type = "formula" }
    "vscode" = { name = "visual-studio-code", type = "cask" }
    "fish"   = { name = "fish", type = "formula" }
  }
  
  name = each.value.name
  type = each.value.type
}

# Configure applications
resource "dotfiles_application" "development_configs" {
  for_each = {
    "neovim" = {
      "init.lua" = {
        target_path = "~/.config/nvim/init.lua"
        strategy   = "symlink"
      }
    }
    "vscode" = {
      "settings.json" = {
        target_path = "~/Library/Application Support/Code/User/settings.json"
        strategy   = "copy"
      }
    }
    "fish" = {
      "config.fish" = {
        target_path = "~/.config/fish/config.fish"
        strategy   = "symlink"
      }
    }
  }
  
  application_name = each.key
  config_mappings = each.value
  
  depends_on = [pkg_package.development_tools]
}
```

### Cross-Platform Configuration

```hcl
data "dotfiles_system" "current" {}

resource "dotfiles_file" "shell_config" {
  source_path = "shell/${data.dotfiles_system.current.platform}/config"
  target_path = data.dotfiles_system.current.platform == "windows" ? 
    "~/AppData/Roaming/shell/config" : 
    "~/.config/shell/config"
}
```

## 🔄 Migration Guide

### From v0.x to v1.0

See [MIGRATION_GUIDE.md](docs/MIGRATION_GUIDE.md) for detailed migration instructions.

**Breaking Changes:**
- Application detection removed (use terraform-provider-package)
- Simplified `dotfiles_application` resource schema
- Updated provider configuration format

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Development Guidelines
- Follow Go conventions and best practices
- Add tests for new functionality
- Update documentation
- Run `make ci` before submitting

## 📄 License

This project is licensed under the MPL-2.0 License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [Terraform Plugin Framework](https://github.com/hashicorp/terraform-plugin-framework)
- [go-git](https://github.com/go-git/go-git) for Git operations
- The Terraform community for feedback and contributions

## 📞 Support

- 📖 [Documentation](https://registry.terraform.io/providers/jamesainslie/dotfiles/latest/docs)
- 🐛 [Issue Tracker](https://github.com/jamesainslie/terraform-provider-dotfiles/issues)
- 💬 [Discussions](https://github.com/jamesainslie/terraform-provider-dotfiles/discussions)
- 📧 [Email Support](mailto:support@example.com)

---

**Made with ❤️ for the Terraform community**