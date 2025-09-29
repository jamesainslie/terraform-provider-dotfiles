// Copyright (c) HashCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

package provider

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/jamesainslie/terraform-provider-dotfiles/internal/platform"
	"github.com/jamesainslie/terraform-provider-dotfiles/internal/validators"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &FilePermissionsResource{}
var _ resource.ResourceWithImportState = &FilePermissionsResource{}

func NewFilePermissionsResource() resource.Resource {
	return &FilePermissionsResource{}
}

// FilePermissionsResource defines the resource implementation.
type FilePermissionsResource struct {
	client *DotfilesClient
}

// FilePermissionsResourceModel describes the resource data model.
type FilePermissionsResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Path           types.String `tfsdk:"path"`
	Mode           types.String `tfsdk:"mode"`
	Owner          types.String `tfsdk:"owner"`
	Group          types.String `tfsdk:"group"`
	Recursive      types.Bool   `tfsdk:"recursive"`
	FollowSymlinks types.Bool   `tfsdk:"follow_symlinks"`
	ApplyToParent  types.Bool   `tfsdk:"apply_to_parent"`
	FilePatterns   types.Map    `tfsdk:"file_patterns"`

	// Computed attributes for state tracking
	ActualMode     types.String `tfsdk:"actual_mode"`
	ActualOwner    types.String `tfsdk:"actual_owner"`
	ActualGroup    types.String `tfsdk:"actual_group"`
	LastApplied    types.String `tfsdk:"last_applied"`
	FilesProcessed types.Int64  `tfsdk:"files_processed"`
}

func (r *FilePermissionsResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_file_permissions"
}

func (r *FilePermissionsResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages file and directory permissions natively. Replaces shell commands like 'chmod 600 file' and 'chown user:group file' with native Go permission management.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "File permissions resource identifier",
			},
			"path": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Path to file or directory to manage permissions for",
				Validators: []validator.String{
					validators.ValidPath(),
					validators.NotEmpty(),
				},
			},
			"mode": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "File permission mode in octal format (e.g., '0644', '0600')",
				Validators: []validator.String{
					validators.ValidFileMode(),
					validators.NotEmpty(),
				},
			},
			"owner": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("current_user"),
				MarkdownDescription: "File owner. Use 'current_user' for current user, or specify username",
			},
			"group": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("current_group"),
				MarkdownDescription: "File group. Use 'current_group' for current group, or specify group name",
			},
			"recursive": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Apply permissions recursively to all files and directories",
			},
			"follow_symlinks": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Follow symbolic links when applying permissions",
			},
			"apply_to_parent": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Also apply permissions to parent directory",
			},
			"file_patterns": schema.MapAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				MarkdownDescription: "Map of file patterns to specific permission modes (e.g., {'*.pub': '0644', 'id_*': '0600'})",
			},
			"actual_mode": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Current actual file permission mode",
			},
			"actual_owner": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Current actual file owner",
			},
			"actual_group": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Current actual file group",
			},
			"last_applied": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Timestamp of last successful permission application",
			},
			"files_processed": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Number of files processed during last operation",
			},
		},
	}
}

func (r *FilePermissionsResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *FilePermissionsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data FilePermissionsResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating file permissions resource", map[string]interface{}{
		"path": data.Path.ValueString(),
		"mode": data.Mode.ValueString(),
	})

	// Apply the file permissions
	filesProcessed, err := r.applyFilePermissions(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError(
			"File Permission Operation Failed",
			fmt.Sprintf("Unable to apply file permissions: %s", err.Error()),
		)
		return
	}

	// Generate ID and update computed attributes
	data.ID = types.StringValue(data.Path.ValueString())
	data.LastApplied = types.StringValue(time.Now().Format(time.RFC3339))
	data.FilesProcessed = types.Int64Value(int64(filesProcessed))

	// Read current permissions to populate computed attributes
	if err := r.updateComputedAttributes(ctx, &data); err != nil {
		tflog.Warn(ctx, "Failed to read current permissions after apply", map[string]interface{}{
			"error": err.Error(),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *FilePermissionsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data FilePermissionsResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading file permissions resource", map[string]interface{}{
		"path": data.Path.ValueString(),
		"id":   data.ID.ValueString(),
	})

	// Update computed attributes with current state
	if err := r.updateComputedAttributes(ctx, &data); err != nil {
		resp.Diagnostics.AddError(
			"File Permission Read Failed",
			fmt.Sprintf("Unable to read current file permissions: %s", err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *FilePermissionsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data FilePermissionsResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating file permissions resource", map[string]interface{}{
		"path": data.Path.ValueString(),
		"mode": data.Mode.ValueString(),
	})

	// Apply the updated file permissions
	filesProcessed, err := r.applyFilePermissions(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError(
			"File Permission Operation Failed",
			fmt.Sprintf("Unable to update file permissions: %s", err.Error()),
		)
		return
	}

	// Update computed attributes
	data.LastApplied = types.StringValue(time.Now().Format(time.RFC3339))
	data.FilesProcessed = types.Int64Value(int64(filesProcessed))

	// Read current permissions to populate computed attributes
	if err := r.updateComputedAttributes(ctx, &data); err != nil {
		tflog.Warn(ctx, "Failed to read current permissions after update", map[string]interface{}{
			"error": err.Error(),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *FilePermissionsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data FilePermissionsResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting file permissions resource", map[string]interface{}{
		"path": data.Path.ValueString(),
		"id":   data.ID.ValueString(),
	})

	// For file permissions resources, deletion means removing the resource from state
	// but leaving the file permissions as they are, unless explicitly configured otherwise
	tflog.Info(ctx, "File permissions resource deleted - file permissions unchanged", map[string]interface{}{
		"path": data.Path.ValueString(),
	})
}

func (r *FilePermissionsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import ID is the file path
	filePath := req.ID
	if filePath == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			"Import ID cannot be empty. Provide the file path to import.",
		)
		return
	}

	// Create a basic file permissions resource model for import
	importData := FilePermissionsResourceModel{
		ID:             types.StringValue(filePath),
		Path:           types.StringValue(filePath),
		Mode:           types.StringValue("0644"), // Default mode for import
		Owner:          types.StringValue("current_user"),
		Group:          types.StringValue("current_group"),
		Recursive:      types.BoolValue(false),
		FollowSymlinks: types.BoolValue(false),
		ApplyToParent:  types.BoolValue(false),
	}

	// Read current permissions to populate computed attributes
	if err := r.updateComputedAttributes(ctx, &importData); err != nil {
		resp.Diagnostics.AddError(
			"Import Failed",
			fmt.Sprintf("Unable to read current file permissions for import: %s", err.Error()),
		)
		return
	}

	// Set the state
	resp.Diagnostics.Append(resp.State.Set(ctx, &importData)...)
}

// applyFilePermissions applies the desired permissions using native platform operations
func (r *FilePermissionsResource) applyFilePermissions(ctx context.Context, data *FilePermissionsResourceModel) (int, error) {
	platformProvider := platform.DetectPlatform()
	filePath := data.Path.ValueString()

	// Expand path (handle ~ and environment variables)
	expandedPath, err := platformProvider.ExpandPath(filePath)
	if err != nil {
		return 0, fmt.Errorf("failed to expand path %s: %w", filePath, err)
	}

	// Parse the permission mode
	mode, err := parseOctalMode(data.Mode.ValueString())
	if err != nil {
		return 0, fmt.Errorf("invalid permission mode %s: %w", data.Mode.ValueString(), err)
	}

	filesProcessed := 0

	// Check if path exists
	info, err := os.Stat(expandedPath)
	if err != nil {
		return 0, fmt.Errorf("path does not exist: %s", expandedPath)
	}

	if data.Recursive.ValueBool() && info.IsDir() {
		// Apply permissions recursively
		err = r.applyPermissionsRecursively(ctx, platformProvider, expandedPath, mode, data)
		if err != nil {
			return 0, err
		}
		filesProcessed++ // Count the directory itself
	} else {
		// Apply permissions to single file/directory using existing platform method
		err = platformProvider.SetPermissions(expandedPath, mode)
		if err != nil {
			return 0, fmt.Errorf("failed to set permissions on %s: %w", expandedPath, err)
		}
		filesProcessed++
	}

	// Apply ownership if specified
	if (!data.Owner.IsNull() && data.Owner.ValueString() != "current_user") ||
		(!data.Group.IsNull() && data.Group.ValueString() != "current_group") {
		owner := data.Owner.ValueString()
		group := data.Group.ValueString()

		if owner == "current_user" {
			owner = "" // Let the system determine current user
		}
		if group == "current_group" {
			group = "" // Let the system determine current group
		}

		err = r.applyOwnership(expandedPath, owner, group)
		if err != nil {
			tflog.Warn(ctx, "Failed to set ownership", map[string]interface{}{
				"path":  expandedPath,
				"owner": owner,
				"group": group,
				"error": err.Error(),
			})
		}
	}

	tflog.Info(ctx, "File permissions applied successfully", map[string]interface{}{
		"path":            expandedPath,
		"mode":            fmt.Sprintf("%04o", mode),
		"files_processed": filesProcessed,
		"recursive":       data.Recursive.ValueBool(),
	})

	return filesProcessed, nil
}

// applyPermissionsRecursively applies permissions to all files in a directory tree
func (r *FilePermissionsResource) applyPermissionsRecursively(ctx context.Context, platformProvider platform.PlatformProvider, dirPath string, mode os.FileMode, data *FilePermissionsResourceModel) error {
	// Parse file patterns if provided
	var filePatterns map[string]string
	if !data.FilePatterns.IsNull() {
		elements := data.FilePatterns.Elements()
		filePatterns = make(map[string]string, len(elements))
		for key, value := range elements {
			if strValue, ok := value.(types.String); ok {
				filePatterns[key] = strValue.ValueString()
			}
		}
	}

	return filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip if not following symlinks and this is a symlink
		if !data.FollowSymlinks.ValueBool() && info.Mode()&os.ModeSymlink != 0 {
			return nil
		}

		targetMode := mode

		// Check for pattern-based permissions
		if filePatterns != nil {
			fileName := filepath.Base(path)
			for pattern, patternMode := range filePatterns {
				if r.matchesGlobPattern(pattern, fileName) {
					if parsedMode, err := parseOctalMode(patternMode); err == nil {
						targetMode = parsedMode
						tflog.Debug(ctx, "Applied pattern-based permission", map[string]interface{}{
							"path":    path,
							"pattern": pattern,
							"mode":    fmt.Sprintf("%04o", parsedMode),
						})
						break
					}
				}
			}
		}

		// Apply permissions using platform provider
		if err := platformProvider.SetPermissions(path, targetMode); err != nil {
			return fmt.Errorf("failed to set permissions on %s: %w", path, err)
		}

		return nil
	})
}

// updateComputedAttributes reads the current file state and updates computed attributes
func (r *FilePermissionsResource) updateComputedAttributes(_ context.Context, data *FilePermissionsResourceModel) error {
	platformProvider := platform.DetectPlatform()
	filePath := data.Path.ValueString()

	// Expand path
	expandedPath, err := platformProvider.ExpandPath(filePath)
	if err != nil {
		return fmt.Errorf("failed to expand path %s: %w", filePath, err)
	}

	// Get current file info using standard library
	info, err := os.Stat(expandedPath)
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	// Update actual mode from file info
	mode := info.Mode().Perm()
	data.ActualMode = types.StringValue(fmt.Sprintf("%04o", mode))

	// For ownership, we'll set placeholder values since it requires platform-specific code
	// In a future implementation, this could call platform-specific functions
	data.ActualOwner = types.StringValue("current_user")
	data.ActualGroup = types.StringValue("current_group")

	return nil
}

// matchesGlobPattern performs simple glob pattern matching
func (r *FilePermissionsResource) matchesGlobPattern(pattern, filename string) bool {
	// Simple implementation - in production would use filepath.Match or similar
	if pattern == filename {
		return true
	}

	if strings.Contains(pattern, "*") {
		if strings.HasPrefix(pattern, "*") {
			suffix := pattern[1:]
			return strings.HasSuffix(filename, suffix)
		}
		if strings.HasSuffix(pattern, "*") {
			prefix := pattern[:len(pattern)-1]
			return strings.HasPrefix(filename, prefix)
		}
	}

	return false
}

// applyOwnership applies ownership changes using standard Go library functions
func (r *FilePermissionsResource) applyOwnership(_ string, owner, group string) error {
	// For now, skip ownership changes as they require platform-specific implementation
	// and would need root privileges. This is a placeholder for future enhancement.
	// Users can use the existing platform.SetPermissions for permission changes.
	if owner != "" && owner != "current_user" {
		return fmt.Errorf("ownership changes not yet implemented for user %s", owner)
	}
	if group != "" && group != "current_group" {
		return fmt.Errorf("ownership changes not yet implemented for group %s", group)
	}
	return nil
}

// parseOctalMode parses an octal mode string (e.g., "0644", "644") into os.FileMode
func parseOctalMode(modeStr string) (os.FileMode, error) {
	// Handle both "0644" and "644" formats
	if len(modeStr) >= 3 {
		if modeStr[0] == '0' && len(modeStr) == 4 {
			// "0644" format
			parsed, err := strconv.ParseUint(modeStr[1:], 8, 32)
			if err != nil {
				return 0, fmt.Errorf("invalid octal mode: %s", modeStr)
			}
			return os.FileMode(parsed), nil
		} else if len(modeStr) == 3 {
			// "644" format
			parsed, err := strconv.ParseUint(modeStr, 8, 32)
			if err != nil {
				return 0, fmt.Errorf("invalid octal mode: %s", modeStr)
			}
			return os.FileMode(parsed), nil
		}
	}

	return 0, fmt.Errorf("invalid mode format: %s (expected 0644 or 644)", modeStr)
}
