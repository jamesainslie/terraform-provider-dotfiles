// Copyright (c) HashCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

package provider

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/jamesainslie/terraform-provider-dotfiles/internal/fileops"
	"github.com/jamesainslie/terraform-provider-dotfiles/internal/platform"
)

// TestEnhancedBackupIntegration tests enhanced backup with file operations.
func TestEnhancedBackupIntegration(t *testing.T) {
	// Setup test environment
	testEnv := setupEnhancedBackupIntegrationEnvironment(t)

	t.Run("Enhanced backup with provider-level configuration", func(t *testing.T) {
		testEnhancedBackupProviderLevelConfig(t, testEnv)
	})

	t.Run("Enhanced backup with file-specific policy", func(t *testing.T) {
		testEnhancedBackupFileSpecificPolicy(t, testEnv)
	})

	t.Run("File operations with enhanced backup", func(t *testing.T) {
		testFileOperationsWithEnhancedBackup(t, testEnv)
	})

	t.Run("Recovery test model validation", func(t *testing.T) {
		testRecoveryModelValidation(t, testEnv)
	})

	t.Run("Backup with recovery validation", func(t *testing.T) {
		testBackupWithRecoveryValidation(t, testEnv)
	})

	t.Run("Timestamped format support", func(t *testing.T) {
		testTimestampedFormatSupport(t, testEnv)
	})

	t.Run("Numbered format support", func(t *testing.T) {
		testNumberedFormatSupport(t, testEnv)
	})

	t.Run("Git-style format support", func(t *testing.T) {
		testGitStyleFormatSupport(t, testEnv)
	})
}

// enhancedBackupIntegrationTestEnv holds the test environment setup
type enhancedBackupIntegrationTestEnv struct {
	tempDir        string
	sourceDir      string
	targetDir      string
	backupDir      string
	sourceFile     string
	initialContent string
}

// setupEnhancedBackupIntegrationEnvironment creates and initializes the test environment
func setupEnhancedBackupIntegrationEnvironment(t *testing.T) *enhancedBackupIntegrationTestEnv {
	tempDir := t.TempDir()

	env := &enhancedBackupIntegrationTestEnv{
		tempDir:        tempDir,
		sourceDir:      filepath.Join(tempDir, "source"),
		targetDir:      filepath.Join(tempDir, "target"),
		backupDir:      filepath.Join(tempDir, "backups"),
		sourceFile:     filepath.Join(tempDir, "source", "config.txt"),
		initialContent: "initial configuration content",
	}

	// Create directories
	if err := os.MkdirAll(env.sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}
	if err := os.MkdirAll(env.targetDir, 0755); err != nil {
		t.Fatalf("Failed to create target directory: %v", err)
	}

	// Create test source file
	if err := os.WriteFile(env.sourceFile, []byte(env.initialContent), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	return env
}

// testEnhancedBackupProviderLevelConfig tests enhanced backup with provider-level configuration
func testEnhancedBackupProviderLevelConfig(t *testing.T, env *enhancedBackupIntegrationTestEnv) {
	// Create enhanced provider model
	providerModel := &EnhancedProviderModel{
		DotfilesRoot:    types.StringValue(env.sourceDir),
		BackupEnabled:   types.BoolValue(true),
		BackupDirectory: types.StringValue(env.backupDir),
		BackupStrategy: &BackupStrategyModel{
			Enabled:         types.BoolValue(true),
			Directory:       types.StringValue(env.backupDir),
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

	validateProviderBackupStrategy(t, providerModel)
}

// validateProviderBackupStrategy validates provider backup strategy configuration
func validateProviderBackupStrategy(t *testing.T, providerModel *EnhancedProviderModel) {
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
}

// testEnhancedBackupFileSpecificPolicy tests enhanced backup with file-specific policy
func testEnhancedBackupFileSpecificPolicy(t *testing.T, env *enhancedBackupIntegrationTestEnv) {
	// Create enhanced file model with backup policy
	fileModel := createEnhancedFileBackupModel(env)

	// Build and validate enhanced backup configuration
	backupConfig, err := buildEnhancedBackupConfig(fileModel.BackupPolicy)
	if err != nil {
		t.Fatalf("Failed to build backup config: %v", err)
	}

	validateFileSpecificBackupConfig(t, backupConfig)
}

// createEnhancedFileBackupModel creates file model for backup testing
func createEnhancedFileBackupModel(env *enhancedBackupIntegrationTestEnv) *EnhancedFileResourceModelWithBackup {
	return &EnhancedFileResourceModelWithBackup{
		EnhancedFileResourceModel: EnhancedFileResourceModel{
			FileResourceModel: FileResourceModel{
				ID:         types.StringValue("test-config"),
				Repository: types.StringValue("test-repo"),
				Name:       types.StringValue("config-file"),
				SourcePath: types.StringValue("config.txt"),
				TargetPath: types.StringValue(filepath.Join(env.targetDir, "config.txt")),
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
}

// validateFileSpecificBackupConfig validates file-specific backup configuration
func validateFileSpecificBackupConfig(t *testing.T, backupConfig *fileops.EnhancedBackupConfig) {
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
}

// testFileOperationsWithEnhancedBackup tests file operations with enhanced backup
func testFileOperationsWithEnhancedBackup(t *testing.T, env *enhancedBackupIntegrationTestEnv) {
	targetFile := filepath.Join(env.targetDir, "enhanced-config.txt")

	// Create initial target file
	if err := os.WriteFile(targetFile, []byte("original target content"), 0644); err != nil {
		t.Fatalf("Failed to create target file: %v", err)
	}

	// Create file manager and config
	platformProvider := platform.DetectPlatform()
	fm := fileops.NewFileManager(platformProvider, false)
	config := createEnhancedBackupConfig(env)

	// Test backup operations
	testBackupCreationAndValidation(t, fm, targetFile, config)
	testBackupUpdateOperations(t, fm, targetFile, config)
}

// createEnhancedBackupConfig creates enhanced backup configuration
func createEnhancedBackupConfig(env *enhancedBackupIntegrationTestEnv) *fileops.EnhancedBackupConfig {
	return &fileops.EnhancedBackupConfig{
		Enabled:        true,
		Directory:      env.backupDir,
		BackupFormat:   "numbered",
		Compression:    true,
		MaxBackups:     3,
		BackupMetadata: true,
		BackupIndex:    true,
		Incremental:    true,
	}
}

// testBackupCreationAndValidation tests backup creation and validation
func testBackupCreationAndValidation(t *testing.T, fm *fileops.FileManager, targetFile string, config *fileops.EnhancedBackupConfig) {
	// Create first enhanced backup
	backupPath1, err := fm.CreateEnhancedBackup(targetFile, config)
	if err != nil {
		t.Fatalf("First enhanced backup failed: %v", err)
	}

	validateBackupCreation(t, backupPath1, config)
}

// validateBackupCreation validates backup creation results
func validateBackupCreation(t *testing.T, backupPath string, config *fileops.EnhancedBackupConfig) {
	if backupPath == "" {
		t.Error("First backup path should not be empty")
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
		t.Error("Backup metadata file should exist")
	}

	// Verify backup index was created
	indexPath := filepath.Join(config.Directory, ".backup_index.json")
	if !pathExists(indexPath) {
		t.Error("Backup index should be created")
	}
}

// testBackupUpdateOperations tests backup update operations
func testBackupUpdateOperations(t *testing.T, fm *fileops.FileManager, targetFile string, config *fileops.EnhancedBackupConfig) {
	// Update target file content
	newContent := "updated target content"
	err := os.WriteFile(targetFile, []byte(newContent), 0644)
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
}

// testRecoveryModelValidation tests recovery model validation
func testRecoveryModelValidation(t *testing.T, env *enhancedBackupIntegrationTestEnv) {
	recoveryModel := &RecoveryTestModel{
		Enabled: types.BoolValue(true),
		Command: types.StringValue("test -r {{.backup_path}}"),
		Timeout: types.StringValue("5s"),
	}

	// Validate recovery model configuration
	if !recoveryModel.Enabled.ValueBool() {
		t.Error("Recovery test should be enabled")
	}
	if !strings.Contains(recoveryModel.Command.ValueString(), "{{.backup_path}}") {
		t.Error("Recovery command should contain backup path placeholder")
	}
	if recoveryModel.Timeout.ValueString() != "5s" {
		t.Error("Recovery timeout should be 5s")
	}
}

// testBackupWithRecoveryValidation tests backup with recovery validation
func testBackupWithRecoveryValidation(t *testing.T, env *enhancedBackupIntegrationTestEnv) {
	targetFile := filepath.Join(env.targetDir, "recovery-test.txt")
	testContent := "recovery test content"

	// Create target file
	if err := os.WriteFile(targetFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create recovery test file: %v", err)
	}

	// Create backup with recovery validation
	platformProvider := platform.DetectPlatform()
	fm := fileops.NewFileManager(platformProvider, false)
	config := createRecoveryValidationConfig(env)

	backupPath, err := fm.CreateEnhancedBackup(targetFile, config)
	if err != nil {
		t.Fatalf("Backup with recovery validation failed: %v", err)
	}

	// Verify backup was created
	if !pathExists(backupPath) {
		t.Error("Recovery validated backup should exist")
	}
}

// createRecoveryValidationConfig creates configuration for recovery validation
func createRecoveryValidationConfig(env *enhancedBackupIntegrationTestEnv) *fileops.EnhancedBackupConfig {
	return &fileops.EnhancedBackupConfig{
		Enabled:        true,
		Directory:      env.backupDir,
		BackupFormat:   "timestamped",
		Compression:    false,
		BackupMetadata: true,
	}
}

// testTimestampedFormatSupport tests timestamped format support
func testTimestampedFormatSupport(t *testing.T, env *enhancedBackupIntegrationTestEnv) {
	config := &fileops.EnhancedBackupConfig{
		Enabled:      true,
		Directory:    env.backupDir,
		BackupFormat: "timestamped",
	}

	err := fileops.ValidateEnhancedBackupConfig(config)
	if err != nil {
		t.Errorf("Timestamped format should be supported: %v", err)
	}
}

// testNumberedFormatSupport tests numbered format support
func testNumberedFormatSupport(t *testing.T, env *enhancedBackupIntegrationTestEnv) {
	config := &fileops.EnhancedBackupConfig{
		Enabled:      true,
		Directory:    env.backupDir,
		BackupFormat: "numbered",
	}

	err := fileops.ValidateEnhancedBackupConfig(config)
	if err != nil {
		t.Errorf("Numbered format should be supported: %v", err)
	}
}

// testGitStyleFormatSupport tests git-style format support
func testGitStyleFormatSupport(t *testing.T, env *enhancedBackupIntegrationTestEnv) {
	config := &fileops.EnhancedBackupConfig{
		Enabled:      true,
		Directory:    env.backupDir,
		BackupFormat: "git_style",
	}

	err := fileops.ValidateEnhancedBackupConfig(config)
	if err != nil {
		t.Errorf("Git-style format should be supported: %v", err)
	}
}

// buildEnhancedBackupConfig builds fileops backup config from provider model.
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

// pathExists checks if a path exists.
func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
