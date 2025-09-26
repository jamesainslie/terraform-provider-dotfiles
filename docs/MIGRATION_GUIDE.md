# Migration Guide: Upgrading to v1.0

This guide helps you migrate from earlier versions of the terraform-provider-dotfiles to v1.0, which includes significant architectural improvements and new features.

## Overview of Changes

### Major Improvements in v1.0

- **Service Layer Architecture**: New modular design with BackupService and TemplateService
- **Enhanced Error Handling**: Structured errors with retry logic and detailed context
- **Advanced Backup System**: Compression, retention policies, and multiple formats
- **Multi-Engine Templates**: Support for Go templates, Handlebars, and Mustache
- **Runtime Validation**: Pre-flight checks for paths, permissions, and dependencies
- **Caching and Performance**: In-memory caching with configurable TTL and concurrency control
- **Git Enhancements**: Submodule support, enhanced authentication, and repository validation

### Breaking Changes

#### 1. Provider Configuration Schema

**Before (v0.x):**
```hcl
provider "dotfiles" {
  dotfiles_root = "~/dotfiles"
  strategy     = "symlink"
  backup_dir   = "~/.backups"
}
```

**After (v1.0):**
```hcl
provider "dotfiles" {
  dotfiles_root     = "~/dotfiles"
  strategy         = "symlink"
  backup_directory = "~/.dotfiles-backups"  # Renamed from backup_dir
  
  # New backup configuration
  backup_strategy {
    enabled     = true
    format      = "timestamped"
    compression = true
    retention {
      max_age   = "30d"
      max_count = 50
    }
  }
}
```

#### 2. Template Engine Configuration

**Before (v0.x):**
```hcl
resource "dotfiles_file" "config" {
  template_engine = "go"
  template_vars = {
    name = "value"
  }
}
```

**After (v1.0):**
```hcl
resource "dotfiles_file" "config" {
  strategy = "template"
  
  template_vars = {
    name = "value"
  }
  
  # Enhanced template configuration
  enhanced_template {
    engine = "go"  # or "handlebars", "mustache"
    platform_variables = true
    strict_mode = true
  }
}
```

#### 3. Backup Configuration

**Before (v0.x):**
```hcl
resource "dotfiles_file" "config" {
  backup_enabled = true
}
```

**After (v1.0):**
```hcl
resource "dotfiles_file" "config" {
  backup_policy {
    enabled = true
    format  = "timestamped"  # New: multiple formats
    retention {
      max_count = 5
      max_age   = "7d"
    }
  }
}
```

#### 4. Git Authentication

**Before (v0.x):**
```hcl
resource "dotfiles_repository" "repo" {
  url      = "git@github.com:user/repo.git"
  ssh_key  = "~/.ssh/id_rsa"
}
```

**After (v1.0):**
```hcl
resource "dotfiles_repository" "repo" {
  url = "git@github.com:user/repo.git"
  
  auth {
    method               = "ssh"
    ssh_key_path         = "~/.ssh/id_rsa"
    ssh_known_hosts_path = "~/.ssh/known_hosts"  # New: enhanced security
  }
  
  # New features
  submodules = true
  depth      = 0
}
```

## Step-by-Step Migration

### Step 1: Update Provider Version

Update your Terraform configuration to use v1.0:

```hcl
terraform {
  required_providers {
    dotfiles = {
      source  = "jamesainslie/dotfiles"
      version = "~> 1.0"  # Updated from 0.x
    }
  }
}
```

### Step 2: Update Provider Configuration

#### Basic Migration

**Old Configuration:**
```hcl
provider "dotfiles" {
  dotfiles_root = "~/dotfiles"
  strategy     = "symlink"
  backup_dir   = "~/.backups"
  template_engine = "go"
}
```

**New Configuration:**
```hcl
provider "dotfiles" {
  dotfiles_root     = "~/dotfiles"
  strategy         = "symlink"
  backup_directory = "~/.dotfiles-backups"  # Renamed
  template_engine  = "go"
  
  # Enhanced configuration options
  log_level       = "info"
  max_concurrency = 10
  dry_run        = false
}
```

#### Advanced Migration with New Features

```hcl
provider "dotfiles" {
  dotfiles_root = "~/dotfiles"
  strategy     = "symlink"
  
  # Enhanced backup system
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
    }
  }
  
  # Performance optimization
  cache_config {
    enabled  = true
    max_size = 1000
    ttl      = "5m"
  }
  
  max_concurrency = 5
}
```

### Step 3: Update Resource Configurations

#### File Resources

**Before:**
```hcl
resource "dotfiles_file" "gitconfig" {
  source      = "git/gitconfig"
  target      = "~/.gitconfig"
  template    = true
  backup      = true
  
  vars = {
    name  = "John Doe"
    email = "john@example.com"
  }
}
```

**After:**
```hcl
resource "dotfiles_file" "gitconfig" {
  source_path = "git/gitconfig"  # Renamed from 'source'
  target_path = "~/.gitconfig"   # Renamed from 'target'
  strategy    = "template"       # Explicit strategy instead of 'template = true'
  
  template_vars = {              # Renamed from 'vars'
    name  = "John Doe"
    email = "john@example.com"
  }
  
  backup_policy {                # Enhanced backup configuration
    enabled = true
    format  = "timestamped"
    retention {
      max_count = 5
      max_age   = "7d"
    }
  }
  
  # New features
  file_mode = "0644"
  post_create_hooks = [
    "git config --global init.defaultBranch main"
  ]
}
```

#### Repository Resources

**Before:**
```hcl
resource "dotfiles_repository" "main" {
  url     = "https://github.com/user/dotfiles.git"
  path    = "~/dotfiles"
  branch  = "main"
  ssh_key = "~/.ssh/id_rsa"
}
```

**After:**
```hcl
resource "dotfiles_repository" "main" {
  url        = "https://github.com/user/dotfiles.git"
  local_path = "~/dotfiles"  # Renamed from 'path'
  branch     = "main"
  
  auth {                     # New structured authentication
    method       = "ssh"
    ssh_key_path = "~/.ssh/id_rsa"
  }
  
  # New features
  submodules = true
  depth      = 0
  
  # Enhanced validation
  validate_on_plan = true
}
```

#### Application Resources (New in v1.0)

```hcl
resource "dotfiles_application" "vscode" {
  application = "vscode"
  
  detection_methods {
    command = ["code", "--version"]
    file    = ["~/Applications/Visual Studio Code.app"]
  }
  
  config_mappings {
    "vscode/settings.json" = "~/Library/Application Support/Code/User/settings.json"
  }
  
  skip_if_not_installed = true
}
```

### Step 4: Update Data Sources

#### System Data Source

**Before:**
```hcl
data "dotfiles_platform" "current" {}
```

**After:**
```hcl
data "dotfiles_system" "current" {}  # Renamed and enhanced

# New attributes available:
# - platform
# - architecture  
# - home_directory
# - config_directory
# - data_directory
# - cache_directory
```

### Step 5: Validate and Apply

1. **Run Terraform Plan**: Check for any configuration issues
   ```bash
   terraform plan
   ```

2. **Review Changes**: Ensure the plan looks correct

3. **Apply Gradually**: Consider applying resources in stages
   ```bash
   terraform apply -target=dotfiles_repository.main
   terraform apply -target=dotfiles_file.gitconfig
   terraform apply  # Apply remaining resources
   ```

## Common Migration Issues and Solutions

### Issue 1: Backup Directory Conflicts

**Problem**: Old backups in different location than new default

**Solution**:
```hcl
provider "dotfiles" {
  backup_directory = "~/.backups"  # Keep old location
  
  backup_strategy {
    enabled   = true
    directory = "~/.backups"       # Specify explicitly
  }
}
```

### Issue 2: Template Engine Compatibility

**Problem**: Templates not rendering correctly with new engine

**Solution**:
```hcl
resource "dotfiles_file" "config" {
  enhanced_template {
    engine = "go"           # Explicitly specify engine
    strict_mode = false     # Disable strict mode for compatibility
  }
}
```

### Issue 3: Permission Changes

**Problem**: Files have different permissions after migration

**Solution**:
```hcl
resource "dotfiles_file" "config" {
  file_mode = "0644"  # Explicitly set permissions
  
  # Or use computed permissions
  permissions {
    preserve_source = true
  }
}
```

### Issue 4: Git Authentication Issues

**Problem**: SSH authentication not working

**Solution**:
```hcl
resource "dotfiles_repository" "repo" {
  auth {
    method                        = "ssh"
    ssh_key_path                 = "~/.ssh/id_rsa"
    ssh_known_hosts_path         = "~/.ssh/known_hosts"
    ssh_skip_host_key_verification = false  # Enable for testing only
  }
}
```

## New Features to Adopt

### 1. Enhanced Backup System

```hcl
provider "dotfiles" {
  backup_strategy {
    enabled     = true
    compression = true
    format      = "timestamped"
    
    retention {
      max_age      = "30d"
      max_count    = 100
      keep_daily   = 7
      keep_weekly  = 4
      keep_monthly = 12
    }
  }
}
```

### 2. Application Detection

```hcl
resource "dotfiles_application" "neovim" {
  application = "neovim"
  
  detection_methods {
    command = ["nvim", "--version"]
    package_manager = {
      homebrew = "neovim"
      apt      = "neovim"
    }
  }
  
  skip_if_not_installed = true
}
```

### 3. Performance Optimization

```hcl
provider "dotfiles" {
  cache_config {
    enabled  = true
    max_size = 1000
    ttl      = "5m"
  }
  
  max_concurrency = 5
}
```

### 4. Enhanced Validation

```hcl
resource "dotfiles_file" "config" {
  # Runtime validation
  validate_source_exists = true
  validate_target_writable = true
  
  # Schema validation
  file_mode = "0644"  # Validates octal format
}
```

## Testing Your Migration

### 1. Dry Run Mode

Test your migration without making changes:

```hcl
provider "dotfiles" {
  dry_run = true  # Enable dry run mode
}
```

### 2. Validate Configuration

```bash
terraform validate
terraform plan
```

### 3. Selective Apply

Apply resources incrementally:

```bash
terraform apply -target=dotfiles_repository.main
terraform apply -target=dotfiles_file.critical_config
```

### 4. Rollback Plan

Keep backups and plan for rollback:

```bash
# Backup current state
cp terraform.tfstate terraform.tfstate.backup

# Test rollback procedure
terraform plan -destroy
```

## Support and Troubleshooting

### Getting Help

1. **Check the Documentation**: [Provider Registry](https://registry.terraform.io/providers/jamesainslie/dotfiles)
2. **Review Examples**: See `examples/` directory in the repository
3. **Open an Issue**: [GitHub Issues](https://github.com/jamesainslie/terraform-provider-dotfiles/issues)
4. **Join Discussions**: [GitHub Discussions](https://github.com/jamesainslie/terraform-provider-dotfiles/discussions)

### Debug Mode

Enable debug logging for troubleshooting:

```bash
export TF_LOG=DEBUG
export TF_LOG_PROVIDER=DEBUG
terraform apply
```

### Common Debugging Steps

1. **Validate Paths**: Ensure all paths are accessible
2. **Check Permissions**: Verify file and directory permissions
3. **Test Git Access**: Manually test Git repository access
4. **Review Logs**: Check Terraform and provider logs
5. **Incremental Testing**: Test one resource at a time

---

**Need additional help?** Open an issue on GitHub with your configuration and error details.
