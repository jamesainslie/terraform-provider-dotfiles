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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ApplicationResource{}

// NewApplicationResource creates a new application resource.
func NewApplicationResource() resource.Resource {
	return &ApplicationResource{}
}

// ApplicationResource defines the application resource implementation.
// This resource manages configuration files for applications, NOT application installation.
// Application installation should be handled by the terraform-provider-package.
type ApplicationResource struct {
	client *DotfilesClient
}

// ApplicationResourceModel describes the application resource data model.
type ApplicationResourceModel struct {
	ID              types.String `tfsdk:"id"`
	ApplicationName types.String `tfsdk:"application_name"`
	ConfigMappings  types.Map    `tfsdk:"config_mappings"`

	// Computed attributes
	ConfiguredFiles types.List   `tfsdk:"configured_files"`
	LastUpdated     types.String `tfsdk:"last_updated"`
}

// ConfigMappingValue represents a single configuration mapping.
type ConfigMappingValue struct {
	TargetPath types.String `tfsdk:"target_path"`
	Strategy   types.String `tfsdk:"strategy"`
}

// Metadata sets the resource type name.
func (r *ApplicationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application"
}

// Schema defines the resource schema.
func (r *ApplicationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages application-specific configuration files. This resource focuses solely on configuration file management - application installation should be handled by terraform-provider-package.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Application configuration resource identifier",
			},
			"application_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Name of the application (used for organization and templating)",
			},
			"config_mappings": schema.MapNestedAttribute{
				Required:            true,
				MarkdownDescription: "Map of source files to target configuration mappings",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"target_path": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "Target path where the configuration file should be placed",
						},
						"strategy": schema.StringAttribute{
							Optional:            true,
							Computed:            true,
							Default:             stringdefault.StaticString("symlink"),
							MarkdownDescription: "Deployment strategy: 'symlink' or 'copy'",
						},
					},
				},
				Default: mapdefault.StaticValue(types.MapValueMust(
					types.ObjectType{
						AttrTypes: map[string]attr.Type{
							"target_path": types.StringType,
							"strategy":    types.StringType,
						},
					},
					map[string]attr.Value{},
				)),
			},
			"configured_files": schema.ListAttribute{
				ElementType:         types.StringType,
				Computed:            true,
				MarkdownDescription: "List of configuration files that were successfully configured",
			},
			"last_updated": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Timestamp when the configuration was last updated",
			},
		},
	}
}

// Configure sets up the resource with the provider client.
func (r *ApplicationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*DotfilesClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *DotfilesClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

// Create handles resource creation.
func (r *ApplicationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ApplicationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Generate resource ID
	data.ID = types.StringValue(fmt.Sprintf("app-%s-%d", data.ApplicationName.ValueString(), time.Now().Unix()))

	tflog.Info(ctx, "Creating application configuration", map[string]interface{}{
		"application": data.ApplicationName.ValueString(),
		"id":          data.ID.ValueString(),
	})

	// Deploy configuration files
	configuredFiles, err := r.deployApplicationConfig(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError(
			"Configuration Deployment Failed",
			fmt.Sprintf("Failed to deploy configuration for application %s: %s", data.ApplicationName.ValueString(), err.Error()),
		)
		return
	}

	// Update computed attributes
	data.ConfiguredFiles = configuredFiles
	data.LastUpdated = types.StringValue(time.Now().Format(time.RFC3339))

	tflog.Info(ctx, "Application configuration created successfully", map[string]interface{}{
		"application":      data.ApplicationName.ValueString(),
		"configured_files": len(configuredFiles.Elements()),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read handles resource reading.
func (r *ApplicationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ApplicationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading application configuration", map[string]interface{}{
		"application": data.ApplicationName.ValueString(),
		"id":          data.ID.ValueString(),
	})

	// Verify configuration files still exist and are properly configured
	err := r.verifyConfigurationFiles(ctx, &data)
	if err != nil {
		tflog.Warn(ctx, "Configuration verification failed", map[string]interface{}{
			"application": data.ApplicationName.ValueString(),
			"error":       err.Error(),
		})
		// Don't fail the read, just log the warning
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update handles resource updates.
func (r *ApplicationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ApplicationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Updating application configuration", map[string]interface{}{
		"application": data.ApplicationName.ValueString(),
		"id":          data.ID.ValueString(),
	})

	// Redeploy configuration files
	configuredFiles, err := r.deployApplicationConfig(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError(
			"Configuration Update Failed",
			fmt.Sprintf("Failed to update configuration for application %s: %s", data.ApplicationName.ValueString(), err.Error()),
		)
		return
	}

	// Update computed attributes
	data.ConfiguredFiles = configuredFiles
	data.LastUpdated = types.StringValue(time.Now().Format(time.RFC3339))

	tflog.Info(ctx, "Application configuration updated successfully", map[string]interface{}{
		"application":      data.ApplicationName.ValueString(),
		"configured_files": len(configuredFiles.Elements()),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete handles resource deletion.
func (r *ApplicationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ApplicationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Deleting application configuration", map[string]interface{}{
		"application": data.ApplicationName.ValueString(),
		"id":          data.ID.ValueString(),
	})

	// Remove configured files (symlinks/copies)
	err := r.removeApplicationConfig(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError(
			"Configuration Removal Failed",
			fmt.Sprintf("Failed to remove configuration for application %s: %s", data.ApplicationName.ValueString(), err.Error()),
		)
		return
	}

	tflog.Info(ctx, "Application configuration deleted successfully", map[string]interface{}{
		"application": data.ApplicationName.ValueString(),
	})
}

// deployApplicationConfig deploys configuration files according to the mappings.
func (r *ApplicationResource) deployApplicationConfig(ctx context.Context, data *ApplicationResourceModel) (types.List, error) {
	var configuredFiles []string

	configMappings := data.ConfigMappings.Elements()
	for sourceFile, mappingValue := range configMappings {
		// Extract the mapping configuration
		mappingObj := mappingValue.(types.Object)
		mappingAttrs := mappingObj.Attributes()

		targetPath := mappingAttrs["target_path"].(types.String).ValueString()
		strategy := mappingAttrs["strategy"].(types.String).ValueString()

		// Expand target path template variables
		expandedTargetPath, err := r.expandTargetPathTemplate(targetPath, data.ApplicationName.ValueString())
		if err != nil {
			return types.ListNull(types.StringType), fmt.Errorf("failed to expand target path template for %s: %w", sourceFile, err)
		}

		// Get source path from dotfiles root
		sourcePath := filepath.Join(r.client.Config.DotfilesRoot, sourceFile)

		// Check if source file exists
		if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
			tflog.Warn(ctx, "Source configuration file does not exist", map[string]interface{}{
				"source_file": sourcePath,
				"application": data.ApplicationName.ValueString(),
			})
			continue
		}

		// Deploy based on strategy
		switch strategy {
		case "symlink":
			err = r.createSymlinkForConfig(ctx, sourcePath, expandedTargetPath)
		case "copy":
			err = r.copyConfigFile(ctx, sourcePath, expandedTargetPath)
		default:
			err = fmt.Errorf("unsupported strategy: %s", strategy)
		}

		if err != nil {
			return types.ListNull(types.StringType), fmt.Errorf("failed to deploy %s using %s strategy: %w", sourceFile, strategy, err)
		}

		configuredFiles = append(configuredFiles, expandedTargetPath)
		tflog.Debug(ctx, "Configuration file deployed", map[string]interface{}{
			"source":   sourcePath,
			"target":   expandedTargetPath,
			"strategy": strategy,
		})
	}

	// Convert to Terraform list type
	configuredFilesList, _ := types.ListValueFrom(ctx, types.StringType, configuredFiles)
	return configuredFilesList, nil
}

// expandTargetPathTemplate expands template variables in target paths.
func (r *ApplicationResource) expandTargetPathTemplate(targetPath, applicationName string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	// Template variables
	replacements := map[string]string{
		"{{.home_dir}}":        homeDir,
		"{{.config_dir}}":      filepath.Join(homeDir, ".config"),
		"{{.app_support_dir}}": filepath.Join(homeDir, "Library", "Application Support"),
		"{{.application}}":     applicationName,
	}

	expandedPath := targetPath
	for template, replacement := range replacements {
		expandedPath = strings.ReplaceAll(expandedPath, template, replacement)
	}

	// Handle tilde expansion
	if strings.HasPrefix(expandedPath, "~/") {
		expandedPath = filepath.Join(homeDir, expandedPath[2:])
	}

	return expandedPath, nil
}

// createSymlinkForConfig creates a symlink for a configuration file.
func (r *ApplicationResource) createSymlinkForConfig(ctx context.Context, sourcePath, targetPath string) error {
	// Create target directory if it doesn't exist
	targetDir := filepath.Dir(targetPath)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory %s: %w", targetDir, err)
	}

	// Remove existing file/symlink if it exists
	if _, err := os.Lstat(targetPath); err == nil {
		if err := os.Remove(targetPath); err != nil {
			return fmt.Errorf("failed to remove existing file %s: %w", targetPath, err)
		}
	}

	// Create symlink
	if err := os.Symlink(sourcePath, targetPath); err != nil {
		return fmt.Errorf("failed to create symlink from %s to %s: %w", sourcePath, targetPath, err)
	}

	return nil
}

// copyConfigFile copies a configuration file to the target location.
func (r *ApplicationResource) copyConfigFile(ctx context.Context, sourcePath, targetPath string) error {
	// Create target directory if it doesn't exist
	targetDir := filepath.Dir(targetPath)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory %s: %w", targetDir, err)
	}

	// Open source file
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to open source file %s: %w", sourcePath, err)
	}
	defer func() {
		if err := sourceFile.Close(); err != nil {
			tflog.Warn(ctx, "Failed to close source file", map[string]interface{}{
				"file":  sourcePath,
				"error": err.Error(),
			})
		}
	}()

	// Create target file
	targetFile, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("failed to create target file %s: %w", targetPath, err)
	}
	defer func() {
		if err := targetFile.Close(); err != nil {
			tflog.Warn(ctx, "Failed to close target file", map[string]interface{}{
				"file":  targetPath,
				"error": err.Error(),
			})
		}
	}()

	// Copy file contents
	if _, err := sourceFile.WriteTo(targetFile); err != nil {
		return fmt.Errorf("failed to copy file contents from %s to %s: %w", sourcePath, targetPath, err)
	}

	return nil
}

// verifyConfigurationFiles verifies that configuration files are still properly configured.
func (r *ApplicationResource) verifyConfigurationFiles(ctx context.Context, data *ApplicationResourceModel) error {
	configuredFiles := data.ConfiguredFiles.Elements()
	
	for _, fileValue := range configuredFiles {
		filePath := fileValue.(types.String).ValueString()
		
		// Check if file still exists
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			return fmt.Errorf("configuration file %s no longer exists", filePath)
		}
	}

	return nil
}

// removeApplicationConfig removes all configured files for the application.
func (r *ApplicationResource) removeApplicationConfig(ctx context.Context, data *ApplicationResourceModel) error {
	configuredFiles := data.ConfiguredFiles.Elements()
	
	for _, fileValue := range configuredFiles {
		filePath := fileValue.(types.String).ValueString()
		
		// Check if file exists before trying to remove
		if _, err := os.Lstat(filePath); os.IsNotExist(err) {
			continue // File doesn't exist, skip
		}
		
		// Remove file/symlink
		if err := os.Remove(filePath); err != nil {
			tflog.Warn(ctx, "Failed to remove configuration file", map[string]interface{}{
				"file":  filePath,
				"error": err.Error(),
			})
			// Continue with other files even if one fails
		} else {
			tflog.Debug(ctx, "Removed configuration file", map[string]interface{}{
				"file": filePath,
			})
		}
	}

	return nil
}
