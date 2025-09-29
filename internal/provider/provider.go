// Copyright (c) HashCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/jamesainslie/terraform-provider-dotfiles/internal/validators"
)

// Ensure DotfilesProvider satisfies various provider interfaces.
var _ provider.Provider = &DotfilesProvider{}

// DotfilesProvider defines the provider implementation.
type DotfilesProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// DotfilesProviderModel describes the provider data model.
type DotfilesProviderModel struct {
	DotfilesRoot       types.String         `tfsdk:"dotfiles_root"`
	BackupEnabled      types.Bool           `tfsdk:"backup_enabled"`
	BackupDirectory    types.String         `tfsdk:"backup_directory"`
	Strategy           types.String         `tfsdk:"strategy"`
	ConflictResolution types.String         `tfsdk:"conflict_resolution"`
	DryRun             types.Bool           `tfsdk:"dry_run"`
	AutoDetectPlatform types.Bool           `tfsdk:"auto_detect_platform"`
	TargetPlatform     types.String         `tfsdk:"target_platform"`
	TemplateEngine     types.String         `tfsdk:"template_engine"`
	LogLevel           types.String         `tfsdk:"log_level"`
	BackupStrategy     *BackupStrategyModel `tfsdk:"backup_strategy"`
	Recovery           *RecoveryModel       `tfsdk:"recovery"`
}

func (p *DotfilesProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "dotfiles"
	resp.Version = p.version
}

func (p *DotfilesProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Terraform provider for managing dotfiles in a declarative, cross-platform manner.",
		Attributes: map[string]schema.Attribute{
			"dotfiles_root": schema.StringAttribute{
				MarkdownDescription: "Root directory of the dotfiles repository. Defaults to ~/dotfiles",
				Optional:            true,
			},
			"backup_enabled": schema.BoolAttribute{
				MarkdownDescription: "Enable automatic backups of existing files before modification. Defaults to true",
				Optional:            true,
			},
			"backup_directory": schema.StringAttribute{
				MarkdownDescription: "Directory to store backup files. Defaults to ~/.dotfiles-backups",
				Optional:            true,
			},
			"strategy": schema.StringAttribute{
				MarkdownDescription: "Default strategy for file management: symlink (default), copy, or template",
				Optional:            true,
			},
			"conflict_resolution": schema.StringAttribute{
				MarkdownDescription: "How to handle conflicts: backup (default), overwrite, skip, or prompt",
				Optional:            true,
			},
			"dry_run": schema.BoolAttribute{
				MarkdownDescription: "Preview changes without applying them. Defaults to false",
				Optional:            true,
			},
			"auto_detect_platform": schema.BoolAttribute{
				MarkdownDescription: "Automatically detect the target platform. Defaults to true",
				Optional:            true,
			},
			"target_platform": schema.StringAttribute{
				MarkdownDescription: "Target platform: auto (default), macos, linux, or windows",
				Optional:            true,
			},
			"template_engine": schema.StringAttribute{
				MarkdownDescription: "Template engine to use: go (default), handlebars, or mustache",
				Optional:            true,
				Validators: []validator.String{
					validators.ValidTemplateEngine(),
				},
			},
			"log_level": schema.StringAttribute{
				MarkdownDescription: "Log level: debug, info (default), warn, or error",
				Optional:            true,
			},
		},
		Blocks: map[string]schema.Block{
			"backup_strategy": GetBackupStrategySchemaBlock(),
			"recovery":        GetRecoverySchemaBlock(),
		},
	}
}

func (p *DotfilesProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data EnhancedProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create and configure the client
	config := &DotfilesConfig{}

	// Map provider data to configuration
	p.mapProviderDataToConfig(&data, config)

	// Set defaults for any empty values
	if err := config.SetDefaults(); err != nil {
		resp.Diagnostics.AddError(
			"Unable to set configuration defaults",
			"An error occurred while setting default configuration values: "+err.Error(),
		)
		return
	}

	// Handle backup strategy configuration
	p.handleBackupStrategyConfig(ctx, &data, config, resp)
	if resp.Diagnostics.HasError() {
		return
	}

	// Handle recovery configuration
	p.handleRecoveryConfig(ctx, &data)

	// Create and validate client
	client, err := p.createAndValidateClient(config, resp)
	if err != nil {
		return
	}

	tflog.Info(ctx, "Configured dotfiles provider", map[string]interface{}{
		"dotfiles_root":   config.DotfilesRoot,
		"backup_enabled":  config.BackupEnabled,
		"strategy":        config.Strategy,
		"target_platform": config.TargetPlatform,
		"dry_run":         config.DryRun,
	})

	resp.DataSourceData = client
	resp.ResourceData = client
}

// mapProviderDataToConfig maps provider data to configuration struct
func (p *DotfilesProvider) mapProviderDataToConfig(data *EnhancedProviderModel, config *DotfilesConfig) {
	if !data.DotfilesRoot.IsNull() {
		config.DotfilesRoot = data.DotfilesRoot.ValueString()
	}

	if !data.BackupEnabled.IsNull() {
		config.BackupEnabled = data.BackupEnabled.ValueBool()
	} else {
		config.BackupEnabled = true // default to true
	}

	if !data.BackupDirectory.IsNull() {
		config.BackupDirectory = data.BackupDirectory.ValueString()
	}

	if !data.Strategy.IsNull() {
		config.Strategy = data.Strategy.ValueString()
	}

	if !data.ConflictResolution.IsNull() {
		config.ConflictResolution = data.ConflictResolution.ValueString()
	}

	if !data.DryRun.IsNull() {
		config.DryRun = data.DryRun.ValueBool()
	} else {
		config.DryRun = false
	}

	if !data.AutoDetectPlatform.IsNull() {
		config.AutoDetectPlatform = data.AutoDetectPlatform.ValueBool()
	} else {
		config.AutoDetectPlatform = true
	}

	if !data.TargetPlatform.IsNull() {
		config.TargetPlatform = data.TargetPlatform.ValueString()
	}

	if !data.TemplateEngine.IsNull() {
		config.TemplateEngine = data.TemplateEngine.ValueString()
	}

	if !data.LogLevel.IsNull() {
		config.LogLevel = data.LogLevel.ValueString()
	}
}

// handleBackupStrategyConfig handles backup strategy configuration and conflict detection
func (p *DotfilesProvider) handleBackupStrategyConfig(ctx context.Context, data *EnhancedProviderModel, config *DotfilesConfig, resp *provider.ConfigureResponse) {
	if data.BackupStrategy == nil {
		return
	}

	// Warn if both top-level and backup_strategy block are used
	if !data.BackupEnabled.IsNull() && !data.BackupStrategy.Enabled.IsNull() {
		resp.Diagnostics.AddWarning(
			"Conflicting backup configuration",
			"Both top-level 'backup_enabled' and 'backup_strategy.enabled' are set. The backup_strategy block takes precedence.",
		)
	}
	if !data.BackupDirectory.IsNull() && !data.BackupStrategy.Directory.IsNull() {
		resp.Diagnostics.AddWarning(
			"Conflicting backup directory configuration",
			"Both top-level 'backup_directory' and 'backup_strategy.directory' are set. The backup_strategy block takes precedence.",
		)
	}

	// Apply backup strategy configuration (overrides top-level settings)
	if !data.BackupStrategy.Enabled.IsNull() {
		config.BackupEnabled = data.BackupStrategy.Enabled.ValueBool()
	}
	if !data.BackupStrategy.Directory.IsNull() {
		config.BackupDirectory = data.BackupStrategy.Directory.ValueString()
	}
	// Additional backup strategy fields can be handled here as needed
	// For now, we keep the existing simple backup configuration approach
}

// handleRecoveryConfig handles recovery configuration
func (p *DotfilesProvider) handleRecoveryConfig(ctx context.Context, data *EnhancedProviderModel) {
	if data.Recovery == nil {
		return
	}

	// Recovery configuration is mainly used by resources
	// Log that recovery features are enabled if configured
	if !data.Recovery.CreateRestoreScripts.IsNull() && data.Recovery.CreateRestoreScripts.ValueBool() {
		tflog.Debug(ctx, "Recovery restore scripts enabled")
	}
	if !data.Recovery.ValidateBackups.IsNull() && data.Recovery.ValidateBackups.ValueBool() {
		tflog.Debug(ctx, "Backup validation enabled")
	}
}

// createAndValidateClient creates and validates the dotfiles client
func (p *DotfilesProvider) createAndValidateClient(config *DotfilesConfig, resp *provider.ConfigureResponse) (*DotfilesClient, error) {
	// Validate configuration
	if err := config.Validate(); err != nil {
		resp.Diagnostics.AddError(
			"Invalid provider configuration",
			"The provider configuration is invalid: "+err.Error(),
		)
		return nil, err
	}

	// Create the client
	client, err := NewDotfilesClient(config)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create dotfiles client",
			"An error occurred while creating the dotfiles client: "+err.Error(),
		)
		return nil, err
	}

	return client, nil
}

func (p *DotfilesProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewRepositoryResource,
		NewFileResource,
		NewSymlinkResource,
		NewDirectoryResource,
		NewApplicationResource,
	}
}

func (p *DotfilesProvider) EphemeralResources(ctx context.Context) []func() ephemeral.EphemeralResource {
	return []func() ephemeral.EphemeralResource{
		// No ephemeral resources planned for initial implementation
	}
}

func (p *DotfilesProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewSystemDataSource,
		NewFileInfoDataSource,
	}
}

func (p *DotfilesProvider) Functions(ctx context.Context) []func() function.Function {
	return []func() function.Function{
		// Functions will be added in later phases
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &DotfilesProvider{
			version: version,
		}
	}
}
