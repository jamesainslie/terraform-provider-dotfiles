# Complete Development Environment Setup
# This example shows how to use terraform-provider-dotfiles with terraform-provider-package
# for a complete development environment setup with proper separation of concerns.

terraform {
  required_version = ">= 1.5"
  
  required_providers {
    # Package management (installation)
    pkg = {
      source  = "jamesainslie/package"
      version = "~> 0.2"
    }
    
    # Configuration management
    dotfiles = {
      source  = "jamesainslie/dotfiles"
      version = "~> 1.0"
    }
  }
}

# Package Provider - Handles application installation
provider "pkg" {
  default_manager = "brew"  # macOS focused
  assume_yes      = true
  sudo_enabled    = false
  update_cache    = "on_change"
}

# Dotfiles Provider - Handles configuration file management
provider "dotfiles" {
  dotfiles_root           = var.dotfiles_path
  backup_enabled          = true
  backup_directory        = "~/.dotfiles-backups"
  strategy               = "symlink"
  conflict_resolution    = "backup"
  template_engine        = "go"
  log_level             = "info"
  
  backup_strategy {
    format          = "timestamped"
    retention_count = 10
    compression     = true
    validation     = true
  }
}

# Get system information for cross-platform logic
data "dotfiles_system" "current" {}

# Step 1: Install Development Tools (using package provider)
resource "pkg_package" "development_tools" {
  for_each = var.development_packages
  
  name = each.value.package_name
  type = each.value.package_type
}

# Step 2: Setup Dotfiles Repository
resource "dotfiles_repository" "main" {
  source_path = var.dotfiles_path
  
  # Optional: Git configuration if using remote repository
  dynamic "git_config" {
    for_each = var.git_repository_url != "" ? [1] : []
    content {
      url                    = var.git_repository_url
      branch                = var.git_branch
      personal_access_token = var.github_token
      clone_depth          = 1
      recurse_submodules   = false
    }
  }
}

# Step 3: Configure Applications (using dotfiles provider)
resource "dotfiles_application" "development_configs" {
  for_each = var.application_configs
  
  application_name = each.key
  config_mappings = each.value.config_mappings
  
  # Ensure applications are installed first
  depends_on = [pkg_package.development_tools]
}

# Step 4: Individual Configuration Files
resource "dotfiles_file" "shell_configs" {
  for_each = var.shell_configs
  
  source_path = each.value.source_path
  target_path = each.value.target_path
  
  is_template     = lookup(each.value, "is_template", false)
  template_engine = lookup(each.value, "template_engine", "go")
  template_vars   = lookup(each.value, "template_vars", {})
  
  file_mode = lookup(each.value, "file_mode", "0644")
}

# Step 5: Directory Synchronization
resource "dotfiles_directory" "config_dirs" {
  for_each = var.config_directories
  
  source_path = each.value.source_path
  target_path = each.value.target_path
  
  recursive       = lookup(each.value, "recursive", true)
  sync_strategy  = lookup(each.value, "sync_strategy", "mirror")
  create_parents = lookup(each.value, "create_parents", true)
}

# Step 6: Symlink Management
resource "dotfiles_symlink" "config_symlinks" {
  for_each = var.symlink_configs
  
  source_path = each.value.source_path
  target_path = each.value.target_path
  
  create_parents = lookup(each.value, "create_parents", true)
}

# Outputs for verification
output "installed_packages" {
  description = "List of installed packages"
  value = {
    for k, v in pkg_package.development_tools : k => {
      name    = v.name
      type    = v.type
      version = v.version
    }
  }
}

output "configured_applications" {
  description = "List of configured applications"
  value = {
    for k, v in dotfiles_application.development_configs : k => {
      application = v.application_name
      files      = v.configured_files
      last_updated = v.last_updated
    }
  }
}

output "system_info" {
  description = "System information"
  value = {
    platform     = data.dotfiles_system.current.platform
    architecture = data.dotfiles_system.current.architecture
    home_dir    = data.dotfiles_system.current.home_directory
    config_dir  = data.dotfiles_system.current.config_directory
  }
}
