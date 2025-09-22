variable "dotfiles_root" {
  description = "Root directory of the dotfiles repository"
  type        = string
  default     = "./test-dotfiles"
}

variable "backup_enabled" {
  description = "Enable automatic backups of existing files"
  type        = bool
  default     = true
}

variable "dry_run" {
  description = "Preview changes without applying them"
  type        = bool
  default     = false
}
