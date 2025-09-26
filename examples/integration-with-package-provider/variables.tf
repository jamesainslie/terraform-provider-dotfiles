# Variables for Complete Development Environment Setup

variable "dotfiles_path" {
  description = "Path to dotfiles directory (local or will be cloned)"
  type        = string
  default     = "~/dotfiles"
}

variable "git_repository_url" {
  description = "Git repository URL for dotfiles (optional)"
  type        = string
  default     = ""
}

variable "git_branch" {
  description = "Git branch to use"
  type        = string
  default     = "main"
}

variable "github_token" {
  description = "GitHub personal access token (for private repos)"
  type        = string
  default     = ""
  sensitive   = true
}

# Package Installation Configuration (terraform-provider-package)
variable "development_packages" {
  description = "Development packages to install"
  type = map(object({
    package_name = string
    package_type = string  # "formula", "cask", "tap"
  }))
  default = {
    # Terminal and Shell
    "fish" = {
      package_name = "fish"
      package_type = "formula"
    }
    "starship" = {
      package_name = "starship"
      package_type = "formula"
    }
    
    # Editors
    "neovim" = {
      package_name = "neovim"
      package_type = "formula"
    }
    "vscode" = {
      package_name = "visual-studio-code"
      package_type = "cask"
    }
    
    # Development Tools
    "git" = {
      package_name = "git"
      package_type = "formula"
    }
    "gh" = {
      package_name = "gh"
      package_type = "formula"
    }
    "terraform" = {
      package_name = "terraform"
      package_type = "formula"
    }
    
    # Languages
    "go" = {
      package_name = "go"
      package_type = "formula"
    }
    "node" = {
      package_name = "node"
      package_type = "formula"
    }
    "python" = {
      package_name = "python@3.12"
      package_type = "formula"
    }
  }
}

# Application Configuration Management (terraform-provider-dotfiles)
variable "application_configs" {
  description = "Application configuration file mappings"
  type = map(object({
    config_mappings = map(object({
      target_path = string
      strategy   = string  # "symlink" or "copy"
    }))
  }))
  default = {
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
        "vscode/snippets/" = {
          target_path = "{{.app_support_dir}}/Code/User/snippets/"
          strategy   = "symlink"
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
        "nvim/after/" = {
          target_path = "~/.config/nvim/after/"
          strategy   = "symlink"
        }
      }
    }
    
    "git" = {
      config_mappings = {
        "git/gitconfig" = {
          target_path = "~/.gitconfig"
          strategy   = "symlink"
        }
        "git/gitignore_global" = {
          target_path = "~/.gitignore_global"
          strategy   = "symlink"
        }
      }
    }
    
    "gh" = {
      config_mappings = {
        "gh/config.yml" = {
          target_path = "~/.config/gh/config.yml"
          strategy   = "symlink"
        }
      }
    }
  }
}

# Shell Configuration Files
variable "shell_configs" {
  description = "Shell configuration files with templating support"
  type = map(object({
    source_path     = string
    target_path     = string
    is_template     = optional(bool, false)
    template_engine = optional(string, "go")
    template_vars   = optional(map(string), {})
    file_mode      = optional(string, "0644")
  }))
  default = {
    "fish_config" = {
      source_path = "fish/config.fish"
      target_path = "~/.config/fish/config.fish"
      is_template = false
    }
    
    "starship_config" = {
      source_path = "starship/starship.toml"
      target_path = "~/.config/starship.toml"
      is_template = false
    }
    
    # Template example with dynamic content
    "ssh_config" = {
      source_path     = "ssh/config.template"
      target_path     = "~/.ssh/config"
      is_template     = true
      template_engine = "go"
      template_vars = {
        hostname = "my-server.example.com"
        username = "myuser"
        port     = "22"
      }
      file_mode = "0600"
    }
  }
}

# Directory Synchronization
variable "config_directories" {
  description = "Configuration directories to synchronize"
  type = map(object({
    source_path    = string
    target_path    = string
    recursive      = optional(bool, true)
    sync_strategy  = optional(string, "mirror")
    create_parents = optional(bool, true)
  }))
  default = {
    "fish_functions" = {
      source_path = "fish/functions"
      target_path = "~/.config/fish/functions"
    }
    
    "fish_completions" = {
      source_path = "fish/completions"
      target_path = "~/.config/fish/completions"
    }
  }
}

# Symlink Configuration
variable "symlink_configs" {
  description = "Symlink configurations"
  type = map(object({
    source_path    = string
    target_path    = string
    create_parents = optional(bool, true)
  }))
  default = {
    "bin_scripts" = {
      source_path = "bin"
      target_path = "~/.local/bin/dotfiles"
    }
  }
}

# Template Variables for Cross-Platform Support
variable "template_variables" {
  description = "Global template variables"
  type        = map(string)
  default = {
    editor       = "nvim"
    browser      = "firefox"
    terminal     = "alacritty"
    git_editor   = "nvim"
    default_shell = "fish"
  }
}
