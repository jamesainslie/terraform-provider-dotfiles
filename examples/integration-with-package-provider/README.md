# Complete Development Environment with Provider Integration

This example demonstrates how to set up a complete development environment using both `terraform-provider-dotfiles` and `terraform-provider-package` with proper separation of concerns.

## Overview

This configuration showcases:

- **Package Installation**: Using `terraform-provider-package` for application lifecycle management
- **Configuration Management**: Using `terraform-provider-dotfiles` for dotfiles and configuration
- **Proper Dependencies**: Ensuring applications are installed before configuration
- **Cross-Platform Support**: Using system data sources and template variables
- **Template Processing**: Dynamic configuration generation
- **Backup Management**: Comprehensive backup strategies

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Development Environment                   │
├─────────────────────────────────────────────────────────────┤
│  terraform-provider-package    │  terraform-provider-dotfiles │
│  ─────────────────────────────  │  ─────────────────────────── │
│  • Install applications        │  • Manage config files       │
│  • Manage services             │  • Process templates          │
│  • Handle dependencies         │  • Create symlinks           │
│  • Version management          │  • Backup configurations     │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
                    ┌─────────────────────┐
                    │   Complete Setup    │
                    │   ───────────────   │
                    │   Applications +    │
                    │   Configurations    │
                    └─────────────────────┘
```

## Quick Start

### 1. Prerequisites

- Terraform 1.5+
- Homebrew (macOS) or appropriate package manager
- Git (for dotfiles repository)

### 2. Setup

```bash
# Clone this example
git clone https://github.com/jamesainslie/terraform-provider-dotfiles.git
cd terraform-provider-dotfiles/examples/integration-with-package-provider

# Copy and customize variables
cp terraform.tfvars.example terraform.tfvars
# Edit terraform.tfvars with your preferences

# Initialize Terraform
terraform init

# Plan the deployment
terraform plan

# Apply the configuration
terraform apply
```

### 3. Customize Your Setup

Edit `terraform.tfvars` to match your preferences:

```hcl
# Your dotfiles repository
git_repository_url = "https://github.com/yourusername/dotfiles.git"

# Add/remove packages
development_packages = {
  "your-app" = {
    package_name = "your-application"
    package_type = "cask"
  }
}

# Configure application mappings
application_configs = {
  "your-app" = {
    config_mappings = {
      "config.json" = {
        target_path = "~/.config/your-app/config.json"
        strategy   = "symlink"
      }
    }
  }
}
```

## What Gets Installed and Configured

### Applications (via terraform-provider-package)

| Category | Applications |
|----------|-------------|
| **Shell & Terminal** | Fish, Starship, Alacritty |
| **Editors** | Neovim, VS Code, Cursor |
| **Development** | Git, GitHub CLI, Terraform, Docker |
| **Languages** | Go, Node.js, Python, Rust |
| **Utilities** | jq, yq, ripgrep, fd, bat, exa |

### Configurations (via terraform-provider-dotfiles)

| Application | Configuration Files |
|-------------|-------------------|
| **VS Code** | settings.json, keybindings.json, snippets/ |
| **Neovim** | init.lua, lua/, after/ |
| **Git** | .gitconfig, .gitignore_global |
| **Fish Shell** | config.fish, functions/, completions/ |
| **SSH** | config (templated) |
| **Starship** | starship.toml |

## Key Features Demonstrated

### 1. Provider Separation

```hcl
# Package installation
resource "pkg_package" "development_tools" {
  for_each = var.development_packages
  name = each.value.package_name
  type = each.value.package_type
}

# Configuration management  
resource "dotfiles_application" "development_configs" {
  for_each = var.application_configs
  application_name = each.key
  config_mappings = each.value.config_mappings
  
  depends_on = [pkg_package.development_tools]  # Proper dependency
}
```

### 2. Template Processing

```hcl
resource "dotfiles_file" "ssh_config" {
  source_path     = "ssh/config.template"
  target_path     = "~/.ssh/config"
  is_template     = true
  template_engine = "go"
  template_vars = {
    hostname = "my-server.example.com"
    username = "myuser"
    port     = "22"
  }
  file_mode = "0600"
}
```

### 3. Cross-Platform Support

```hcl
data "dotfiles_system" "current" {}

# Use template variables for platform-specific paths
resource "dotfiles_application" "vscode" {
  config_mappings = {
    "settings.json" = {
      target_path = "{{.app_support_dir}}/Code/User/settings.json"
      strategy   = "symlink"
    }
  }
}
```

### 4. Backup Management

```hcl
provider "dotfiles" {
  backup_enabled = true
  
  backup_strategy {
    format          = "timestamped"
    retention_count = 10
    compression     = true
    validation     = true
  }
}
```

## Directory Structure Expected

Your dotfiles repository should follow this structure:

```
~/dotfiles/
├── fish/
│   ├── config.fish
│   ├── functions/
│   └── completions/
├── nvim/
│   ├── init.lua
│   ├── lua/
│   └── after/
├── vscode/
│   ├── settings.json
│   ├── keybindings.json
│   └── snippets/
├── git/
│   ├── gitconfig
│   └── gitignore_global
├── ssh/
│   └── config.template
├── starship/
│   └── starship.toml
└── bin/
    └── your-scripts
```

## Template Examples

### SSH Config Template (`ssh/config.template`)

```bash
# SSH Configuration Template
Host {{.hostname}}
    HostName {{.hostname}}
    User {{.username}}
    Port {{.port}}
    IdentityFile ~/.ssh/id_ed25519
    IdentitiesOnly yes
```

### Git Config Template (`git/gitconfig.template`)

```ini
[user]
    name = {{.user_name}}
    email = {{.user_email}}
    
[core]
    editor = {{.editor}}
    excludesfile = ~/.gitignore_global
    
[init]
    defaultBranch = main
```

## Customization Options

### Add New Applications

1. **Add to package list**:
```hcl
development_packages = {
  "new-app" = {
    package_name = "new-application"
    package_type = "cask"
  }
}
```

2. **Add configuration mapping**:
```hcl
application_configs = {
  "new-app" = {
    config_mappings = {
      "new-app/config.json" = {
        target_path = "~/.config/new-app/config.json"
        strategy   = "symlink"
      }
    }
  }
}
```

### Platform-Specific Configurations

```hcl
# Use system data for conditional logic
locals {
  is_macos = data.dotfiles_system.current.platform == "darwin"
  is_linux = data.dotfiles_system.current.platform == "linux"
}

resource "dotfiles_file" "platform_config" {
  source_path = local.is_macos ? "macos/config" : "linux/config"
  target_path = "~/.config/app/config"
}
```

## Troubleshooting

### Common Issues

1. **Package not found**:
   ```bash
   # Search for correct package name
   brew search your-app-name
   ```

2. **Configuration not applied**:
   ```bash
   # Check if application was installed first
   terraform state show pkg_package.your-app
   ```

3. **Template errors**:
   ```bash
   # Validate template syntax
   terraform plan  # Will show template processing errors
   ```

### Debugging

```bash
# Enable debug logging
export TF_LOG=DEBUG
terraform apply

# Check provider logs
export DOTFILES_LOG_LEVEL=debug
terraform apply
```

## Advanced Features

### Conditional Configuration

```hcl
# Only configure if application is in package list
resource "dotfiles_application" "conditional_app" {
  count = contains(keys(var.development_packages), "optional-app") ? 1 : 0
  
  application_name = "optional-app"
  config_mappings = {
    "config.json" = {
      target_path = "~/.config/optional-app/config.json"
      strategy   = "symlink"
    }
  }
}
```

### Dynamic Template Variables

```hcl
locals {
  template_vars = merge(var.template_variables, {
    platform = data.dotfiles_system.current.platform
    arch     = data.dotfiles_system.current.architecture
    home     = data.dotfiles_system.current.home_directory
  })
}
```

### Backup Verification

```hcl
# Check if backups were created
data "dotfiles_file_info" "backup_check" {
  file_path = "~/.dotfiles-backups"
}

output "backup_directory_exists" {
  value = data.dotfiles_file_info.backup_check.exists
}
```

## Integration with CI/CD

### GitHub Actions Example

```yaml
name: Deploy Dotfiles
on: [push]
jobs:
  deploy:
    runs-on: macos-latest
    steps:
      - uses: actions/checkout@v3
      - uses: hashicorp/setup-terraform@v2
      - name: Deploy
        run: |
          terraform init
          terraform apply -auto-approve
        env:
          TF_VAR_github_token: ${{ secrets.GITHUB_TOKEN }}
```

## Next Steps

1. **Customize** the configuration for your specific needs
2. **Add** more applications and configurations
3. **Create** templates for dynamic content
4. **Set up** automated deployment
5. **Contribute** improvements back to the community

## Support

- [Provider Documentation](https://registry.terraform.io/providers/jamesainslie/dotfiles/latest/docs)
- [GitHub Issues](https://github.com/jamesainslie/terraform-provider-dotfiles/issues)
- [terraform-provider-package](https://github.com/jamesainslie/terraform-provider-package)

---

**Happy configuring!** 🚀
