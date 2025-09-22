terraform {
  required_providers {
    dotfiles = {
      source = "jamesainslie/dotfiles"
    }
  }
}

provider "dotfiles" {
  dotfiles_root = "./test-dotfiles"
  dry_run       = true # Safe for testing
}

# Test the system data source
data "dotfiles_system" "current" {}

# Test a simple repository
resource "dotfiles_repository" "test" {
  name        = "test-repo"
  source_path = "./test-dotfiles"
  description = "Test repository"
}

output "test_outputs" {
  value = {
    platform = data.dotfiles_system.current.platform
    repo_id  = dotfiles_repository.test.id
  }
}
