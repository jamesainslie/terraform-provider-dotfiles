## 0.1.0 (September 22, 2025)

FEATURES:

* **Core Provider Framework**: Complete Terraform provider implementation with configuration validation and default value handling
* **File Management**: Full CRUD operations for individual dotfiles with template processing, backup, and conflict resolution
* **Symlink Management**: Full CRUD operations for creating and managing symbolic links with parent directory creation
* **Repository Management**: Support for local dotfiles repositories with path validation and caching
* **Go Template Engine**: Advanced template processing with custom functions for configuration paths, platform detection, and text manipulation
* **Cross-Platform Support**: Native support for macOS, Linux, and Windows with platform-specific file operations
* **Backup System**: Automatic file backup with configurable backup directories and conflict resolution strategies
* **Git Integration**: Repository validation, URL parsing, and authentication configuration for Git-based dotfiles
* **Data Sources**: System information and file metadata data sources for configuration discovery

RESOURCES:

* `dotfiles_file` - Manages individual configuration files with template processing
* `dotfiles_symlink` - Creates and manages symbolic links to dotfiles
* `dotfiles_directory` - Directory resource for organized configuration management
* `dotfiles_repository` - Local repository resource for dotfiles organization

DATA SOURCES:

* `dotfiles_system` - System information including platform, architecture, and environment details
* `dotfiles_file_info` - File metadata and existence checking for configuration discovery

IMPROVEMENTS:

* **Comprehensive Testing**: 300+ tests with 36.8% code coverage including unit, integration, and end-to-end tests
* **CI/CD Integration**: Complete GitHub Actions workflow with linting, testing, and documentation generation
* **Documentation**: Auto-generated provider documentation with examples and usage guides
* **Error Handling**: Robust error handling with detailed diagnostic messages
* **Logging**: Configurable logging levels for debugging and monitoring
* **Dry Run Mode**: Test configuration changes without applying them

EXAMPLES:

* **Basic Setup**: Simple file and symlink management examples
* **GitHub Integration**: Complete GitHub-based dotfiles repository management
* **Team Configuration**: Multi-user dotfiles management examples
* **Complete Environment**: Comprehensive example showcasing all provider features

TECHNICAL DETAILS:

* **Protocol Version**: Terraform Plugin Protocol v6.0
* **Go Version**: Compatible with Go 1.19+
* **Platforms**: darwin/amd64, linux/amd64, windows/amd64, and additional architectures
* **Dependencies**: Minimal external dependencies with secure Git operations
* **Architecture**: Modular design with separated concerns for file operations, templating, and platform abstraction
