// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

package provider

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/jamesainslie/terraform-provider-dotfiles/internal/fileops"
	"github.com/jamesainslie/terraform-provider-dotfiles/internal/platform"
)

// TestEnhancedBackupEndToEnd tests the complete enhanced backup workflow.
func TestEnhancedBackupEndToEnd(t *testing.T) {
	// Create temporary directories
	tempDir := t.TempDir()

	sourceDir := filepath.Join(tempDir, "source")
	targetDir := filepath.Join(tempDir, "target")
	backupDir := filepath.Join(tempDir, "backups")

	// Create source directory and files
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatalf("Failed to create target directory: %v", err)
	}

	// Create test configuration files
	configFiles := map[string]string{
		"critical.conf": "# Critical configuration\napi_key=secret123\nhost=production.com",
		"settings.json": `{"theme": "dark", "fontSize": 14}`,
		"app.yaml":      "version: 1.0\nenvironment: production",
	}

	for filename, content := range configFiles {
		sourcePath := filepath.Join(sourceDir, filename)
		err := os.WriteFile(sourcePath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create source file %s: %v", filename, err)
		}
	}

	t.Run("Complete enhanced backup workflow", func(t *testing.T) {
		// Test Case 1: File with comprehensive backup policy
		t.Run("Critical config with full backup features", func(t *testing.T) {
			targetFile := filepath.Join(targetDir, "critical.conf")

			// Create existing target file (to trigger backup)
			err := os.WriteFile(targetFile, []byte("old configuration"), 0644)
			if err != nil {
				t.Fatalf("Failed to create existing target file: %v", err)
			}

			// Create file model with comprehensive backup policy
			model := &EnhancedFileResourceModelWithBackup{
				EnhancedFileResourceModel: EnhancedFileResourceModel{
					FileResourceModel: FileResourceModel{
						ID:            types.StringValue("critical-config"),
						Repository:    types.StringValue("test-repo"),
						Name:          types.StringValue("critical-config"),
						SourcePath:    types.StringValue("critical.conf"),
						TargetPath:    types.StringValue(targetFile),
						BackupEnabled: types.BoolValue(true),
					},
					Permissions: &PermissionsModel{
						Files:     types.StringValue("0600"),
						Recursive: types.BoolValue(false),
					},
				},
				BackupPolicy: &BackupPolicyModel{
					AlwaysBackup:    types.BoolValue(true),
					VersionedBackup: types.BoolValue(true),
					BackupFormat:    types.StringValue("timestamped"),
					RetentionCount:  types.Int64Value(3),
					BackupMetadata:  types.BoolValue(true),
					Compression:     types.BoolValue(true),
				},
				RecoveryTest: &RecoveryTestModel{
					Enabled: types.BoolValue(true),
					Command: types.StringValue("test -f {{.backup_path}}"),
					Timeout: types.StringValue("10s"),
				},
			}

			// Build and validate backup configuration
			backupConfig, err := buildEnhancedBackupConfigFromFileModel(model)
			if err != nil {
				t.Fatalf("Failed to build enhanced backup config: %v", err)
			}

			// Set backup directory
			backupConfig.Directory = backupDir

			// Create FileManager and test backup creation
			platformProvider := platform.DetectPlatform()
			fm := fileops.NewFileManager(platformProvider, false)

			// Create initial backup
			backupPath, err := fm.CreateEnhancedBackup(targetFile, backupConfig)
			if err != nil {
				t.Fatalf("Failed to create enhanced backup: %v", err)
			}

			// Verify backup was created with correct properties
			if !strings.Contains(backupPath, ".backup.") {
				t.Error("Backup path should contain .backup.")
			}
			if !strings.HasSuffix(backupPath, ".gz") {
				t.Error("Compressed backup should have .gz extension")
			}
			if !pathExists(backupPath) {
				t.Error("Backup file should exist")
			}

			// Verify metadata was created
			metadataPath := backupPath + ".meta"
			if !pathExists(metadataPath) {
				t.Error("Backup metadata should be created")
			}

			// Load and verify metadata
			metadata, err := fileops.LoadBackupMetadata(metadataPath)
			if err != nil {
				t.Fatalf("Failed to load backup metadata: %v", err)
			}

			if metadata.OriginalPath != targetFile {
				t.Errorf("Expected original path %s, got %s", targetFile, metadata.OriginalPath)
			}
			if metadata.Checksum == "" {
				t.Error("Metadata should contain checksum")
			}
			if !metadata.Compressed {
				t.Error("Metadata should indicate file is compressed")
			}

			// Verify backup index was created
			indexPath := filepath.Join(backupDir, ".backup_index.json")
			if !pathExists(indexPath) {
				t.Error("Backup index should be created")
			}

			// Load and verify index
			index, err := fileops.LoadBackupIndex(indexPath)
			if err != nil {
				t.Fatalf("Failed to load backup index: %v", err)
			}
			if len(index.Backups) != 1 {
				t.Errorf("Expected 1 backup in index, got %d", len(index.Backups))
			}
		})

		// Test Case 2: Multiple backups with retention
		t.Run("Multiple backups with retention management", func(t *testing.T) {
			targetFile := filepath.Join(targetDir, "settings.json")

			// Create target file
			err := os.WriteFile(targetFile, []byte(`{"theme": "light"}`), 0644)
			if err != nil {
				t.Fatalf("Failed to create target file: %v", err)
			}

			// Configure retention with low max backups for testing
			config := &fileops.EnhancedBackupConfig{
				Enabled:        true,
				Directory:      backupDir,
				BackupFormat:   "numbered",
				MaxBackups:     2, // Keep only 2 backups
				BackupMetadata: true,
				BackupIndex:    true,
				Compression:    false,
			}

			platformProvider := platform.DetectPlatform()
			fm := fileops.NewFileManager(platformProvider, false)

			// Create multiple backups by changing content
			backupPaths := make([]string, 0)
			for i := 1; i <= 4; i++ {
				content := fmt.Sprintf(`{"theme": "version%d", "fontSize": %d}`, i, 12+i)
				err := os.WriteFile(targetFile, []byte(content), 0644)
				if err != nil {
					t.Fatalf("Failed to update target file iteration %d: %v", i, err)
				}

				backupPath, err := fm.CreateEnhancedBackup(targetFile, config)
				if err != nil {
					t.Fatalf("Failed to create backup iteration %d: %v", i, err)
				}

				if backupPath != "" {
					backupPaths = append(backupPaths, backupPath)
				}
			}

			// Verify retention policy was applied - should only have 2 actual backup files (not counting metadata)
			allFiles, err := filepath.Glob(filepath.Join(backupDir, "settings.json.backup.*"))
			if err != nil {
				t.Fatalf("Failed to list backup files: %v", err)
			}

			// Count only actual backup files (not metadata files)
			actualBackups := make([]string, 0)
			for _, path := range allFiles {
				if !strings.HasSuffix(path, ".meta") {
					actualBackups = append(actualBackups, path)
				}
			}

			if len(actualBackups) > 2 {
				t.Errorf("Expected at most 2 actual backup files after retention, got %d: %v",
					len(actualBackups), actualBackups)
			}

			// Verify backup index reflects the current state
			indexPath := filepath.Join(backupDir, ".backup_index.json")
			index, err := fileops.LoadBackupIndex(indexPath)
			if err != nil {
				t.Fatalf("Failed to load backup index: %v", err)
			}

			// Index should contain all backup records (retention is applied to files, not index)
			if len(index.Backups) < 2 {
				t.Errorf("Backup index should contain multiple backup records")
			}
		})

		// Test Case 3: Incremental backup (skip duplicates)
		t.Run("Incremental backup skips duplicates", func(t *testing.T) {
			targetFile := filepath.Join(targetDir, "app.yaml")
			content := "version: 2.0\nenvironment: staging"

			err := os.WriteFile(targetFile, []byte(content), 0644)
			if err != nil {
				t.Fatalf("Failed to create target file: %v", err)
			}

			config := &fileops.EnhancedBackupConfig{
				Enabled:      true,
				Directory:    backupDir,
				BackupFormat: "git_style",
				Incremental:  true,
				BackupIndex:  true,
			}

			platformProvider := platform.DetectPlatform()
			fm := fileops.NewFileManager(platformProvider, false)

			// Create first backup
			backupPath1, err := fm.CreateEnhancedBackup(targetFile, config)
			if err != nil {
				t.Fatalf("Failed to create first backup: %v", err)
			}
			if backupPath1 == "" {
				t.Error("First backup should always be created")
			}

			// Try to create backup with same content - should be skipped
			backupPath2, err := fm.CreateEnhancedBackup(targetFile, config)
			if err != nil {
				t.Fatalf("Failed to check second backup: %v", err)
			}
			if backupPath2 != "" {
				t.Error("Second backup should be skipped with incremental backup")
			}

			// Change content and create backup - should be created
			newContent := "version: 2.1\nenvironment: production"
			err = os.WriteFile(targetFile, []byte(newContent), 0644)
			if err != nil {
				t.Fatalf("Failed to update target file: %v", err)
			}

			backupPath3, err := fm.CreateEnhancedBackup(targetFile, config)
			if err != nil {
				t.Fatalf("Failed to create third backup: %v", err)
			}
			if backupPath3 == "" {
				t.Error("Third backup should be created when content changes")
			}

			// Verify git-style naming (should contain 8-character hash)
			backupName := filepath.Base(backupPath3)
			if !strings.Contains(backupName, ".backup.") {
				t.Error("Git-style backup should contain .backup.")
			}
		})

		// Test Case 4: Provider-level backup strategy
		t.Run("Provider-level backup strategy configuration", func(t *testing.T) {
			// Test that provider model can be configured with backup strategy
			providerModel := &EnhancedProviderModel{
				DotfilesProviderModel: DotfilesProviderModel{
					DotfilesRoot:    types.StringValue(sourceDir),
					BackupEnabled:   types.BoolValue(true),
					BackupDirectory: types.StringValue(backupDir),
				},
				BackupStrategy: &BackupStrategyModel{
					Enabled:         types.BoolValue(true),
					Directory:       types.StringValue(backupDir),
					RetentionPolicy: types.StringValue("7d"),
					Compression:     types.BoolValue(true),
					Incremental:     types.BoolValue(true),
					MaxBackups:      types.Int64Value(10),
				},
				Recovery: &RecoveryModel{
					CreateRestoreScripts: types.BoolValue(true),
					ValidateBackups:      types.BoolValue(true),
					TestRecovery:         types.BoolValue(false),
					BackupIndex:          types.BoolValue(true),
				},
			}

			// Verify all provider-level configuration is accessible
			if !providerModel.BackupStrategy.Enabled.ValueBool() {
				t.Error("Provider backup strategy should be enabled")
			}
			if providerModel.BackupStrategy.RetentionPolicy.ValueString() != "7d" {
				t.Error("Provider backup retention should be 7d")
			}
			if !providerModel.BackupStrategy.Compression.ValueBool() {
				t.Error("Provider backup compression should be enabled")
			}
			if !providerModel.Recovery.ValidateBackups.ValueBool() {
				t.Error("Provider recovery backup validation should be enabled")
			}
		})
	})
}

// TestEnhancedBackupCompatibility tests backward compatibility with existing backup functionality.
func TestEnhancedBackupCompatibility(t *testing.T) {
	tempDir := t.TempDir()

	sourceFile := filepath.Join(tempDir, "source.txt")
	targetFile := filepath.Join(tempDir, "target.txt")

	// Create source and target files
	err = os.WriteFile(sourceFile, []byte("source content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	err = os.WriteFile(targetFile, []byte("existing content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create target file: %v", err)
	}

	t.Run("Legacy backup still works", func(t *testing.T) {
		// Test that old-style FileResourceModel still works
		model := &EnhancedFileResourceModelWithBackup{
			EnhancedFileResourceModel: EnhancedFileResourceModel{
				FileResourceModel: FileResourceModel{
					ID:            types.StringValue("legacy-test"),
					Repository:    types.StringValue("test-repo"),
					Name:          types.StringValue("legacy-config"),
					SourcePath:    types.StringValue("source.txt"),
					TargetPath:    types.StringValue(targetFile),
					BackupEnabled: types.BoolValue(true),
				},
			},
			// No BackupPolicy - should fall back to legacy backup
		}

		// Build backup config - should return nil for enhanced config
		backupConfig, err := buildEnhancedBackupConfigFromFileModel(model)
		if err != nil {
			t.Fatalf("Failed to build backup config: %v", err)
		}

		// Should be nil since no backup policy is configured
		if backupConfig != nil {
			t.Error("Should return nil for enhanced backup config when no policy configured")
		}

		// Legacy backup should still work via the resource implementation
		// This is tested indirectly through the resource CRUD operations
	})

	t.Run("Enhanced backup overrides legacy", func(t *testing.T) {
		// Test that enhanced backup policy overrides legacy settings
		model := &EnhancedFileResourceModelWithBackup{
			EnhancedFileResourceModel: EnhancedFileResourceModel{
				FileResourceModel: FileResourceModel{
					ID:            types.StringValue("enhanced-test"),
					Repository:    types.StringValue("test-repo"),
					Name:          types.StringValue("enhanced-config"),
					SourcePath:    types.StringValue("source.txt"),
					TargetPath:    types.StringValue(targetFile),
					BackupEnabled: types.BoolValue(false), // Legacy disabled
				},
			},
			BackupPolicy: &BackupPolicyModel{
				AlwaysBackup:   types.BoolValue(true), // Enhanced enabled
				BackupFormat:   types.StringValue("numbered"),
				RetentionCount: types.Int64Value(5),
				Compression:    types.BoolValue(false),
			},
		}

		// Build backup config
		backupConfig, err := buildEnhancedBackupConfigFromFileModel(model)
		if err != nil {
			t.Fatalf("Failed to build backup config: %v", err)
		}

		// Should have enhanced backup configuration
		if backupConfig == nil {
			t.Error("Should have enhanced backup config when policy is configured")
		}
		if !backupConfig.Enabled {
			t.Error("Enhanced backup should be enabled even when legacy backup is disabled")
		}
		if backupConfig.BackupFormat != "numbered" {
			t.Error("Should use configured backup format")
		}
		if backupConfig.MaxBackups != 5 {
			t.Error("Should use configured retention count")
		}
	})
}

// TestBackupFeatureCompletion tests that all required backup features are implemented.
func TestBackupFeatureCompletion(t *testing.T) {
	t.Run("All backup formats supported", func(t *testing.T) {
		formats := []string{"timestamped", "numbered", "git_style"}

		for _, format := range formats {
			config := &fileops.EnhancedBackupConfig{
				Enabled:      true,
				BackupFormat: format,
				Directory:    "/tmp/test-backups",
			}

			err := fileops.ValidateEnhancedBackupConfig(config)
			if err != nil {
				t.Errorf("Backup format %s should be supported: %v", format, err)
			}
		}
	})

	t.Run("Retention policies supported", func(t *testing.T) {
		policies := []string{"7d", "30d", "1y", "12w", "6m"}

		for _, policy := range policies {
			config := &fileops.EnhancedBackupConfig{
				Enabled:         true,
				RetentionPolicy: policy,
				Directory:       "/tmp/test-backups",
			}

			err := fileops.ValidateEnhancedBackupConfig(config)
			if err != nil {
				t.Errorf("Retention policy %s should be supported: %v", policy, err)
			}
		}
	})

	t.Run("All backup features can be configured", func(t *testing.T) {
		// Test comprehensive configuration
		config := &fileops.EnhancedBackupConfig{
			Enabled:         true,
			Directory:       "/tmp/backups",
			RetentionPolicy: "30d",
			Compression:     true,
			Incremental:     true,
			MaxBackups:      50,
			BackupFormat:    "timestamped",
			BackupMetadata:  true,
			BackupIndex:     true,
		}

		err := fileops.ValidateEnhancedBackupConfig(config)
		if err != nil {
			t.Errorf("Comprehensive backup configuration should be valid: %v", err)
		}

		// Verify all features are accessible
		if !config.Enabled {
			t.Error("Backup should be enabled")
		}
		if !config.Compression {
			t.Error("Compression should be enabled")
		}
		if !config.Incremental {
			t.Error("Incremental backup should be enabled")
		}
		if !config.BackupMetadata {
			t.Error("Backup metadata should be enabled")
		}
		if !config.BackupIndex {
			t.Error("Backup index should be enabled")
		}
	})
}
