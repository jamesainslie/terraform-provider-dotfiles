# Implement missing provider features

This pull request implements five provider features identified as missing:

- Permission management with rule-based octal modes and glob matching
- Post-create, post-update, and pre-destroy command hooks
- Backup and recovery options (timestamped/numbered/git-style), optional compression, incremental mode, retention and metadata/index
- Template processing (Go templates with default functions, plus Handlebars/Mustache-compatible syntax mapping), platform-specific variables, and custom functions
- Application detection (command/file checks, Homebrew/package manager detection, version constraints) with optional conditional resource behavior

## Features

### Permission management
```hcl
resource "dotfiles_file" "ssh_config" {
  permissions = {
    directory = "0700"
    files     = "0600"
    recursive = true
  }

  permission_rules = {
    "id_*"        = "0600"
    "*.pub"       = "0644"
    "known_hosts" = "0600"
  }
}
```

### Post-creation/update/destroy hooks
```hcl
resource "dotfiles_file" "scripts" {
  post_create_commands = [
    "find ~/.local/bin -name '*.sh' -exec chmod +x {} \\;",
    "rehash"
  ]

  post_update_commands = [
    "systemctl --user reload environment"
  ]

  pre_destroy_commands = [
    "backup-custom-scripts ~/.local/bin/custom/"
  ]
}
```

### Backup and recovery
```hcl
provider "dotfiles" {
  backup_strategy = {
    enabled          = true
    retention_policy = "30d"
    compression      = true
    incremental      = true
    max_backups      = 50
  }
}

resource "dotfiles_file" "config" {
  backup_policy = {
    versioned_backup = true
    backup_format    = "timestamped"
    retention_count  = 10
    compression      = true
    backup_metadata  = true
  }
}
```

### Template processing
```hcl
resource "dotfiles_file" "gitconfig" {
  is_template     = true
  template_engine = "go"

  template_vars = {
    user_name  = var.git_user_name
    user_email = var.git_user_email
  }

  platform_template_vars = {
    macos = {
      credential_helper = "osxkeychain"
      diff_tool         = "opendiff"
    }
    linux = {
      credential_helper = "cache"
      diff_tool         = "vimdiff"
    }
  }
}
```

### Application detection
```hcl
resource "dotfiles_application" "cursor" {
  application            = "cursor"
  detect_installation    = true
  skip_if_not_installed  = true
  min_version            = "0.17.0"
}

resource "dotfiles_file" "vscode_config" {
  require_application  = "code"
  skip_if_app_missing  = true
}
```

## Notes
- Tests cover unit, integration, and end-to-end scenarios.
- No breaking changes expected for existing configurations.
- Documentation updates are included under `docs/`.

