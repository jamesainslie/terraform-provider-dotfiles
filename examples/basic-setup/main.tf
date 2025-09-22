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
  backup_directory = "./.dotfiles-backups"
  strategy         = "symlink"
  dry_run          = false
}

# System information data source
data "dotfiles_system" "current" {}

# Example repository
resource "dotfiles_repository" "main" {
  name                  = "personal-dotfiles"
  source_path          = "./test-dotfiles"
  description          = "Personal development environment dotfiles"
  default_backup_enabled = true
  default_file_mode    = "0644"
  default_dir_mode     = "0755"
}

# Example file resource
resource "dotfiles_file" "gitconfig" {
  repository    = dotfiles_repository.main.id
  name         = "git-config"
  source_path  = "git/gitconfig"
  target_path  = "~/.gitconfig"
  is_template  = false
  file_mode    = "0644"
}

# Example symlink resource
resource "dotfiles_symlink" "fish_config" {
  repository     = dotfiles_repository.main.id
  name          = "fish-configuration"
  source_path   = "fish"
  target_path   = "~/.config/fish"
  force_update  = false
  create_parents = true
}

# Example directory resource
resource "dotfiles_directory" "tools_config" {
  repository            = dotfiles_repository.main.id
  name                 = "tools-configuration"
  source_path          = "tools"
  target_path          = "~/.config/tools"
  recursive            = true
  preserve_permissions = true
}

# Output system information
output "system_info" {
  value = {
    platform     = data.dotfiles_system.current.platform
    architecture = data.dotfiles_system.current.architecture
    home_dir     = data.dotfiles_system.current.home_dir
    config_dir   = data.dotfiles_system.current.config_dir
  }
}

output "repository_id" {
  value = dotfiles_repository.main.id
}
