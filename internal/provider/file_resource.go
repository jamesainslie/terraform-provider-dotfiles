// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/jamesainslie/terraform-provider-dotfiles/internal/fileops"
	"github.com/jamesainslie/terraform-provider-dotfiles/internal/platform"
	"github.com/jamesainslie/terraform-provider-dotfiles/internal/template"
	"github.com/jamesainslie/terraform-provider-dotfiles/internal/utils"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &FileResource{}

func NewFileResource() resource.Resource {
	return &FileResource{}
}

// FileResource defines the resource implementation.
type FileResource struct {
	client *DotfilesClient
}

// FileResourceModel describes the resource data model.
type FileResourceModel struct {
	ID            types.String `tfsdk:"id"`
	Repository    types.String `tfsdk:"repository"`
	Name          types.String `tfsdk:"name"`
	SourcePath    types.String `tfsdk:"source_path"`
	TargetPath    types.String `tfsdk:"target_path"`
	IsTemplate    types.Bool   `tfsdk:"is_template"`
	FileMode      types.String `tfsdk:"file_mode"`
	BackupEnabled types.Bool   `tfsdk:"backup_enabled"`

	// Template variables (for template processing)
	TemplateVars types.Map `tfsdk:"template_vars"`

	// Computed attributes for state tracking
	ContentHash  types.String `tfsdk:"content_hash"`
	LastModified types.String `tfsdk:"last_modified"`
	FileExists   types.Bool   `tfsdk:"file_exists"`
}

func (r *FileResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_file"
}

func (r *FileResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages individual dotfiles",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "File identifier",
			},
			"repository": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Repository ID this file belongs to",
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "File name/identifier",
			},
			"source_path": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Path to source file in repository",
			},
			"target_path": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Target path where file should be placed",
			},
			"is_template": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Whether the file should be processed as a template",
			},
			"file_mode": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "File permissions (e.g., '0644')",
			},
			"backup_enabled": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Whether to backup existing files",
			},
			"template_vars": schema.MapAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Variables for template processing",
			},
			"content_hash": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "SHA256 hash of file content",
			},
			"last_modified": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Last modification timestamp",
			},
			"file_exists": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the target file exists",
			},
		},
	}
}

func (r *FileResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *FileResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data FileResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating file resource", map[string]interface{}{
		"name":        data.Name.ValueString(),
		"source_path": data.SourcePath.ValueString(),
		"target_path": data.TargetPath.ValueString(),
		"is_template": data.IsTemplate.ValueBool(),
	})

	// Get repository information (for local path if it's a Git repository)
	repositoryLocalPath, err := r.getRepositoryLocalPath(data.Repository.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Repository not found",
			fmt.Sprintf("Could not find repository %s: %s", data.Repository.ValueString(), err.Error()),
		)
		return
	}

	// Build source file path
	sourcePath := filepath.Join(repositoryLocalPath, data.SourcePath.ValueString())
	targetPath := data.TargetPath.ValueString()

	// Expand target path
	platformProvider := platform.DetectPlatform()
	expandedTargetPath, err := platformProvider.ExpandPath(targetPath)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid target path",
			fmt.Sprintf("Could not expand target path %s: %s", targetPath, err.Error()),
		)
		return
	}

	// Create file manager
	fileManager := fileops.NewFileManager(platformProvider, r.client.Config.DryRun)

	// Determine file mode
	fileMode := "0644" // default
	if !data.FileMode.IsNull() {
		fileMode = data.FileMode.ValueString()
	}

	// Check if backup is needed
	backupEnabled := r.client.Config.BackupEnabled
	if !data.BackupEnabled.IsNull() {
		backupEnabled = data.BackupEnabled.ValueBool()
	}

	var finalErr error

	if data.IsTemplate.ValueBool() {
		// Process template
		templateVars := make(map[string]interface{})

		// Add template variables if provided
		if !data.TemplateVars.IsNull() {
			elements := data.TemplateVars.Elements()
			for key, value := range elements {
				if strValue, ok := value.(types.String); ok {
					templateVars[key] = strValue.ValueString()
				}
			}
		}

		// Add system context
		systemInfo := r.client.GetPlatformInfo()
		context := template.CreateTemplateContext(systemInfo, templateVars)

		if backupEnabled && utils.PathExists(expandedTargetPath) {
			// Create backup first
			_, err := fileManager.CreateBackup(expandedTargetPath, r.client.Config.BackupDirectory)
			if err != nil {
				resp.Diagnostics.AddWarning(
					"Backup failed",
					fmt.Sprintf("Could not create backup of %s: %s", expandedTargetPath, err.Error()),
				)
			}
		}

		// Process template
		finalErr = fileManager.ProcessTemplate(sourcePath, expandedTargetPath, context, fileMode)
	} else {
		// Regular file copy
		if backupEnabled {
			finalErr = fileManager.CopyFileWithBackup(sourcePath, expandedTargetPath, fileMode, r.client.Config.BackupDirectory)
		} else {
			finalErr = fileManager.CopyFile(sourcePath, expandedTargetPath, fileMode)
		}
	}

	if finalErr != nil {
		resp.Diagnostics.AddError(
			"File operation failed",
			fmt.Sprintf("Could not create file %s: %s", expandedTargetPath, finalErr.Error()),
		)
		return
	}

	// Update computed attributes
	if err := r.updateComputedAttributes(ctx, &data, expandedTargetPath); err != nil {
		resp.Diagnostics.AddWarning(
			"Could not update file metadata",
			fmt.Sprintf("File created successfully but could not update metadata: %s", err.Error()),
		)
	}

	// Set ID and save state
	data.ID = data.Name
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	tflog.Info(ctx, "File resource created successfully", map[string]interface{}{
		"name":        data.Name.ValueString(),
		"target_path": expandedTargetPath,
		"is_template": data.IsTemplate.ValueBool(),
	})
}

func (r *FileResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data FileResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading file resource", map[string]interface{}{
		"name":        data.Name.ValueString(),
		"target_path": data.TargetPath.ValueString(),
	})

	// Expand target path to check current state
	targetPath := data.TargetPath.ValueString()
	if targetPath != "" {
		platformProvider := platform.DetectPlatform()
		expandedTargetPath, err := platformProvider.ExpandPath(targetPath)
		if err != nil {
			resp.Diagnostics.AddError(
				"Invalid target path",
				fmt.Sprintf("Could not expand target path %s: %s", targetPath, err.Error()),
			)
			return
		}

		// Update computed attributes with current file state
		if err := r.updateComputedAttributes(ctx, &data, expandedTargetPath); err != nil {
			resp.Diagnostics.AddWarning(
				"Could not read file metadata",
				fmt.Sprintf("Could not update file metadata: %s", err.Error()),
			)
		}

		// Check for drift if file doesn't exist
		if !data.FileExists.ValueBool() {
			resp.Diagnostics.AddWarning(
				"Managed file not found",
				fmt.Sprintf("The managed file %s no longer exists", expandedTargetPath),
			)
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *FileResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data FileResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating file resource", map[string]interface{}{
		"name":        data.Name.ValueString(),
		"source_path": data.SourcePath.ValueString(),
		"target_path": data.TargetPath.ValueString(),
	})

	// Get repository local path
	repositoryLocalPath, err := r.getRepositoryLocalPath(data.Repository.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Repository not found",
			fmt.Sprintf("Could not find repository %s: %s", data.Repository.ValueString(), err.Error()),
		)
		return
	}

	// Build paths
	sourcePath := filepath.Join(repositoryLocalPath, data.SourcePath.ValueString())
	targetPath := data.TargetPath.ValueString()

	// Expand target path
	platformProvider := platform.DetectPlatform()
	expandedTargetPath, err := platformProvider.ExpandPath(targetPath)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid target path",
			fmt.Sprintf("Could not expand target path %s: %s", targetPath, err.Error()),
		)
		return
	}

	// Create file manager
	fileManager := fileops.NewFileManager(platformProvider, r.client.Config.DryRun)

	// Determine file mode
	fileMode := "0644" // default
	if !data.FileMode.IsNull() {
		fileMode = data.FileMode.ValueString()
	}

	// Check if backup is needed
	backupEnabled := r.client.Config.BackupEnabled
	if !data.BackupEnabled.IsNull() {
		backupEnabled = data.BackupEnabled.ValueBool()
	}

	var finalErr error

	if data.IsTemplate.ValueBool() {
		// Process template for update
		templateVars := make(map[string]interface{})

		// Add template variables if provided
		if !data.TemplateVars.IsNull() {
			elements := data.TemplateVars.Elements()
			for key, value := range elements {
				if strValue, ok := value.(types.String); ok {
					templateVars[key] = strValue.ValueString()
				}
			}
		}

		// Add system context
		systemInfo := r.client.GetPlatformInfo()
		context := template.CreateTemplateContext(systemInfo, templateVars)

		if backupEnabled && utils.PathExists(expandedTargetPath) {
			// Create backup before update
			_, err := fileManager.CreateBackup(expandedTargetPath, r.client.Config.BackupDirectory)
			if err != nil {
				resp.Diagnostics.AddWarning(
					"Backup failed",
					fmt.Sprintf("Could not create backup before update: %s", err.Error()),
				)
			}
		}

		// Process template
		finalErr = fileManager.ProcessTemplate(sourcePath, expandedTargetPath, context, fileMode)
	} else {
		// Regular file copy for update
		if backupEnabled {
			finalErr = fileManager.CopyFileWithBackup(sourcePath, expandedTargetPath, fileMode, r.client.Config.BackupDirectory)
		} else {
			finalErr = fileManager.CopyFile(sourcePath, expandedTargetPath, fileMode)
		}
	}

	if finalErr != nil {
		resp.Diagnostics.AddError(
			"File update failed",
			fmt.Sprintf("Could not update file %s: %s", expandedTargetPath, finalErr.Error()),
		)
		return
	}

	// Update computed attributes
	if err := r.updateComputedAttributes(ctx, &data, expandedTargetPath); err != nil {
		resp.Diagnostics.AddWarning(
			"Could not update file metadata",
			fmt.Sprintf("File updated successfully but could not update metadata: %s", err.Error()),
		)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	tflog.Info(ctx, "File resource updated successfully", map[string]interface{}{
		"name":        data.Name.ValueString(),
		"target_path": expandedTargetPath,
	})
}

func (r *FileResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data FileResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting file resource", map[string]interface{}{
		"name":        data.Name.ValueString(),
		"target_path": data.TargetPath.ValueString(),
	})

	// For file resources, we typically remove the managed file
	// but preserve any backups
	targetPath := data.TargetPath.ValueString()
	if targetPath != "" {
		// Expand target path
		platformProvider := platform.DetectPlatform()
		expandedTargetPath, err := platformProvider.ExpandPath(targetPath)
		if err != nil {
			resp.Diagnostics.AddWarning(
				"Could not expand target path",
				fmt.Sprintf("Could not expand target path for cleanup: %s", err.Error()),
			)
			return
		}

		// Remove the file if it exists
		if utils.PathExists(expandedTargetPath) {
			err := os.Remove(expandedTargetPath)
			if err != nil {
				resp.Diagnostics.AddWarning(
					"Could not remove file",
					fmt.Sprintf("Could not remove file %s: %s", expandedTargetPath, err.Error()),
				)
			} else {
				tflog.Info(ctx, "File resource removed", map[string]interface{}{
					"target_path": expandedTargetPath,
				})
			}
		}
	}
}

// getRepositoryLocalPath returns the local path for a repository.
func (r *FileResource) getRepositoryLocalPath(repositoryID string) (string, error) {
	// For now, assume repository ID maps to the dotfiles root
	// TODO: Implement proper repository lookup when repository state management is added
	_ = repositoryID // TODO: Use repositoryID when repository lookup is implemented
	return r.client.Config.DotfilesRoot, nil
}

// updateComputedAttributes updates computed attributes for state tracking.
func (r *FileResource) updateComputedAttributes(ctx context.Context, data *FileResourceModel, targetPath string) error {
	_ = ctx // Context reserved for future logging
	// Check if file exists
	exists := utils.PathExists(targetPath)
	data.FileExists = types.BoolValue(exists)

	if exists {
		// Get file info
		info, err := os.Stat(targetPath)
		if err != nil {
			return fmt.Errorf("failed to stat file: %w", err)
		}

		// Set last modified time
		data.LastModified = types.StringValue(info.ModTime().Format(time.RFC3339))

		// Calculate content hash
		content, err := os.ReadFile(targetPath)
		if err != nil {
			return fmt.Errorf("failed to read file for hash: %w", err)
		}

		hash := sha256.Sum256(content)
		data.ContentHash = types.StringValue(fmt.Sprintf("%x", hash))
	} else {
		data.LastModified = types.StringNull()
		data.ContentHash = types.StringNull()
	}

	return nil
}
