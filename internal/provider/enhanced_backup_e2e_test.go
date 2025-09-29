// Copyright (c) HashCorp, Inc.
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
	// Setup test environment
	testEnv := setupEnhancedBackupTestEnvironment(t)

	t.Run("Complete enhanced backup workflow", func(t *testing.T) {
		testCriticalConfigBackupFeatures(t, testEnv)
		testRetentionManagementFeatures(t, testEnv)
		testTemplateProcessingWithBackup(t, testEnv)
		testRecoveryValidationFeatures(t, testEnv)
	})
}

// enhancedBackupTestEnv holds the test environment setup
type enhancedBackupTestEnv struct {
	tempDir   string
	sourceDir string
	targetDir string
	backupDir string
}

// setupEnhancedBackupTestEnvironment creates and initializes the test environment
func setupEnhancedBackupTestEnvironment(t *testing.T) *enhancedBackupTestEnv {
	tempDir := t.TempDir()
	env := &enhancedBackupTestEnv{
		tempDir:   tempDir,
		sourceDir: filepath.Join(tempDir, "source"),
		targetDir: filepath.Join(tempDir, "target"),
		backupDir: filepath.Join(tempDir, "backups"),
	}

	createTestDirectories(t, env)
	createTestConfigFiles(t, env)

	return env
}

// createTestDirectories creates the necessary directories for testing
func createTestDirectories(t *testing.T, env *enhancedBackupTestEnv) {
	if err := os.MkdirAll(env.sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}

	if err := os.MkdirAll(env.targetDir, 0755); err != nil {
		t.Fatalf("Failed to create target directory: %v", err)
	}
}

// createTestConfigFiles creates test configuration files
func createTestConfigFiles(t *testing.T, env *enhancedBackupTestEnv) {
	configFiles := map[string]string{
		"critical.conf": "# Critical configuration\napi_key=secret123\nhost=production.com",
		"settings.json": `{"theme": "dark", "fontSize": 14}`,
		"app.yaml":      "version: 1.0\nenvironment: production",
	}

	for filename, content := range configFiles {
		sourcePath := filepath.Join(env.sourceDir, filename)
		err := os.WriteFile(sourcePath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create source file %s: %v", filename, err)
		}
	}
}

// testCriticalConfigBackupFeatures tests comprehensive backup policy features
func testCriticalConfigBackupFeatures(t *testing.T, env *enhancedBackupTestEnv) {
	targetFile := filepath.Join(env.targetDir, "critical.conf")

	// Create existing target file (to trigger backup)
	err := os.WriteFile(targetFile, []byte("old configuration"), 0644)
	if err != nil {
		t.Fatalf("Failed to create existing target file: %v", err)
	}

	// Create file model with comprehensive backup policy
	model := createCriticalConfigModel(targetFile)

	// Test backup creation and validation
	testCriticalConfigBackup(t, env, targetFile, model)
}

// createCriticalConfigModel creates the model for critical config testing
func createCriticalConfigModel(targetFile string) *EnhancedFileResourceModelWithBackup {
	return &EnhancedFileResourceModelWithBackup{
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
}

// testCriticalConfigBackup performs the backup creation and validation
func testCriticalConfigBackup(t *testing.T, env *enhancedBackupTestEnv, targetFile string, model *EnhancedFileResourceModelWithBackup) {
	// Build and validate backup configuration
	backupConfig, err := buildEnhancedBackupConfigFromFileModel(model)
	if err != nil {
		t.Fatalf("Failed to build enhanced backup config: %v", err)
	}

	// Set backup directory
	backupConfig.Directory = env.backupDir

	// Create FileManager and test backup creation
	platformProvider := platform.DetectPlatform()
	fm := fileops.NewFileManager(platformProvider, false)

	// Create initial backup
	backupPath, err := fm.CreateEnhancedBackup(targetFile, backupConfig)
	if err != nil {
		t.Fatalf("Failed to create enhanced backup: %v", err)
	}

	// Validate all backup features
	validateCriticalConfigBackup(t, env, backupPath, targetFile)
}

// validateCriticalConfigBackup validates all aspects of the critical config backup
func validateCriticalConfigBackup(t *testing.T, env *enhancedBackupTestEnv, backupPath, targetFile string) {
	validateBackupFileProperties(t, backupPath)
	validateBackupMetadataFile(t, backupPath, targetFile)
	validateBackupIndexFile(t, env)
}

// validateBackupFileProperties validates basic backup file properties
func validateBackupFileProperties(t *testing.T, backupPath string) {
	if !strings.Contains(backupPath, ".backup.") {
		t.Error("Backup path should contain .backup.")
	}
	if !strings.HasSuffix(backupPath, ".gz") {
		t.Error("Compressed backup should have .gz extension")
	}
	if !pathExists(backupPath) {
		t.Error("Backup file should exist")
	}
}

// validateBackupMetadataFile validates backup metadata
func validateBackupMetadataFile(t *testing.T, backupPath, targetFile string) {
	metadataPath := backupPath + ".meta"
	if !pathExists(metadataPath) {
		t.Error("Backup metadata should be created")
		return
	}

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
}

// validateBackupIndexFile validates backup index
func validateBackupIndexFile(t *testing.T, env *enhancedBackupTestEnv) {
	indexPath := filepath.Join(env.backupDir, ".backup_index.json")
	if !pathExists(indexPath) {
		t.Error("Backup index should be created")
		return
	}

	index, err := fileops.LoadBackupIndex(indexPath)
	if err != nil {
		t.Fatalf("Failed to load backup index: %v", err)
	}
	if len(index.Backups) != 1 {
		t.Errorf("Expected 1 backup in index, got %d", len(index.Backups))
	}
}

// testRetentionManagementFeatures tests backup retention features
func testRetentionManagementFeatures(t *testing.T, env *enhancedBackupTestEnv) {
	targetFile := filepath.Join(env.targetDir, "settings.json")

	// Create target file
	err := os.WriteFile(targetFile, []byte(`{"theme": "light"}`), 0644)
	if err != nil {
		t.Fatalf("Failed to create target file: %v", err)
	}

	// Create model for retention testing
	model := createRetentionTestModel(targetFile)

	// Test multiple backups with retention
	testMultipleBackupsWithRetention(t, env, targetFile, model)
}

// createRetentionTestModel creates a model for retention testing
func createRetentionTestModel(targetFile string) *EnhancedFileResourceModelWithBackup {
	return &EnhancedFileResourceModelWithBackup{
		EnhancedFileResourceModel: EnhancedFileResourceModel{
			FileResourceModel: FileResourceModel{
				ID:            types.StringValue("retention-test"),
				Repository:    types.StringValue("test-repo"),
				Name:          types.StringValue("settings-config"),
				SourcePath:    types.StringValue("settings.json"),
				TargetPath:    types.StringValue(targetFile),
				BackupEnabled: types.BoolValue(true),
			},
		},
		BackupPolicy: &BackupPolicyModel{
			AlwaysBackup:    types.BoolValue(true),
			VersionedBackup: types.BoolValue(true),
			BackupFormat:    types.StringValue("numbered"),
			RetentionCount:  types.Int64Value(2), // Keep only 2 backups
			BackupMetadata:  types.BoolValue(false),
			Compression:     types.BoolValue(false),
		},
	}
}

// testMultipleBackupsWithRetention tests multiple backup creation and retention
func testMultipleBackupsWithRetention(t *testing.T, env *enhancedBackupTestEnv, targetFile string, model *EnhancedFileResourceModelWithBackup) {
	backupConfig, err := buildEnhancedBackupConfigFromFileModel(model)
	if err != nil {
		t.Fatalf("Failed to build backup config: %v", err)
	}
	backupConfig.Directory = env.backupDir

	platformProvider := platform.DetectPlatform()
	fm := fileops.NewFileManager(platformProvider, false)

	// Create multiple backups to test retention
	for i := 1; i <= 4; i++ {
		// Update file content
		content := fmt.Sprintf(`{"theme": "version_%d"}`, i)
		err := os.WriteFile(targetFile, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to update file for backup %d: %v", i, err)
		}

		// Create backup
		_, err = fm.CreateEnhancedBackup(targetFile, backupConfig)
		if err != nil {
			t.Fatalf("Failed to create backup %d: %v", i, err)
		}
	}

	// Verify retention policy was applied
	validateRetentionPolicy(t, env, targetFile)
}

// validateRetentionPolicy validates that only the expected number of backups remain
func validateRetentionPolicy(t *testing.T, env *enhancedBackupTestEnv, targetFile string) {
	backupPattern := filepath.Join(env.backupDir, filepath.Base(targetFile)+".backup.*")
	matches, err := filepath.Glob(backupPattern)
	if err != nil {
		t.Fatalf("Failed to glob backup files: %v", err)
	}

	if len(matches) > 2 {
		t.Errorf("Expected at most 2 backups due to retention policy, found %d", len(matches))
	}
}

// testTemplateProcessingWithBackup tests template processing with backup
func testTemplateProcessingWithBackup(t *testing.T, env *enhancedBackupTestEnv) {
	targetFile := filepath.Join(env.targetDir, "app.yaml")

	// Create initial target file
	err := os.WriteFile(targetFile, []byte("version: 0.9"), 0644)
	if err != nil {
		t.Fatalf("Failed to create target file: %v", err)
	}

	// Create model for template testing
	model := createTemplateTestModel(targetFile)

	// Test template processing with backup
	testTemplateBackupProcessing(t, env, targetFile, model)
}

// createTemplateTestModel creates a model for template testing
func createTemplateTestModel(targetFile string) *EnhancedFileResourceModelWithBackup {
	return &EnhancedFileResourceModelWithBackup{
		EnhancedFileResourceModel: EnhancedFileResourceModel{
			FileResourceModel: FileResourceModel{
				ID:            types.StringValue("template-test"),
				Repository:    types.StringValue("test-repo"),
				Name:          types.StringValue("app-config"),
				SourcePath:    types.StringValue("app.yaml"),
				TargetPath:    types.StringValue(targetFile),
				BackupEnabled: types.BoolValue(true),
			},
		},
		BackupPolicy: &BackupPolicyModel{
			AlwaysBackup:   types.BoolValue(true),
			BackupFormat:   types.StringValue("timestamped"),
			BackupMetadata: types.BoolValue(true),
			Compression:    types.BoolValue(false),
		},
	}
}

// testTemplateBackupProcessing tests the template backup processing logic
func testTemplateBackupProcessing(t *testing.T, env *enhancedBackupTestEnv, targetFile string, model *EnhancedFileResourceModelWithBackup) {
	backupConfig, err := buildEnhancedBackupConfigFromFileModel(model)
	if err != nil {
		t.Fatalf("Failed to build backup config: %v", err)
	}
	backupConfig.Directory = env.backupDir

	platformProvider := platform.DetectPlatform()
	fm := fileops.NewFileManager(platformProvider, false)

	// Create backup before template processing
	backupPath, err := fm.CreateEnhancedBackup(targetFile, backupConfig)
	if err != nil {
		t.Fatalf("Failed to create template backup: %v", err)
	}

	// Verify backup was created
	if !pathExists(backupPath) {
		t.Error("Template backup should exist")
	}

	// Verify metadata
	if backupConfig.BackupMetadata {
		metadataPath := backupPath + ".meta"
		if !pathExists(metadataPath) {
			t.Error("Template backup metadata should exist")
		}
	}
}

// testRecoveryValidationFeatures tests recovery and validation features
func testRecoveryValidationFeatures(t *testing.T, env *enhancedBackupTestEnv) {
	targetFile := filepath.Join(env.targetDir, "recovery-test.conf")

	// Create target file
	err := os.WriteFile(targetFile, []byte("test configuration"), 0644)
	if err != nil {
		t.Fatalf("Failed to create target file: %v", err)
	}

	// Create model with recovery testing
	model := createRecoveryTestModel(targetFile)

	// Test recovery features
	testRecoveryFeatures(t, env, targetFile, model)
}

// createRecoveryTestModel creates a model for recovery testing
func createRecoveryTestModel(targetFile string) *EnhancedFileResourceModelWithBackup {
	return &EnhancedFileResourceModelWithBackup{
		EnhancedFileResourceModel: EnhancedFileResourceModel{
			FileResourceModel: FileResourceModel{
				ID:            types.StringValue("recovery-test"),
				Repository:    types.StringValue("test-repo"),
				Name:          types.StringValue("recovery-config"),
				SourcePath:    types.StringValue("recovery-test.conf"),
				TargetPath:    types.StringValue(targetFile),
				BackupEnabled: types.BoolValue(true),
			},
		},
		BackupPolicy: &BackupPolicyModel{
			AlwaysBackup:   types.BoolValue(true),
			BackupFormat:   types.StringValue("timestamped"),
			BackupMetadata: types.BoolValue(true),
			Compression:    types.BoolValue(true),
		},
		RecoveryTest: &RecoveryTestModel{
			Enabled: types.BoolValue(true),
			Command: types.StringValue("test -f {{.backup_path}}"),
			Timeout: types.StringValue("5s"),
		},
	}
}

// testRecoveryFeatures tests the recovery validation functionality
func testRecoveryFeatures(t *testing.T, env *enhancedBackupTestEnv, targetFile string, model *EnhancedFileResourceModelWithBackup) {
	backupConfig, err := buildEnhancedBackupConfigFromFileModel(model)
	if err != nil {
		t.Fatalf("Failed to build backup config: %v", err)
	}
	backupConfig.Directory = env.backupDir

	platformProvider := platform.DetectPlatform()
	fm := fileops.NewFileManager(platformProvider, false)

	// Create backup for recovery testing
	backupPath, err := fm.CreateEnhancedBackup(targetFile, backupConfig)
	if err != nil {
		t.Fatalf("Failed to create recovery backup: %v", err)
	}

	// Verify backup exists for recovery test
	if !pathExists(backupPath) {
		t.Error("Recovery backup should exist")
	}

	// Test recovery validation
	validateRecoveryFeatures(t, backupPath)
}

// validateRecoveryFeatures validates recovery functionality
func validateRecoveryFeatures(t *testing.T, backupPath string) {
	// Basic validation that backup file exists and is accessible
	if !pathExists(backupPath) {
		t.Error("Recovery backup validation failed - file does not exist")
	}

	// Check if file is readable
	_, err := os.Stat(backupPath)
	if err != nil {
		t.Errorf("Recovery backup validation failed - cannot access file: %v", err)
	}
}
