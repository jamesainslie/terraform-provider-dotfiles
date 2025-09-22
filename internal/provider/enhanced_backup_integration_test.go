// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

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

// TestEnhancedBackupIntegration tests enhanced backup with file operations
func TestEnhancedBackupIntegration(t *testing.T) {
	// Create temporary directories for testing
	tempDir, err := os.MkdirTemp("", "enhanced-backup-integration-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create source and target directories
	sourceDir := filepath.Join(tempDir, "source")
	targetDir := filepath.Join(tempDir, "target")
	backupDir := filepath.Join(tempDir, "backups")

	err = os.MkdirAll(sourceDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}

	err = os.MkdirAll(targetDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create target directory: %v", err)
	}

	// Create test source file
	sourceFile := filepath.Join(sourceDir, "config.txt")
	initialContent := "initial configuration content"
	err = os.WriteFile(sourceFile, []byte(initialContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	t.Run("Enhanced backup with provider-level configuration", func(t *testing.T) {
		// Create enhanced provider model
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
				MaxBackups:      types.Int64Value(5),
			},
			Recovery: &RecoveryModel{
				CreateRestoreScripts: types.BoolValue(true),
				ValidateBackups:      types.BoolValue(true),
				BackupIndex:          types.BoolValue(true),
			},
		}

		// Verify provider model configuration
		if !providerModel.BackupStrategy.Enabled.ValueBool() {
			t.Error("Backup strategy should be enabled")
		}
		if providerModel.BackupStrategy.RetentionPolicy.ValueString() != "7d" {
			t.Error("Retention policy should be 7d")
		}
		if !providerModel.BackupStrategy.Compression.ValueBool() {
			t.Error("Compression should be enabled")
		}
		if providerModel.BackupStrategy.MaxBackups.ValueInt64() != 5 {
			t.Error("Max backups should be 5")
		}
	})

	t.Run("Enhanced backup with file-specific policy", func(t *testing.T) {
		// Create enhanced file model with backup policy
		fileModel := &EnhancedFileResourceModelWithBackup{
			EnhancedFileResourceModel: EnhancedFileResourceModel{
				FileResourceModel: FileResourceModel{
					ID:         types.StringValue("test-config"),
					Repository: types.StringValue("test-repo"),
					Name:       types.StringValue("config-file"),
					SourcePath: types.StringValue("config.txt"),
					TargetPath: types.StringValue(filepath.Join(targetDir, "config.txt")),
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
				Command: types.StringValue("test -r {{.backup_path}}"),
				Timeout: types.StringValue("10s"),
			},
		}

		// Build enhanced backup configuration from model
		backupConfig, err := buildEnhancedBackupConfig(fileModel.BackupPolicy)
		if err != nil {
			t.Fatalf("Failed to build backup config: %v", err)
		}

		// Verify backup configuration
		if !backupConfig.Enabled {
			t.Error("Backup should be enabled")
		}
		if backupConfig.BackupFormat != "timestamped" {
			t.Error("Backup format should be timestamped")
		}
		if backupConfig.MaxBackups != 3 {
			t.Error("Max backups should be 3")
		}
		if !backupConfig.Compression {
			t.Error("Compression should be enabled")
		}
		if !backupConfig.BackupMetadata {
			t.Error("Backup metadata should be enabled")
		}
	})

	t.Run("File operations with enhanced backup", func(t *testing.T) {
		targetFile := filepath.Join(targetDir, "enhanced-config.txt")

		// Create initial target file
		err = os.WriteFile(targetFile, []byte("original target content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create target file: %v", err)
		}

		// Create file manager
		platformProvider := platform.DetectPlatform()
		fm := fileops.NewFileManager(platformProvider, false)

		// Configure enhanced backup
		config := &fileops.EnhancedBackupConfig{
			Enabled:        true,
			Directory:      backupDir,
			BackupFormat:   "numbered",
			Compression:    true,
			MaxBackups:     3,
			BackupMetadata: true,
			BackupIndex:    true,
			Incremental:    true,
		}

		// Create first enhanced backup
		backupPath1, err := fm.CreateEnhancedBackup(targetFile, config)
		if err != nil {
			t.Fatalf("First enhanced backup failed: %v", err)
		}

		// Verify backup was created
		if backupPath1 == "" {
			t.Error("First backup path should not be empty")
		}
		if !strings.HasSuffix(backupPath1, ".gz") {
			t.Error("Compressed backup should have .gz extension")
		}
		if !pathExists(backupPath1) {
			t.Error("Backup file should exist")
		}

		// Verify metadata was created
		metadataPath := backupPath1 + ".meta"
		if !pathExists(metadataPath) {
			t.Error("Backup metadata file should exist")
		}

		// Verify backup index was created
		indexPath := filepath.Join(backupDir, ".backup_index.json")
		if !pathExists(indexPath) {
			t.Error("Backup index should be created")
		}

		// Update target file content
		newContent := "updated target content"
		err = os.WriteFile(targetFile, []byte(newContent), 0644)
		if err != nil {
			t.Fatalf("Failed to update target file: %v", err)
		}

		// Create second enhanced backup
		backupPath2, err := fm.CreateEnhancedBackup(targetFile, config)
		if err != nil {
			t.Fatalf("Second enhanced backup failed: %v", err)
		}

		// Should create new backup since content changed
		if backupPath2 == "" {
			t.Error("Second backup should be created when content changes")
		}
		if backupPath1 == backupPath2 {
			t.Errorf("Second backup should have different path: path1=%s, path2=%s", backupPath1, backupPath2)
		}

		// Try to create backup with same content (should be skipped with incremental)
		backupPath3, err := fm.CreateEnhancedBackup(targetFile, config)
		if err != nil {
			t.Fatalf("Third enhanced backup check failed: %v", err)
		}

		// Should skip backup since content hasn't changed
		if backupPath3 != "" {
			t.Error("Third backup should be skipped with incremental backup")
		}

		// Verify backup index contains both backups
		index, err := fileops.LoadBackupIndex(indexPath)
		if err != nil {
			t.Fatalf("Failed to load backup index: %v", err)
		}
		if len(index.Backups) < 2 {
			t.Errorf("Expected at least 2 backups in index, got %d", len(index.Backups))
		}

		// Test retention - create more backups than max allowed
		for i := 0; i < 5; i++ {
			content := fmt.Sprintf("content version %d", i+3)
			err = os.WriteFile(targetFile, []byte(content), 0644)
			if err != nil {
				t.Fatalf("Failed to update file for retention test: %v", err)
			}

			_, err = fm.CreateEnhancedBackup(targetFile, config)
			if err != nil {
				t.Fatalf("Backup creation failed during retention test: %v", err)
			}
		}

		// Check that retention policy was applied
		backupFiles, err := filepath.Glob(filepath.Join(backupDir, "enhanced-config.txt.backup.*.gz"))
		if err != nil {
			t.Fatalf("Failed to list backup files: %v", err)
		}

		if len(backupFiles) > int(config.MaxBackups) {
			t.Errorf("Expected at most %d backup files after retention, got %d",
				config.MaxBackups, len(backupFiles))
		}
	})
}

// buildEnhancedBackupConfig builds fileops backup config from provider model
func buildEnhancedBackupConfig(policy *BackupPolicyModel) (*fileops.EnhancedBackupConfig, error) {
	if policy == nil {
		return nil, nil
	}

	config := &fileops.EnhancedBackupConfig{
		Enabled:        policy.AlwaysBackup.ValueBool(),
		BackupFormat:   policy.BackupFormat.ValueString(),
		MaxBackups:     policy.RetentionCount.ValueInt64(),
		BackupMetadata: policy.BackupMetadata.ValueBool(),
		Compression:    policy.Compression.ValueBool(),
		Incremental:    policy.VersionedBackup.ValueBool(),
		BackupIndex:    true, // Always enable for file-level policies
	}

	return config, fileops.ValidateEnhancedBackupConfig(config)
}

// pathExists checks if a path exists
func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// TestRecoveryAndValidation tests backup recovery and validation features
func TestRecoveryAndValidation(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "recovery-validation-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test file
	testFile := filepath.Join(tempDir, "test.conf")
	testContent := "test configuration for recovery"
	err = os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	backupDir := filepath.Join(tempDir, "backups")
	platformProvider := platform.DetectPlatform()
	fm := fileops.NewFileManager(platformProvider, false)

	t.Run("Recovery test model validation", func(t *testing.T) {
		model := &RecoveryTestModel{
			Enabled: types.BoolValue(true),
			Command: types.StringValue("test -f {{.backup_path}}"),
			Timeout: types.StringValue("30s"),
		}

		if !model.Enabled.ValueBool() {
			t.Error("Recovery test should be enabled")
		}
		if !strings.Contains(model.Command.ValueString(), "{{.backup_path}}") {
			t.Error("Recovery test command should contain backup_path template")
		}
		if model.Timeout.ValueString() != "30s" {
			t.Error("Recovery test timeout should be 30s")
		}
	})

	t.Run("Backup with recovery validation", func(t *testing.T) {
		config := &fileops.EnhancedBackupConfig{
			Enabled:        true,
			Directory:      backupDir,
			BackupFormat:   "timestamped",
			BackupMetadata: true,
			BackupIndex:    true,
		}

		// Create backup
		backupPath, err := fm.CreateEnhancedBackup(testFile, config)
		if err != nil {
			t.Fatalf("Failed to create backup: %v", err)
		}

		// Load and verify metadata
		metadataPath := backupPath + ".meta"
		metadata, err := fileops.LoadBackupMetadata(metadataPath)
		if err != nil {
			t.Fatalf("Failed to load backup metadata: %v", err)
		}

		// Verify metadata content
		if metadata.OriginalPath != testFile {
			t.Errorf("Expected original path %s, got %s", testFile, metadata.OriginalPath)
		}
		if metadata.BackupPath != backupPath {
			t.Errorf("Expected backup path %s, got %s", backupPath, metadata.BackupPath)
		}
		if metadata.Checksum == "" {
			t.Error("Backup metadata should include checksum")
		}
		if metadata.OriginalSize == 0 {
			t.Error("Backup metadata should include original size")
		}

		// Test recovery validation by checking if backup is readable
		backupContent, err := os.ReadFile(backupPath)
		if err != nil {
			t.Fatalf("Failed to read backup file: %v", err)
		}
		if string(backupContent) != testContent {
			t.Error("Backup content should match original")
		}
	})
}

// TestBackupFormatSupport tests different backup format implementations
func TestBackupFormatSupport(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "backup-format-support-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testFile := filepath.Join(tempDir, "format-test.txt")
	err = os.WriteFile(testFile, []byte("format test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	backupDir := filepath.Join(tempDir, "backups")
	platformProvider := platform.DetectPlatform()
	fm := fileops.NewFileManager(platformProvider, false)

	t.Run("Timestamped format support", func(t *testing.T) {
		policy := &BackupPolicyModel{
			BackupFormat:   types.StringValue("timestamped"),
			RetentionCount: types.Int64Value(5),
			Compression:    types.BoolValue(false),
		}

		config, err := buildEnhancedBackupConfig(policy)
		if err != nil {
			t.Fatalf("Failed to build config: %v", err)
		}
		config.Directory = backupDir
		config.Enabled = true

		backupPath, err := fm.CreateEnhancedBackup(testFile, config)
		if err != nil {
			t.Fatalf("Timestamped backup failed: %v", err)
		}

		if !strings.Contains(backupPath, ".backup.") {
			t.Error("Timestamped backup should contain .backup. in name")
		}
	})

	t.Run("Numbered format support", func(t *testing.T) {
		policy := &BackupPolicyModel{
			BackupFormat:   types.StringValue("numbered"),
			RetentionCount: types.Int64Value(5),
			Compression:    types.BoolValue(false),
		}

		config, err := buildEnhancedBackupConfig(policy)
		if err != nil {
			t.Fatalf("Failed to build config: %v", err)
		}
		config.Directory = backupDir
		config.Enabled = true

		backupPath, err := fm.CreateEnhancedBackup(testFile, config)
		if err != nil {
			t.Fatalf("Numbered backup failed: %v", err)
		}

		if !strings.Contains(backupPath, ".backup.001") {
			t.Error("First numbered backup should end with .backup.001")
		}
	})

	t.Run("Git-style format support", func(t *testing.T) {
		policy := &BackupPolicyModel{
			BackupFormat:   types.StringValue("git_style"),
			RetentionCount: types.Int64Value(5),
			Compression:    types.BoolValue(false),
		}

		config, err := buildEnhancedBackupConfig(policy)
		if err != nil {
			t.Fatalf("Failed to build config: %v", err)
		}
		config.Directory = backupDir
		config.Enabled = true

		backupPath, err := fm.CreateEnhancedBackup(testFile, config)
		if err != nil {
			t.Fatalf("Git-style backup failed: %v", err)
		}

		backupName := filepath.Base(backupPath)
		if !strings.Contains(backupName, ".backup.") {
			t.Error("Git-style backup should contain .backup. in name")
		}

		// Should contain 8-character hash
		parts := strings.Split(backupName, ".")
		hashPart := parts[len(parts)-1]
		if len(hashPart) != 8 {
			t.Errorf("Git-style backup should end with 8-character hash, got %s", hashPart)
		}
	})
}
