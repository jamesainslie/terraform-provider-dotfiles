// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

package provider

import (
	"context"
	"os"
	"path/filepath"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
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
	DotfilesRoot       types.String `tfsdk:"dotfiles_root"`
	BackupEnabled      types.Bool   `tfsdk:"backup_enabled"`
	BackupDirectory    types.String `tfsdk:"backup_directory"`
	Strategy           types.String `tfsdk:"strategy"`
	ConflictResolution types.String `tfsdk:"conflict_resolution"`
	DryRun             types.Bool   `tfsdk:"dry_run"`
	AutoDetectPlatform types.Bool   `tfsdk:"auto_detect_platform"`
	TargetPlatform     types.String `tfsdk:"target_platform"`
	TemplateEngine     types.String `tfsdk:"template_engine"`
	LogLevel           types.String `tfsdk:"log_level"`
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
				MarkdownDescription: "Template engine to use: go (default), handlebars, or none",
				Optional:            true,
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
	var data DotfilesProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create and configure the client
	config := &DotfilesConfig{}

	// Set configuration values with defaults
	if !data.DotfilesRoot.IsNull() {
		config.DotfilesRoot = data.DotfilesRoot.ValueString()
	} else {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to get user home directory",
				"Could not determine user home directory: "+err.Error(),
			)
			return
		}
		config.DotfilesRoot = filepath.Join(homeDir, "dotfiles")
	}

	if !data.BackupEnabled.IsNull() {
		config.BackupEnabled = data.BackupEnabled.ValueBool()
	} else {
		config.BackupEnabled = true // default to true
	}

	if !data.BackupDirectory.IsNull() {
		config.BackupDirectory = data.BackupDirectory.ValueString()
	} else {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to get user home directory",
				"Could not determine user home directory for backup directory: "+err.Error(),
			)
			return
		}
		config.BackupDirectory = filepath.Join(homeDir, ".dotfiles-backups")
	}

	if !data.Strategy.IsNull() {
		config.Strategy = data.Strategy.ValueString()
	} else {
		config.Strategy = "symlink"
	}

	if !data.ConflictResolution.IsNull() {
		config.ConflictResolution = data.ConflictResolution.ValueString()
	} else {
		config.ConflictResolution = "backup"
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
	} else {
		config.TargetPlatform = "auto"
	}

	if !data.TemplateEngine.IsNull() {
		config.TemplateEngine = data.TemplateEngine.ValueString()
	} else {
		config.TemplateEngine = "go"
	}

	if !data.LogLevel.IsNull() {
		config.LogLevel = data.LogLevel.ValueString()
	} else {
		config.LogLevel = "info"
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		resp.Diagnostics.AddError(
			"Invalid provider configuration",
			"The provider configuration is invalid: "+err.Error(),
		)
		return
	}

	// Create the client
	client, err := NewDotfilesClient(config)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create dotfiles client",
			"An error occurred while creating the dotfiles client: "+err.Error(),
		)
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
