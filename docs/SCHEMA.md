# Schema Documentation

This document provides comprehensive schema documentation for all resources and data sources in the terraform-provider-dotfiles.

## Provider Configuration

### Arguments

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `dotfiles_root` | `string` | No | `"~/dotfiles"` | Root directory containing dotfiles |
| `strategy` | `string` | No | `"symlink"` | Default deployment strategy |
| `conflict_resolution` | `string` | No | `"backup"` | How to handle file conflicts |
| `target_platform` | `string` | No | `"auto"` | Target platform for deployment |
| `template_engine` | `string` | No | `"go"` | Default template engine |
| `backup_enabled` | `bool` | No | `true` | Enable automatic backups |
| `backup_directory` | `string` | No | `"~/.dotfiles-backups"` | Directory for storing backups |
| `dry_run` | `bool` | No | `false` | Preview mode without making changes |
| `log_level` | `string` | No | `"info"` | Logging verbosity level |
| `max_concurrency` | `number` | No | `10` | Maximum concurrent operations |
| `auto_detect_platform` | `bool` | No | `true` | Automatically detect target platform |

### Nested Blocks

#### `backup_strategy`

Enhanced backup configuration block.

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `enabled` | `bool` | No | `true` | Enable enhanced backup system |
| `directory` | `string` | No | `"~/.dotfiles-backups"` | Backup storage directory |
| `format` | `string` | No | `"timestamped"` | Backup naming format |
| `compression` | `bool` | No | `false` | Enable backup compression |
| `retention` | `block` | No | - | Backup retention policy |

##### `retention` (within `backup_strategy`)

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `max_age` | `string` | No | `"30d"` | Maximum age of backups to keep |
| `max_count` | `number` | No | `50` | Maximum number of backups to keep |
| `keep_daily` | `number` | No | `7` | Number of daily backups to keep |
| `keep_weekly` | `number` | No | `4` | Number of weekly backups to keep |
| `keep_monthly` | `number` | No | `12` | Number of monthly backups to keep |

#### `cache_config`

Performance caching configuration.

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `enabled` | `bool` | No | `true` | Enable in-memory caching |
| `max_size` | `number` | No | `1000` | Maximum cache entries |
| `ttl` | `string` | No | `"5m"` | Time-to-live for cache entries |
| `cleanup_interval` | `string` | No | `"10m"` | Cache cleanup frequency |

### Valid Values

#### `strategy`
- `"symlink"` - Create symbolic links
- `"copy"` - Copy files directly  
- `"template"` - Process as templates

#### `conflict_resolution`
- `"backup"` - Create backup before overwriting
- `"overwrite"` - Overwrite without backup
- `"skip"` - Skip conflicting files
- `"prompt"` - Interactive prompt (not supported in Terraform)

#### `target_platform`
- `"auto"` - Automatically detect platform
- `"macos"` - Target macOS systems
- `"linux"` - Target Linux systems  
- `"windows"` - Target Windows systems

#### `template_engine`
- `"go"` - Go text/template engine
- `"handlebars"` - Handlebars template engine
- `"mustache"` - Mustache template engine
- `"none"` - Disable template processing

#### `log_level`
- `"debug"` - Verbose debugging information
- `"info"` - General information
- `"warn"` - Warning messages only
- `"error"` - Error messages only

#### `format` (for backups)
- `"timestamped"` - Append timestamp to filename
- `"numbered"` - Append incremental number
- `"git-style"` - Use git-style backup naming

## Resources

### `dotfiles_repository`

Manages dotfiles repositories (local or Git-based).

#### Arguments

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `url` | `string` | Yes | - | Repository URL (Git) or local path |
| `local_path` | `string` | Yes | - | Local path to clone/manage repository |
| `branch` | `string` | No | `"main"` | Git branch to use |
| `tag` | `string` | No | - | Specific Git tag to checkout |
| `commit` | `string` | No | - | Specific Git commit to checkout |
| `submodules` | `bool` | No | `false` | Initialize and update Git submodules |
| `depth` | `number` | No | `1` | Clone depth (0 for full clone) |
| `single_branch` | `bool` | No | `true` | Clone only specified branch |
| `validate_on_plan` | `bool` | No | `true` | Validate repository during plan |

#### Nested Blocks

##### `auth`

Git authentication configuration.

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `method` | `string` | Yes | - | Authentication method |
| `username` | `string` | No | - | Username for HTTPS auth |
| `personal_access_token` | `string` | No | - | Personal access token |
| `ssh_key_path` | `string` | No | - | Path to SSH private key |
| `ssh_passphrase` | `string` | No | - | SSH key passphrase |
| `ssh_known_hosts_path` | `string` | No | `"~/.ssh/known_hosts"` | SSH known hosts file |
| `ssh_skip_host_key_verification` | `bool` | No | `false` | Skip SSH host key verification |

###### Valid `method` values:
- `"none"` - No authentication
- `"ssh"` - SSH key authentication
- `"https"` - HTTPS with token/password
- `"auto"` - Automatically detect method

#### Computed Attributes

| Name | Type | Description |
|------|------|-------------|
| `id` | `string` | Resource identifier |
| `exists` | `bool` | Whether repository exists locally |
| `is_git_repository` | `bool` | Whether path is a Git repository |
| `current_branch` | `string` | Currently checked out branch |
| `current_commit` | `string` | Current commit hash |
| `last_updated` | `string` | Last update timestamp |
| `remote_url` | `string` | Configured remote URL |
| `submodule_count` | `number` | Number of submodules |

### `dotfiles_file`

Manages individual configuration files.

#### Arguments

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `source_path` | `string` | Yes | - | Source file path (relative to repository) |
| `target_path` | `string` | Yes | - | Target deployment path |
| `strategy` | `string` | No | Provider default | Deployment strategy for this file |
| `file_mode` | `string` | No | `"0644"` | File permissions (octal) |
| `create_parents` | `bool` | No | `true` | Create parent directories if needed |
| `template_vars` | `map(any)` | No | `{}` | Variables for template processing |
| `repository_id` | `string` | No | - | Repository resource ID |
| `post_create_hooks` | `list(string)` | No | `[]` | Commands to run after file creation |
| `post_update_hooks` | `list(string)` | No | `[]` | Commands to run after file updates |
| `validate_source_exists` | `bool` | No | `true` | Validate source file exists |
| `validate_target_writable` | `bool` | No | `true` | Validate target is writable |

#### Nested Blocks

##### `enhanced_template`

Advanced template processing configuration.

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `engine` | `string` | No | Provider default | Template engine to use |
| `platform_variables` | `bool` | No | `false` | Include platform-specific variables |
| `strict_mode` | `bool` | No | `false` | Enable strict template validation |
| `custom_functions` | `map(string)` | No | `{}` | Custom template functions |
| `custom_delimiters` | `block` | No | - | Custom template delimiters |

###### `custom_delimiters` (within `enhanced_template`)

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `left` | `string` | Yes | - | Left delimiter |
| `right` | `string` | Yes | - | Right delimiter |

##### `backup_policy`

File-specific backup configuration.

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `enabled` | `bool` | No | Provider default | Enable backups for this file |
| `format` | `string` | No | Provider default | Backup naming format |
| `retention` | `block` | No | - | Retention policy for this file |

###### `retention` (within `backup_policy`)

Same schema as provider-level retention policy.

##### `permissions`

Advanced permission configuration.

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `preserve_source` | `bool` | No | `false` | Preserve source file permissions |
| `inherit_parent` | `bool` | No | `false` | Inherit parent directory permissions |

#### Computed Attributes

| Name | Type | Description |
|------|------|-------------|
| `id` | `string` | Resource identifier |
| `target_exists` | `bool` | Whether target file exists |
| `target_size` | `number` | Target file size in bytes |
| `target_mode` | `string` | Target file permissions |
| `target_modified_time` | `string` | Target file modification time |
| `source_checksum` | `string` | Source file checksum |
| `target_checksum` | `string` | Target file checksum |
| `template_processed` | `bool` | Whether template was processed |
| `backup_created` | `bool` | Whether backup was created |
| `backup_path` | `string` | Path to created backup |

### `dotfiles_symlink`

Creates and manages symbolic links.

#### Arguments

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `source_path` | `string` | Yes | - | Source path for symlink target |
| `target_path` | `string` | Yes | - | Symlink path to create |
| `create_parents` | `bool` | No | `true` | Create parent directories |
| `force` | `bool` | No | `false` | Force creation even if target exists |
| `repository_id` | `string` | No | - | Repository resource ID |

#### Nested Blocks

##### `backup_policy`

Same schema as `dotfiles_file` backup policy.

##### `permissions`

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `link_mode` | `string` | No | `"0755"` | Symlink permissions |
| `follow_target` | `bool` | No | `false` | Apply permissions to target |

#### Computed Attributes

| Name | Type | Description |
|------|------|-------------|
| `id` | `string` | Resource identifier |
| `link_exists` | `bool` | Whether symlink exists |
| `target_exists` | `bool` | Whether symlink target exists |
| `link_target` | `string` | Current symlink target |
| `is_broken` | `bool` | Whether symlink is broken |
| `target_type` | `string` | Type of symlink target (file/directory) |

### `dotfiles_directory`

Manages directory structures and synchronization.

#### Arguments

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `source_path` | `string` | Yes | - | Source directory path |
| `target_path` | `string` | Yes | - | Target directory path |
| `strategy` | `string` | No | `"copy"` | Directory synchronization strategy |
| `recursive` | `bool` | No | `true` | Recursively process subdirectories |
| `create_parents` | `bool` | No | `true` | Create parent directories |
| `exclude_patterns` | `list(string)` | No | `[]` | Patterns to exclude from sync |
| `include_patterns` | `list(string)` | No | `[]` | Patterns to include in sync |
| `repository_id` | `string` | No | - | Repository resource ID |

#### Nested Blocks

##### `permissions`

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `file_mode` | `string` | No | `"0644"` | Default file permissions |
| `directory_mode` | `string` | No | `"0755"` | Default directory permissions |
| `preserve_source` | `bool` | No | `false` | Preserve source permissions |

##### `sync_options`

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `delete_extra` | `bool` | No | `false` | Delete files not in source |
| `update_only` | `bool` | No | `false` | Only update existing files |
| `skip_newer` | `bool` | No | `false` | Skip files newer than source |

#### Computed Attributes

| Name | Type | Description |
|------|------|-------------|
| `id` | `string` | Resource identifier |
| `directory_exists` | `bool` | Whether target directory exists |
| `file_count` | `number` | Number of files in directory |
| `total_size` | `number` | Total size of directory contents |
| `last_synced` | `string` | Last synchronization timestamp |

### `dotfiles_application`

Manages application-specific configuration with detection.

#### Arguments

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `application` | `string` | Yes | - | Application name |
| `strategy` | `string` | No | `"symlink"` | Configuration deployment strategy |
| `skip_if_not_installed` | `bool` | No | `true` | Skip if application not detected |
| `warn_on_version_mismatch` | `bool` | No | `false` | Warn on version constraint violations |
| `repository_id` | `string` | No | - | Repository resource ID |

#### Nested Blocks

##### `detection_methods`

Application detection configuration.

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `command` | `list(string)` | No | `[]` | Command to test for application |
| `file` | `list(string)` | No | `[]` | Files/paths that indicate installation |
| `package_manager` | `map(string)` | No | `{}` | Package names by package manager |
| `brew_cask` | `string` | No | - | Homebrew Cask name |
| `registry_key` | `string` | No | - | Windows registry key |

##### `version_constraints`

Version requirement configuration.

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `min_version` | `string` | No | - | Minimum required version |
| `max_version` | `string` | No | - | Maximum supported version |
| `exact_version` | `string` | No | - | Exact version requirement |

##### `config_mappings`

Configuration file mappings.

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| Source path (key) | `string` | Yes | - | Target path (value) |

#### Computed Attributes

| Name | Type | Description |
|------|------|-------------|
| `id` | `string` | Resource identifier |
| `installed` | `bool` | Whether application is installed |
| `version` | `string` | Detected application version |
| `installation_path` | `string` | Application installation path |
| `detection_method` | `string` | Method used for detection |
| `last_checked` | `string` | Last detection check timestamp |
| `detection_result` | `string` | Detailed detection results |

## Data Sources

### `dotfiles_system`

Provides system information for configuration.

#### Computed Attributes

| Name | Type | Description |
|------|------|-------------|
| `id` | `string` | Data source identifier |
| `platform` | `string` | Operating system platform |
| `architecture` | `string` | System architecture |
| `hostname` | `string` | System hostname |
| `username` | `string` | Current username |
| `home_directory` | `string` | User home directory |
| `config_directory` | `string` | User configuration directory |
| `data_directory` | `string` | User data directory |
| `cache_directory` | `string` | User cache directory |
| `runtime_directory` | `string` | User runtime directory |
| `shell` | `string` | Default user shell |
| `environment_variables` | `map(string)` | Environment variables |

### `dotfiles_file_info`

Provides information about existing files.

#### Arguments

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `path` | `string` | Yes | - | File path to inspect |

#### Computed Attributes

| Name | Type | Description |
|------|------|-------------|
| `id` | `string` | Data source identifier |
| `exists` | `bool` | Whether file exists |
| `size` | `number` | File size in bytes |
| `mode` | `string` | File permissions |
| `is_directory` | `bool` | Whether path is a directory |
| `is_symlink` | `bool` | Whether path is a symlink |
| `symlink_target` | `string` | Symlink target (if applicable) |
| `modified_time` | `string` | Last modification time |
| `checksum` | `string` | File content checksum |
| `mime_type` | `string` | MIME type of file |

## Validation Rules

### Path Validation

All path arguments are validated using the following rules:

- Must not be empty
- Must not contain null bytes
- Must be valid for target platform
- Environment variable expansion is supported (`$VAR`, `${VAR}`)
- Tilde expansion is supported (`~`, `~/path`)

### File Mode Validation

File mode arguments must be valid octal permissions:

- Format: `"0644"`, `"0755"`, etc.
- Range: `0000` to `0777`
- Must be 3 or 4 digits
- Must contain only octal digits (0-7)

### Template Engine Validation

Template engine values must be one of:

- `"go"` - Go text/template
- `"handlebars"` - Handlebars templates
- `"mustache"` - Mustache templates
- `"none"` - No template processing

### Environment Variable Expansion

The provider supports environment variable expansion in string values:

- `$VAR` - Simple variable expansion
- `${VAR}` - Braced variable expansion
- `${VAR:-default}` - Variable with default value
- `${VAR:+value}` - Conditional value if variable is set

## Examples

### Basic Configuration

```hcl
provider "dotfiles" {
  dotfiles_root = "~/dotfiles"
  strategy     = "symlink"
  
  backup_strategy {
    enabled = true
    format  = "timestamped"
  }
}
```

### Advanced Configuration

```hcl
provider "dotfiles" {
  dotfiles_root     = "~/dotfiles"
  strategy         = "symlink"
  template_engine  = "handlebars"
  max_concurrency  = 5
  
  backup_strategy {
    enabled     = true
    directory   = "~/.config/dotfiles/backups"
    format      = "git-style"
    compression = true
    
    retention {
      max_age      = "90d"
      max_count    = 200
      keep_daily   = 14
      keep_weekly  = 8
      keep_monthly = 24
    }
  }
  
  cache_config {
    enabled    = true
    max_size   = 2000
    ttl        = "10m"
  }
}
```

### Resource Examples

```hcl
# Repository management
resource "dotfiles_repository" "main" {
  url        = "git@github.com:user/dotfiles.git"
  local_path = "~/dotfiles"
  branch     = "main"
  submodules = true
  
  auth {
    method       = "ssh"
    ssh_key_path = "~/.ssh/id_ed25519"
  }
}

# File deployment with template
resource "dotfiles_file" "gitconfig" {
  source_path = "git/gitconfig.template"
  target_path = "~/.gitconfig"
  strategy    = "template"
  file_mode   = "0644"
  
  template_vars = {
    name  = "John Doe"
    email = "john@example.com"
  }
  
  enhanced_template {
    engine = "handlebars"
    platform_variables = true
  }
  
  backup_policy {
    enabled = true
    format  = "numbered"
  }
  
  depends_on = [dotfiles_repository.main]
}

# Application configuration
resource "dotfiles_application" "vscode" {
  application = "vscode"
  
  detection_methods {
    command = ["code", "--version"]
    file    = [
      "~/Applications/Visual Studio Code.app",
      "/Applications/Visual Studio Code.app"
    ]
  }
  
  config_mappings = {
    "vscode/settings.json"    = "~/Library/Application Support/Code/User/settings.json"
    "vscode/keybindings.json" = "~/Library/Application Support/Code/User/keybindings.json"
  }
  
  skip_if_not_installed = true
}

# System information
data "dotfiles_system" "current" {}

# File inspection
data "dotfiles_file_info" "existing_config" {
  path = "~/.existing-config"
}
```
