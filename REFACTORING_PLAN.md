# Terraform Provider Dotfiles - Pure Declarative Architecture Refactoring Plan

## Executive Summary

This document outlines a comprehensive refactoring plan to eliminate arbitrary shell command execution from the terraform-provider-dotfiles and replace it with pure declarative resource management. This addresses critical security vulnerabilities, improves maintainability, and aligns with Terraform's declarative paradigm.

**Goal**: Transform imperative shell commands into declarative Terraform resources that manage system state natively in Go.

## ⚠️ Architectural Update

**IMPORTANT**: After further analysis, **service management has been excluded** from this provider's scope to eliminate functional overlap with [terraform-provider-package](https://github.com/jamesainslie/terraform-provider-package), which already provides comprehensive service management.

**Revised Scope**:
- **Dotfiles Provider**: Configuration file operations (files, templates, permissions, backups, notifications)
- **Package Provider**: System-level operations (packages, services, repositories)

This change eliminates duplication and maintains clear separation of concerns. Users should use both providers together for complete system management.

## Table of Contents

- [1. Problem Statement](#1-problem-statement)
- [2. Current Architecture Analysis](#2-current-architecture-analysis)
- [3. Proposed Solution: Pure Declarative Architecture](#3-proposed-solution-pure-declarative-architecture)
- [4. New Resource Design (Revised)](#4-new-resource-design-revised)
- [5. Implementation Roadmap](#5-implementation-roadmap)
- [6. Technical Implementation Details](#6-technical-implementation-details)
- [7. Migration Strategy](#7-migration-strategy)
- [8. Testing Strategy](#8-testing-strategy)
- [9. Documentation Updates](#9-documentation-updates)
- [10. Risk Assessment](#10-risk-assessment)
- [11. Success Metrics](#11-success-metrics)

## 1. Problem Statement

### 1.1 Current Issues

The current implementation allows arbitrary shell command execution through fields like:
- `post_create_commands`
- `post_update_commands` 
- `pre_destroy_commands`

**Critical Problems:**
- **Security Risk**: G204 CodeQL alerts - arbitrary command execution
- **Cross-platform Issues**: Shell commands are platform-specific
- **Non-deterministic Behavior**: Commands can have side effects
- **Poor Testability**: Shell execution is difficult to test reliably
- **Architectural Violation**: Imperative commands in declarative infrastructure

### 1.2 Examples of Current Shell Usage

```hcl
# Current problematic patterns
resource "dotfiles_file" "config" {
  source_path = "app/config"
  target_path = "~/.config/app"
  
  # ❌ Security risk - arbitrary command execution
  post_create_commands = [
    "systemctl --user restart app",
    "chmod 600 ~/.config/app/secrets",
    "killall -HUP app-daemon"
  ]
}
```

## 2. Current Architecture Analysis

### 2.1 Existing Provider Structure

```
Resources: 5
├── dotfiles_repository   (Git repository management)
├── dotfiles_file         ❌ Contains shell command execution
├── dotfiles_symlink      (Symlink management)
├── dotfiles_directory    (Directory synchronization)
└── dotfiles_application  (Multi-file configurations)

Data Sources: 2
├── dotfiles_system       (Platform information)
└── dotfiles_file_info    (File metadata)
```

### 2.2 Shell Command Usage Patterns

Based on codebase analysis, shell commands are used for:

| Use Case | Current Shell Commands | Security Risk |
|----------|----------------------|---------------|
| Service Management | `systemctl restart nginx`, `launchctl load app` | High |
| Permission Changes | `chmod 600 file`, `chown user:group file` | Medium |
| Backup Operations | `cp file file.backup`, `tar czf backup.tar.gz file` | Medium |
| Config Reloads | `source ~/.bashrc`, `fish -c reload` | Medium |
| Application Restarts | `killall Terminal`, `pkill -f "app-name"` | High |
| Notifications | `echo "Config updated"`, `notify-send "Done"` | Low |

## 3. Proposed Solution: Pure Declarative Architecture

### 3.1 Design Principles

1. **Zero Shell Execution**: All operations implemented natively in Go
2. **Declarative State Management**: Resources declare desired state, not actions
3. **Cross-platform Consistency**: Platform abstraction for all operations
4. **Type Safety**: Strongly typed resource configurations
5. **Idempotent Operations**: All operations can be run multiple times safely
6. **Terraform-native**: Follow Terraform framework patterns and conventions

### 3.2 Architecture Transformation

```
BEFORE: Imperative Commands
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Terraform     │───▶│  Dotfiles       │───▶│  Shell Commands │
│   Configuration │    │  Provider       │    │  (Arbitrary)    │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                │
                                ▼
                       ❌ SECURITY RISK

AFTER: Declarative Resources  
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Terraform     │───▶│  Dotfiles       │───▶│  Native Go      │
│   Configuration │    │  Provider       │    │  Operations     │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                │
                                ▼
                       ✅ SECURE & DECLARATIVE
```

## 4. New Resource Design (Revised)

**⚠️ Service Management Removed**: Service management (originally planned as `dotfiles_service`) has been excluded to avoid overlap with [terraform-provider-package](https://github.com/jamesainslie/terraform-provider-package). Use `pkg_service` from the package provider for service management operations.

**Revised Focus**: Dotfiles-specific configuration file management operations only.

### 4.1 File Permission Resource

**Purpose**: Replace permission-changing shell commands with native permission management.

```hcl
# New declarative permission management
resource "dotfiles_file_permissions" "ssh_key_perms" {
  path      = "~/.ssh/id_rsa"
  mode      = "0600"
  owner     = "current_user"  # current_user, specific username
  group     = "current_group" # current_group, specific group  
  recursive = false
  
  # Platform-specific behaviors
  follow_symlinks = false
  apply_to_parent = false
  
  depends_on = [dotfiles_file.ssh_key]
}

# Batch permission management
resource "dotfiles_file_permissions" "ssh_directory_perms" {
  path      = "~/.ssh"
  mode      = "0700"
  recursive = true
  
  # Pattern-based permissions
  file_patterns = {
    "id_*"     = "0600"
    "*.pub"    = "0644"
    "config"   = "0600"
    "known_hosts" = "0644"
  }
  
  depends_on = [dotfiles_directory.ssh_dir]
}
```

### 4.2 Application State Resource

**Purpose**: Replace application restart/reload commands with declarative application state management.

```hcl
# New declarative application management
resource "dotfiles_application_state" "terminal_restart" {
  application     = "Terminal"
  desired_state   = "restarted"     # running, stopped, restarted
  restart_method  = "graceful"      # graceful, force, signal
  signal          = "SIGHUP"        # for signal-based restarts
  timeout         = "10s"
  
  # Restart conditions
  restart_when = {
    config_changed = dotfiles_file.terminal_config.content_hash
    always         = false
  }
  
  # Platform-specific application identifiers
  identifiers = {
    darwin  = "com.apple.Terminal"
    linux   = "gnome-terminal"  
    windows = "WindowsTerminal.exe"
  }
  
  depends_on = [dotfiles_file.terminal_config]
}
```

### 4.3 Backup State Resource

**Purpose**: Replace backup shell commands with declarative backup management.

```hcl
# New declarative backup management
resource "dotfiles_backup" "config_safety_backup" {
  source_path = "~/.vimrc"
  backup_path = "~/.dotfiles-backups/vimrc"
  
  # Backup triggers
  triggers = ["destroy", "update", "manual"]
  
  # Backup format and behavior
  format         = "copy"        # copy, archive, git_commit
  compression    = true          # for archive format
  retention      = 10            # keep last 10 backups
  
  # Backup validation
  verify_backup = true
  checksum_type = "sha256"
  
  # Conditional backup
  backup_when = {
    file_exists    = true
    size_gt        = "0"
    modified_since = "1h"
  }
  
  depends_on = [dotfiles_file.vimrc]
}
```

### 4.4 Directory State Resource

**Purpose**: Replace directory creation/management commands with declarative directory state.

```hcl
# New declarative directory management
resource "dotfiles_directory_state" "config_directories" {
  directories = {
    "~/.config/fish" = {
      mode           = "0755"
      create_parents = true
      ensure         = "present"  # present, absent
    }
    "~/.local/share/apps" = {
      mode           = "0755"  
      create_parents = true
      owner          = "current_user"
    }
  }
  
  # Cleanup behavior
  cleanup_empty = false
  remove_on_destroy = false
}
```

### 4.5 Notification Resource

**Purpose**: Replace notification shell commands with declarative notification management.

```hcl
# New declarative notification system
resource "dotfiles_notification" "config_update_alert" {
  message = "Dotfiles configuration updated successfully"
  
  # Notification types
  desktop {
    enabled = true
    title   = "Dotfiles Update"
    icon    = "info"
    timeout = "5s"
  }
  
  log {
    enabled = true
    level   = "info"
    format  = "json"
  }
  
  webhook {
    enabled = false
    url     = "https://hooks.slack.com/..."
    payload = jsonencode({
      text = "Dotfiles updated on ${data.dotfiles_system.current.hostname}"
    })
  }
  
  # Conditional notifications
  notify_when = {
    always      = false
    on_change   = true
    on_error    = true
  }
  
  depends_on = [dotfiles_file.main_config]
}
```

### 4.7 Enhanced File Resource

**Purpose**: Remove shell command fields and add built-in declarative operations.

```hcl
# Enhanced file resource (no shell commands)
resource "dotfiles_file" "gitconfig" {
  repository  = dotfiles_repository.main.id
  name        = "git-config"
  source_path = "git/gitconfig.template"
  target_path = "~/.gitconfig"
  is_template = true
  
  # Built-in permission management
  permissions {
    mode              = "0644"
    apply_immediately = true
  }
  
  # Built-in backup management  
  backup_policy {
    enabled         = true
    format          = "timestamped"
    retention_count = 5
    verify_restore  = true
  }
  
  # Built-in validation
  validation {
    syntax_check = {
      enabled = true
      command = "git config --list --file={{.path}}"
    }
    
    required_commands = ["git"]
    
    content_rules = {
      max_size = "10MB"
      encoding = "utf-8"
    }
  }
  
  # Template processing
  template_vars = {
    user_name  = "John Doe"
    user_email = "john@example.com"
  }
}

# Service management moved to terraform-provider-package
# Use pkg_service for system-level service management
resource "pkg_service" "git_daemon" {
  service_name = "git-daemon"  # if applicable
  state        = "restarted"
  
  depends_on = [dotfiles_file.gitconfig]
}

# Or use dotfiles application state for process management
resource "dotfiles_application_state" "git_reload" {
  application     = "git"
  desired_state   = "config_reloaded"
  signal          = "SIGHUP"
  
  depends_on = [dotfiles_file.gitconfig]
}
```

## 5. Implementation Roadmap

### 5.1 Phase 1: Foundation Infrastructure (Weeks 1-2)

#### Week 1: Core Interfaces and Abstractions
- **Create base resource interfaces**
  ```go
  type DeclarativeResource interface {
      GetDesiredState() interface{}
      GetActualState(ctx context.Context) (interface{}, error)
      ApplyState(ctx context.Context) error
      ValidateState() error
      DetectDrift() (bool, error)
  }
  ```

- **Build platform abstraction layer**
  ```go
  type PlatformManager interface {
      // ServiceManager() removed - use terraform-provider-package instead
      FileManager() FileManagerInterface  
      ProcessManager() ProcessManagerInterface
      NotificationManager() NotificationManagerInterface
  }
  ```

- **Implement state management system**
  - Resource state tracking
  - Drift detection mechanisms
  - Rollback capabilities

#### Week 2: Logging and Validation Framework
- **Enhanced logging system**
  - Structured logging with context
  - Operation tracing
  - Performance metrics

- **Comprehensive validation framework**
  - Type-safe resource validation
  - Cross-resource dependency validation
  - Platform compatibility checking

### 5.2 Phase 2: Essential Resources Implementation (Weeks 3-5)

#### Week 3: Service Management Resource
```go
// Implementation priority: HIGH
type ServiceResource struct {
    Name          string
    DesiredState  string  // running, stopped, restarted, reloaded
    Scope         string  // user, system
    RestartMethod string  // graceful, force, signal
    Timeout       time.Duration
    Platform      platform.Provider
}
```

**Features to implement:**
- Cross-platform service detection
- systemd integration (Linux)
- launchd integration (macOS)
- Windows service manager integration
- User vs system service handling
- Graceful vs forced restart methods

#### Week 4: File Permission Resource
```go
// Implementation priority: HIGH
type FilePermissionResource struct {
    Path          string
    Mode          os.FileMode
    Owner         string
    Group         string
    Recursive     bool
    FollowSymlinks bool
    FilePatterns  map[string]string
}
```

**Features to implement:**
- Native Go permission management
- Recursive permission application
- Pattern-based permissions
- Platform-specific permission handling
- Symlink permission handling

#### Week 5: Application State Resource
```go
// Implementation priority: MEDIUM
type ApplicationStateResource struct {
    Application   string
    DesiredState  string
    RestartMethod string
    Signal        string
    Timeout       time.Duration
    Identifiers   map[string]string  // platform -> identifier
}
```

**Features to implement:**
- Cross-platform process management
- Application identification strategies
- Signal-based restart mechanisms
- Graceful shutdown procedures

### 5.3 Phase 3: Advanced State Management (Weeks 6-7)

#### Week 6: Backup and Directory Resources
- **Backup State Resource**
  - Multiple backup formats (copy, archive, git)
  - Retention policies
  - Backup verification and validation
  - Incremental backup support

- **Directory State Resource**  
  - Recursive directory creation
  - Permission propagation
  - Directory cleanup policies
  - Parent directory handling

#### Week 7: Notification and Enhanced File Resources
- **Notification Resource**
  - Desktop notifications (cross-platform)
  - Webhook integrations
  - Structured logging
  - Conditional notification triggers

- **Enhanced File Resource**
  - Remove all shell command fields
  - Integrate permission management
  - Add built-in validation
  - Expand template processing

### 5.4 Phase 4: Migration and Polish (Weeks 8-9)

#### Week 8: Migration Tooling
- **Configuration migration tool**
  ```bash
  terraform-provider-dotfiles migrate \
    --input ./main.tf \
    --output ./main-declarative.tf \
    --backup ./main.tf.backup
  ```

- **Migration validation**
  - Detect unconvertible patterns
  - Generate migration reports
  - Validate converted configurations

#### Week 9: Documentation and Testing
- **Comprehensive documentation**
  - Migration guides
  - Resource reference documentation
  - Best practices guide
  - Example configurations

- **Performance optimization**
  - Resource operation batching
  - Dependency graph optimization
  - Memory usage optimization

## 6. Technical Implementation Details

### 6.1 Resource State Management Pattern

```go
// Universal pattern for all declarative resources
type ResourceState interface {
    // Get the desired state as declared in Terraform configuration
    GetDesiredState() interface{}
    
    // Get the actual current state of the managed resource
    GetActualState(ctx context.Context) (interface{}, error)
    
    // Apply changes to make actual state match desired state
    ApplyState(ctx context.Context) error
    
    // Validate that the desired state is achievable and valid
    ValidateState() error
    
    // Detect if actual state has drifted from desired state
    DetectDrift() (bool, error)
    
    // Get a human-readable description of the resource
    Description() string
}

// Example implementation for service resource
type ServiceResourceState struct {
    Name         string
    DesiredState string
    Scope        string
    Platform     platform.Provider
}

func (s *ServiceResourceState) GetDesiredState() interface{} {
    return ServiceState{
        Name:  s.Name,
        State: s.DesiredState,
        Scope: s.Scope,
    }
}

func (s *ServiceResourceState) GetActualState(ctx context.Context) (interface{}, error) {
    manager := s.Platform.ServiceManager()
    status, err := manager.GetServiceStatus(s.Name, s.Scope == "user")
    if err != nil {
        return nil, fmt.Errorf("failed to get service status: %w", err)
    }
    
    return ServiceState{
        Name:  s.Name,
        State: status.State,
        Scope: s.Scope,
    }, nil
}

func (s *ServiceResourceState) ApplyState(ctx context.Context) error {
    manager := s.Platform.ServiceManager()
    userLevel := s.Scope == "user"
    
    switch s.DesiredState {
    case "running":
        return manager.StartService(s.Name, userLevel)
    case "stopped":
        return manager.StopService(s.Name, userLevel)
    case "restarted":
        return manager.RestartService(s.Name, userLevel)
    case "reloaded":
        return manager.ReloadService(s.Name, userLevel)
    default:
        return fmt.Errorf("unsupported desired state: %s", s.DesiredState)
    }
}
```

### 6.2 Cross-Platform Abstraction Layer

```go
// Platform abstraction interfaces
type ServiceManager interface {
    StartService(name string, userLevel bool) error
    StopService(name string, userLevel bool) error
    RestartService(name string, userLevel bool) error
    ReloadService(name string, userLevel bool) error
    GetServiceStatus(name string, userLevel bool) (ServiceStatus, error)
    ServiceExists(name string, userLevel bool) bool
}

type FileManager interface {
    SetPermissions(path string, mode os.FileMode) error
    SetOwnership(path string, owner, group string) error
    CreateBackup(source, backup string, format BackupFormat) error
    ValidateBackup(backup string) error
}

type ProcessManager interface {
    FindProcessesByName(name string) ([]Process, error)
    SendSignalToProcess(pid int, signal os.Signal) error
    TerminateProcess(pid int, graceful bool) error
    IsProcessRunning(pid int) bool
}

type NotificationManager interface {
    SendDesktopNotification(title, message string, level NotificationLevel) error
    WriteLogNotification(message string, level LogLevel) error
    SendWebhookNotification(url string, payload interface{}) error
}

// Platform-specific implementations
type SystemdServiceManager struct{}  // Linux
type LaunchdServiceManager struct{}  // macOS  
type WindowsServiceManager struct{}  // Windows

type UnixFileManager struct{}        // Unix-like systems
type WindowsFileManager struct{}     // Windows

type UnixProcessManager struct{}     // Unix-like systems
type WindowsProcessManager struct{}  // Windows
```

### 6.3 Resource Schema Definitions

```go
// Service resource schema
func serviceResourceSchema() schema.Schema {
    return schema.Schema{
        Description: "Manages system service state",
        Attributes: map[string]schema.Attribute{
            "name": schema.StringAttribute{
                Description: "Service name",
                Required:    true,
            },
            "desired_state": schema.StringAttribute{
                Description: "Desired service state",
                Required:    true,
                Validators: []validator.String{
                    stringvalidator.OneOf("running", "stopped", "restarted", "reloaded"),
                },
            },
            "scope": schema.StringAttribute{
                Description: "Service scope (user or system)",
                Optional:    true,
                Computed:    true,
                Default:     stringdefault.StaticString("user"),
                Validators: []validator.String{
                    stringvalidator.OneOf("user", "system"),
                },
            },
            "restart_method": schema.StringAttribute{
                Description: "How to restart the service",
                Optional:    true,
                Computed:    true,
                Default:     stringdefault.StaticString("graceful"),
                Validators: []validator.String{
                    stringvalidator.OneOf("graceful", "force", "signal"),
                },
            },
            "timeout": schema.StringAttribute{
                Description: "Timeout for service operations",
                Optional:    true,
                Computed:    true,
                Default:     stringdefault.StaticString("30s"),
            },
            "only_if": schema.SingleNestedAttribute{
                Description: "Conditions for when to manage the service",
                Optional:    true,
                Attributes: map[string]schema.Attribute{
                    "service_exists": schema.BoolAttribute{
                        Description: "Only manage if service exists",
                        Optional:    true,
                    },
                    "file_changed": schema.StringAttribute{
                        Description: "Only manage if file hash changed",
                        Optional:    true,
                    },
                },
            },
        },
    }
}
```

### 6.4 Error Handling and Validation

```go
// Comprehensive error handling
type ResourceError struct {
    Type        string
    Resource    string
    Operation   string
    Underlying  error
    Context     map[string]interface{}
    Recoverable bool
}

func (r *ResourceError) Error() string {
    return fmt.Sprintf("%s error in %s during %s: %v", 
        r.Type, r.Resource, r.Operation, r.Underlying)
}

// Validation framework
type ResourceValidator struct {
    Rules []ValidationRule
}

type ValidationRule interface {
    Validate(ctx context.Context, resource interface{}) error
    Description() string
}

// Example validation rules
type ServiceExistsRule struct {
    ServiceName string
    Scope       string
}

func (s *ServiceExistsRule) Validate(ctx context.Context, resource interface{}) error {
    // Check if service exists before trying to manage it
    manager := platform.GetServiceManager()
    if !manager.ServiceExists(s.ServiceName, s.Scope == "user") {
        return fmt.Errorf("service %s does not exist in %s scope", s.ServiceName, s.Scope)
    }
    return nil
}
```

## 7. Migration Strategy

### 7.1 Deprecation Timeline

#### Phase 1: Soft Deprecation (v1.2.0)
- Add deprecation warnings for shell command fields
- Provide migration examples in warnings
- Document new declarative resources
- Maintain full backward compatibility

```go
// Example warning implementation
if !data.PostCreateCommands.IsNull() && !data.PostCreateCommands.IsUnknown() {
    resp.Diagnostics.AddWarning(
        "Shell commands are deprecated",
        "The 'post_create_commands' field is deprecated and will be removed in v2.0.0. "+
        "Consider using the new 'dotfiles_service' resource instead. "+
        "See migration guide: https://registry.terraform.io/providers/jamesainslie/dotfiles/latest/docs/guides/migration",
    )
}
```

#### Phase 2: Migration Period (v1.3.0 - v1.9.0)  
- Release migration tool
- Provide comprehensive examples
- Add configuration validation warnings
- Gradual feature enhancement of new resources

#### Phase 3: Hard Deprecation (v2.0.0)
- Remove shell command fields entirely
- Breaking change with major version bump
- Comprehensive upgrade guide
- Migration tool support

### 7.2 Migration Tool Implementation

```bash
#!/bin/bash
# terraform-provider-dotfiles migration tool

# Usage: migrate-config --input main.tf --output main-v2.tf --format hcl

terraform-provider-dotfiles migrate-config \
    --input ./config \
    --output ./config-v2 \
    --format hcl \
    --validate \
    --backup
```

**Migration Tool Features:**
- Parse existing Terraform configurations
- Identify shell command usage patterns
- Convert to equivalent declarative resources
- Generate migration reports
- Validate converted configurations
- Create backup copies of original files

### 7.3 Common Migration Patterns

#### Pattern 1: Service Management
```hcl
# BEFORE (v1.x)
resource "dotfiles_file" "nginx_config" {
  source_path = "nginx/nginx.conf"
  target_path = "/etc/nginx/nginx.conf"
  
  post_create_commands = ["systemctl restart nginx"]
  post_update_commands = ["systemctl reload nginx"]
}

# AFTER (v2.x) - Use package provider for service management
resource "dotfiles_file" "nginx_config" {
  source_path = "nginx/nginx.conf"
  target_path = "/etc/nginx/nginx.conf"
}

# Use terraform-provider-package for service management
resource "pkg_service" "nginx" {
  service_name = "nginx"
  state        = "running"
  startup      = "enabled"
  
  depends_on = [dotfiles_file.nginx_config]
}
```

#### Pattern 2: Permission Management
```hcl
# BEFORE (v1.x)
resource "dotfiles_file" "ssh_key" {
  source_path = "ssh/id_rsa"
  target_path = "~/.ssh/id_rsa"
  
  post_create_commands = ["chmod 600 ~/.ssh/id_rsa"]
}

# AFTER (v2.x)
resource "dotfiles_file" "ssh_key" {
  source_path = "ssh/id_rsa"
  target_path = "~/.ssh/id_rsa"
  
  permissions {
    mode = "0600"
    apply_immediately = true
  }
}
```

#### Pattern 3: Backup Operations
```hcl
# BEFORE (v1.x)
resource "dotfiles_file" "important_config" {
  source_path = "app/config.yml"
  target_path = "~/.config/app/config.yml"
  
  pre_destroy_commands = ["cp ~/.config/app/config.yml ~/.config/app/config.yml.backup"]
}

# AFTER (v2.x)
resource "dotfiles_file" "important_config" {
  source_path = "app/config.yml" 
  target_path = "~/.config/app/config.yml"
  
  backup_policy {
    enabled = true
    format = "timestamped"
    triggers = ["destroy", "update"]
  }
}
```

## 8. Testing Strategy

### 8.1 Unit Testing

```go
// Example unit tests for service resource
func TestServiceResource_Apply(t *testing.T) {
    tests := []struct {
        name          string
        desiredState  string
        currentState  string
        shouldRestart bool
        expectError   bool
    }{
        {
            name:          "start stopped service",
            desiredState:  "running",
            currentState:  "stopped", 
            shouldRestart: true,
            expectError:   false,
        },
        {
            name:          "restart running service",
            desiredState:  "restarted",
            currentState:  "running",
            shouldRestart: true, 
            expectError:   false,
        },
        {
            name:          "invalid desired state",
            desiredState:  "invalid",
            currentState:  "running",
            shouldRestart: false,
            expectError:   true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            mockManager := &MockServiceManager{
                CurrentState: tt.currentState,
            }
            
            resource := &ServiceResource{
                Name:         "test-service",
                DesiredState: tt.desiredState,
                Manager:      mockManager,
            }

            err := resource.Apply(context.Background())
            
            if tt.expectError && err == nil {
                t.Error("expected error but got none")
            }
            if !tt.expectError && err != nil {
                t.Errorf("unexpected error: %v", err)
            }
            
            if tt.shouldRestart && !mockManager.RestartCalled {
                t.Error("expected service restart but it wasn't called")
            }
        })
    }
}
```

### 8.2 Integration Testing

```go
// Cross-platform integration tests
func TestServiceManagement_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    // Create a test service for each platform
    testCases := map[string]struct {
        serviceName string
        setup       func() error
        cleanup     func() error
    }{
        "linux": {
            serviceName: "test-dotfiles-service",
            setup:       setupSystemdTestService,
            cleanup:     cleanupSystemdTestService,
        },
        "darwin": {
            serviceName: "com.test.dotfiles-service", 
            setup:       setupLaunchdTestService,
            cleanup:     cleanupLaunchdTestService,
        },
    }

    testCase, ok := testCases[runtime.GOOS]
    if !ok {
        t.Skipf("no test case for platform: %s", runtime.GOOS)
    }

    // Setup test service
    if err := testCase.setup(); err != nil {
        t.Fatalf("failed to setup test service: %v", err)
    }
    defer testCase.cleanup()

    // Test service management operations
    manager := platform.GetServiceManager()
    
    // Test start
    err := manager.StartService(testCase.serviceName, true)
    if err != nil {
        t.Errorf("failed to start service: %v", err)
    }
    
    // Test status
    status, err := manager.GetServiceStatus(testCase.serviceName, true)
    if err != nil {
        t.Errorf("failed to get service status: %v", err)
    }
    
    if status.State != "running" {
        t.Errorf("expected service to be running, got: %s", status.State)
    }
    
    // Test stop
    err = manager.StopService(testCase.serviceName, true)
    if err != nil {
        t.Errorf("failed to stop service: %v", err)
    }
}
```

### 8.3 Acceptance Testing

```go
// Terraform acceptance tests
func TestAccFileResource_WithServiceRestart(t *testing.T) {
    resource.Test(t, resource.TestCase{
        ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
        Steps: []resource.TestStep{
            {
                Config: testAccFileResourceWithServiceRestart(),
                Check: resource.ComposeAggregateTestCheckFunc(
                    resource.TestCheckResourceAttr("dotfiles_file.config", "target_path", "/tmp/test-config"),
                    resource.TestCheckResourceAttr("pkg_service.restart", "service_name", "test-service"),
                    resource.TestCheckResourceAttr("pkg_service.restart", "state", "running"),
                ),
            },
        },
    })
}

func testAccFileResourceWithServiceRestart() string {
    return `
resource "dotfiles_repository" "test" {
  name        = "test-repo"
  source_path = "/tmp/test-dotfiles"
}

resource "dotfiles_file" "config" {
  repository  = dotfiles_repository.test.id
  name        = "test-config"
  source_path = "config.yml"
  target_path = "/tmp/test-config"
}

# Use terraform-provider-package for service management
resource "pkg_service" "restart" {
  service_name = "test-service"
  state        = "running"
  startup      = "enabled"
  depends_on   = [dotfiles_file.config]
}
`
}
```

## 9. Documentation Updates

### 9.1 Migration Documentation

**New Documentation Structure:**
```
docs/
├── guides/
│   ├── migration-v1-to-v2.md
│   ├── shell-commands-to-resources.md
│   └── best-practices-v2.md
├── resources/
│   ├── service.md
│   ├── file_permissions.md
│   ├── application_state.md
│   ├── backup.md
│   ├── directory_state.md
│   └── notification.md
└── examples/
    ├── service-management.md
    ├── permission-management.md
    └── backup-strategies.md
```

### 9.2 Resource Documentation Template

```markdown
# Service Management - Use Package Provider

Service management has been moved to terraform-provider-package.
Use `pkg_service` from the package provider for all service operations.

## Example Usage - Use Package Provider

```hcl
# Use terraform-provider-package for service management
resource "pkg_service" "nginx" {
  service_name = "nginx"
  state        = "running"
  startup      = "enabled"
  
  health_check = {
    type = "http"
    url  = "http://localhost:80"
  }
}
```

## Package Provider Reference

See [terraform-provider-package documentation](https://registry.terraform.io/providers/jamesainslie/package) for complete `pkg_service` resource reference.

Key attributes:
- `service_name` (Required) - Service name to manage
- `state` (Required) - Desired service state: `running`, `stopped`
- `startup` (Optional) - Service startup configuration: `enabled`, `disabled`
- `health_check` (Optional) - Health check configuration for monitoring
- `management_strategy` (Optional) - Platform-specific management strategy

## Platform Support

| Platform | Service Manager | User Services | System Services |
|----------|----------------|---------------|-----------------|
| Linux    | systemd        | ✅            | ✅              |
| macOS    | launchd        | ✅            | ✅              |
| Windows  | Service Control| ❌            | ✅              |

## Migration from Shell Commands

### Before (v1.x)
```hcl
resource "dotfiles_file" "config" {
  source_path = "nginx/nginx.conf"
  target_path = "/etc/nginx/nginx.conf"
  post_create_commands = ["systemctl restart nginx"]
}
```

### After (v2.x) - Use Package Provider
```hcl
resource "dotfiles_file" "config" {
  source_path = "nginx/nginx.conf"
  target_path = "/etc/nginx/nginx.conf"
}

# Use terraform-provider-package for service management
resource "pkg_service" "nginx" {
  service_name = "nginx"
  state        = "running"
  startup      = "enabled"
  depends_on   = [dotfiles_file.config]
}
```
```

## 10. Risk Assessment

### 10.1 Technical Risks

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| **Breaking Changes** | High | High | Comprehensive migration tools and documentation |
| **Cross-platform Compatibility** | Medium | Medium | Extensive testing on all target platforms |
| **Performance Regression** | Low | Low | Benchmarking and performance testing |
| **Resource Dependency Issues** | Medium | Medium | Dependency graph validation and testing |

### 10.2 User Experience Risks

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| **Migration Complexity** | High | Medium | Automated migration tools and clear guides |
| **Feature Parity Loss** | Medium | Low | Comprehensive feature mapping and testing |
| **Learning Curve** | Medium | High | Examples, tutorials, and gradual migration path |

### 10.3 Security Risks

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| **Privilege Escalation** | High | Low | Careful permission validation and principle of least privilege |
| **Service Management Abuse** | Medium | Low | Service existence validation and scope restrictions |
| **Path Traversal** | Medium | Low | Path validation and home directory restrictions |

## 11. Success Metrics

### 11.1 Security Metrics
- ✅ **Zero CodeQL G204 alerts** - Complete elimination of shell command execution
- ✅ **Reduced attack surface** - No arbitrary command execution vectors
- ✅ **Improved permission handling** - Native Go permission management

### 11.2 Quality Metrics  
- ✅ **100% test coverage** for new declarative resources
- ✅ **Cross-platform compatibility** - All resources work on Linux, macOS, Windows
- ✅ **Performance benchmarks** - No significant performance regression

### 11.3 User Experience Metrics
- ✅ **Migration success rate** - >90% of configurations can be automatically migrated
- ✅ **Feature parity** - All shell command use cases covered by declarative resources
- ✅ **Documentation completeness** - Migration guides and examples for all resources

### 11.4 Adoption Metrics
- ✅ **Community feedback** - Positive reception of new declarative approach
- ✅ **Issue reduction** - Fewer security and compatibility related issues
- ✅ **Terraform best practices alignment** - Better adherence to Terraform patterns

---

## Conclusion

This refactoring plan transforms the terraform-provider-dotfiles from an imperative, shell-command-based system to a pure declarative resource management system. This addresses critical security vulnerabilities while improving maintainability, testability, and user experience.

The phased approach ensures minimal disruption to existing users while providing a clear migration path. The comprehensive testing strategy and extensive documentation will ensure a smooth transition to the new architecture.

**Key Benefits:**
- ✅ **Eliminates security risks** from arbitrary shell command execution
- ✅ **Improves reliability** through native Go implementations
- ✅ **Enhances cross-platform support** with unified abstractions
- ✅ **Maintains Terraform best practices** with pure declarative resources
- ✅ **Provides better user experience** with type-safe configurations

This refactoring represents a fundamental improvement in the provider's architecture and positions it as a secure, reliable solution for dotfiles management in the Terraform ecosystem.
