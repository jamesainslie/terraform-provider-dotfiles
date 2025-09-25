// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

package provider

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/jamesainslie/terraform-provider-dotfiles/internal/platform"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ApplicationResource{}

func NewApplicationResource() resource.Resource {
	return &ApplicationResource{}
}

// ApplicationResource defines the application resource implementation.
type ApplicationResource struct {
	client *DotfilesClient
}

// ApplicationResourceModel describes the application resource data model.
type ApplicationResourceModel struct {
	ID                   types.String `tfsdk:"id"`
	Repository           types.String `tfsdk:"repository"`
	Application          types.String `tfsdk:"application"`
	SourcePath           types.String `tfsdk:"source_path"`
	DetectInstallation   types.Bool   `tfsdk:"detect_installation"`
	SkipIfNotInstalled   types.Bool   `tfsdk:"skip_if_not_installed"`
	WarnIfNotInstalled   types.Bool   `tfsdk:"warn_if_not_installed"`
	MinVersion           types.String `tfsdk:"min_version"`
	MaxVersion           types.String `tfsdk:"max_version"`
	ConditionalOperation types.Bool   `tfsdk:"conditional"`
	ConfigStrategy       types.String `tfsdk:"config_strategy"`

	// Detection configuration blocks
	DetectionMethods []DetectionMethodModel `tfsdk:"detection_methods"`
	ConfigMappings   *ConfigMappingModel    `tfsdk:"config_mappings"`

	// Computed attributes
	Installed        types.Bool   `tfsdk:"installed"`
	Version          types.String `tfsdk:"version"`
	InstallationPath types.String `tfsdk:"installation_path"`
	LastChecked      types.String `tfsdk:"last_checked"`
	DetectionResult  types.String `tfsdk:"detection_result"`
}

func (r *ApplicationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application"
}

func (r *ApplicationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages application-specific dotfiles with conditional installation detection",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Application resource identifier",
			},
			"repository": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Repository ID this application belongs to",
			},
			"application": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Application name/identifier",
			},
			"source_path": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Path to application configuration in repository",
			},
			"detect_installation": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				MarkdownDescription: "Enable application installation detection",
			},
			"skip_if_not_installed": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Skip configuration if application is not installed",
			},
			"warn_if_not_installed": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				MarkdownDescription: "Warn if application is not installed",
			},
			"min_version": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Minimum required application version",
			},
			"max_version": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Maximum supported application version",
			},
			"conditional": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				MarkdownDescription: "Enable conditional configuration based on installation",
			},
			"config_strategy": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("symlink"),
				MarkdownDescription: "Configuration strategy: symlink, copy, merge, template",
			},
			"installed": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the application is detected as installed",
			},
			"version": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Detected application version",
			},
			"installation_path": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Path where application is installed",
			},
			"last_checked": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Last time installation was checked",
			},
			"detection_result": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Result of the last detection attempt",
			},
		},
		Blocks: map[string]schema.Block{
			"detection_methods": GetDetectionMethodsSchemaBlock(),
			"config_mappings":   GetConfigMappingsSchemaBlock(),
		},
	}
}

func (r *ApplicationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*DotfilesClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			"Expected *DotfilesClient, got something else. Please report this issue to the provider developers.",
		)
		return
	}

	r.client = client
}

func (r *ApplicationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ApplicationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating application resource", map[string]interface{}{
		"application":         data.Application.ValueString(),
		"source_path":         data.SourcePath.ValueString(),
		"detect_installation": data.DetectInstallation.ValueBool(),
	})

	// Perform application detection
	detectionResult := r.performApplicationDetection(ctx, &data)

	// Update computed attributes with detection results
	data.Installed = types.BoolValue(detectionResult.Installed)
	data.Version = types.StringValue(detectionResult.Version)
	data.InstallationPath = types.StringValue(detectionResult.InstallationPath)
	data.LastChecked = types.StringValue(time.Now().Format(time.RFC3339))
	data.DetectionResult = types.StringValue(detectionResult.Method)

	// Handle conditional behavior
	if data.SkipIfNotInstalled.ValueBool() && !detectionResult.Installed {
		tflog.Info(ctx, "Skipping application configuration - not installed", map[string]interface{}{
			"application": data.Application.ValueString(),
		})
		// Set ID and save state (but don't configure files)
		data.ID = data.Application
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}

	if data.WarnIfNotInstalled.ValueBool() && !detectionResult.Installed {
		resp.Diagnostics.AddWarning(
			"Application not installed",
			fmt.Sprintf("Application %s is not installed but configuration will proceed", data.Application.ValueString()),
		)
	}

	// Deploy configuration files if config_mappings is specified
	if data.ConfigMappings != nil {
		err := r.deployApplicationConfig(ctx, &data)
		if err != nil {
			resp.Diagnostics.AddError(
				"Failed to deploy application configuration",
				fmt.Sprintf("Error deploying config for %s: %v", data.Application.ValueString(), err),
			)
			return
		}
	}

	// Set ID and save state
	data.ID = data.Application
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	tflog.Info(ctx, "Application resource created successfully", map[string]interface{}{
		"application": data.Application.ValueString(),
		"installed":   detectionResult.Installed,
		"version":     detectionResult.Version,
	})
}

func (r *ApplicationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ApplicationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Re-perform application detection to check current state
	if data.DetectInstallation.ValueBool() {
		detectionResult := r.performApplicationDetection(ctx, &data)
		// Update computed attributes
		data.Installed = types.BoolValue(detectionResult.Installed)
		data.Version = types.StringValue(detectionResult.Version)
		data.InstallationPath = types.StringValue(detectionResult.InstallationPath)
		data.LastChecked = types.StringValue(time.Now().Format(time.RFC3339))
		data.DetectionResult = types.StringValue(detectionResult.Method)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ApplicationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ApplicationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating application resource", map[string]interface{}{
		"application":         data.Application.ValueString(),
		"source_path":         data.SourcePath.ValueString(),
		"detect_installation": data.DetectInstallation.ValueBool(),
	})

	// Perform application detection (same as Create method)
	detectionResult := r.performApplicationDetection(ctx, &data)

	// Update computed attributes with detection results
	data.Installed = types.BoolValue(detectionResult.Installed)
	data.Version = types.StringValue(detectionResult.Version)
	data.InstallationPath = types.StringValue(detectionResult.InstallationPath)
	data.LastChecked = types.StringValue(time.Now().Format(time.RFC3339))
	data.DetectionResult = types.StringValue(detectionResult.Method)

	// Handle conditional behavior
	if data.SkipIfNotInstalled.ValueBool() && !detectionResult.Installed {
		tflog.Info(ctx, "Skipping application configuration update - not installed", map[string]interface{}{
			"application": data.Application.ValueString(),
		})
		// Update ID and save state (but don't configure files)
		data.ID = data.Application
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}

	if data.WarnIfNotInstalled.ValueBool() && !detectionResult.Installed {
		resp.Diagnostics.AddWarning(
			"Application not installed",
			fmt.Sprintf("Application %s is not installed but configuration update will proceed", data.Application.ValueString()),
		)
	}

	// Deploy configuration files if config_mappings is specified
	if data.ConfigMappings != nil {
		err := r.deployApplicationConfig(ctx, &data)
		if err != nil {
			resp.Diagnostics.AddError(
				"Failed to deploy application configuration",
				fmt.Sprintf("Error deploying config for %s: %v", data.Application.ValueString(), err),
			)
			return
		}
	}

	// Set ID and save state
	data.ID = data.Application
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	tflog.Info(ctx, "Application resource updated successfully", map[string]interface{}{
		"application": data.Application.ValueString(),
		"installed":   detectionResult.Installed,
		"version":     detectionResult.Version,
	})
}

func (r *ApplicationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ApplicationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting application resource", map[string]interface{}{
		"application": data.Application.ValueString(),
	})

	// TODO: Implement cleanup of application configuration files
	// This would remove symlinks/files created by this resource
}

// ApplicationDetectionResult represents the result of application detection.
type ApplicationDetectionResult struct {
	Installed        bool
	Version          string
	InstallationPath string
	Method           string
}

// performApplicationDetection performs application detection using configured methods.
func (r *ApplicationResource) performApplicationDetection(ctx context.Context, data *ApplicationResourceModel) *ApplicationDetectionResult {
	if !data.DetectInstallation.ValueBool() {
		return &ApplicationDetectionResult{
			Installed: true, // Assume installed if detection is disabled
			Method:    "disabled",
		}
	}

	appName := data.Application.ValueString()

	// Use default detection methods since blocks are handled separately
	detectionMethods := []string{"command", "file"}

	// Try each detection method
	for _, method := range detectionMethods {
		result := r.tryDetectionMethod(ctx, appName, method)
		if result.Installed {
			tflog.Info(ctx, "Application detected", map[string]interface{}{
				"application": appName,
				"method":      method,
				"version":     result.Version,
				"path":        result.InstallationPath,
			})
			return result
		}
	}

	// No detection method succeeded
	return &ApplicationDetectionResult{
		Installed: false,
		Method:    "not_found",
	}
}

// tryDetectionMethod attempts to detect application using specified method.
func (r *ApplicationResource) tryDetectionMethod(ctx context.Context, appName, method string) *ApplicationDetectionResult {
	platformProvider := platform.DetectPlatform()

	switch method {
	case "command":
		return r.detectByCommand(ctx, appName)
	case "file":
		return r.detectByFile(ctx, appName, platformProvider)
	case "brew_cask":
		return r.detectByBrewCask(ctx, appName)
	case "package_manager":
		return r.detectByPackageManager(ctx, appName, platformProvider)
	default:
		return &ApplicationDetectionResult{Installed: false}
	}
}

// detectByCommand detects application by running a command.
func (r *ApplicationResource) detectByCommand(ctx context.Context, appName string) *ApplicationDetectionResult {
	// Use command -v to check if command exists in PATH
	cmd := fmt.Sprintf("command -v %s", appName)

	// Execute command using the same shell execution logic as hooks
	err := executeShellCommand(ctx, cmd)
	if err != nil {
		return &ApplicationDetectionResult{Installed: false}
	}

	// TODO: Extract version information if possible
	return &ApplicationDetectionResult{
		Installed: true,
		Method:    "command",
		Version:   "unknown", // Could be enhanced to extract version
	}
}

// detectByFile detects application by checking file/directory existence.
func (r *ApplicationResource) detectByFile(ctx context.Context, appName string, platformProvider platform.PlatformProvider) *ApplicationDetectionResult {
	// Use context for structured logging and to avoid unused parameter warnings
	tflog.Debug(ctx, "detectByFile invoked", map[string]interface{}{
		"application": appName,
		"platform":    platformProvider.GetPlatform(),
	})
	// Define common application installation paths by platform
	var searchPaths []string

	switch platformProvider.GetPlatform() {
	case "macos":
		searchPaths = []string{
			fmt.Sprintf("/Applications/%s.app", capitalizeFirst(appName)),
			fmt.Sprintf("/Applications/%s.app", appName),
			fmt.Sprintf("/System/Applications/%s.app", capitalizeFirst(appName)),
			// For testing, also check in common test locations
			fmt.Sprintf("./%s.app", capitalizeFirst(appName)),
			fmt.Sprintf("./%s.app", appName),
		}
	case "linux":
		searchPaths = []string{
			fmt.Sprintf("/usr/bin/%s", appName),
			fmt.Sprintf("/usr/local/bin/%s", appName),
			fmt.Sprintf("/opt/%s", appName),
			// For testing
			fmt.Sprintf("./%s", appName),
		}
	case "windows":
		searchPaths = []string{
			fmt.Sprintf("C:\\Program Files\\%s", capitalizeFirst(appName)),
			fmt.Sprintf("C:\\Program Files (x86)\\%s", capitalizeFirst(appName)),
			// For testing
			fmt.Sprintf(".\\%s", appName),
		}
	}

	// Check each path
	for _, path := range searchPaths {
		var expandedPath string
		var err error

		// Handle relative paths for testing
		if strings.HasPrefix(path, "./") || strings.HasPrefix(path, ".\\") {
			// Use working directory as base for relative paths
			expandedPath = path
		} else {
			expandedPath, err = platformProvider.ExpandPath(path)
			if err != nil {
				continue
			}
		}

		if _, err := os.Stat(expandedPath); err == nil {
			return &ApplicationDetectionResult{
				Installed:        true,
				InstallationPath: expandedPath,
				Method:           "file",
				Version:          "unknown", // Could be enhanced to extract version from app bundle
			}
		}
	}

	return &ApplicationDetectionResult{Installed: false}
}

// detectByBrewCask detects application installed via Homebrew cask.
func (r *ApplicationResource) detectByBrewCask(ctx context.Context, appName string) *ApplicationDetectionResult {
	cmd := fmt.Sprintf("brew list --cask | grep -q '^%s$'", appName)

	err := executeShellCommand(ctx, cmd)
	if err != nil {
		return &ApplicationDetectionResult{Installed: false}
	}

	return &ApplicationDetectionResult{
		Installed: true,
		Method:    "brew_cask",
		Version:   "unknown", // Could be enhanced to get brew cask version
	}
}

// detectByPackageManager detects application via system package manager.
func (r *ApplicationResource) detectByPackageManager(ctx context.Context, appName string, platformProvider platform.PlatformProvider) *ApplicationDetectionResult {
	var cmd string

	switch platformProvider.GetPlatform() {
	case "macos":
		cmd = fmt.Sprintf("brew list | grep -q '^%s$'", appName)
	case "linux":
		// Try common Linux package managers
		managers := []string{
			fmt.Sprintf("dpkg -l | grep -q ' %s '", appName),      // Debian/Ubuntu
			fmt.Sprintf("rpm -qa | grep -q '^%s-'", appName),      // RedHat/CentOS
			fmt.Sprintf("pacman -Q %s > /dev/null 2>&1", appName), // Arch
		}

		// Try each package manager
		for _, managerCmd := range managers {
			if err := executeShellCommand(ctx, managerCmd); err == nil {
				return &ApplicationDetectionResult{
					Installed: true,
					Method:    "package_manager",
				}
			}
		}
		return &ApplicationDetectionResult{Installed: false}
	default:
		return &ApplicationDetectionResult{Installed: false}
	}

	err := executeShellCommand(ctx, cmd)
	if err != nil {
		return &ApplicationDetectionResult{Installed: false}
	}

	return &ApplicationDetectionResult{
		Installed: true,
		Method:    "package_manager",
	}
}

// capitalizeFirst capitalizes the first letter of a string.
func capitalizeFirst(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// deployApplicationConfig deploys configuration files based on config_mappings.
func (r *ApplicationResource) deployApplicationConfig(ctx context.Context, data *ApplicationResourceModel) error {
	if data.ConfigMappings == nil {
		return nil
	}

	tflog.Debug(ctx, "Deploying application configuration", map[string]interface{}{
		"application": data.Application.ValueString(),
		"source_path": data.SourcePath.ValueString(),
	})

	// Get the repository local path
	repositoryLocalPath := r.getRepositoryLocalPath(data.Repository.ValueString())
	sourcePath := data.SourcePath.ValueString()

	// Resolve full source path
	var fullSourcePath string
	if strings.HasPrefix(sourcePath, "/") {
		fullSourcePath = sourcePath
	} else {
		fullSourcePath = fmt.Sprintf("%s/%s", repositoryLocalPath, sourcePath)
	}

	// Check if source path exists
	if _, err := os.Stat(fullSourcePath); os.IsNotExist(err) {
		return fmt.Errorf("source path does not exist: %s", fullSourcePath)
	}

	// Deploy based on target path configuration
	var targetPath string
	if !data.ConfigMappings.TargetPath.IsNull() {
		targetPath = data.ConfigMappings.TargetPath.ValueString()
	} else if !data.ConfigMappings.TargetPathTemplate.IsNull() {
		// Expand target path template
		template := data.ConfigMappings.TargetPathTemplate.ValueString()
		expandedPath, err := r.expandTargetPathTemplate(template, data)
		if err != nil {
			return fmt.Errorf("failed to expand target path template: %w", err)
		}
		targetPath = expandedPath
	} else {
		return fmt.Errorf("either target_path or target_path_template must be specified")
	}

	// Expand tilde in target path
	if strings.HasPrefix(targetPath, "~") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("unable to get home directory: %w", err)
		}
		targetPath = strings.Replace(targetPath, "~", homeDir, 1)
	}

	// Create target directory if it doesn't exist
	targetDir := filepath.Dir(targetPath)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory %s: %w", targetDir, err)
	}

	// Deploy using the configured strategy
	strategy := data.ConfigStrategy.ValueString()
	if strategy == "" {
		strategy = r.client.Config.Strategy // Use provider default
	}

	tflog.Debug(ctx, "Deploying config file", map[string]interface{}{
		"source":   fullSourcePath,
		"target":   targetPath,
		"strategy": strategy,
	})

	switch strategy {
	case "symlink":
		return r.createSymlinkForConfig(fullSourcePath, targetPath)
	case "copy":
		return r.copyConfigFile(fullSourcePath, targetPath)
	default:
		return fmt.Errorf("unsupported config strategy: %s", strategy)
	}
}

// expandTargetPathTemplate expands a target path template with application variables.
func (r *ApplicationResource) expandTargetPathTemplate(template string, data *ApplicationResourceModel) (string, error) {
	// Get platform provider for directory paths
	platformProvider := platform.DetectPlatform()

	// Replace common template variables
	result := template

	// Replace {{.home_dir}}
	if strings.Contains(result, "{{.home_dir}}") {
		homeDir, err := platformProvider.GetHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		result = strings.ReplaceAll(result, "{{.home_dir}}", homeDir)
	}

	// Replace {{.config_dir}}
	if strings.Contains(result, "{{.config_dir}}") {
		configDir, err := platformProvider.GetConfigDir()
		if err != nil {
			return "", fmt.Errorf("failed to get config directory: %w", err)
		}
		result = strings.ReplaceAll(result, "{{.config_dir}}", configDir)
	}

	// Replace {{.app_support_dir}}
	if strings.Contains(result, "{{.app_support_dir}}") {
		appSupportDir, err := platformProvider.GetAppSupportDir()
		if err != nil {
			return "", fmt.Errorf("failed to get app support directory: %w", err)
		}
		result = strings.ReplaceAll(result, "{{.app_support_dir}}", appSupportDir)
	}

	// Replace {{.application}}
	result = strings.ReplaceAll(result, "{{.application}}", data.Application.ValueString())

	return result, nil
}

// createSymlinkForConfig creates a symlink from source to target.
func (r *ApplicationResource) createSymlinkForConfig(source, target string) error {
	// Remove existing file/symlink if it exists
	if _, err := os.Lstat(target); err == nil {
		if err := os.Remove(target); err != nil {
			return fmt.Errorf("failed to remove existing target %s: %w", target, err)
		}
	}

	// Create symlink
	if err := os.Symlink(source, target); err != nil {
		return fmt.Errorf("failed to create symlink from %s to %s: %w", source, target, err)
	}

	return nil
}

// copyConfigFile copies a file from source to target.
func (r *ApplicationResource) copyConfigFile(source, target string) error {
	sourceFile, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("failed to open source file %s: %w", source, err)
	}
	defer sourceFile.Close()

	// Remove existing file if it exists
	if _, err := os.Stat(target); err == nil {
		if err := os.Remove(target); err != nil {
			return fmt.Errorf("failed to remove existing target %s: %w", target, err)
		}
	}

	targetFile, err := os.Create(target)
	if err != nil {
		return fmt.Errorf("failed to create target file %s: %w", target, err)
	}
	defer targetFile.Close()

	// Copy content
	if _, err := targetFile.ReadFrom(sourceFile); err != nil {
		return fmt.Errorf("failed to copy content from %s to %s: %w", source, target, err)
	}

	// Copy permissions
	sourceInfo, err := sourceFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to get source file info: %w", err)
	}

	if err := targetFile.Chmod(sourceInfo.Mode()); err != nil {
		return fmt.Errorf("failed to set target file permissions: %w", err)
	}

	return nil
}

// getRepositoryLocalPath returns the local path for a repository.
func (r *ApplicationResource) getRepositoryLocalPath(repositoryID string) string {
	// For now, assume repository ID maps to the dotfiles root
	// TODO: Implement proper repository lookup when repository state management is added
	_ = repositoryID // TODO: Use repositoryID when repository lookup is implemented
	return r.client.Config.DotfilesRoot
}
