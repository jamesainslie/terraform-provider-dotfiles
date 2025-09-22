variable "github_token" {
  description = "GitHub Personal Access Token for private repository access"
  type        = string
  sensitive   = true
  default     = ""
  
  validation {
    condition     = can(regex("^$|^gh[ps]_[A-Za-z0-9_]{36,}$", var.github_token))
    error_message = "GitHub token must be empty or a valid GitHub token (starts with ghp_ or ghs_)."
  }
}

variable "enterprise_github_token" {
  description = "Enterprise GitHub Personal Access Token"
  type        = string
  sensitive   = true
  default     = ""
}

variable "ssh_passphrase" {
  description = "Passphrase for SSH private key (if required)"
  type        = string
  sensitive   = true
  default     = ""
}

variable "dotfiles_repository_url" {
  description = "URL of the dotfiles repository to clone"
  type        = string
  default     = "https://github.com/jamesainslie/dotfiles.git"
  
  validation {
    condition = can(regex("^(https://github\\.com/|git@github\\.com:|https://.*\\.git$)", var.dotfiles_repository_url))
    error_message = "Repository URL must be a valid Git URL."
  }
}

variable "git_branch" {
  description = "Git branch to checkout"
  type        = string
  default     = "main"
}

variable "enable_auto_update" {
  description = "Enable automatic repository updates"
  type        = bool
  default     = true
}

variable "update_interval" {
  description = "Interval for automatic repository updates (e.g., '1h', '30m')"
  type        = string
  default     = "1h"
  
  validation {
    condition = can(regex("^([0-9]+[smh]|never)$", var.update_interval))
    error_message = "Update interval must be in format like '30m', '1h', '2h' or 'never'."
  }
}
