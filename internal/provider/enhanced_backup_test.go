// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestEnhancedBackupProviderSchema tests the enhanced backup configuration schema.
func TestEnhancedBackupProviderSchema(t *testing.T) {
	t.Run("Provider schema should support backup_strategy block", func(t *testing.T) {
		dotfilesProvider := &DotfilesProvider{}
		ctx := context.Background()

		req := provider.SchemaRequest{}
		resp := &provider.SchemaResponse{}

		dotfilesProvider.Schema(ctx, req, resp)

		if resp.Diagnostics.HasError() {
			t.Fatalf("Provider schema validation failed: %v", resp.Diagnostics)
		}

		// Check for backup_strategy block
		if _, exists := resp.Schema.Blocks["backup_strategy"]; !exists {
			t.Error("backup_strategy block should be defined in provider schema")
		}

		// Check for recovery block
		if _, exists := resp.Schema.Blocks["recovery"]; !exists {
			t.Error("recovery block should be defined in provider schema")
		}
	})
}

// TestEnhancedFileResourceBackupSchema tests file resource backup policy schema.
func TestEnhancedFileResourceBackupSchema(t *testing.T) {
	t.Run("FileResource should support backup_policy block", func(t *testing.T) {
		r := NewFileResource()
		ctx := context.Background()

		req := resource.SchemaRequest{}
		resp := &resource.SchemaResponse{}

		r.Schema(ctx, req, resp)

		if resp.Diagnostics.HasError() {
			t.Fatalf("Schema validation failed: %v", resp.Diagnostics)
		}

		// Check for backup_policy block
		if _, exists := resp.Schema.Blocks["backup_policy"]; !exists {
			t.Error("backup_policy block should be defined in file resource schema")
		}

		// Check for recovery_test block
		if _, exists := resp.Schema.Blocks["recovery_test"]; !exists {
			t.Error("recovery_test block should be defined in file resource schema")
		}
	})
}

// TestBackupStrategyModel tests the backup strategy configuration model.
func TestBackupStrategyModel(t *testing.T) {
	// Test enhanced provider model with backup strategy
	model := &EnhancedProviderModel{
		DotfilesRoot:    types.StringValue("~/dotfiles"),
		BackupEnabled:   types.BoolValue(true),
		BackupDirectory: types.StringValue("~/.dotfiles-backups"),
		BackupStrategy: &BackupStrategyModel{
			Enabled:         types.BoolValue(true),
			Directory:       types.StringValue("~/.dotfiles-backups"),
			RetentionPolicy: types.StringValue("30d"),
			Compression:     types.BoolValue(true),
			Incremental:     types.BoolValue(true),
			MaxBackups:      types.Int64Value(50),
		},
		Recovery: &RecoveryModel{
			CreateRestoreScripts: types.BoolValue(true),
			ValidateBackups:      types.BoolValue(true),
			TestRecovery:         types.BoolValue(false),
			BackupIndex:          types.BoolValue(true),
		},
	}

	// Verify backup strategy model
	if !model.BackupStrategy.Enabled.ValueBool() {
		t.Error("BackupStrategy enabled not set correctly")
	}
	if model.BackupStrategy.RetentionPolicy.ValueString() != "30d" {
		t.Error("BackupStrategy retention policy not set correctly")
	}
	if !model.BackupStrategy.Compression.ValueBool() {
		t.Error("BackupStrategy compression not set correctly")
	}
	if model.BackupStrategy.MaxBackups.ValueInt64() != 50 {
		t.Error("BackupStrategy max backups not set correctly")
	}

	// Verify recovery model
	if !model.Recovery.CreateRestoreScripts.ValueBool() {
		t.Error("Recovery create restore scripts not set correctly")
	}
	if !model.Recovery.BackupIndex.ValueBool() {
		t.Error("Recovery backup index not set correctly")
	}
}

// TestBackupPolicyModel tests the file-specific backup policy model.
func TestBackupPolicyModel(t *testing.T) {
	// Test enhanced file model with backup policy
	model := &EnhancedFileResourceModelWithBackup{
		EnhancedFileResourceModel: EnhancedFileResourceModel{
			FileResourceModel: FileResourceModel{
				ID:         types.StringValue("test-file"),
				Repository: types.StringValue("test-repo"),
				Name:       types.StringValue("critical-config"),
				SourcePath: types.StringValue("config/important.conf"),
				TargetPath: types.StringValue("~/.config/important.conf"),
			},
		},
		BackupPolicy: &BackupPolicyModel{
			AlwaysBackup:    types.BoolValue(true),
			VersionedBackup: types.BoolValue(true),
			BackupFormat:    types.StringValue("timestamped"),
			RetentionCount:  types.Int64Value(10),
			BackupMetadata:  types.BoolValue(true),
			Compression:     types.BoolValue(true),
		},
		RecoveryTest: &RecoveryTestModel{
			Enabled: types.BoolValue(true),
			Command: types.StringValue("validate-config {{.backup_path}}"),
			Timeout: types.StringValue("30s"),
		},
	}

	// Verify backup policy model
	if !model.BackupPolicy.AlwaysBackup.ValueBool() {
		t.Error("BackupPolicy always backup not set correctly")
	}
	if model.BackupPolicy.BackupFormat.ValueString() != "timestamped" {
		t.Error("BackupPolicy backup format not set correctly")
	}
	if model.BackupPolicy.RetentionCount.ValueInt64() != 10 {
		t.Error("BackupPolicy retention count not set correctly")
	}

	// Verify recovery test model
	if !model.RecoveryTest.Enabled.ValueBool() {
		t.Error("RecoveryTest enabled not set correctly")
	}
	if model.RecoveryTest.Command.ValueString() != "validate-config {{.backup_path}}" {
		t.Error("RecoveryTest command not set correctly")
	}
}
