# Migration Guide - Terraform Provider Dotfiles

This guide helps you migrate from previous versions of terraform-provider-dotfiles to the current version, which implements separation of concerns with terraform-provider-package.

## Overview of Changes

The provider has undergone a significant refactoring to focus solely on **configuration file management**, removing application detection and installation logic. This creates a cleaner separation of concerns:

- **terraform-provider-dotfiles**: Configuration file management (symlinks, templates, backups)
- **terraform-provider-package**: Application installation and lifecycle management

## Breaking Changes Summary

### v0.x → v1.0

| Component | Change | Action Required |
|-----------|--------|-----------------|
| **Application Detection** | ❌ Removed entirely | Use terraform-provider-package |
| **dotfiles_application** | 🔄 Schema simplified | Update resource configuration |
| **Provider Config** | 🔄 Enhanced backup options | Update provider block |
| **File Resource** | ✅ No breaking changes | Optional: leverage new features |

---

## 1. Provider Configuration Migration

### Before (v0.x)
```hcl
provider "dotfiles" {
  dotfiles_root = var.dotfiles_source_path
}
```

### After (v1.0)
```hcl
provider "dotfiles" {
  dotfiles_root           = var.dotfiles_source_path
  backup_enabled          = true
  backup_directory        = "~/.dotfiles-backups"
  strategy               = "symlink"
  conflict_resolution    = "backup"
  template_engine        = "go"
  log_level             = "info"
  dry_run               = false
  
  # Enhanced backup configuration
  backup_strategy {
    format          = "timestamped"
    retention_count = 10
    compression     = true
    validation     = true
  }
}
```

**Migration Steps:**
1. Add new provider configuration options
2. Configure backup strategy if needed
3. Set default behavior preferences

---

## 2. Application Resource Migration

### Before (v0.x) - ❌ No Longer Works
```hcl
resource "dotfiles_application" "vscode" {
  repository = dotfiles_repository.main.id
  application = "vscode"
  source_path = "vscode"
  
  # Application detection (REMOVED)
  detect_installation   = true
  skip_if_not_installed = true
  warn_if_not_installed = false
  
  detection_methods {
    type = "command"
    test = "code --version"
  }
  
  detection_methods {
    type = "file"
    path = "/Applications/Visual Studio Code.app"
  }
  
  version_constraints = {
    min_version = "1.70.0"
  }
  
  # Old config mapping format
  config_mappings {
    "settings.json" = "~/Library/Application Support/Code/User/settings.json"
    "keybindings.json" = "~/Library/Application Support/Code/User/keybindings.json"
  }
}
```

### After (v1.0) - ✅ New Simplified Schema
```hcl
# Step 1: Install application using package provider
resource "pkg_package" "vscode" {
  name = "visual-studio-code"
  type = "cask"
}

# Step 2: Configure application using dotfiles provider
resource "dotfiles_application" "vscode" {
  application_name = "vscode"
  
  config_mappings = {
    "vscode/settings.json" = {
      target_path = "~/Library/Application Support/Code/User/settings.json"
      strategy   = "symlink"
    }
    "vscode/keybindings.json" = {
      target_path = "~/Library/Application Support/Code/User/keybindings.json"
      strategy   = "copy"
    }
    "vscode/snippets/" = {
      target_path = "{{.app_support_dir}}/Code/User/snippets/"
      strategy   = "symlink"
    }
  }
  
  # Ensure application is installed first
  depends_on = [pkg_package.vscode]
}
```

**Key Changes:**
- ❌ **Removed**: `repository`, `detection_methods`, `skip_if_not_installed`, `version_constraints`
- 🔄 **Changed**: `application` → `application_name`
- 🔄 **Enhanced**: `config_mappings` now uses object syntax with `target_path` and `strategy`
- ✅ **Added**: Template variables in target paths
- ✅ **Added**: Strategy selection per mapping (`symlink` or `copy`)

---

## 3. Repository Resource Migration

### Before (v0.x)
```hcl
resource "dotfiles_repository" "main" {
  name        = "main-dotfiles"
  source_path = var.source_path
  description = "Main dotfiles repository"
  
  git_personal_access_token = var.git_token
  git_ssh_private_key_path  = var.ssh_key_path
  
  default_backup_enabled = true
  default_file_mode     = "0644"
  default_dir_mode      = "0755"
}
```

### After (v1.0)
```hcl
resource "dotfiles_repository" "main" {
  source_path = var.source_path
  
  # Git configuration moved to separate block
  git_config {
    url                    = var.git_repository_url
    branch                = var.git_branch
    personal_access_token = var.git_token
    ssh_key_path         = var.ssh_key_path
    clone_depth          = 1
    recurse_submodules   = false
  }
}
```

**Key Changes:**
- ❌ **Removed**: `name`, `description`, `default_*` attributes (handled by provider config)
- 🔄 **Restructured**: Git configuration moved to `git_config` block
- ✅ **Added**: `clone_depth`, `recurse_submodules` options

---

## 4. File Resource Migration

The file resource has minimal breaking changes but gains new capabilities:

### Before (v0.x)
```hcl
resource "dotfiles_file" "gitconfig" {
  name        = "gitconfig"
  repository  = dotfiles_repository.main.id
  source_path = "git/gitconfig.template"
  target_path = "~/.gitconfig"
  
  is_template     = true
  template_engine = "go"
  template_vars   = {
    user_name  = "John Doe"
    user_email = "john@example.com"
  }
  
  backup_policy {
    always_backup     = true
    versioned_backup  = true
    backup_format     = "timestamped"
    retention_count   = 5
  }
  
  require_application = "git"
  skip_if_app_missing = false
}
```

### After (v1.0)
```hcl
resource "dotfiles_file" "gitconfig" {
  source_path = "git/gitconfig.template"
  target_path = "~/.gitconfig"
  
  is_template     = true
  template_engine = "go"
  template_vars   = {
    user_name  = "John Doe"
    user_email = "john@example.com"
  }
  
  file_mode = "0644"
}
```

**Key Changes:**
- ❌ **Removed**: `name`, `repository`, `backup_policy`, `require_application`, `skip_if_app_missing`
- ✅ **Simplified**: Backup handled by provider configuration
- ✅ **Enhanced**: Better error handling and retry logic

---

## 5. Variable Updates

### Update terraform.tfvars

**Add package definitions:**
```hcl
# NEW: Package installation (terraform-provider-package)
development_packages = {
  "vscode" = {
    package_name = "visual-studio-code"
    package_type = "cask"
  }
  "neovim" = {
    package_name = "neovim"
    package_type = "formula"
  }
  "fish" = {
    package_name = "fish"
    package_type = "formula"
  }
}

# UPDATED: Simplified application configs
application_configs = {
  "vscode" = {
    config_mappings = {
      "vscode/settings.json" = {
        target_path = "~/Library/Application Support/Code/User/settings.json"
        strategy   = "symlink"
      }
      "vscode/keybindings.json" = {
        target_path = "~/Library/Application Support/Code/User/keybindings.json"
        strategy   = "copy"
      }
    }
  }
  "neovim" = {
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
}
```

### Update variables.tf

**Remove old variables:**
```hcl
# REMOVE: No longer needed
variable "application_configs" {
  type = map(object({
    optional = bool
    detection_methods = list(object({
      type = string
      test = string
      path = string
    }))
  }))
}
```

**Add new variables:**
```hcl
# ADD: Package management
variable "development_packages" {
  description = "Development packages to install via terraform-provider-package"
  type = map(object({
    package_name = string
    package_type = string
  }))
  default = {}
}

# UPDATE: Simplified application configs
variable "application_configs" {
  description = "Application configuration file mappings"
  type = map(object({
    config_mappings = map(object({
      target_path = string
      strategy   = string
    }))
  }))
  default = {}
}

# ADD: Git repository configuration
variable "git_repository_url" {
  description = "Git repository URL for dotfiles"
  type        = string
  default     = ""
}

variable "git_branch" {
  description = "Git branch to use"
  type        = string
  default     = "main"
}
```

---

## 6. Template Path Variables

The new application resource supports template variables in target paths:

### Available Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `{{.home_dir}}` | User home directory | `/Users/john` |
| `{{.config_dir}}` | User config directory | `~/.config` |
| `{{.app_support_dir}}` | App support directory (macOS) | `~/Library/Application Support` |
| `{{.application}}` | Application name | `vscode` |

### Usage Example

```hcl
resource "dotfiles_application" "cross_platform" {
  application_name = "myapp"
  
  config_mappings = {
    "config.json" = {
      target_path = "{{.config_dir}}/{{.application}}/config.json"
      strategy   = "symlink"
    }
    "settings.json" = {
      target_path = "{{.app_support_dir}}/{{.application}}/settings.json"
      strategy   = "copy"
    }
  }
}
```

---

## 7. Integration with terraform-provider-package

### Add Package Provider

**Update versions.tf:**
```hcl
terraform {
  required_providers {
    pkg = {
      source = "jamesainslie/package"
      version = "~> 0.2"
    }
    dotfiles = {
      source = "jamesainslie/dotfiles"
      version = "~> 1.0"
    }
  }
}

provider "pkg" {
  default_manager = "brew"  # or "apt", "winget"
  assume_yes      = true
  sudo_enabled    = false
}
```

### Module Integration Pattern

**In main.tf:**
```hcl
# Install packages first
module "packages" {
  source = "./modules/packages"
  
  development_packages = var.development_packages
}

# Configure applications after installation
module "dotfiles" {
  source = "./modules/dotfiles"
  
  application_configs = var.application_configs
  
  depends_on = [module.packages]
}
```

**In modules/packages/main.tf:**
```hcl
resource "pkg_package" "development_tools" {
  for_each = var.development_packages
  
  name = each.value.package_name
  type = each.value.package_type
}
```

**In modules/dotfiles/main.tf:**
```hcl
resource "dotfiles_application" "development_configs" {
  for_each = var.application_configs
  
  application_name = each.key
  config_mappings = each.value.config_mappings
}
```

---

## 8. Step-by-Step Migration Process

### Phase 1: Preparation
1. **Backup your current configuration:**
   ```bash
   cp -r ~/dotfiles/terraform-devenv ~/dotfiles/terraform-devenv.backup
   ```

2. **Install terraform-provider-package:**
   ```bash
   # Add to versions.tf and run:
   terraform init -upgrade
   ```

### Phase 2: Provider Updates
3. **Update provider configuration** in `versions.tf` and provider blocks
4. **Add package provider configuration**

### Phase 3: Resource Migration
5. **Update application resources:**
   - Remove detection-related attributes
   - Update `config_mappings` to new object format
   - Add `depends_on` for package resources

6. **Update repository resources:**
   - Move Git config to `git_config` block
   - Remove deprecated attributes

### Phase 4: Variable Updates
7. **Update variables.tf** with new variable definitions
8. **Update terraform.tfvars** with package and config mappings

### Phase 5: Testing
9. **Plan and validate:**
   ```bash
   terraform plan
   ```

10. **Apply changes:**
    ```bash
    terraform apply
    ```

### Phase 6: Cleanup
11. **Remove old state** if needed:
    ```bash
    terraform state rm dotfiles_application.old_resource
    ```

---

## 9. Common Migration Issues

### Issue: Application Detection Errors
**Error:** `Application detection methods no longer supported`

**Solution:** Remove all detection-related configuration and use terraform-provider-package:
```hcl
# OLD - Remove this
detect_installation = true
detection_methods { ... }

# NEW - Add this
resource "pkg_package" "app" {
  name = "application-name"
  type = "cask"  # or "formula"
}
```

### Issue: Config Mappings Format Error
**Error:** `config_mappings must be map of objects`

**Solution:** Update to new object format:
```hcl
# OLD
config_mappings {
  "file.json" = "~/target/file.json"
}

# NEW
config_mappings = {
  "file.json" = {
    target_path = "~/target/file.json"
    strategy   = "symlink"
  }
}
```

### Issue: Repository Dependencies
**Error:** `repository attribute no longer exists`

**Solution:** Remove repository references from application resources:
```hcl
# OLD - Remove this
resource "dotfiles_application" "app" {
  repository = dotfiles_repository.main.id
  # ...
}

# NEW - No repository reference needed
resource "dotfiles_application" "app" {
  application_name = "app"
  # ...
}
```

### Issue: Backup Policy Errors
**Error:** `backup_policy block no longer supported`

**Solution:** Configure backup at provider level:
```hcl
# OLD - Remove from resources
backup_policy {
  always_backup = true
  # ...
}

# NEW - Configure in provider
provider "dotfiles" {
  backup_enabled = true
  backup_strategy {
    format = "timestamped"
    retention_count = 10
  }
}
```

---

## 10. Rollback Plan

If you need to rollback to v0.x:

1. **Restore backup configuration:**
   ```bash
   cp -r ~/dotfiles/terraform-devenv.backup ~/dotfiles/terraform-devenv
   ```

2. **Downgrade provider version:**
   ```hcl
   dotfiles = {
     source  = "jamesainslie/dotfiles"
     version = "~> 0.2"  # Previous version
   }
   ```

3. **Run terraform init:**
   ```bash
   terraform init -upgrade
   ```

---

## 11. Benefits After Migration

### Improved Separation of Concerns
- **Cleaner architecture**: Each provider has a single responsibility
- **Better maintainability**: Focused codebase for each provider
- **Reduced complexity**: Simpler schemas and logic

### Enhanced Reliability
- **Fewer edge cases**: No complex application detection logic
- **Better error handling**: Structured errors with retry logic
- **Improved testing**: Focused unit and integration tests

### Better Composability
- **Mix and match**: Use providers independently or together
- **Clear dependencies**: Explicit `depends_on` relationships
- **Flexible deployment**: Choose appropriate provider for each task

### Future-Proof Design
- **Single responsibility**: Each provider can evolve independently
- **Clear boundaries**: Well-defined interfaces between providers
- **Easier contributions**: Focused development efforts

---

## 12. Support and Resources

- **Migration Issues**: [GitHub Issues](https://github.com/jamesainslie/terraform-provider-dotfiles/issues)
- **Documentation**: [Provider Registry Docs](https://registry.terraform.io/providers/jamesainslie/dotfiles/latest/docs)
- **Examples**: [GitHub Examples](https://github.com/jamesainslie/terraform-provider-dotfiles/tree/main/examples)
- **Package Provider**: [terraform-provider-package](https://github.com/jamesainslie/terraform-provider-package)

**Need help?** Open an issue with:
- Your current configuration
- Target configuration
- Specific error messages
- Platform information (OS, Terraform version)

---

**Migration completed successfully?** 🎉 You now have a cleaner, more maintainable infrastructure setup with proper separation of concerns!