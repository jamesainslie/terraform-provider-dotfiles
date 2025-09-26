# Terraform Provider Dotfiles - Schema Documentation

This document provides comprehensive schema documentation for all resources and data sources in the terraform-provider-dotfiles.

## Provider Configuration

### Provider Block

```hcl
provider "dotfiles" {
  # Required
  dotfiles_root = "~/dotfiles"
  
  # Backup Configuration
  backup_enabled    = true
  backup_directory  = "~/.dotfiles-backups"
  
  # Deployment Strategy
  strategy            = "symlink"
  conflict_resolution = "backup"
  
  # Template Engine
  template_engine = "go"
  
  # Behavior
  dry_run   = false
  log_level = "info"
  
  # Enhanced Backup Strategy (optional block)
  backup_strategy {
    format          = "timestamped"
    retention_count = 10
    compression     = true
    validation     = true
  }
}
```

### Provider Arguments

| Argument | Type | Required | Default | Description |
|----------|------|----------|---------|-------------|
| `dotfiles_root` | `string` | Yes | - | Root directory containing dotfiles |
| `backup_enabled` | `bool` | No | `true` | Enable automatic backups |
| `backup_directory` | `string` | No | `~/.dotfiles-backups` | Directory for backup files |
| `strategy` | `string` | No | `symlink` | Default deployment strategy (`symlink`, `copy`, `template`) |
| `conflict_resolution` | `string` | No | `backup` | How to handle conflicts (`backup`, `overwrite`, `skip`) |
| `template_engine` | `string` | No | `go` | Default template engine (`go`, `handlebars`, `mustache`) |
| `dry_run` | `bool` | No | `false` | Preview changes without applying |
| `log_level` | `string` | No | `info` | Logging level (`debug`, `info`, `warn`, `error`) |

### Backup Strategy Block

| Argument | Type | Required | Default | Description |
|----------|------|----------|---------|-------------|
| `format` | `string` | No | `timestamped` | Backup format (`timestamped`, `versioned`, `simple`) |
| `retention_count` | `number` | No | `10` | Number of backups to retain |
| `compression` | `bool` | No | `true` | Enable backup compression |
| `validation` | `bool` | No | `true` | Validate backup integrity |

## Resources

### dotfiles_repository

Manages dotfiles repositories (Git or local directories).

#### Schema

```hcl
resource "dotfiles_repository" "example" {
  source_path = "~/dotfiles"
  
  # Optional Git configuration
  git_config {
    url                    = "https://github.com/user/dotfiles.git"
    branch                = "main"
    personal_access_token = var.github_token
    ssh_key_path         = "~/.ssh/id_ed25519"
    clone_depth          = 1
    recurse_submodules   = false
  }
}
```

#### Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `source_path` | `string` | Yes | Path to the dotfiles repository |

#### Git Config Block (Optional)

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `url` | `string` | Yes | Git repository URL |
| `branch` | `string` | No | Git branch to use (default: `main`) |
| `personal_access_token` | `string` | No | GitHub/GitLab personal access token |
| `ssh_key_path` | `string` | No | Path to SSH private key |
| `clone_depth` | `number` | No | Git clone depth (default: `1`) |
| `recurse_submodules` | `bool` | No | Clone submodules (default: `false`) |

#### Attributes

| Attribute | Type | Description |
|-----------|------|-------------|
| `id` | `string` | Repository identifier |
| `local_path` | `string` | Local path to cloned repository |
| `last_updated` | `string` | Timestamp of last update |

---

### dotfiles_file

Manages individual configuration files with templating support.

#### Schema

```hcl
resource "dotfiles_file" "example" {
  source_path = "git/gitconfig"
  target_path = "~/.gitconfig"
  
  # File permissions
  file_mode = "0644"
  
  # Template processing (optional)
  is_template     = true
  template_engine = "go"
  template_vars = {
    user_name  = "John Doe"
    user_email = "john@example.com"
  }
  
  # Platform-specific variables (optional)
  platform_template_vars = {
    editor = data.dotfiles_system.current.platform == "darwin" ? "code" : "vim"
  }
}
```

#### Arguments

| Argument | Type | Required | Default | Description |
|----------|------|----------|---------|-------------|
| `source_path` | `string` | Yes | - | Path to source file in repository |
| `target_path` | `string` | Yes | - | Target path for deployed file |
| `file_mode` | `string` | No | `0644` | File permissions (octal format) |
| `is_template` | `bool` | No | `false` | Process file as template |
| `template_engine` | `string` | No | Provider default | Template engine to use |
| `template_vars` | `map(string)` | No | `{}` | Template variables |
| `platform_template_vars` | `map(string)` | No | `{}` | Platform-specific template variables |

#### Attributes

| Attribute | Type | Description |
|-----------|------|-------------|
| `id` | `string` | File resource identifier |
| `file_exists` | `bool` | Whether target file exists |
| `last_modified` | `string` | Last modification timestamp |
| `checksum` | `string` | File content checksum |

---

### dotfiles_symlink

Creates and manages symbolic links to configuration files.

#### Schema

```hcl
resource "dotfiles_symlink" "example" {
  source_path = "fish"
  target_path = "~/.config/fish"
  
  create_parents = true
}
```

#### Arguments

| Argument | Type | Required | Default | Description |
|----------|------|----------|---------|-------------|
| `source_path` | `string` | Yes | - | Path to source file/directory |
| `target_path` | `string` | Yes | - | Target path for symlink |
| `create_parents` | `bool` | No | `false` | Create parent directories if missing |

#### Attributes

| Attribute | Type | Description |
|-----------|------|-------------|
| `id` | `string` | Symlink resource identifier |
| `link_target` | `string` | Actual symlink target |
| `exists` | `bool` | Whether symlink exists |
| `last_checked` | `string` | Last verification timestamp |

---

### dotfiles_directory

Manages directory synchronization and operations.

#### Schema

```hcl
resource "dotfiles_directory" "example" {
  source_path = "config"
  target_path = "~/.config"
  
  recursive          = true
  sync_strategy     = "mirror"
  create_parents    = true
  preserve_permissions = true
  
  # Directory permissions
  directory_mode = "0755"
}
```

#### Arguments

| Argument | Type | Required | Default | Description |
|----------|------|----------|---------|-------------|
| `source_path` | `string` | Yes | - | Path to source directory |
| `target_path` | `string` | Yes | - | Target directory path |
| `recursive` | `bool` | No | `true` | Process directory recursively |
| `sync_strategy` | `string` | No | `mirror` | Sync strategy (`mirror`, `merge`, `overlay`) |
| `create_parents` | `bool` | No | `false` | Create parent directories |
| `preserve_permissions` | `bool` | No | `true` | Preserve source permissions |
| `directory_mode` | `string` | No | `0755` | Directory permissions |

#### Attributes

| Attribute | Type | Description |
|-----------|------|-------------|
| `id` | `string` | Directory resource identifier |
| `directory_exists` | `bool` | Whether target directory exists |
| `file_count` | `number` | Number of files synchronized |
| `last_synced` | `string` | Last synchronization timestamp |

---

### dotfiles_application

Manages application-specific configuration file mappings.

> **Note**: This resource focuses solely on configuration file management. Application installation should be handled by [terraform-provider-package](https://github.com/jamesainslie/terraform-provider-package).

#### Schema

```hcl
resource "dotfiles_application" "example" {
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
    "vscode/snippets/" = {
      target_path = "{{.app_support_dir}}/Code/User/snippets/"
      strategy   = "symlink"
    }
  }
}
```

#### Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `application_name` | `string` | Yes | Name of the application (used for organization and templating) |
| `config_mappings` | `map(object)` | Yes | Map of source files to target configuration mappings |

#### Config Mappings Object

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `target_path` | `string` | Yes | - | Target path for the configuration file |
| `strategy` | `string` | No | `symlink` | Deployment strategy (`symlink` or `copy`) |

#### Template Variables in Target Paths

| Variable | Description | Example |
|----------|-------------|---------|
| `{{.home_dir}}` | User home directory | `/Users/john` |
| `{{.config_dir}}` | User config directory | `~/.config` |
| `{{.app_support_dir}}` | Application support directory (macOS) | `~/Library/Application Support` |
| `{{.application}}` | Application name | `vscode` |

#### Attributes

| Attribute | Type | Description |
|-----------|------|-------------|
| `id` | `string` | Application resource identifier |
| `configured_files` | `list(string)` | List of successfully configured files |
| `last_updated` | `string` | Last update timestamp |

## Data Sources

### dotfiles_system

Provides system information for conditional logic and platform-specific configurations.

#### Schema

```hcl
data "dotfiles_system" "current" {}
```

#### Attributes

| Attribute | Type | Description |
|-----------|------|-------------|
| `platform` | `string` | Operating system (`darwin`, `linux`, `windows`) |
| `architecture` | `string` | CPU architecture (`amd64`, `arm64`) |
| `home_directory` | `string` | User home directory path |
| `config_directory` | `string` | User configuration directory |
| `shell` | `string` | Default shell |
| `username` | `string` | Current username |

---

### dotfiles_file_info

Provides information about file existence and properties.

#### Schema

```hcl
data "dotfiles_file_info" "check" {
  file_path = "~/.vimrc"
}
```

#### Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `file_path` | `string` | Yes | Path to file to check |

#### Attributes

| Attribute | Type | Description |
|-----------|------|-------------|
| `exists` | `bool` | Whether file exists |
| `is_directory` | `bool` | Whether path is a directory |
| `is_symlink` | `bool` | Whether path is a symlink |
| `size` | `number` | File size in bytes |
| `mode` | `string` | File permissions |
| `modified_time` | `string` | Last modification time |

## Validation Rules

### Path Validation

- Paths must be valid for the target platform
- Tilde (`~`) expansion is supported
- Environment variable expansion is supported (`$HOME`, `${HOME}`)
- Relative paths are resolved from the dotfiles root

### File Mode Validation

- Must be valid octal format (e.g., `0644`, `0755`)
- Must be between `0000` and `0777`
- Validates against platform-specific restrictions

### Template Engine Validation

- Must be one of: `go`, `handlebars`, `mustache`
- Template syntax is validated before processing
- Template variables must be strings

### Template Name Validation

- Template names must be valid identifiers
- Cannot contain special characters except `_` and `-`
- Must not be empty or whitespace-only

## Error Handling

The provider includes comprehensive error handling with structured error types:

### Error Types

| Type | Description | Retryable |
|------|-------------|-----------|
| `ValidationError` | Invalid configuration or input | No |
| `ConfigurationError` | Provider configuration issues | No |
| `PermissionError` | File system permission issues | No |
| `IOError` | File system I/O errors | Yes |
| `TemplateError` | Template processing errors | No |
| `GitError` | Git operation errors | Yes |
| `NetworkError` | Network-related errors | Yes |

### Retry Configuration

The provider automatically retries certain operations with exponential backoff:

- **Max Attempts**: 3
- **Base Delay**: 1 second
- **Max Delay**: 30 seconds
- **Multiplier**: 2.0

## Environment Variables

| Variable | Description | Schema Equivalent |
|----------|-------------|-------------------|
| `DOTFILES_ROOT` | Root directory for dotfiles | `dotfiles_root` |
| `DOTFILES_BACKUP_DIR` | Backup directory | `backup_directory` |
| `DOTFILES_DRY_RUN` | Enable dry-run mode | `dry_run` |
| `DOTFILES_LOG_LEVEL` | Logging level | `log_level` |
| `DOTFILES_GIT_TOKEN` | Git personal access token | `git_config.personal_access_token` |
| `DOTFILES_SSH_KEY_PATH` | SSH private key path | `git_config.ssh_key_path` |

## Platform-Specific Considerations

### macOS (Darwin)

- Application Support directory: `~/Library/Application Support`
- Preferences directory: `~/Library/Preferences`
- LaunchAgents directory: `~/Library/LaunchAgents`

### Linux

- Config directory: `~/.config` (XDG Base Directory)
- Data directory: `~/.local/share`
- State directory: `~/.local/state`

### Windows

- AppData Roaming: `%APPDATA%`
- AppData Local: `%LOCALAPPDATA%`
- User Profile: `%USERPROFILE%`

## Examples

### Complete Provider Configuration

```hcl
provider "dotfiles" {
  dotfiles_root           = "~/dotfiles"
  backup_enabled          = true
  backup_directory        = "~/.dotfiles-backups"
  strategy               = "symlink"
  conflict_resolution    = "backup"
  template_engine        = "go"
  log_level             = "info"
  dry_run               = false
  
  backup_strategy {
    format          = "timestamped"
    retention_count = 10
    compression     = true
    validation     = true
  }
}
```

### Multi-Platform Configuration

```hcl
data "dotfiles_system" "current" {}

resource "dotfiles_file" "shell_config" {
  source_path = "shell/${data.dotfiles_system.current.platform}/config"
  target_path = data.dotfiles_system.current.platform == "windows" ? 
    "~/AppData/Roaming/shell/config" : 
    "~/.config/shell/config"
  
  is_template = true
  template_vars = {
    platform = data.dotfiles_system.current.platform
    arch     = data.dotfiles_system.current.architecture
  }
}
```

### Application Integration Pattern

```hcl
# Install application (using terraform-provider-package)
resource "pkg_package" "neovim" {
  name = "neovim"
  type = "formula"
}

# Configure application (using terraform-provider-dotfiles)
resource "dotfiles_application" "neovim" {
  application_name = "neovim"
  
  config_mappings = {
    "nvim/init.lua" = {
      target_path = "~/.config/nvim/init.lua"
      strategy   = "symlink"
    }
    "nvim/lua/" = {
      target_path = "~/.config/nvim/lua/"
      strategy   = "symlink"
    }
  }
  
  depends_on = [pkg_package.neovim]
}
```

## Migration from v0.x

See [MIGRATION_GUIDE.md](MIGRATION_GUIDE.md) for detailed migration instructions from previous versions.

---

For more examples and use cases, see the [examples](../examples/) directory in this repository.