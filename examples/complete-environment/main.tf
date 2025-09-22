terraform {
  required_providers {
    dotfiles = {
      source = "jamesainslie/dotfiles"
    }
  }
}

provider "dotfiles" {
  dotfiles_root    = "./test-dotfiles"
  backup_enabled   = true
  backup_directory = "./backups"
  strategy         = "copy" # Use copy for this example
  dry_run          = false
}

# System information
data "dotfiles_system" "current" {}

# Local repository
resource "dotfiles_repository" "local" {
  name        = "local-dotfiles"
  source_path = "./test-dotfiles"
  description = "Local test dotfiles repository"
}

# Copy SSH config with strict permissions
resource "dotfiles_file" "ssh_config" {
  repository     = dotfiles_repository.local.id
  name           = "ssh-configuration"
  source_path    = "ssh/config"
  target_path    = "~/.ssh/config"
  file_mode      = "0600"
  backup_enabled = true
}

# Process gitconfig template with variables
resource "dotfiles_file" "gitconfig" {
  repository  = dotfiles_repository.local.id
  name        = "git-configuration"
  source_path = "templates/gitconfig.template"
  target_path = "~/.gitconfig"
  is_template = true
  file_mode   = "0644"

  template_vars = {
    user_name  = "Test User"
    user_email = "test@example.com"
    editor     = "vim"
    gpg_key    = "ABC123DEF456"
  }
}

# Create symlink to fish configuration
resource "dotfiles_symlink" "fish_config" {
  repository     = dotfiles_repository.local.id
  name           = "fish-configuration"
  source_path    = "fish"
  target_path    = "~/.config/fish"
  create_parents = true
  force_update   = false
}

# Process fish config template
resource "dotfiles_file" "fish_config_file" {
  repository  = dotfiles_repository.local.id
  name        = "fish-config-file"
  source_path = "templates/config.fish.template"
  target_path = "~/.config/fish/config.fish"
  is_template = true
  file_mode   = "0644"

  template_vars = {
    user_name = "Test User"
    editor    = "vim"
  }
}

# Copy tools configuration
resource "dotfiles_file" "cursor_config" {
  repository     = dotfiles_repository.local.id
  name           = "cursor-configuration"
  source_path    = "tools/cursor.json"
  target_path    = "~/.config/cursor/settings.json"
  file_mode      = "0644"
  backup_enabled = true
}

# Outputs to show computed values
output "system_info" {
  value = {
    platform     = data.dotfiles_system.current.platform
    architecture = data.dotfiles_system.current.architecture
    home_dir     = data.dotfiles_system.current.home_dir
    config_dir   = data.dotfiles_system.current.config_dir
  }
}

output "repository_info" {
  value = {
    id          = dotfiles_repository.local.id
    local_path  = dotfiles_repository.local.local_path
    last_update = dotfiles_repository.local.last_update
  }
}

output "file_states" {
  value = {
    ssh_config = {
      exists        = dotfiles_file.ssh_config.file_exists
      content_hash  = dotfiles_file.ssh_config.content_hash
      last_modified = dotfiles_file.ssh_config.last_modified
    }
    gitconfig = {
      exists        = dotfiles_file.gitconfig.file_exists
      content_hash  = dotfiles_file.gitconfig.content_hash
      last_modified = dotfiles_file.gitconfig.last_modified
    }
  }
}

output "symlink_state" {
  value = {
    fish_config = {
      exists        = dotfiles_symlink.fish_config.link_exists
      is_symlink    = dotfiles_symlink.fish_config.is_symlink
      link_target   = dotfiles_symlink.fish_config.link_target
      last_modified = dotfiles_symlink.fish_config.last_modified
    }
  }
}
