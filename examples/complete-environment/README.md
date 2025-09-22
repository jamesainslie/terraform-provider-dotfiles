# Complete Environment Example

This example demonstrates the full functionality of the Terraform Dotfiles Provider, including:

- File copying with permission management
- Template processing with variables and conditionals
- Symlink creation with parent directory support
- Backup system for conflict resolution
- Cross-platform compatibility

## Features Demonstrated

### File Operations
- **SSH config**: Copied with secure permissions (0600)
- **Cursor settings**: JSON configuration file copying
- **Git config**: Template processing with user variables

### Template Processing  
- **User variables**: Name, email, editor preferences
- **System context**: Platform detection and paths
- **Feature flags**: Conditional configuration based on enabled features
- **Platform-specific content**: macOS vs Linux specific configurations

### Symlink Management
- **Fish configuration**: Directory symlink with parent creation
- **Cross-platform**: Automatic platform detection and appropriate handling

### Backup System
- **Conflict resolution**: Automatic backup of existing files
- **Timestamped backups**: Organized backup storage
- **Safe operations**: Never lose existing configuration

## Quick Start

1. **Set up the example**:
   ```bash
   cd examples/complete-environment
   ```

2. **Initialize Terraform**:
   ```bash
   terraform init
   ```

3. **Plan the changes**:
   ```bash
   terraform plan
   ```

4. **Apply the configuration**:
   ```bash
   terraform apply
   ```

## File Structure

```
test-dotfiles/
├── templates/              # Template files for processing
│   ├── gitconfig.template  # Git config with user variables
│   └── config.fish.template # Fish shell with platform detection
├── ssh/
│   └── config             # SSH configuration
├── fish/
│   └── config.fish        # Basic fish configuration
└── tools/
    └── cursor.json        # Cursor editor settings
```

## Generated Files

After applying, you'll have:

- `~/.ssh/config` - SSH configuration (mode 0600)
- `~/.gitconfig` - Processed Git configuration with your variables
- `~/.config/fish/` - Symlinked fish configuration directory
- `~/.config/fish/config.fish` - Processed fish configuration
- `~/.config/cursor/settings.json` - Cursor editor settings

## Template Variables

The example uses these template variables:

```hcl
template_vars = {
  user_name  = "Test User"
  user_email = "test@example.com"
  editor     = "vim"
  gpg_key    = "ABC123DEF456"
}
```

## System Context

Templates automatically have access to:

- `{{.system.platform}}` - Operating system (macos, linux, windows)
- `{{.system.architecture}}` - CPU architecture
- `{{.system.home_dir}}` - User home directory
- `{{.system.config_dir}}` - Configuration directory

## Features

Templates can use conditional logic:

```
{{if .features.docker}}
# Docker configuration
{{end}}

{{if eq .system.platform "macos"}}
# macOS specific settings
{{end}}
```

## Outputs

The configuration outputs useful information:

- **System information**: Platform, architecture, directories
- **Repository state**: Local path, last update timestamp
- **File states**: Existence, content hashes, modification times
- **Symlink states**: Target validation, symlink verification

## Verification

After applying, verify the setup:

```bash
# Check files were created
ls -la ~/.ssh/config ~/.gitconfig ~/.config/fish/

# Check SSH permissions
stat -c "%a" ~/.ssh/config  # Should be 600

# Check symlink
ls -la ~/.config/fish  # Should show symlink arrow

# Check processed template content
cat ~/.gitconfig  # Should contain your variables, not {{.user_name}}

# Check backups (if any conflicts occurred)
ls -la ./backups/
```

This example demonstrates the complete capabilities of the Terraform Dotfiles Provider for declarative dotfiles management.
