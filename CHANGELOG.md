# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Enhanced file permissions resource with pattern-based validation
- Refactored test architecture to reduce cyclomatic complexity
- New file permissions resource test suite

### Fixed

- Resolved linting errors and GoReleaser configuration deprecation warnings
- Fixed whitespace issues in file permissions resource
- Removed outdated test artifacts and coverage files

### Security

- **BREAKING**: Removed shell command execution from all resources (G204
  vulnerability fix)
- Eliminated arbitrary command execution hooks (`post_create_commands`,
  `post_update_commands`, `pre_destroy_commands`)

### Changed

- Refactored resource schemas to remove shell command fields
- Simplified platform provider interfaces
- Removed service management functionality (moved to terraform-provider-package)

## [0.1.1] - 2024-09-29

### Fixed in v0.1.1

- Symlink drift detection when directory exists instead of symlink
- Platform provider method compatibility issues

### Changed in v0.1.1

- Prepared architecture for v0.1.0 release

## [0.1.0] - 2024-09-28

### Added in v0.1.0

- Initial release of terraform-provider-dotfiles
- File resource for dotfile management
- Symlink resource for configuration linking
- Directory resource for folder synchronization  
- Repository resource for Git repository management
- Application resource for multi-file configurations
- System and file_info data sources
- Template processing with multiple engines (Go, Handlebars, Mustache)
- Platform-aware operations (macOS, Linux, Windows)
- Backup and recovery functionality
- Permission management capabilities

### Security in v0.1.0

- GPG-signed releases
- Secure file operations with proper validation
