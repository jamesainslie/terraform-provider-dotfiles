# Getting Started - Terraform Dotfiles Provider Implementation

## Quick Start Guide

This guide provides step-by-step instructions to begin implementing the Terraform Dotfiles Provider, following the Phase 1 timeline from the implementation plan.

## Prerequisites

### Development Environment Setup
```bash
# Required tools
- Go 1.23+
- Terraform 1.0+
- Make
- Git

# Verify installations
go version      # Should show 1.23+
terraform version  # Should show 1.0+
```

### Platform Testing Setup
```bash
# For cross-platform testing, ensure access to:
- macOS development environment
- Linux environment (VM or container)
- Windows environment (VM or WSL2)
```

## Phase 1 Implementation Steps

### Week 1: Project Foundation

#### Step 1: Update Project Metadata (Day 1)

1. **Update go.mod**
```bash
# Replace module path
sed -i '' 's|github.com/hashicorp/terraform-provider-scaffolding-framework|github.com/jamesainslie/terraform-provider-dotfiles|g' go.mod

# Update go.mod manually if needed
go mod tidy
```

2. **Update main.go**
```go
// Update provider address in main.go
Address: "registry.terraform.io/jamesainslie/dotfiles",
```

3. **Update terraform-registry-manifest.json**
```json
{
    "version": 1,
    "metadata": {
        "protocol_versions": ["6.0"]
    }
}
```

4. **Clean up scaffolding code**
```bash
# Remove example files
rm internal/provider/example_*
rm -rf examples/data-sources/scaffolding_example/
rm -rf examples/resources/scaffolding_example/
rm -rf examples/ephemeral-resources/scaffolding_example/
```

#### Step 2: Create Project Structure (Day 2)

```bash
# Create directory structure
mkdir -p internal/{platform,fileops,template,models,security,state,utils}
mkdir -p tests/{acceptance,fixtures,helpers}
mkdir -p tests/fixtures/{dotfiles,configs}
mkdir -p examples/{basic-setup,complete-environment,team-dotfiles}

# Create initial files
touch internal/platform/{platform.go,detector.go,darwin.go,linux.go}
touch internal/fileops/{operations.go,backup.go,validation.go}
touch internal/template/{engine.go,go_template.go,context.go}
touch internal/models/{repository.go,file.go,symlink.go}
```

#### Step 3: Provider Configuration Schema (Days 3-4)

1. **Update internal/provider/provider.go**
```go
package provider

import (
    "context"
    "github.com/hashicorp/terraform-plugin-framework/provider"
    "github.com/hashicorp/terraform-plugin-framework/provider/schema"
    "github.com/hashicorp/terraform-plugin-framework/types"
)

type DotfilesProvider struct {
    version string
}

type DotfilesProviderModel struct {
    DotfilesRoot       types.String `tfsdk:"dotfiles_root"`
    BackupEnabled      types.Bool   `tfsdk:"backup_enabled"`
    BackupDirectory    types.String `tfsdk:"backup_directory"`
    Strategy           types.String `tfsdk:"strategy"`
    ConflictResolution types.String `tfsdk:"conflict_resolution"`
    TargetPlatform     types.String `tfsdk:"target_platform"`
    AutoDetectPlatform types.Bool   `tfsdk:"auto_detect_platform"`
    DryRun            types.Bool   `tfsdk:"dry_run"`
}

func (p *DotfilesProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
    resp.TypeName = "dotfiles"
    resp.Version = p.version
}

func (p *DotfilesProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
    resp.Schema = schema.Schema{
        MarkdownDescription: "Terraform provider for managing dotfiles in a declarative way",
        Attributes: map[string]schema.Attribute{
            "dotfiles_root": schema.StringAttribute{
                MarkdownDescription: "Root directory of the dotfiles repository",
                Optional:            true,
            },
            "backup_enabled": schema.BoolAttribute{
                MarkdownDescription: "Enable automatic backups of existing files",
                Optional:            true,
            },
            "backup_directory": schema.StringAttribute{
                MarkdownDescription: "Directory to store backup files",
                Optional:            true,
            },
            "strategy": schema.StringAttribute{
                MarkdownDescription: "Default strategy for file management (symlink, copy, template)",
                Optional:            true,
            },
            "conflict_resolution": schema.StringAttribute{
                MarkdownDescription: "How to handle conflicts (backup, overwrite, skip)",
                Optional:            true,
            },
            "target_platform": schema.StringAttribute{
                MarkdownDescription: "Target platform (auto, macos, linux, windows)",
                Optional:            true,
            },
            "auto_detect_platform": schema.BoolAttribute{
                MarkdownDescription: "Automatically detect the target platform",
                Optional:            true,
            },
            "dry_run": schema.BoolAttribute{
                MarkdownDescription: "Preview changes without applying them",
                Optional:            true,
            },
        },
    }
}
```

2. **Create internal/provider/config.go**
```go
package provider

import (
    "context"
    "os"
    "path/filepath"
)

type DotfilesConfig struct {
    DotfilesRoot       string
    BackupEnabled      bool
    BackupDirectory    string
    Strategy           string
    ConflictResolution string
    TargetPlatform     string
    AutoDetectPlatform bool
    DryRun            bool
}

func (c *DotfilesConfig) Validate() error {
    // Validate configuration
    if c.DotfilesRoot == "" {
        homeDir, err := os.UserHomeDir()
        if err != nil {
            return err
        }
        c.DotfilesRoot = filepath.Join(homeDir, "dotfiles")
    }
    
    // Set defaults
    if c.Strategy == "" {
        c.Strategy = "symlink"
    }
    
    if c.ConflictResolution == "" {
        c.ConflictResolution = "backup"
    }
    
    return nil
}
```

#### Step 4: Build and Test (Day 5)

```bash
# Build the provider
go build -v .

# Run initial tests
go test -v ./internal/provider/

# Test provider loading
terraform init
terraform plan  # Should work without errors
```

### Week 2: Platform Abstraction Implementation

#### Step 5: Platform Interface (Day 1)

**Create internal/platform/platform.go**
```go
package platform

import (
    "os"
    "runtime"
)

type PlatformProvider interface {
    GetPlatform() string
    GetArchitecture() string
    GetHomeDir() (string, error)
    GetConfigDir() (string, error)
    ResolvePath(path string) (string, error)
    CreateSymlink(source, target string) error
    CopyFile(source, target string) error
    SetPermissions(path string, mode os.FileMode) error
}

type BasePlatform struct {
    platform     string
    architecture string
}

func DetectPlatform() PlatformProvider {
    switch runtime.GOOS {
    case "darwin":
        return &DarwinProvider{BasePlatform{platform: "darwin", architecture: runtime.GOARCH}}
    case "linux":
        return &LinuxProvider{BasePlatform{platform: "linux", architecture: runtime.GOARCH}}
    case "windows":
        return &WindowsProvider{BasePlatform{platform: "windows", architecture: runtime.GOARCH}}
    default:
        return &LinuxProvider{BasePlatform{platform: runtime.GOOS, architecture: runtime.GOARCH}}
    }
}
```

#### Step 6: Platform Implementations (Days 2-3)

**Create internal/platform/darwin.go**
```go
package platform

import (
    "os"
    "path/filepath"
)

type DarwinProvider struct {
    BasePlatform
}

func (p *DarwinProvider) GetHomeDir() (string, error) {
    return os.UserHomeDir()
}

func (p *DarwinProvider) GetConfigDir() (string, error) {
    home, err := p.GetHomeDir()
    if err != nil {
        return "", err
    }
    return filepath.Join(home, ".config"), nil
}

func (p *DarwinProvider) ResolvePath(path string) (string, error) {
    if filepath.IsAbs(path) {
        return path, nil
    }
    
    if path[0] == '~' {
        home, err := p.GetHomeDir()
        if err != nil {
            return "", err
        }
        return filepath.Join(home, path[1:]), nil
    }
    
    return filepath.Abs(path)
}

// Implement other interface methods...
```

#### Step 7: File Operations (Days 4-5)

**Create internal/fileops/operations.go**
```go
package fileops

import (
    "fmt"
    "io"
    "os"
    "path/filepath"
)

type FileOperations struct {
    dryRun bool
}

func NewFileOperations(dryRun bool) *FileOperations {
    return &FileOperations{dryRun: dryRun}
}

func (f *FileOperations) CopyFile(source, target string) error {
    if f.dryRun {
        fmt.Printf("DRY RUN: Would copy %s to %s\n", source, target)
        return nil
    }
    
    sourceFile, err := os.Open(source)
    if err != nil {
        return err
    }
    defer sourceFile.Close()
    
    err = os.MkdirAll(filepath.Dir(target), 0755)
    if err != nil {
        return err
    }
    
    targetFile, err := os.Create(target)
    if err != nil {
        return err
    }
    defer targetFile.Close()
    
    _, err = io.Copy(targetFile, sourceFile)
    return err
}

func (f *FileOperations) CreateSymlink(source, target string) error {
    if f.dryRun {
        fmt.Printf("DRY RUN: Would create symlink %s -> %s\n", target, source)
        return nil
    }
    
    err := os.MkdirAll(filepath.Dir(target), 0755)
    if err != nil {
        return err
    }
    
    return os.Symlink(source, target)
}
```

### Week 3: Core Resources Implementation

#### Step 8: Repository Resource (Days 1-2)

**Create internal/provider/repository_resource.go**
```go
package provider

import (
    "context"
    "fmt"
    
    "github.com/hashicorp/terraform-plugin-framework/resource"
    "github.com/hashicorp/terraform-plugin-framework/resource/schema"
    "github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &RepositoryResource{}

func NewRepositoryResource() resource.Resource {
    return &RepositoryResource{}
}

type RepositoryResource struct {
    client *DotfilesClient
}

type RepositoryResourceModel struct {
    ID                    types.String `tfsdk:"id"`
    Name                  types.String `tfsdk:"name"`
    SourcePath           types.String `tfsdk:"source_path"`
    Description          types.String `tfsdk:"description"`
    DefaultBackupEnabled types.Bool   `tfsdk:"default_backup_enabled"`
    DefaultFileMode      types.String `tfsdk:"default_file_mode"`
    DefaultDirMode       types.String `tfsdk:"default_dir_mode"`
}

func (r *RepositoryResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
    resp.TypeName = req.ProviderTypeName + "_repository"
}

func (r *RepositoryResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
    resp.Schema = schema.Schema{
        MarkdownDescription: "Manages a dotfiles repository configuration",
        Attributes: map[string]schema.Attribute{
            "id": schema.StringAttribute{
                Computed:            true,
                MarkdownDescription: "Repository identifier",
            },
            "name": schema.StringAttribute{
                Required:            true,
                MarkdownDescription: "Repository name",
            },
            "source_path": schema.StringAttribute{
                Required:            true,
                MarkdownDescription: "Path to the dotfiles repository",
            },
            "description": schema.StringAttribute{
                Optional:            true,
                MarkdownDescription: "Repository description",
            },
            // Add other attributes...
        },
    }
}

// Implement other resource methods (Configure, Create, Read, Update, Delete)...
```

#### Step 9: File Resource (Days 3-4)

Follow similar pattern for `file_resource.go` with file-specific schema and operations.

#### Step 10: Symlink Resource (Day 5)

Implement `symlink_resource.go` with symlink-specific functionality.

### Testing Each Week

#### Daily Testing Routine
```bash
# Run unit tests
go test -v ./internal/...

# Run acceptance tests (when available)
TF_ACC=1 go test -v ./internal/provider/

# Build and manual test
go build -v .
terraform init
terraform plan
```

#### Weekly Integration Testing
```bash
# Create test configuration
cat > test.tf << 'EOF'
terraform {
  required_providers {
    dotfiles = {
      source = "jamesainslie/dotfiles"
    }
  }
}

provider "dotfiles" {
  dotfiles_root = "./test-dotfiles"
  backup_enabled = true
}

resource "dotfiles_repository" "test" {
  name = "test-repo"
  source_path = "./test-dotfiles"
}
EOF

# Test the configuration
terraform init
terraform plan
terraform apply
```

## Development Workflow

### Daily Development Process
1. **Morning**: Review TODO list and select next task
2. **Implementation**: Write code following established patterns
3. **Testing**: Run tests after each significant change
4. **Documentation**: Update code comments and documentation
5. **Evening**: Update TODO list and commit progress

### Code Quality Checks
```bash
# Format code
go fmt ./...

# Run linter
golangci-lint run

# Run security scan
gosec ./...

# Check dependencies
go mod verify
```

### Git Workflow
```bash
# Feature branch workflow
git checkout -b feature/repository-resource
# ... implement feature ...
git add .
git commit -m "feat: implement dotfiles_repository resource"
git push origin feature/repository-resource
# ... create PR ...
```

## Troubleshooting Common Issues

### Build Issues
```bash
# Module path issues
go mod tidy
go clean -modcache

# Dependency conflicts
go get -u all
```

### Testing Issues
```bash
# Provider registration issues
export TF_ACC=1
export TF_LOG=DEBUG

# Platform-specific testing
GOOS=linux go test ./internal/platform/
```

### Development Environment Issues
```bash
# Terraform caching issues
rm -rf .terraform/
terraform init

# Go build cache issues
go clean -cache
```

## Next Steps After Week 1

1. **Review Progress**: Ensure foundation is solid
2. **Platform Testing**: Test on different operating systems
3. **Community Feedback**: Share progress with potential users
4. **Documentation**: Keep docs updated with implementation

## Resource Links

- [Terraform Plugin Framework Documentation](https://developer.hashicorp.com/terraform/plugin/framework)
- [Go Documentation](https://golang.org/doc/)
- [Cross-Platform Go Development](https://golang.org/doc/code)
- [Terraform Provider Best Practices](https://developer.hashicorp.com/terraform/plugin/best-practices)

## Success Metrics for Week 1

- [ ] Provider builds without errors
- [ ] Basic provider configuration works
- [ ] Platform detection functional
- [ ] File operations library created
- [ ] First resource (repository) implemented
- [ ] Tests pass on primary development platform

This guide provides the concrete steps needed to begin implementation. Each step builds on the previous one, creating a solid foundation for the full provider implementation.
