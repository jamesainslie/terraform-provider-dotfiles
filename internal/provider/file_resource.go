// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

package provider

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
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
	// Get post-hooks attributes and merge with base attributes
	baseAttributes := map[string]schema.Attribute{
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
			MarkdownDescription: "File permissions (e.g., '0644') - deprecated, use permissions block",
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
		"permission_rules": GetPermissionRulesAttribute(),
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
	}

	// Add post-hooks attributes
	postHooksAttrs := GetPostHooksAttributes()
	for key, attr := range postHooksAttrs {
		baseAttributes[key] = attr
	}

	// Add enhanced template attributes
	templateAttrs := GetEnhancedTemplateAttributes()
	for key, attr := range templateAttrs {
		baseAttributes[key] = attr
	}

	// Add application detection attributes
	appDetectionAttrs := GetApplicationDetectionAttributes()
	for key, attr := range appDetectionAttrs {
		baseAttributes[key] = attr
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages individual dotfiles with comprehensive features: permissions, backup, templates, hooks, and application detection",
		Attributes:          baseAttributes,
		Blocks: map[string]schema.Block{
			"permissions":   GetPermissionsSchemaBlock(),
			"backup_policy": GetBackupPolicySchemaBlock(),
			"recovery_test": GetRecoveryTestSchemaBlock(),
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
	var data EnhancedFileResourceModelWithApplicationDetection

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

	// Check application requirements before proceeding
	if !data.RequireApplication.IsNull() {
		appDetectionConfig := buildApplicationDetectionConfig(&data)

		shouldSkip := r.checkApplicationRequirements(ctx, appDetectionConfig, &resp.Diagnostics)

		if shouldSkip {
			tflog.Info(ctx, "Skipping file resource - required application not available", map[string]interface{}{
				"required_app": appDetectionConfig.RequiredApplication,
			})
			// Set ID and save state without creating file
			data.ID = data.Name
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			return
		}
	}

	// Get repository information (for local path if it's a Git repository)
	repositoryLocalPath := r.getRepositoryLocalPath(data.Repository.ValueString())

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

	// Build permission configuration
	permConfig, err := buildFilePermissionConfig(&data.EnhancedFileResourceModel)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid permission configuration",
			fmt.Sprintf("Failed to build permission config: %s", err.Error()),
		)
		return
	}

	// Build enhanced backup configuration
	enhancedBackupConfig, err := buildEnhancedBackupConfigFromAppModel(&data)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid backup configuration",
			fmt.Sprintf("Failed to build backup config: %s", err.Error()),
		)
		return
	}

	// Handle backup - use enhanced if available, otherwise fall back to legacy
	if utils.PathExists(expandedTargetPath) {
		if enhancedBackupConfig != nil && enhancedBackupConfig.Enabled {
			// Use enhanced backup
			enhancedBackupConfig.Directory = r.client.Config.BackupDirectory
			_, err := fileManager.CreateEnhancedBackup(expandedTargetPath, enhancedBackupConfig)
			if err != nil {
				resp.Diagnostics.AddWarning(
					"Enhanced backup failed",
					fmt.Sprintf("Could not create enhanced backup of %s: %s", expandedTargetPath, err.Error()),
				)
			}
		} else {
			// Fall back to legacy backup
			backupEnabled := r.client.Config.BackupEnabled
			if !data.BackupEnabled.IsNull() {
				backupEnabled = data.BackupEnabled.ValueBool()
			}

			if backupEnabled {
				_, err := fileManager.CreateBackup(expandedTargetPath, r.client.Config.BackupDirectory)
				if err != nil {
					resp.Diagnostics.AddWarning(
						"Backup failed",
						fmt.Sprintf("Could not create backup of %s: %s", expandedTargetPath, err.Error()),
					)
				}
			}
		}
	}

	var finalErr error

	if data.IsTemplate.ValueBool() {
		// Build enhanced template configuration
		templateConfig, err := buildEnhancedTemplateConfigFromAppModel(&data)
		if err != nil {
			resp.Diagnostics.AddError(
				"Invalid template configuration",
				fmt.Sprintf("Failed to build template config: %s", err.Error()),
			)
			return
		}

		// Process template with enhanced features
		finalErr = r.processEnhancedTemplate(sourcePath, expandedTargetPath, templateConfig, permConfig)
	} else {
		// Regular file copy with enhanced permissions
		finalErr = fileManager.CopyFileWithPermissions(sourcePath, expandedTargetPath, permConfig)
	}

	if finalErr != nil {
		resp.Diagnostics.AddError(
			"File operation failed",
			fmt.Sprintf("Could not create file %s: %s", expandedTargetPath, finalErr.Error()),
		)
		return
	}

	// Execute post-create commands
	if err := executePostCommands(ctx, data.PostCreateCommands, "post-create"); err != nil {
		resp.Diagnostics.AddWarning(
			"Post-create commands failed",
			fmt.Sprintf("File created successfully but post-create commands failed: %s", err.Error()),
		)
	}

	// Update computed attributes
	if err := r.updateComputedAttributes(ctx, &data.FileResourceModel, expandedTargetPath); err != nil {
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
	var data EnhancedFileResourceModelWithApplicationDetection

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
		if err := r.updateComputedAttributes(ctx, &data.FileResourceModel, expandedTargetPath); err != nil {
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
	var data EnhancedFileResourceModelWithApplicationDetection

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
	repositoryLocalPath := r.getRepositoryLocalPath(data.Repository.ValueString())

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

	// Build permission configuration
	permConfig, err := buildFilePermissionConfig(&data.EnhancedFileResourceModel)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid permission configuration",
			fmt.Sprintf("Failed to build permission config: %s", err.Error()),
		)
		return
	}

	// Build enhanced backup configuration
	enhancedBackupConfig, err := buildEnhancedBackupConfigFromAppModel(&data)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid backup configuration",
			fmt.Sprintf("Failed to build backup config: %s", err.Error()),
		)
		return
	}

	// Handle backup before update - use enhanced if available, otherwise fall back to legacy
	if utils.PathExists(expandedTargetPath) {
		if enhancedBackupConfig != nil && enhancedBackupConfig.Enabled {
			// Use enhanced backup
			enhancedBackupConfig.Directory = r.client.Config.BackupDirectory
			_, err := fileManager.CreateEnhancedBackup(expandedTargetPath, enhancedBackupConfig)
			if err != nil {
				resp.Diagnostics.AddWarning(
					"Enhanced backup failed",
					fmt.Sprintf("Could not create enhanced backup before update: %s", err.Error()),
				)
			}
		} else {
			// Fall back to legacy backup
			backupEnabled := r.client.Config.BackupEnabled
			if !data.BackupEnabled.IsNull() {
				backupEnabled = data.BackupEnabled.ValueBool()
			}

			if backupEnabled {
				_, err := fileManager.CreateBackup(expandedTargetPath, r.client.Config.BackupDirectory)
				if err != nil {
					resp.Diagnostics.AddWarning(
						"Backup failed",
						fmt.Sprintf("Could not create backup before update: %s", err.Error()),
					)
				}
			}
		}
	}

	var finalErr error

	if data.IsTemplate.ValueBool() {
		// Build enhanced template configuration
		templateConfig, err := buildEnhancedTemplateConfigFromAppModel(&data)
		if err != nil {
			resp.Diagnostics.AddError(
				"Invalid template configuration",
				fmt.Sprintf("Failed to build template config: %s", err.Error()),
			)
			return
		}

		// Process template with enhanced features
		finalErr = r.processEnhancedTemplate(sourcePath, expandedTargetPath, templateConfig, permConfig)
	} else {
		// Regular file copy with enhanced permissions
		finalErr = fileManager.CopyFileWithPermissions(sourcePath, expandedTargetPath, permConfig)
	}

	if finalErr != nil {
		resp.Diagnostics.AddError(
			"File update failed",
			fmt.Sprintf("Could not update file %s: %s", expandedTargetPath, finalErr.Error()),
		)
		return
	}

	// Execute post-update commands
	if err := executePostCommands(ctx, data.PostUpdateCommands, "post-update"); err != nil {
		resp.Diagnostics.AddWarning(
			"Post-update commands failed",
			fmt.Sprintf("File updated successfully but post-update commands failed: %s", err.Error()),
		)
	}

	// Update computed attributes
	if err := r.updateComputedAttributes(ctx, &data.FileResourceModel, expandedTargetPath); err != nil {
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
	var data EnhancedFileResourceModelWithApplicationDetection

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting file resource", map[string]interface{}{
		"name":        data.Name.ValueString(),
		"target_path": data.TargetPath.ValueString(),
	})

	// Execute pre-destroy commands
	if err := executePostCommands(ctx, data.PreDestroyCommands, "pre-destroy"); err != nil {
		resp.Diagnostics.AddWarning(
			"Pre-destroy commands failed",
			fmt.Sprintf("Pre-destroy commands failed: %s", err.Error()),
		)
	}

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
func (r *FileResource) getRepositoryLocalPath(repositoryID string) string {
	// For now, assume repository ID maps to the dotfiles root
	// TODO: Implement proper repository lookup when repository state management is added
	_ = repositoryID // TODO: Use repositoryID when repository lookup is implemented
	return r.client.Config.DotfilesRoot
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

// buildEnhancedBackupConfigFromFileModel builds enhanced backup config from file model.
func buildEnhancedBackupConfigFromFileModel(data *EnhancedFileResourceModelWithBackup) (*fileops.EnhancedBackupConfig, error) {
	if data.BackupPolicy == nil {
		return nil, nil
	}

	config := &fileops.EnhancedBackupConfig{
		Enabled:        data.BackupPolicy.AlwaysBackup.ValueBool() || !data.BackupPolicy.AlwaysBackup.IsNull(),
		BackupFormat:   data.BackupPolicy.BackupFormat.ValueString(),
		MaxBackups:     data.BackupPolicy.RetentionCount.ValueInt64(),
		BackupMetadata: data.BackupPolicy.BackupMetadata.ValueBool(),
		Compression:    data.BackupPolicy.Compression.ValueBool(),
		Incremental:    data.BackupPolicy.VersionedBackup.ValueBool(),
		BackupIndex:    true, // Always enable for file-level policies
	}

	// Set defaults if not specified
	if config.BackupFormat == "" {
		config.BackupFormat = "timestamped"
	}
	if config.MaxBackups == 0 {
		config.MaxBackups = 5
	}

	return config, fileops.ValidateEnhancedBackupConfig(config)
}

// buildEnhancedBackupConfigFromTemplateModel builds enhanced backup config from template model.
func buildEnhancedBackupConfigFromTemplateModel(data *EnhancedFileResourceModelWithTemplate) (*fileops.EnhancedBackupConfig, error) {
	return buildEnhancedBackupConfigFromFileModel(&data.EnhancedFileResourceModelWithBackup)
}

// buildEnhancedTemplateConfigFromAppModel builds template config from application detection model.
func buildEnhancedTemplateConfigFromAppModel(data *EnhancedFileResourceModelWithApplicationDetection) (*EnhancedTemplateConfig, error) {
	return buildEnhancedTemplateConfig(&data.EnhancedFileResourceModelWithTemplate)
}

// buildEnhancedBackupConfigFromAppModel builds backup config from application detection model.
func buildEnhancedBackupConfigFromAppModel(data *EnhancedFileResourceModelWithApplicationDetection) (*fileops.EnhancedBackupConfig, error) {
	return buildEnhancedBackupConfigFromTemplateModel(&data.EnhancedFileResourceModelWithTemplate)
}

// buildEnhancedTemplateConfig builds template configuration from template model.
func buildEnhancedTemplateConfig(data *EnhancedFileResourceModelWithTemplate) (*EnhancedTemplateConfig, error) {
	config := &EnhancedTemplateConfig{
		Engine:          "go", // default
		UserVars:        make(map[string]interface{}),
		PlatformVars:    make(map[string]map[string]interface{}),
		CustomFunctions: make(map[string]interface{}),
	}

	// Set template engine
	if !data.TemplateEngine.IsNull() {
		config.Engine = data.TemplateEngine.ValueString()
	}

	// Parse template vars
	if !data.TemplateVars.IsNull() {
		elements := data.TemplateVars.Elements()
		for key, value := range elements {
			if strValue, ok := value.(types.String); ok {
				config.UserVars[key] = strValue.ValueString()
			}
		}
	}

	// Parse platform template vars
	if !data.PlatformTemplateVars.IsNull() {
		elements := data.PlatformTemplateVars.Elements()
		for platform, platformVarsValue := range elements {
			if objValue, ok := platformVarsValue.(types.Object); ok {
				platformMap := make(map[string]interface{})
				objAttrs := objValue.Attributes()
				for key, attrValue := range objAttrs {
					if strValue, ok := attrValue.(types.String); ok {
						platformMap[key] = strValue.ValueString()
					}
				}
				config.PlatformVars[platform] = platformMap
			}
		}
	}

	// Parse template functions (simple string mappings for now)
	if !data.TemplateFunctions.IsNull() {
		elements := data.TemplateFunctions.Elements()
		for name, funcValue := range elements {
			if strValue, ok := funcValue.(types.String); ok {
				// For now, store as string values - could be enhanced to support actual functions
				config.CustomFunctions[name] = strValue.ValueString()
			}
		}
	}

	return config, ValidateEnhancedTemplateConfig(config)
}

// EnhancedTemplateConfig represents enhanced template configuration.
type EnhancedTemplateConfig struct {
	Engine          string
	UserVars        map[string]interface{}
	PlatformVars    map[string]map[string]interface{}
	CustomFunctions map[string]interface{}
}

// ValidateEnhancedTemplateConfig validates enhanced template configuration.
func ValidateEnhancedTemplateConfig(config *EnhancedTemplateConfig) error {
	if config == nil {
		return nil
	}

	// Validate template engine
	validEngines := []string{"go", "handlebars", "mustache"}
	valid := false
	for _, engine := range validEngines {
		if config.Engine == engine {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("invalid template engine: %s (must be one of: %v)", config.Engine, validEngines)
	}

	return nil
}

// buildFilePermissionConfig builds a PermissionConfig from the enhanced model data.
func buildFilePermissionConfig(data *EnhancedFileResourceModel) (*fileops.PermissionConfig, error) {
	config := &fileops.PermissionConfig{}

	// Handle permissions block
	if data.Permissions != nil {
		if !data.Permissions.Directory.IsNull() {
			config.DirectoryMode = data.Permissions.Directory.ValueString()
		}
		if !data.Permissions.Files.IsNull() {
			config.FileMode = data.Permissions.Files.ValueString()
		}
		if !data.Permissions.Recursive.IsNull() {
			config.Recursive = data.Permissions.Recursive.ValueBool()
		}
	}

	// Handle permission rules
	if !data.PermissionRules.IsNull() && !data.PermissionRules.IsUnknown() {
		config.Rules = make(map[string]string)
		elements := data.PermissionRules.Elements()
		for pattern, permValue := range elements {
			if strPerm, ok := permValue.(types.String); ok {
				config.Rules[pattern] = strPerm.ValueString()
			}
		}
	}

	// Fallback to legacy file_mode if no permissions are set
	if config.FileMode == "" && !data.FileMode.IsNull() {
		config.FileMode = data.FileMode.ValueString()
	}

	return config, fileops.ValidatePermissionConfig(config)
}

// executePostCommands executes post-creation/update commands.
func executePostCommands(ctx context.Context, commands types.List, operation string) error {
	if commands.IsNull() || commands.IsUnknown() {
		return nil
	}

	tflog.Debug(ctx, fmt.Sprintf("Executing %s commands", operation))

	elements := commands.Elements()
	for i, cmdValue := range elements {
		if strCmd, ok := cmdValue.(types.String); ok {
			cmd := strCmd.ValueString()
			tflog.Debug(ctx, fmt.Sprintf("Executing %s command %d: %s", operation, i+1, cmd))

			if err := executeShellCommand(ctx, cmd); err != nil {
				return fmt.Errorf("command %d failed: %w", i+1, err)
			}
		}
	}

	return nil
}

// processEnhancedTemplate processes a template with enhanced features.
func (r *FileResource) processEnhancedTemplate(sourcePath, targetPath string, config *EnhancedTemplateConfig, permConfig *fileops.PermissionConfig) error {
	// Create template engine based on configuration
	var engine template.TemplateEngine
	var err error

	if len(config.CustomFunctions) > 0 {
		engine, err = template.CreateTemplateEngineWithFunctions(config.Engine, config.CustomFunctions)
	} else {
		engine, err = template.CreateTemplateEngine(config.Engine)
	}
	if err != nil {
		return fmt.Errorf("failed to create template engine: %w", err)
	}

	// Build comprehensive template context
	systemInfo := r.client.GetPlatformInfo()
	templateContext := template.BuildPlatformAwareTemplateContext(
		systemInfo,
		config.UserVars,
		config.PlatformVars,
	)

	// Process template file
	err = engine.ProcessTemplateFile(sourcePath, targetPath, templateContext, permConfig.FileMode)
	if err != nil {
		return fmt.Errorf("failed to process template file: %w", err)
	}

	// Apply permissions after template processing
	err = r.fileManager().ApplyPermissions(targetPath, permConfig)
	if err != nil {
		return fmt.Errorf("failed to apply permissions after template processing: %w", err)
	}

	return nil
}

// fileManager creates a file manager instance for this resource.
func (r *FileResource) fileManager() *fileops.FileManager {
	platformProvider := platform.DetectPlatform()
	return fileops.NewFileManager(platformProvider, r.client.Config.DryRun)
}

// checkApplicationRequirements checks if required applications are available.
func (r *FileResource) checkApplicationRequirements(ctx context.Context, config *ApplicationDetectionConfig, diagnostics *diag.Diagnostics) bool {
	if config.RequiredApplication == "" {
		return false // No application required
	}

	// Create a temporary ApplicationResource for detection
	appResource := &ApplicationResource{client: r.client}

	// Create model for detection
	detectionModel := ApplicationResourceModel{
		Application:        types.StringValue(config.RequiredApplication),
		DetectInstallation: types.BoolValue(true),
		// Detection methods will use defaults since blocks are handled separately
	}

	// Perform detection
	result := appResource.performApplicationDetection(ctx, &detectionModel)

	// Check if application is installed
	if !result.Installed {
		if config.SkipIfMissing {
			return true // Skip this resource
		}
		// Add warning if configured to warn
		diagnostics.AddWarning(
			"Required application not found",
			fmt.Sprintf("Required application %s is not installed", config.RequiredApplication),
		)
	}

	// Check version compatibility if version info is available
	if result.Version != "" && result.Version != "unknown" {
		if !isVersionCompatible(result.Version, config.MinVersion, config.MaxVersion) {
			message := fmt.Sprintf("Application %s version %s is not compatible", config.RequiredApplication, result.Version)
			if config.MinVersion != "" {
				message += fmt.Sprintf(" (min: %s)", config.MinVersion)
			}
			if config.MaxVersion != "" {
				message += fmt.Sprintf(" (max: %s)", config.MaxVersion)
			}

			if config.SkipIfMissing {
				diagnostics.AddWarning("Application version incompatible", message+" - skipping configuration")
				return true // Skip this resource
			} else {
				diagnostics.AddWarning("Application version incompatible", message+" - proceeding anyway")
			}
		}
	}

	return false // Don't skip
}

// buildApplicationDetectionConfig builds application detection config from model.
func buildApplicationDetectionConfig(data *EnhancedFileResourceModelWithApplicationDetection) *ApplicationDetectionConfig {
	config := &ApplicationDetectionConfig{}

	if !data.RequireApplication.IsNull() {
		config.RequiredApplication = data.RequireApplication.ValueString()
	}
	if !data.ApplicationVersionMin.IsNull() {
		config.MinVersion = data.ApplicationVersionMin.ValueString()
	}
	if !data.ApplicationVersionMax.IsNull() {
		config.MaxVersion = data.ApplicationVersionMax.ValueString()
	}
	if !data.SkipIfAppMissing.IsNull() {
		config.SkipIfMissing = data.SkipIfAppMissing.ValueBool()
	}

	return config
}

// isVersionCompatible checks if a version is within specified bounds.
func isVersionCompatible(detected, minVersion, maxVersion string) bool {
	// Simplified version comparison for now
	// A real implementation would use proper semantic versioning
	if minVersion == "" && maxVersion == "" {
		return true
	}

	// Basic string comparison (would need proper semver library in production)
	if minVersion != "" && detected < minVersion {
		return false
	}
	if maxVersion != "" && detected > maxVersion {
		return false
	}

	return true
}

// executeShellCommand executes a shell command safely.
func executeShellCommand(ctx context.Context, cmdStr string) error {
	// Parse command and arguments
	parts := strings.Fields(cmdStr)
	if len(parts) == 0 {
		return fmt.Errorf("empty command")
	}

	// Use shell to execute complex commands
	var cmd *exec.Cmd

	// Determine shell based on OS
	var shell, shellFlag string
	if strings.Contains(os.Getenv("SHELL"), "fish") {
		shell = "fish"
		shellFlag = "-c"
	} else if runtime.GOOS == "windows" {
		shell = "cmd"
		shellFlag = "/c"
	} else {
		shell = "sh"
		shellFlag = "-c"
	}

	cmd = exec.CommandContext(ctx, shell, shellFlag, cmdStr)

	// Set environment variables
	cmd.Env = os.Environ()

	// Capture output
	output, err := cmd.CombinedOutput()

	if err != nil {
		tflog.Error(ctx, fmt.Sprintf("Command failed: %s", cmdStr), map[string]interface{}{
			"error":  err.Error(),
			"output": string(output),
		})
		return fmt.Errorf("command '%s' failed: %w (output: %s)", cmdStr, err, string(output))
	}

	tflog.Info(ctx, fmt.Sprintf("Command executed successfully: %s", cmdStr), map[string]interface{}{
		"output": string(output),
	})

	return nil
}
