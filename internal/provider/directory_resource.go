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
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/jamesainslie/terraform-provider-dotfiles/internal/utils"
)

var _ resource.Resource = &DirectoryResource{}

func NewDirectoryResource() resource.Resource {
	return &DirectoryResource{}
}

type DirectoryResource struct {
	client *DotfilesClient
}

type DirectoryResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	Repository          types.String `tfsdk:"repository"`
	Name                types.String `tfsdk:"name"`
	SourcePath          types.String `tfsdk:"source_path"`
	TargetPath          types.String `tfsdk:"target_path"`
	Recursive           types.Bool   `tfsdk:"recursive"`
	PreservePermissions types.Bool   `tfsdk:"preserve_permissions"`

	// Computed attributes
	DirectoryExists types.Bool   `tfsdk:"directory_exists"`
	FileCount       types.Int64  `tfsdk:"file_count"`
	LastSynced      types.String `tfsdk:"last_synced"`
}

func (r *DirectoryResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_directory"
}

func (r *DirectoryResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages directory structures and their contents",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Directory identifier",
			},
			"repository": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Repository ID this directory belongs to",
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Directory name/identifier",
			},
			"source_path": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Path to source directory in repository",
			},
			"target_path": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Target directory path",
			},
			"recursive": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				MarkdownDescription: "Process directory recursively. Defaults to true",
			},
			"preserve_permissions": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				MarkdownDescription: "Preserve file permissions. Defaults to true",
			},
			"directory_exists": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the target directory exists",
			},
			"file_count": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Number of files in the directory",
			},
			"last_synced": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Timestamp when the directory was last synced",
			},
		},
	}
}

func (r *DirectoryResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*DotfilesClient)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", "Expected *DotfilesClient")
		return
	}
	r.client = client
}

func (r *DirectoryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data DirectoryResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating directory resource", map[string]interface{}{
		"name":        data.Name.ValueString(),
		"source_path": data.SourcePath.ValueString(),
		"target_path": data.TargetPath.ValueString(),
		"recursive":   data.Recursive.ValueBool(),
	})

	// Resolve source and target paths
	sourcePath, targetPath, err := r.resolvePaths(&data)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to resolve paths",
			fmt.Sprintf("Error resolving paths for directory %s: %v", data.Name.ValueString(), err),
		)
		return
	}

	// Create or sync the directory
	err = r.syncDirectory(ctx, sourcePath, targetPath, &data)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to sync directory",
			fmt.Sprintf("Error syncing directory %s: %v", data.Name.ValueString(), err),
		)
		return
	}

	// Update computed attributes
	err = r.updateComputedAttributes(ctx, &data, targetPath)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to update computed attributes",
			fmt.Sprintf("Error updating attributes for directory %s: %v", data.Name.ValueString(), err),
		)
		return
	}

	data.ID = data.Name
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	tflog.Info(ctx, "Directory resource created successfully", map[string]interface{}{
		"name":       data.Name.ValueString(),
		"file_count": data.FileCount.ValueInt64(),
	})
}

func (r *DirectoryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data DirectoryResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Resolve target path to check current state
	_, targetPath, err := r.resolvePaths(&data)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to resolve paths",
			fmt.Sprintf("Error resolving paths for directory %s: %v", data.Name.ValueString(), err),
		)
		return
	}

	// Update computed attributes with current state
	err = r.updateComputedAttributes(ctx, &data, targetPath)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to update computed attributes",
			fmt.Sprintf("Error updating attributes for directory %s: %v", data.Name.ValueString(), err),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DirectoryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data DirectoryResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating directory resource", map[string]interface{}{
		"name":        data.Name.ValueString(),
		"source_path": data.SourcePath.ValueString(),
		"target_path": data.TargetPath.ValueString(),
	})

	// Resolve source and target paths
	sourcePath, targetPath, err := r.resolvePaths(&data)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to resolve paths",
			fmt.Sprintf("Error resolving paths for directory %s: %v", data.Name.ValueString(), err),
		)
		return
	}

	// Re-sync the directory with updated configuration
	err = r.syncDirectory(ctx, sourcePath, targetPath, &data)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to sync directory",
			fmt.Sprintf("Error syncing directory %s: %v", data.Name.ValueString(), err),
		)
		return
	}

	// Update computed attributes
	err = r.updateComputedAttributes(ctx, &data, targetPath)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to update computed attributes",
			fmt.Sprintf("Error updating attributes for directory %s: %v", data.Name.ValueString(), err),
		)
		return
	}

	data.ID = data.Name
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	tflog.Info(ctx, "Directory resource updated successfully", map[string]interface{}{
		"name":       data.Name.ValueString(),
		"file_count": data.FileCount.ValueInt64(),
	})
}

func (r *DirectoryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data DirectoryResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting directory resource", map[string]interface{}{
		"name":        data.Name.ValueString(),
		"target_path": data.TargetPath.ValueString(),
	})

	// Resolve target path
	_, targetPath, err := r.resolvePaths(&data)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to resolve paths",
			fmt.Sprintf("Error resolving paths for directory %s: %v", data.Name.ValueString(), err),
		)
		return
	}

	// Check if target exists before attempting deletion
	if !utils.PathExists(targetPath) {
		tflog.Info(ctx, "Target directory does not exist, nothing to delete", map[string]interface{}{
			"target_path": targetPath,
		})
		return
	}

	// Safety check: Don't delete system directories or directories outside expected paths
	if err := r.validateDeletionSafety(targetPath); err != nil {
		resp.Diagnostics.AddError(
			"Unsafe directory deletion",
			fmt.Sprintf("Cannot delete directory %s: %v", targetPath, err),
		)
		return
	}

	// Perform recursive deletion if configured
	if data.Recursive.ValueBool() {
		err = os.RemoveAll(targetPath)
	} else {
		err = os.Remove(targetPath)
	}

	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to delete directory",
			fmt.Sprintf("Error deleting directory %s: %v", targetPath, err),
		)
		return
	}

	tflog.Info(ctx, "Directory resource deleted successfully", map[string]interface{}{
		"name":        data.Name.ValueString(),
		"target_path": targetPath,
	})
}

// resolvePaths resolves the source and target paths for the directory.
func (r *DirectoryResource) resolvePaths(data *DirectoryResourceModel) (string, string, error) {
	// Get repository local path
	repositoryLocalPath := r.getRepositoryLocalPath(data.Repository.ValueString())
	sourcePath := data.SourcePath.ValueString()

	// Resolve full source path
	var fullSourcePath string
	if strings.HasPrefix(sourcePath, "/") {
		fullSourcePath = sourcePath
	} else {
		fullSourcePath = filepath.Join(repositoryLocalPath, sourcePath)
	}

	// Resolve target path
	targetPath := data.TargetPath.ValueString()
	if strings.HasPrefix(targetPath, "~") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", "", fmt.Errorf("unable to get home directory: %w", err)
		}
		targetPath = strings.Replace(targetPath, "~", homeDir, 1)
	}

	// Convert to absolute paths
	fullSourcePath, err := filepath.Abs(fullSourcePath)
	if err != nil {
		return "", "", fmt.Errorf("failed to get absolute source path: %w", err)
	}

	targetPath, err = filepath.Abs(targetPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to get absolute target path: %w", err)
	}

	return fullSourcePath, targetPath, nil
}

// syncDirectory synchronizes the source directory to the target location.
func (r *DirectoryResource) syncDirectory(ctx context.Context, sourcePath, targetPath string, data *DirectoryResourceModel) error {
	// Check if source exists
	if !utils.PathExists(sourcePath) {
		return fmt.Errorf("source directory does not exist: %s", sourcePath)
	}

	// Create target directory if it doesn't exist
	if err := os.MkdirAll(targetPath, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	if data.Recursive.ValueBool() {
		return r.syncDirectoryRecursive(ctx, sourcePath, targetPath, data)
	} else {
		return r.syncDirectoryShallow(ctx, sourcePath, targetPath, data)
	}
}

// syncDirectoryRecursive recursively syncs directories.
func (r *DirectoryResource) syncDirectoryRecursive(ctx context.Context, sourcePath, targetPath string, data *DirectoryResourceModel) error {
	_ = ctx // Context reserved for future logging
	return filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Calculate relative path
		relPath, err := filepath.Rel(sourcePath, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		targetFile := filepath.Join(targetPath, relPath)

		if info.IsDir() {
			// Create directory
			if err := os.MkdirAll(targetFile, info.Mode()); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", targetFile, err)
			}
		} else {
			// Copy file
			if err := r.copyFile(path, targetFile, data); err != nil {
				return fmt.Errorf("failed to copy file %s: %w", path, err)
			}
		}

		return nil
	})
}

// syncDirectoryShallow syncs only the top-level directory contents.
func (r *DirectoryResource) syncDirectoryShallow(ctx context.Context, sourcePath, targetPath string, data *DirectoryResourceModel) error {
	_ = ctx // Context reserved for future logging
	entries, err := os.ReadDir(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to read source directory: %w", err)
	}

	for _, entry := range entries {
		sourceFile := filepath.Join(sourcePath, entry.Name())
		targetFile := filepath.Join(targetPath, entry.Name())

		info, err := entry.Info()
		if err != nil {
			return fmt.Errorf("failed to get file info for %s: %w", entry.Name(), err)
		}

		if info.IsDir() {
			// Create directory
			if err := os.MkdirAll(targetFile, info.Mode()); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", targetFile, err)
			}
		} else {
			// Copy file
			if err := r.copyFile(sourceFile, targetFile, data); err != nil {
				return fmt.Errorf("failed to copy file %s: %w", sourceFile, err)
			}
		}
	}

	return nil
}

// copyFile copies a single file with optional permission preservation.
func (r *DirectoryResource) copyFile(sourcePath, targetPath string, data *DirectoryResourceModel) error {
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	// Create target directory if needed
	targetDir := filepath.Dir(targetPath)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	targetFile, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("failed to create target file: %w", err)
	}
	defer targetFile.Close()

	// Copy content
	if _, err := targetFile.ReadFrom(sourceFile); err != nil {
		return fmt.Errorf("failed to copy file content: %w", err)
	}

	// Preserve permissions if requested
	if data.PreservePermissions.ValueBool() {
		sourceInfo, err := sourceFile.Stat()
		if err != nil {
			return fmt.Errorf("failed to get source file permissions: %w", err)
		}

		if err := targetFile.Chmod(sourceInfo.Mode()); err != nil {
			return fmt.Errorf("failed to set target file permissions: %w", err)
		}
	}

	return nil
}

// updateComputedAttributes updates computed attributes based on current directory state.
func (r *DirectoryResource) updateComputedAttributes(ctx context.Context, data *DirectoryResourceModel, targetPath string) error {
	_ = ctx // Context reserved for future logging
	// Check if directory exists
	exists := utils.PathExists(targetPath)
	data.DirectoryExists = types.BoolValue(exists)

	if exists {
		// Count files
		fileCount, err := r.countFiles(targetPath, data.Recursive.ValueBool())
		if err != nil {
			return fmt.Errorf("failed to count files: %w", err)
		}
		data.FileCount = types.Int64Value(fileCount)
	} else {
		data.FileCount = types.Int64Value(0)
	}

	// Set last synced timestamp
	data.LastSynced = types.StringValue(time.Now().Format(time.RFC3339))

	return nil
}

// countFiles counts the number of files in a directory.
func (r *DirectoryResource) countFiles(dirPath string, recursive bool) (int64, error) {
	var count int64

	if recursive {
		err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				count++
			}
			return nil
		})
		return count, err
	} else {
		entries, err := os.ReadDir(dirPath)
		if err != nil {
			return 0, err
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				count++
			}
		}
		return count, nil
	}
}

// validateDeletionSafety performs safety checks before deleting a directory.
func (r *DirectoryResource) validateDeletionSafety(targetPath string) error {
	// Get absolute path
	absPath, err := filepath.Abs(targetPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// List of paths that should never be deleted
	dangerousPaths := []string{
		"/",
		"/bin",
		"/boot",
		"/dev",
		"/etc",
		"/lib",
		"/proc",
		"/root",
		"/sbin",
		"/sys",
		"/tmp",
		"/usr",
		"/var",
	}

	for _, dangerous := range dangerousPaths {
		if absPath == dangerous || strings.HasPrefix(absPath+"/", dangerous+"/") {
			return fmt.Errorf("refusing to delete system directory: %s", absPath)
		}
	}

	// Don't delete if outside of home directory or known safe paths
	homeDir, err := os.UserHomeDir()
	if err == nil {
		if !strings.HasPrefix(absPath, homeDir) && !strings.HasPrefix(absPath, "/tmp") {
			return fmt.Errorf("refusing to delete directory outside home directory: %s", absPath)
		}
	}

	return nil
}

// getRepositoryLocalPath returns the local path for a repository.
func (r *DirectoryResource) getRepositoryLocalPath(repositoryID string) string {
	// For now, assume repository ID maps to the dotfiles root
	// TODO: Implement proper repository lookup when repository state management is added
	_ = repositoryID // TODO: Use repositoryID when repository lookup is implemented
	return r.client.Config.DotfilesRoot
}
