# Example configuration with deprecated strategy field usage
# This file demonstrates patterns that need migration

terraform {
  required_providers {
    dotfiles = {
      source = "jamesainslie/dotfiles"
    }
  }
}

provider "dotfiles" {
  dotfiles_root = "./dotfiles"
}

# Repository setup
resource "dotfiles_repository" "main" {
  source_path = "./dotfiles"
}

# DEPRECATED: dotfiles_file with strategy field - these need migration
resource "dotfiles_file" "gitconfig" {
  repository  = dotfiles_repository.main.id
  name        = "git-config"
  source_path = "git/gitconfig"
  target_path = "~/.gitconfig"
  file_mode   = "0644"
}

# MIGRATED: dotfiles_file with strategy=symlink → dotfiles_symlink
resource "dotfiles_symlink" "nvim_config" {
  repository  = dotfiles_repository.main.id
  name        = "neovim-config"
  source_path = "nvim/"
  target_path = "~/.config/nvim/"
}

# MIGRATED: strategy=template → is_template=true
resource "dotfiles_file" "fish_template" {
  repository    = dotfiles_repository.main.id
  name          = "fish-config"
  source_path   = "templates/config.fish.template"
  target_path   = "~/.config/fish/config.fish"
  is_template = true
  file_mode     = "0644"
  
  template_vars = {
    user_name = "John Doe"
  }
}

# CORRECT: These don't need migration
resource "dotfiles_symlink" "existing_symlink" {
  repository     = dotfiles_repository.main.id
  name           = "existing-link"
  source_path    = "tools"
  target_path    = "~/.config/tools"
  create_parents = true
}

resource "dotfiles_application" "development_tools" {
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
  }
}
