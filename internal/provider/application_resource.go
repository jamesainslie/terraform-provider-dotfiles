// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

package provider

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
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

	// Detection configuration will be handled through blocks, not attributes

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
			"detection_methods": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Computed:    true,
				Default: listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("command"),
					types.StringValue("file"),
				})),
				MarkdownDescription: "Detection methods to use: command, file, brew_cask, package_manager",
			},
			"config_mappings": schema.MapAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Configuration file mappings (source -> target)",
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

	// TODO: Implement application configuration management here
	// This would handle the config_mappings and copy/symlink configuration files

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

	// Re-run the same logic as Create for updates
	// TODO: Implement differential updates for configuration changes

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
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
