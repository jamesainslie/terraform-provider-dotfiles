terraform {
  required_version = ">= 1.5"
  required_providers {
    dotfiles = {
      source  = "jamesainslie/dotfiles"
      version = "~> 0.1"
    }
  }
}

provider "dotfiles" {
  dotfiles_root    = "~/.terraform-dotfiles-cache"
  backup_enabled   = true
  backup_directory = "~/.dotfiles-backups"
  strategy         = "symlink"
}

# Example 1: Public GitHub repository (no authentication required)
resource "dotfiles_repository" "public_dotfiles" {
  name        = "public-dotfiles"
  source_path = "https://github.com/example/public-dotfiles.git"
  description = "Public dotfiles repository"

  git_branch = "main"

  default_backup_enabled = true
  default_file_mode      = "0644"
  default_dir_mode       = "0755"
}

# Example 2: Private GitHub repository with Personal Access Token
resource "dotfiles_repository" "private_dotfiles" {
  name        = "private-dotfiles"
  source_path = "https://github.com/jamesainslie/dotfiles.git"
  description = "Private dotfiles repository with PAT authentication"

  git_branch                = "main"
  git_personal_access_token = var.github_token # From variables or environment
  git_username              = "jamesainslie"   # Optional when using PAT

  default_backup_enabled = true
}

# Example 3: GitHub repository with SSH authentication
resource "dotfiles_repository" "ssh_dotfiles" {
  name        = "ssh-dotfiles"
  source_path = "git@github.com:jamesainslie/dotfiles.git"
  description = "Dotfiles repository with SSH key authentication"

  git_ssh_private_key_path = "~/.ssh/id_ed25519"
  git_ssh_passphrase       = var.ssh_passphrase # Optional if key has passphrase

  default_backup_enabled = true
}

# Example 4: Enterprise GitHub repository
resource "dotfiles_repository" "enterprise_dotfiles" {
  name        = "enterprise-dotfiles"
  source_path = "https://github.enterprise.com/company/dotfiles.git"
  description = "Enterprise GitHub dotfiles"

  git_personal_access_token = var.enterprise_github_token
  git_update_interval       = "1h" # Check for updates every hour
}

# System information data source
data "dotfiles_system" "current" {}

# Example file resource using the GitHub repository
resource "dotfiles_file" "gitconfig" {
  repository  = dotfiles_repository.private_dotfiles.id
  name        = "git-configuration"
  source_path = "git/gitconfig"
  target_path = "~/.gitconfig"
  is_template = false
  file_mode   = "0644"
}

# Example symlink resource using the GitHub repository
resource "dotfiles_symlink" "fish_config" {
  repository     = dotfiles_repository.private_dotfiles.id
  name           = "fish-shell-config"
  source_path    = "fish"
  target_path    = "~/.config/fish"
  force_update   = false
  create_parents = true
}

# Output repository information
output "repository_info" {
  value = {
    public_repo = {
      id          = dotfiles_repository.public_dotfiles.id
      local_path  = dotfiles_repository.public_dotfiles.local_path
      last_commit = dotfiles_repository.public_dotfiles.last_commit
      last_update = dotfiles_repository.public_dotfiles.last_update
    }

    private_repo = {
      id          = dotfiles_repository.private_dotfiles.id
      local_path  = dotfiles_repository.private_dotfiles.local_path
      last_commit = dotfiles_repository.private_dotfiles.last_commit
      last_update = dotfiles_repository.private_dotfiles.last_update
    }
  }

  description = "Information about managed dotfiles repositories"
}

output "system_info" {
  value = {
    platform     = data.dotfiles_system.current.platform
    architecture = data.dotfiles_system.current.architecture
    home_dir     = data.dotfiles_system.current.home_dir
    config_dir   = data.dotfiles_system.current.config_dir
  }
}
