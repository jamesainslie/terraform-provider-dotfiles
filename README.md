# Terraform Dotfiles Provider

A Terraform provider for managing dotfiles in a declarative, cross-platform manner. This provider enables Infrastructure as Code principles for personal development environment configuration.

## Features

- **GitHub Repository Support**: Clone and manage dotfiles from GitHub repositories with secure authentication
- **Cross-Platform**: Works on macOS, Linux, and Windows
- **Secure Authentication**: Supports GitHub Personal Access Tokens and SSH keys
- **Multiple Strategies**: File copying, symbolic linking, and template processing
- **Backup System**: Automatic backup of existing files before modification
- **Platform Detection**: Automatic detection of operating system and application paths

## Quick Start

### Installation

```hcl
terraform {
  required_providers {
    dotfiles = {
      source = "jamesainslie/dotfiles"
    }
  }
}
```

### Basic Usage

```hcl
provider "dotfiles" {
  dotfiles_root = "~/dotfiles"
  backup_enabled = true
  strategy = "symlink"
}

# Local repository
resource "dotfiles_repository" "local" {
  name        = "local-dotfiles"
  source_path = "~/dotfiles"
}

# GitHub repository  
resource "dotfiles_repository" "github" {
  name        = "github-dotfiles"
  source_path = "https://github.com/username/dotfiles.git"
  git_personal_access_token = var.github_token
}

# File management
resource "dotfiles_file" "gitconfig" {
  repository  = dotfiles_repository.github.id
  source_path = "git/gitconfig"
  target_path = "~/.gitconfig"
}

# Symlink management
resource "dotfiles_symlink" "fish_config" {
  repository  = dotfiles_repository.github.id
  source_path = "fish"
  target_path = "~/.config/fish"
}
```

## Resources

- **dotfiles_repository**: Manages dotfiles repositories (local or Git)
- **dotfiles_file**: Manages individual files with templating support
- **dotfiles_symlink**: Creates and manages symbolic links
- **dotfiles_directory**: Manages directory structures

## Data Sources

- **dotfiles_system**: Provides system information (platform, paths, etc.)
- **dotfiles_file_info**: Provides information about existing files

## Authentication

### GitHub Personal Access Token

```hcl
resource "dotfiles_repository" "private" {
  source_path = "https://github.com/user/private-repo.git"
  git_personal_access_token = var.github_token
}
```

### SSH Authentication

```hcl
resource "dotfiles_repository" "ssh" {
  source_path = "git@github.com:user/repo.git"
  git_ssh_private_key_path = "~/.ssh/id_ed25519"
}
```

### Environment Variables

The provider automatically detects these environment variables:
- `GITHUB_TOKEN`
- `GH_TOKEN`

## Platform Support

- **macOS**: Full support with Application Support directories
- **Linux**: XDG Base Directory specification compliance
- **Windows**: AppData directory support with symlink fallbacks

## Documentation

See the [examples](./examples/) directory for complete usage examples:
- [Basic Setup](./examples/basic-setup/): Local dotfiles management
- [GitHub Repository](./examples/github-repository/): GitHub integration with authentication

## Development

### Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.23

### Building

```bash
git clone https://github.com/jamesainslie/terraform-provider-dotfiles
cd terraform-provider-dotfiles
go build -v .
```

### Testing

```bash
# Run all tests
go test ./...

# Run with coverage
go test ./... -cover

# Run acceptance tests
TF_ACC=1 go test -v ./internal/provider/
```

### Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes with tests
4. Ensure all tests pass: `go test ./...`
5. Run documentation generation: `make generate`
6. Submit a pull request

## License

This project is licensed under the MPL-2.0 License - see the [LICENSE](LICENSE) file for details.

## Status

This provider is currently in development. The foundation is complete with GitHub repository support, cross-platform compatibility, and comprehensive testing (40% coverage, 269 tests).

### Current Status
- ✅ Provider framework and configuration
- ✅ GitHub repository support with authentication
- ✅ Cross-platform abstraction layer
- ✅ Resource and data source schemas
- ✅ Comprehensive testing infrastructure

### Planned Features
- 🔄 Template processing engine
- 🔄 Backup and conflict resolution system
- 🔄 Advanced application management
- 🔄 Security features and validation