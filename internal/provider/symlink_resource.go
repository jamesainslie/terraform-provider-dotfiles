// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

package provider

import (
	"context"
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
	"github.com/jamesainslie/terraform-provider-dotfiles/internal/utils"
)

var _ resource.Resource = &SymlinkResource{}

func NewSymlinkResource() resource.Resource {
	return &SymlinkResource{}
}

type SymlinkResource struct {
	client *DotfilesClient
}

type SymlinkResourceModel struct {
	ID            types.String `tfsdk:"id"`
	Repository    types.String `tfsdk:"repository"`
	Name          types.String `tfsdk:"name"`
	SourcePath    types.String `tfsdk:"source_path"`
	TargetPath    types.String `tfsdk:"target_path"`
	ForceUpdate   types.Bool   `tfsdk:"force_update"`
	CreateParents types.Bool   `tfsdk:"create_parents"`

	// Enhanced fields
	Permissions     *PermissionsModel `tfsdk:"permissions"`
	PermissionRules types.Map         `tfsdk:"permission_rules"`

	// Computed attributes
	LinkExists   types.Bool   `tfsdk:"link_exists"`
	IsSymlink    types.Bool   `tfsdk:"is_symlink"`
	LinkTarget   types.String `tfsdk:"link_target"`
	LastModified types.String `tfsdk:"last_modified"`
}

func (r *SymlinkResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_symlink"
}

func (r *SymlinkResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	// Build base attributes
	baseAttributes := map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Symlink identifier",
		},
		"repository": schema.StringAttribute{
			Required:            true,
			MarkdownDescription: "Repository ID this symlink belongs to",
		},
		"name": schema.StringAttribute{
			Required:            true,
			MarkdownDescription: "Symlink name/identifier",
		},
		"source_path": schema.StringAttribute{
			Required:            true,
			MarkdownDescription: "Path to source in repository",
		},
		"target_path": schema.StringAttribute{
			Required:            true,
			MarkdownDescription: "Target symlink path",
		},
		"force_update": schema.BoolAttribute{
			Optional:            true,
			MarkdownDescription: "Force update existing symlinks",
		},
		"create_parents": schema.BoolAttribute{
			Optional:            true,
			MarkdownDescription: "Create parent directories",
		},
		"permission_rules": GetPermissionRulesAttribute(),
		"link_exists": schema.BoolAttribute{
			Computed:            true,
			MarkdownDescription: "Whether the symlink exists",
		},
		"is_symlink": schema.BoolAttribute{
			Computed:            true,
			MarkdownDescription: "Whether the target is actually a symlink",
		},
		"link_target": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "The target that the symlink points to",
		},
		"last_modified": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Last modification timestamp of the symlink",
		},
	}

	// Shell command hooks removed for security reasons (G204 vulnerability)
	// Use terraform-provider-package for service management operations

	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages symbolic links to dotfiles with comprehensive permission management",
		Attributes:          baseAttributes,
		Blocks: map[string]schema.Block{
			"permissions": GetPermissionsSchemaBlock(),
		},
	}
}

func (r *SymlinkResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*DotfilesClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			"Expected *DotfilesClient, got something else.",
		)
		return
	}

	r.client = client
}

func (r *SymlinkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data SymlinkResourceModel

	tflog.Debug(ctx, "=== SYMLINK CREATE START ===")
	tflog.Debug(ctx, "Getting plan data from request")
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Failed to get plan data", map[string]interface{}{
			"diagnostics_count": len(resp.Diagnostics),
		})
		return
	}

	tflog.Debug(ctx, "Creating symlink resource", map[string]interface{}{
		"name":           data.Name.ValueString(),
		"repository":     data.Repository.ValueString(),
		"source_path":    data.SourcePath.ValueString(),
		"target_path":    data.TargetPath.ValueString(),
		"force_update":   data.ForceUpdate.ValueBool(),
		"create_parents": data.CreateParents.ValueBool(),
	})

	// Get repository local path
	repositoryID := data.Repository.ValueString()
	repositoryLocalPath := r.getRepositoryLocalPath(repositoryID)
	tflog.Debug(ctx, "Retrieved repository local path", map[string]interface{}{
		"repository_id":   repositoryID,
		"repository_path": repositoryLocalPath,
	})

	// Build source path
	sourcePath := filepath.Join(repositoryLocalPath, data.SourcePath.ValueString())
	targetPath := data.TargetPath.ValueString()
	tflog.Debug(ctx, "Built file paths", map[string]interface{}{
		"raw_source_path":  data.SourcePath.ValueString(),
		"full_source_path": sourcePath,
		"raw_target_path":  targetPath,
	})

	// Expand target path
	platformProvider := platform.DetectPlatform()
	tflog.Debug(ctx, "Detected platform", map[string]interface{}{
		"platform_type": fmt.Sprintf("%T", platformProvider),
	})

	expandedTargetPath, err := platformProvider.ExpandPath(targetPath)
	if err != nil {
		tflog.Error(ctx, "Failed to expand target path", map[string]interface{}{
			"error":       err.Error(),
			"target_path": targetPath,
		})
		resp.Diagnostics.AddError(
			"Invalid target path",
			fmt.Sprintf("Could not expand target path %s: %s", targetPath, err.Error()),
		)
		return
	}
	tflog.Debug(ctx, "Expanded target path", map[string]interface{}{
		"original_target": targetPath,
		"expanded_target": expandedTargetPath,
	})

	// Expand source path
	expandedSourcePath, err := platformProvider.ExpandPath(sourcePath)
	if err != nil {
		tflog.Error(ctx, "Failed to expand source path", map[string]interface{}{
			"error":       err.Error(),
			"source_path": sourcePath,
		})
		resp.Diagnostics.AddError(
			"Invalid source path",
			fmt.Sprintf("Could not expand source path %s: %s", sourcePath, err.Error()),
		)
		return
	}
	tflog.Debug(ctx, "Expanded source path", map[string]interface{}{
		"original_source": sourcePath,
		"expanded_source": expandedSourcePath,
	})

	// Verify source exists
	tflog.Debug(ctx, "Checking if source exists", map[string]interface{}{
		"expanded_source": expandedSourcePath,
	})
	sourceExists := utils.PathExists(expandedSourcePath)
	if !sourceExists {
		tflog.Error(ctx, "Source path does not exist", map[string]interface{}{
			"expanded_source": expandedSourcePath,
		})
		resp.Diagnostics.AddError(
			"Source not found",
			fmt.Sprintf("Source path does not exist: %s", expandedSourcePath),
		)
		return
	}
	tflog.Debug(ctx, "Source path verified", map[string]interface{}{
		"expanded_source": expandedSourcePath,
	})

	// Create file manager
	dryRun := r.client.Config.DryRun
	tflog.Debug(ctx, "Creating file manager", map[string]interface{}{
		"dry_run": dryRun,
	})
	fileManager := fileops.NewFileManager(platformProvider, dryRun)

	// Handle existing target
	tflog.Debug(ctx, "Checking if target exists", map[string]interface{}{
		"expanded_target": expandedTargetPath,
	})
	targetExists := utils.PathExists(expandedTargetPath)
	tflog.Debug(ctx, "Target existence check result", map[string]interface{}{
		"target_exists": targetExists,
	})

	if targetExists {
		if !data.ForceUpdate.ValueBool() {
			// Create backup if enabled
			if r.client.Config.BackupEnabled {
				_, err := fileManager.CreateBackup(expandedTargetPath, r.client.Config.BackupDirectory)
				if err != nil {
					resp.Diagnostics.AddWarning(
						"Backup failed",
						fmt.Sprintf("Could not create backup of existing target: %s", err.Error()),
					)
				}
			}
		}

		// Remove existing target (handle both files and directories)
		tflog.Debug(ctx, "Statting existing target for removal", map[string]interface{}{
			"expanded_target": expandedTargetPath,
		})
		info, err := os.Stat(expandedTargetPath)
		if err != nil {
			tflog.Error(ctx, "Failed to stat existing target", map[string]interface{}{
				"error":           err.Error(),
				"expanded_target": expandedTargetPath,
			})
			resp.Diagnostics.AddError(
				"Could not stat existing target",
				fmt.Sprintf("Could not stat existing target at %s: %s", expandedTargetPath, err.Error()),
			)
			return
		}

		isDir := info.IsDir()
		tflog.Debug(ctx, "Target stat results", map[string]interface{}{
			"is_directory": isDir,
			"mode":         info.Mode().String(),
			"size":         info.Size(),
		})

		if isDir {
			tflog.Debug(ctx, "Removing existing directory with RemoveAll")
			// Use RemoveAll for directories
			err = os.RemoveAll(expandedTargetPath)
		} else {
			tflog.Debug(ctx, "Removing existing file with Remove")
			// Use Remove for files
			err = os.Remove(expandedTargetPath)
		}

		if err != nil {
			tflog.Error(ctx, "Failed to remove existing target", map[string]interface{}{
				"error":           err.Error(),
				"expanded_target": expandedTargetPath,
				"was_directory":   isDir,
			})
			resp.Diagnostics.AddError(
				"Could not remove existing target",
				fmt.Sprintf("Could not remove existing target at %s: %s", expandedTargetPath, err.Error()),
			)
			return
		}
		tflog.Debug(ctx, "Successfully removed existing target", map[string]interface{}{
			"expanded_target": expandedTargetPath,
			"was_directory":   isDir,
		})
	}

	// Create symlink
	tflog.Debug(ctx, "Creating symlink", map[string]interface{}{
		"source_path":    expandedSourcePath,
		"target_path":    expandedTargetPath,
		"create_parents": data.CreateParents.ValueBool(),
	})

	var finalErr error
	if data.CreateParents.ValueBool() {
		tflog.Debug(ctx, "Using CreateSymlinkWithParents")
		finalErr = fileManager.CreateSymlinkWithParents(expandedSourcePath, expandedTargetPath)
	} else {
		tflog.Debug(ctx, "Using CreateSymlink")
		finalErr = fileManager.CreateSymlink(expandedSourcePath, expandedTargetPath)
	}

	if finalErr != nil {
		tflog.Error(ctx, "Symlink creation failed", map[string]interface{}{
			"error":       finalErr.Error(),
			"source_path": expandedSourcePath,
			"target_path": expandedTargetPath,
		})
		resp.Diagnostics.AddError(
			"Symlink creation failed",
			fmt.Sprintf("Could not create symlink %s -> %s: %s", expandedTargetPath, expandedSourcePath, finalErr.Error()),
		)
		return
	}
	tflog.Debug(ctx, "Symlink created successfully")

	// Update computed attributes
	tflog.Debug(ctx, "Updating computed attributes")
	if err := r.updateComputedAttributes(ctx, &data, expandedTargetPath); err != nil {
		tflog.Warn(ctx, "Failed to update computed attributes, setting defaults", map[string]interface{}{
			"error": err.Error(),
		})
		// Set default values for computed attributes if update fails
		data.LinkExists = types.BoolValue(utils.PathExists(expandedTargetPath))
		data.IsSymlink = types.BoolValue(false)
		data.LinkTarget = types.StringNull()
		data.LastModified = types.StringValue(time.Now().Format(time.RFC3339))

		resp.Diagnostics.AddWarning(
			"Could not update symlink metadata",
			fmt.Sprintf("Symlink created successfully but could not update metadata: %s", err.Error()),
		)
	}

	// Set ID and save state
	tflog.Debug(ctx, "Setting resource ID and saving state", map[string]interface{}{
		"id": data.Name.ValueString(),
	})
	data.ID = data.Name
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	tflog.Info(ctx, "Symlink resource created successfully", map[string]interface{}{
		"name":        data.Name.ValueString(),
		"target_path": expandedTargetPath,
		"source_path": expandedSourcePath,
		"link_exists": data.LinkExists.ValueBool(),
		"is_symlink":  data.IsSymlink.ValueBool(),
	})
	tflog.Debug(ctx, "=== SYMLINK CREATE END ===")
}

func (r *SymlinkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data SymlinkResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading symlink resource", map[string]interface{}{
		"name":        data.Name.ValueString(),
		"target_path": data.TargetPath.ValueString(),
	})

	// Expand target path to check current state
	platformProvider := platform.DetectPlatform()
	expandedTargetPath, err := platformProvider.ExpandPath(data.TargetPath.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to expand target path",
			fmt.Sprintf("Error expanding target path for symlink %s: %v", data.Name.ValueString(), err),
		)
		return
	}

	// Update computed attributes with current state
	err = r.updateComputedAttributes(ctx, &data, expandedTargetPath)
	if err != nil {
		resp.Diagnostics.AddWarning(
			"Failed to update computed attributes",
			fmt.Sprintf("Error updating attributes for symlink %s: %v", data.Name.ValueString(), err),
		)
		// Set default values if update fails
		data.LinkExists = types.BoolValue(utils.PathExists(expandedTargetPath))
		data.IsSymlink = types.BoolValue(utils.IsSymlink(expandedTargetPath))
		data.LinkTarget = types.StringNull()
		data.LastModified = types.StringValue(time.Now().Format(time.RFC3339))
	}

	// Check if symlink exists in correct form
	if data.LinkExists.ValueBool() {
		if data.IsSymlink.ValueBool() {
			// Case 1: Symlink exists - check if target is correct
			actualTarget, err := os.Readlink(expandedTargetPath)
			if err != nil {
				tflog.Warn(ctx, "Could not read symlink target", map[string]interface{}{
					"target_path": expandedTargetPath,
					"error":       err.Error(),
				})
				// Symlink is corrupted - remove from state to trigger recreation
				tflog.Info(ctx, "Removing corrupted symlink from state to trigger recreation")
				return
			} else {
				// Expand the source path to compare
				repositoryLocalPath := r.getRepositoryLocalPath(data.Repository.ValueString())
				sourcePath := filepath.Join(repositoryLocalPath, data.SourcePath.ValueString())
				expandedSourcePath, err := platformProvider.ExpandPath(sourcePath)
				if err == nil {
					// Make paths absolute for comparison
					expectedTarget, _ := filepath.Abs(expandedSourcePath)
					actualTargetAbs, _ := filepath.Abs(actualTarget)

					if expectedTarget != actualTargetAbs {
						tflog.Info(ctx, "Symlink points to wrong target - removing from state", map[string]interface{}{
							"expected": expectedTarget,
							"actual":   actualTargetAbs,
						})
						// Wrong target - remove from state to trigger recreation
						return
					}
				}
			}
		} else {
			// Case 2: CRITICAL - Directory/file exists instead of symlink!
			// Remove from state to trigger recreation
			tflog.Info(ctx, "Directory/file exists instead of expected symlink - removing from state", map[string]interface{}{
				"target_path": expandedTargetPath,
				"expected":    "symlink",
				"actual":      "directory or file",
			})

			// Log details for debugging
			if info, err := os.Lstat(expandedTargetPath); err == nil {
				tflog.Info(ctx, "Target details", map[string]interface{}{
					"mode":   info.Mode().String(),
					"is_dir": info.IsDir(),
					"size":   info.Size(),
				})
			}

			// Don't set state - this will cause Terraform to detect the resource is gone
			// and needs to be recreated
			resp.State.RemoveResource(ctx)
			return
		}
	} else {
		// Case 3: Nothing exists at target path - resource is gone
		tflog.Info(ctx, "Symlink target does not exist - removing from state", map[string]interface{}{
			"target_path": expandedTargetPath,
		})
		// Don't set state - resource doesn't exist anymore
		resp.State.RemoveResource(ctx)
		return
	}

	// Only set state if symlink exists and is correct
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SymlinkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data SymlinkResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set ID and save state
	data.ID = data.Name
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SymlinkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data SymlinkResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting symlink resource", map[string]interface{}{
		"name":        data.Name.ValueString(),
		"target_path": data.TargetPath.ValueString(),
	})

	// Remove the symlink
	targetPath := data.TargetPath.ValueString()
	if targetPath != "" {
		platformProvider := platform.DetectPlatform()
		expandedTargetPath, err := platformProvider.ExpandPath(targetPath)
		if err != nil {
			resp.Diagnostics.AddWarning(
				"Could not expand target path",
				fmt.Sprintf("Could not expand target path for cleanup: %s", err.Error()),
			)
			return
		}

		if utils.PathExists(expandedTargetPath) {
			// Handle both files and directories (same logic as Create function)
			info, err := os.Lstat(expandedTargetPath) // Use Lstat to handle symlinks properly
			if err != nil {
				resp.Diagnostics.AddWarning(
					"Could not stat target for removal",
					fmt.Sprintf("Could not stat target %s for removal: %s", expandedTargetPath, err.Error()),
				)
			} else {
				// Remove based on what it actually is
				if info.IsDir() {
					err = os.RemoveAll(expandedTargetPath)
				} else {
					err = os.Remove(expandedTargetPath)
				}

				if err != nil {
					resp.Diagnostics.AddWarning(
						"Could not remove target",
						fmt.Sprintf("Could not remove target %s: %s", expandedTargetPath, err.Error()),
					)
				} else {
					tflog.Info(ctx, "Target removed successfully", map[string]interface{}{
						"target_path":   expandedTargetPath,
						"was_directory": info.IsDir(),
					})
				}
			}
		}
	}
}

// getRepositoryLocalPath returns the local path for a repository.
func (r *SymlinkResource) getRepositoryLocalPath(repositoryID string) string {
	// For now, assume repository ID maps to the dotfiles root
	// NOTE: Repository lookup feature planned for v0.2.0 - will enable proper multi-repository support
	_ = repositoryID // Placeholder for future repository lookup implementation
	return r.client.Config.DotfilesRoot
}

// updateComputedAttributes updates computed attributes for state tracking.
func (r *SymlinkResource) updateComputedAttributes(ctx context.Context, data *SymlinkResourceModel, targetPath string) error {
	tflog.Debug(ctx, "Updating computed attributes for symlink", map[string]interface{}{
		"target_path": targetPath,
	})

	// Check if link exists
	exists := utils.PathExists(targetPath)
	data.LinkExists = types.BoolValue(exists)
	tflog.Debug(ctx, "Link exists check", map[string]interface{}{
		"exists": exists,
	})

	if exists {
		// Check if it's actually a symlink
		isSymlink := utils.IsSymlink(targetPath)
		data.IsSymlink = types.BoolValue(isSymlink)
		tflog.Debug(ctx, "Symlink check", map[string]interface{}{
			"is_symlink": isSymlink,
		})

		if isSymlink {
			// Get symlink target
			linkTarget, err := os.Readlink(targetPath)
			if err != nil {
				tflog.Warn(ctx, "Failed to read symlink target", map[string]interface{}{
					"error": err.Error(),
				})
				data.LinkTarget = types.StringNull()
				// Return error for critical symlink read failures
				return fmt.Errorf("failed to read symlink target: %w", err)
			} else {
				data.LinkTarget = types.StringValue(linkTarget)
				tflog.Debug(ctx, "Symlink target read", map[string]interface{}{
					"target": linkTarget,
				})
			}
		} else {
			data.LinkTarget = types.StringNull()
		}

		// Get modification time
		info, err := os.Lstat(targetPath) // Use Lstat to get symlink info, not target info
		if err != nil {
			tflog.Warn(ctx, "Failed to stat symlink, using current time", map[string]interface{}{
				"error": err.Error(),
			})
			data.LastModified = types.StringValue(time.Now().Format(time.RFC3339))
			// Don't return error for stat failures - they're not critical
		} else {
			data.LastModified = types.StringValue(info.ModTime().Format(time.RFC3339))
			tflog.Debug(ctx, "Modification time set", map[string]interface{}{
				"mod_time": info.ModTime().Format(time.RFC3339),
			})
		}
	} else {
		data.IsSymlink = types.BoolValue(false)
		data.LinkTarget = types.StringNull()
		data.LastModified = types.StringNull()
		tflog.Debug(ctx, "Link does not exist, set default values")
	}

	tflog.Debug(ctx, "Computed attributes updated successfully")
	return nil
}
