# Terraform Provider Dotfiles - Examples

This directory contains comprehensive examples demonstrating various use cases and integration patterns for the terraform-provider-dotfiles.

##  Provider Separation of Concerns

The terraform-provider-dotfiles focuses solely on **configuration file management**. For application installation and lifecycle management, use [terraform-provider-package](https://github.com/jamesainslie/terraform-provider-package).

| Provider | Responsibility | Examples |
|----------|----------------|----------|
| `terraform-provider-dotfiles` | Configuration Management | File operations, Git repos, templates, backups |
| `terraform-provider-package` | Application Lifecycle | Package install/remove, service management |

##  Available Examples

###  [Integration with Package Provider](./integration-with-package-provider/)
**Complete development environment setup showing proper separation of concerns**

- Package installation using terraform-provider-package
- Configuration management using terraform-provider-dotfiles  
- Proper dependency management with `depends_on`
- Cross-platform support and template processing
- Comprehensive backup strategies

**Use this example when:** Setting up a complete development environment with both application installation and configuration management.

###  [Basic Setup](./basic-setup/)
**Simple dotfiles management without external dependencies**

- Local dotfiles repository management
- Basic file and symlink operations
- Minimal provider configuration

**Use this example when:** Getting started with simple dotfiles management or working with local-only configurations.

###  [Complete Environment](./complete-environment/)
**Advanced configuration with all provider features**

- Git repository integration
- Template processing with multiple engines
- Advanced backup configurations
- Permission management
- Cross-platform considerations

**Use this example when:** You need comprehensive dotfiles management with advanced features like templating and Git integration.

###  [GitHub Repository](./github-repository/)
**Git-based dotfiles management with authentication**

- GitHub repository cloning
- SSH and token authentication
- Branch and submodule management
- Automated Git operations

**Use this example when:** Your dotfiles are stored in a Git repository and you need automated cloning and synchronization.

###  [Data Sources](./data-sources/)
**Using provider data sources for conditional logic**

- System information gathering
- File existence checking
- Platform-specific configurations
- Dynamic template variables

**Use this example when:** You need conditional logic based on system information or file existence.

##  Quick Start Guide

### 1. Choose Your Use Case

| Scenario | Recommended Example |
|----------|-------------------|
| **Complete dev environment** | [Integration with Package Provider](./integration-with-package-provider/) |
| **Simple dotfiles only** | [Basic Setup](./basic-setup/) |
| **Git-based dotfiles** | [GitHub Repository](./github-repository/) |
| **Advanced features** | [Complete Environment](./complete-environment/) |

### 2. Copy and Customize

```bash
# Choose an example
cd examples/integration-with-package-provider

# Copy configuration
cp terraform.tfvars.example terraform.tfvars

# Edit with your preferences
vim terraform.tfvars

# Initialize and apply
terraform init
terraform apply
```

### 3. Adapt to Your Needs

Each example is designed to be a starting point. Customize:

- Package lists and application configurations
- Template variables and processing
- Backup strategies and retention policies
- Platform-specific logic

##  Common Patterns

### Application Installation + Configuration

```hcl
# Install application (terraform-provider-package)
resource "pkg_package" "vscode" {
  name = "visual-studio-code"
  type = "cask"
}

# Configure application (terraform-provider-dotfiles)
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

### Template Processing

```hcl
resource "dotfiles_file" "gitconfig" {
  source_path = "git/gitconfig.template"
  target_path = "~/.gitconfig"
  
  is_template     = true
  template_engine = "go"
  template_vars = {
    user_name  = "John Doe"
    user_email = "john@example.com"
    editor     = "nvim"
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

### Conditional Resources

```hcl
resource "dotfiles_application" "optional_app" {
  count = var.install_optional_apps ? 1 : 0
  
  application_name = "optional-tool"
  config_mappings = {
    "config.json" = {
      target_path = "~/.config/optional-tool/config.json"
      strategy   = "symlink"
    }
  }
}
```

##  Configuration Patterns

### Provider Configuration

```hcl
provider "dotfiles" {
  dotfiles_root           = "~/dotfiles"
  backup_enabled          = true
  backup_directory        = "~/.dotfiles-backups"
  strategy               = "symlink"
  conflict_resolution    = "backup"
  template_engine        = "go"
  
  backup_strategy {
    format          = "timestamped"
    retention_count = 10
    compression     = true
    validation     = true
  }
}
```

### Variable Organization

```hcl
# Group related configurations
variable "development_packages" {
  description = "Development tools to install"
  type = map(object({
    package_name = string
    package_type = string
  }))
}

variable "application_configs" {
  description = "Application configuration mappings"
  type = map(object({
    config_mappings = map(object({
      target_path = string
      strategy   = string
    }))
  }))
}
```

##  Best Practices

### 1. Separation of Concerns
- Use terraform-provider-package for installation
- Use terraform-provider-dotfiles for configuration
- Maintain clear dependencies with `depends_on`

### 2. Template Organization
- Keep templates in dedicated directories
- Use meaningful variable names
- Validate template syntax before deployment

### 3. Backup Strategy
- Enable automatic backups for important configurations
- Set appropriate retention policies
- Test restore procedures regularly

### 4. Cross-Platform Support
- Use system data sources for platform detection
- Organize platform-specific configurations clearly
- Test on multiple platforms when possible

### 5. Version Control
- Store Terraform configurations in version control
- Use meaningful commit messages
- Tag releases for stable configurations

##  Troubleshooting

### Common Issues

1. **Application not configured**:
   - Check if package was installed first
   - Verify `depends_on` relationships
   - Confirm application is in package list

2. **Template processing errors**:
   - Validate template syntax
   - Check variable definitions
   - Ensure template engine is correct

3. **Permission issues**:
   - Verify file modes are correct
   - Check directory permissions
   - Ensure backup directory is writable

4. **Cross-platform issues**:
   - Test platform detection logic
   - Verify path formats for each OS
   - Check application installation paths

### Debugging Commands

```bash
# Enable debug logging
export TF_LOG=DEBUG
export DOTFILES_LOG_LEVEL=debug

# Plan with detailed output
terraform plan -detailed-exitcode

# Apply with logging
terraform apply

# Check resource state
terraform state show resource.name

# Validate configuration
terraform validate
```

##  Additional Resources

### Documentation
- [Provider Registry Docs](https://registry.terraform.io/providers/jamesainslie/dotfiles/latest/docs)
- [Schema Documentation](../docs/SCHEMA.md)
- [Migration Guide](../docs/MIGRATION_GUIDE.md)

### Related Providers
- [terraform-provider-package](https://github.com/jamesainslie/terraform-provider-package)
- [Terraform Plugin Framework](https://github.com/hashicorp/terraform-plugin-framework)

### Community
- [GitHub Issues](https://github.com/jamesainslie/terraform-provider-dotfiles/issues)
- [GitHub Discussions](https://github.com/jamesainslie/terraform-provider-dotfiles/discussions)

##  Contributing

Found an issue with an example or have a suggestion for improvement?

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

### Example Contribution Guidelines

- **New Examples**: Should demonstrate a specific use case or integration pattern
- **Documentation**: Keep README files updated with changes
- **Testing**: Ensure examples work on multiple platforms when possible
- **Variables**: Use clear, descriptive variable names and documentation

---

**Ready to get started?** Choose an example that matches your use case and start customizing! 