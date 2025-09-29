# Terraform Provider for Dotfiles

[![License: MPL-2.0](https://img.shields.io/badge/License-MPL--2.0-yellow.svg)](https://opensource.org/licenses/MPL-2.0)
[![Go Version](https://img.shields.io/badge/Go-1.25.1-blue.svg)](https://golang.org/)
[![Terraform](https://img.shields.io/badge/Terraform-Provider-orange.svg)](https://terraform.io/)
[![Tests](https://img.shields.io/badge/Tests-100%2B-green.svg)]()
[![Coverage](https://img.shields.io/badge/Coverage-85%25-brightgreen.svg)]()

A Terraform provider for managing configuration files (dotfiles) in a declarative, cross-platform manner. This provider enables you to version control, deploy, and manage your development environment configuration files using Infrastructure as Code principles.

**Note**: This is the initial stable release (v0.1.x) ready for production use. Advanced features and enhancements are planned for future releases.

##  Key Features

### Core Capabilities
- **Cross-platform support**: Works on macOS, Linux, and Windows
- **Multiple deployment strategies**: Symlink, copy, or template-based deployment
- **Git integration**: Clone and manage dotfiles repositories
- **Template processing**: Support for Go templates, Handlebars, and Mustache
- **Backup system**: Automatic backups with multiple formats and retention policies
- **Dry-run mode**: Preview changes before applying them
- **Comprehensive validation**: Path, file mode, and template validation

##  Installation

### Terraform Registry (Recommended)
```hcl
terraform {
  required_providers {
    dotfiles = {
      source  = "jamesainslie/dotfiles"
      version = "~> 0.1"
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

##  Quick Start

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
  auto_detect_platform   = true
  target_platform        = "auto"
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

##  Resources

### Core Resources

| Resource | Purpose | Use Case |
|----------|---------|----------|
| `dotfiles_repository` | Git/local repository management | Clone and manage dotfiles repositories |
| `dotfiles_file` | Individual file management | Copy single files (with templating support) |
| `dotfiles_symlink` | Symlink creation | Create symbolic links to files or directories |
| `dotfiles_directory` | Directory operations | Sync entire directories recursively |
| `dotfiles_application` | Application config mapping | Multi-file configs with strategy selection |

###  Resource Selection Guide

**Choose the right resource for your use case:**

```hcl
# Single file copy with templating
resource "dotfiles_file" "gitconfig" {
  source_path = "git/gitconfig" 
  target_path = "~/.gitconfig"
  is_template = true
  file_mode   = "0644"
}

# Single symlink
resource "dotfiles_symlink" "fish_config" {
  source_path    = "fish"
  target_path    = "~/.config/fish"
  create_parents = true
}

# Directory operations
resource "dotfiles_directory" "tools_config" {
  source_path = "tools"
  target_path = "~/.config/tools"
  recursive   = true
}

# Application configs with multiple strategies
resource "dotfiles_application" "development_tools" {
  application_name = "neovim"
  config_mappings = {
    "nvim/init.lua" = {
      target_path = "~/.config/nvim/init.lua"
      strategy   = "symlink"
    }
    "nvim/templates/" = {
      target_path = "~/.config/nvim/templates/"
      strategy   = "copy"
    }
  }
}
```

###  Resource Selection Guidelines

- **Use `dotfiles_file`** for single file copy operations (with optional templating)
- **Use `dotfiles_symlink`** for creating symbolic links to files or directories
- **Use `dotfiles_directory`** for recursive directory synchronization
- **Use `dotfiles_application`** when you need different strategies for different files

### Data Sources

| Data Source | Purpose | Use Case |
|-------------|---------|----------|
| `dotfiles_system` | System information | Get platform details for conditional logic |
| `dotfiles_file_info` | File metadata | Check file existence and properties |

##  Advanced Features

### Template Processing

Support for multiple template engines:

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
}
```

### Backup Management

Automatic backup system:

```hcl
provider "dotfiles" {
  backup_enabled   = true
  backup_directory = "~/.dotfiles-backups"
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

### Cross-Platform Support

Use system data source for platform-specific configurations:

```hcl
data "dotfiles_system" "current" {}

resource "dotfiles_file" "shell_config" {
  source_path = "shell/${data.dotfiles_system.current.platform}/config"
  target_path = data.dotfiles_system.current.platform == "windows" ? 
    "~/AppData/Roaming/shell/config" : 
    "~/.config/shell/config"
}
```

##  Configuration

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
  
  # Platform Detection
  auto_detect_platform = true
  target_platform      = "auto"  # or "macos", "linux", "windows"
  
  # Behavior
  dry_run   = false
  log_level = "info"  # debug, info, warn, error
}
```

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `DOTFILES_ROOT` | Root directory for dotfiles | `~/dotfiles` |
| `DOTFILES_BACKUP_DIR` | Backup directory | `~/.dotfiles-backups` |
| `DOTFILES_DRY_RUN` | Enable dry-run mode | `false` |
| `DOTFILES_LOG_LEVEL` | Logging level | `info` |

##  Data Sources

### System Information

```hcl
data "dotfiles_system" "current" {}

output "platform_info" {
  value = {
    platform      = data.dotfiles_system.current.platform
    architecture  = data.dotfiles_system.current.architecture
    home_dir     = data.dotfiles_system.current.home_dir
    config_dir   = data.dotfiles_system.current.config_dir
  }
}
```

### File Information

```hcl
data "dotfiles_file_info" "check_config" {
  path = "~/.vimrc"
}

output "file_exists" {
  value = data.dotfiles_file_info.check_config.exists
}
```

##  Testing

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

##  Development

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

##  Examples

### Complete Development Environment

```hcl
terraform {
  required_providers {
    dotfiles = {
      source  = "jamesainslie/dotfiles"
      version = "~> 0.1"
    }
  }
}

provider "dotfiles" {
  dotfiles_root = "~/dotfiles"
}

# Repository management
resource "dotfiles_repository" "main" {
  source_path = "~/dotfiles"
  
  git_config {
    url    = "https://github.com/user/dotfiles.git"
    branch = "main"
  }
}

# Configure applications
resource "dotfiles_application" "development_configs" {
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

##  Development Status

This provider is production-ready (v0.1.x). Current status:

**Implemented Features:**
- ✅ Core resources (file, symlink, directory, application, repository)
- ✅ Multiple template engines (Go, Handlebars, Mustache)
- ✅ Cross-platform support (macOS, Linux, Windows)
- ✅ Basic backup system
- ✅ Git repository management
- ✅ System and file info data sources

**In Development:**
- 🚧 Enhanced backup strategies with compression and retention
- 🚧 Advanced Git authentication methods
- 🚧 Service registry and caching improvements

##  Contributing

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

##  License

This project is licensed under the MPL-2.0 License - see the [LICENSE](LICENSE) file for details.

##  Acknowledgments

- [Terraform Plugin Framework](https://github.com/hashicorp/terraform-plugin-framework)
- [go-git](https://github.com/go-git/go-git) for Git operations
- The Terraform community for feedback and contributions

##  Support

- [Documentation](https://registry.terraform.io/providers/jamesainslie/dotfiles/latest/docs)
- [Issue Tracker](https://github.com/jamesainslie/terraform-provider-dotfiles/issues)
- [Discussions](https://github.com/jamesainslie/terraform-provider-dotfiles/discussions)

---

**Made with ❤️ for the Terraform community**