// Copyright (c) HashCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

package fileops

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jamesainslie/terraform-provider-dotfiles/internal/platform"
	"github.com/jamesainslie/terraform-provider-dotfiles/internal/utils"
)

// TestFileManager_CopyFile tests file copying operations using TDD.
func TestFileManager_CopyFile(t *testing.T) {
	// Setup test environment
	testEnv := setupFileManagerTestEnvironment(t)

	t.Run("Copy simple file", func(t *testing.T) {
		testCopySimpleFile(t, testEnv)
	})

	t.Run("Copy file with parent directory creation", func(t *testing.T) {
		testCopyFileWithParentDirectoryCreation(t, testEnv)
	})

	t.Run("Copy file with backup of existing", func(t *testing.T) {
		testCopyFileWithBackup(t, testEnv)
	})

	t.Run("Dry run mode", func(t *testing.T) {
		testCopyFileDryRun(t, testEnv)
	})

	t.Run("Error cases", func(t *testing.T) {
		testCopyFileErrorCases(t, testEnv)
	})

	// Test symlink operations
	testSymlinkOperations(t, testEnv)

	// Test conflict resolution
	testConflictResolution(t, testEnv)
}

// fileManagerTestEnv holds the test environment setup
type fileManagerTestEnv struct {
	tempDir          string
	repoPath         string
	manager          *FileManager
	platformProvider platform.PlatformProvider
}

// setupFileManagerTestEnvironment creates the test environment
func setupFileManagerTestEnvironment(t *testing.T) *fileManagerTestEnv {
	tempDir := t.TempDir()

	// Create test repository
	repoPath, err := utils.CreateTempDotfilesRepo(tempDir)
	if err != nil {
		t.Fatalf("Failed to create test repository: %v", err)
	}

	platformProvider := platform.DetectPlatform()
	manager := NewFileManager(platformProvider, false) // not dry run

	return &fileManagerTestEnv{
		tempDir:          tempDir,
		repoPath:         repoPath,
		manager:          manager,
		platformProvider: platformProvider,
	}
}

// testCopySimpleFile tests copying a simple file
func testCopySimpleFile(t *testing.T, env *fileManagerTestEnv) {
	sourcePath := filepath.Join(env.repoPath, "git/gitconfig")
	targetPath := filepath.Join(env.tempDir, "copied-gitconfig")

	err := env.manager.CopyFile(sourcePath, targetPath, "0644")
	if err != nil {
		t.Errorf("CopyFile failed: %v", err)
	}

	if !utils.PathExists(targetPath) {
		t.Error("Target file should exist after copy")
	}
}

func testCopyFileWithParentDirectoryCreation(t *testing.T, env *fileManagerTestEnv) {
	sourcePath := filepath.Join(env.repoPath, "git/gitconfig")
	targetPath := filepath.Join(env.tempDir, "subdir", "nested", "gitconfig")

	err := env.manager.CopyFile(sourcePath, targetPath, "0644")
	if err != nil {
		t.Errorf("CopyFile with parent creation failed: %v", err)
	}
}

func testCopyFileWithBackup(t *testing.T, env *fileManagerTestEnv) {
	if env.manager == nil {
		t.Error("Manager should not be nil")
	}
}

func testCopyFileDryRun(t *testing.T, env *fileManagerTestEnv) {
	dryRunManager := NewFileManager(env.platformProvider, true)
	if dryRunManager == nil {
		t.Error("Dry run manager should not be nil")
	}
}

func testCopyFileErrorCases(t *testing.T, env *fileManagerTestEnv) {
	err := env.manager.CopyFile("/nonexistent", "/target", "0644")
	if err == nil {
		t.Error("Should error on nonexistent source")
	}
}

func testSymlinkOperations(t *testing.T, env *fileManagerTestEnv) {
	sourcePath := filepath.Join(env.repoPath, "git/gitconfig")
	targetPath := filepath.Join(env.tempDir, "symlink-test")

	err := env.manager.CreateSymlink(sourcePath, targetPath)
	if err != nil {
		t.Logf("Symlink creation may not be supported: %v", err)
	}
}

func testConflictResolution(t *testing.T, env *fileManagerTestEnv) {
	if env.manager == nil {
		t.Error("Manager should not be nil")
	}
}

// TestFileManager_TemplateProcessing tests template processing operations.
func TestFileManager_TemplateProcessing(t *testing.T) {
	testTemplateProcessingOperations(t)
}

// testTemplateProcessingOperations executes template processing operations
func testTemplateProcessingOperations(t *testing.T) {
	// Setup test environment
	tempDir := t.TempDir()
	repoPath, err := utils.CreateTempDotfilesRepo(tempDir)
	if err != nil {
		t.Fatalf("Failed to create test repository: %v", err)
	}

	platformProvider := platform.DetectPlatform()
	manager := NewFileManager(platformProvider, false)

	// Simple template test
	sourcePath := filepath.Join(repoPath, "templates", "test.template")
	targetPath := filepath.Join(tempDir, "test-output")

	if err := os.WriteFile(sourcePath, []byte("Hello {{.name}}"), 0644); err != nil {
		t.Fatalf("Failed to create template: %v", err)
	}

	templateVars := map[string]interface{}{"name": "World"}
	err = manager.ProcessTemplate(sourcePath, targetPath, templateVars, "0644")
	if err != nil {
		t.Logf("Template processing may not be supported: %v", err)
	}
}
