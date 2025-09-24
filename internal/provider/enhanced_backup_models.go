// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// BackupStrategyModel defines the backup strategy configuration block.
type BackupStrategyModel struct {
	Enabled         types.Bool   `tfsdk:"enabled"`
	Directory       types.String `tfsdk:"directory"`
	RetentionPolicy types.String `tfsdk:"retention_policy"`
	Compression     types.Bool   `tfsdk:"compression"`
	Incremental     types.Bool   `tfsdk:"incremental"`
	MaxBackups      types.Int64  `tfsdk:"max_backups"`
}

// RecoveryModel defines the recovery configuration block.
type RecoveryModel struct {
	CreateRestoreScripts types.Bool `tfsdk:"create_restore_scripts"`
	ValidateBackups      types.Bool `tfsdk:"validate_backups"`
	TestRecovery         types.Bool `tfsdk:"test_recovery"`
	BackupIndex          types.Bool `tfsdk:"backup_index"`
}

// BackupPolicyModel defines file-specific backup policy.
type BackupPolicyModel struct {
	AlwaysBackup    types.Bool   `tfsdk:"always_backup"`
	VersionedBackup types.Bool   `tfsdk:"versioned_backup"`
	BackupFormat    types.String `tfsdk:"backup_format"`
	RetentionCount  types.Int64  `tfsdk:"retention_count"`
	BackupMetadata  types.Bool   `tfsdk:"backup_metadata"`
	Compression     types.Bool   `tfsdk:"compression"`
}

// RecoveryTestModel defines recovery testing configuration.
type RecoveryTestModel struct {
	Enabled types.Bool   `tfsdk:"enabled"`
	Command types.String `tfsdk:"command"`
	Timeout types.String `tfsdk:"timeout"`
}

// EnhancedProviderModel extends DotfilesProviderModel with enhanced backup features.
// Since DotfilesProviderModel now includes backup_strategy and recovery fields,
// this is now just an alias for backward compatibility.
type EnhancedProviderModel = DotfilesProviderModel

// EnhancedFileResourceModelWithBackup extends EnhancedFileResourceModel with backup features.
type EnhancedFileResourceModelWithBackup struct {
	EnhancedFileResourceModel
	BackupPolicy *BackupPolicyModel `tfsdk:"backup_policy"`
	RecoveryTest *RecoveryTestModel `tfsdk:"recovery_test"`
}

// EnhancedSymlinkResourceModelWithBackup extends EnhancedSymlinkResourceModel with backup features.
type EnhancedSymlinkResourceModelWithBackup struct {
	EnhancedSymlinkResourceModel
	BackupPolicy *BackupPolicyModel `tfsdk:"backup_policy"`
	RecoveryTest *RecoveryTestModel `tfsdk:"recovery_test"`
}

// GetBackupStrategySchemaBlock returns the schema block for backup strategy.
func GetBackupStrategySchemaBlock() schema.SingleNestedBlock {
	return schema.SingleNestedBlock{
		MarkdownDescription: "Enhanced backup strategy configuration",
		Attributes: map[string]schema.Attribute{
			"enabled": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				MarkdownDescription: "Enable enhanced backup features",
			},
			"directory": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("~/.dotfiles-backups"),
				MarkdownDescription: "Directory to store backup files",
			},
			"retention_policy": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("30d"),
				MarkdownDescription: "Backup retention policy (e.g., '30d', '7d', '1y')",
			},
			"compression": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Enable backup compression (gzip)",
			},
			"incremental": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				MarkdownDescription: "Only backup when content changes",
			},
			"max_backups": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(50),
				MarkdownDescription: "Maximum number of backups to keep per file",
			},
		},
	}
}

// GetRecoverySchemaBlock returns the schema block for recovery configuration.
func GetRecoverySchemaBlock() schema.SingleNestedBlock {
	return schema.SingleNestedBlock{
		MarkdownDescription: "Recovery and validation configuration",
		Attributes: map[string]schema.Attribute{
			"create_restore_scripts": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Generate restore scripts for backups",
			},
			"validate_backups": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				MarkdownDescription: "Validate backup integrity with checksums",
			},
			"test_recovery": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Test backup recovery functionality",
			},
			"backup_index": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				MarkdownDescription: "Create searchable backup index",
			},
		},
	}
}

// GetBackupPolicySchemaBlock returns the schema block for file-specific backup policy.
func GetBackupPolicySchemaBlock() schema.SingleNestedBlock {
	return schema.SingleNestedBlock{
		MarkdownDescription: "File-specific backup policy configuration",
		Attributes: map[string]schema.Attribute{
			"always_backup": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Force backup even if globally disabled",
			},
			"versioned_backup": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Keep multiple backup versions",
			},
			"backup_format": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("timestamped"),
				MarkdownDescription: "Backup naming format: timestamped, numbered, or git_style",
			},
			"retention_count": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(5),
				MarkdownDescription: "Number of backup versions to retain",
			},
			"backup_metadata": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Store backup metadata (checksums, timestamps)",
			},
			"compression": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Compress this file's backups",
			},
		},
	}
}

// GetRecoveryTestSchemaBlock returns the schema block for recovery testing.
func GetRecoveryTestSchemaBlock() schema.SingleNestedBlock {
	return schema.SingleNestedBlock{
		MarkdownDescription: "Recovery testing configuration",
		Attributes: map[string]schema.Attribute{
			"enabled": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Enable recovery testing for this file",
			},
			"command": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Command to validate backup ({{.backup_path}} template available)",
			},
			"timeout": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("30s"),
				MarkdownDescription: "Timeout for recovery test commands",
			},
		},
	}
}
