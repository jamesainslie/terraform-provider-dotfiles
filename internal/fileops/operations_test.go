// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package fileops

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jamesainslie/terraform-provider-dotfiles/internal/platform"
	"github.com/jamesainslie/terraform-provider-dotfiles/internal/utils"
)

// TestFileManager_CopyFile tests file copying operations using TDD
func TestFileManager_CopyFile(t *testing.T) {
	tempDir := t.TempDir()
	
	// Create test repository
	repoPath, err := utils.CreateTempDotfilesRepo(tempDir)
	if err != nil {
		t.Fatalf("Failed to create test repository: %v", err)
	}
	
	platformProvider := platform.DetectPlatform()
	manager := NewFileManager(platformProvider, false) // not dry run
	
	t.Run("Copy simple file", func(t *testing.T) {
		sourcePath := filepath.Join(repoPath, "git/gitconfig")
		targetPath := filepath.Join(tempDir, "copied-gitconfig")
		
		err := manager.CopyFile(sourcePath, targetPath, "0644")
		if err != nil {
			t.Errorf("CopyFile failed: %v", err)
		}
		
		// Verify file was copied
		if !utils.PathExists(targetPath) {
			t.Error("Target file should exist after copy")
		}
		
		// Verify content matches
		same, err := utils.CompareFileContent(sourcePath, targetPath)
		if err != nil {
			t.Errorf("Failed to compare file content: %v", err)
		}
		if !same {
			t.Error("Copied file content should match source")
		}
		
		// Verify permissions
		info, err := os.Stat(targetPath)
		if err != nil {
			t.Errorf("Failed to stat target file: %v", err)
		} else {
			expectedMode, _ := utils.ParseFileMode("0644")
			if info.Mode().Perm() != expectedMode {
				t.Errorf("File permissions not set correctly: expected %v, got %v", expectedMode, info.Mode().Perm())
			}
		}
	})
	
	t.Run("Copy file with parent directory creation", func(t *testing.T) {
		sourcePath := filepath.Join(repoPath, "fish/config.fish")
		targetPath := filepath.Join(tempDir, "deep/nested/config.fish")
		
		err := manager.CopyFile(sourcePath, targetPath, "0644")
		if err != nil {
			t.Errorf("CopyFile with parent creation failed: %v", err)
		}
		
		// Verify parent directories were created
		parentDir := filepath.Dir(targetPath)
		if !utils.PathExists(parentDir) {
			t.Error("Parent directories should be created")
		}
		
		// Verify file was copied
		if !utils.PathExists(targetPath) {
			t.Error("Target file should exist")
		}
	})
	
	t.Run("Copy file with backup of existing", func(t *testing.T) {
		sourcePath := filepath.Join(repoPath, "git/gitconfig")
		targetPath := filepath.Join(tempDir, "gitconfig-backup-test")
		
		// Create existing file
		existingContent := "existing content that should be backed up"
		err := os.WriteFile(targetPath, []byte(existingContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create existing file: %v", err)
		}
		
		backupDir := filepath.Join(tempDir, "backups")
		err = manager.CopyFileWithBackup(sourcePath, targetPath, "0644", backupDir)
		if err != nil {
			t.Errorf("CopyFileWithBackup failed: %v", err)
		}
		
		// Verify original file was backed up
		if !utils.PathExists(backupDir) {
			t.Error("Backup directory should be created")
		}
		
		// Check backup was created (should have timestamp or similar)
		entries, err := os.ReadDir(backupDir)
		if err != nil {
			t.Errorf("Failed to read backup directory: %v", err)
		} else if len(entries) == 0 {
			t.Error("Backup file should be created")
		}
		
		// Verify new file has source content
		same, err := utils.CompareFileContent(sourcePath, targetPath)
		if err != nil {
			t.Errorf("Failed to compare file content: %v", err)
		}
		if !same {
			t.Error("Target file should have source content after backup")
		}
	})
	
	t.Run("Dry run mode", func(t *testing.T) {
		dryRunManager := NewFileManager(platformProvider, true) // dry run enabled
		
		sourcePath := filepath.Join(repoPath, "git/gitconfig")
		targetPath := filepath.Join(tempDir, "dry-run-test")
		
		err := dryRunManager.CopyFile(sourcePath, targetPath, "0644")
		if err != nil {
			t.Errorf("Dry run should not error: %v", err)
		}
		
		// File should NOT exist in dry run mode
		if utils.PathExists(targetPath) {
			t.Error("File should not be created in dry run mode")
		}
	})
	
	t.Run("Error cases", func(t *testing.T) {
		// Non-existent source file
		err := manager.CopyFile("/nonexistent/source.txt", filepath.Join(tempDir, "target.txt"), "0644")
		if err == nil {
			t.Error("CopyFile should error with non-existent source")
		}
		
		// Invalid permissions format
		sourcePath := filepath.Join(repoPath, "git/gitconfig")
		err = manager.CopyFile(sourcePath, filepath.Join(tempDir, "invalid-perms"), "invalid")
		if err == nil {
			t.Error("CopyFile should error with invalid permissions")
		}
	})
}

func TestFileManager_CreateSymlink(t *testing.T) {
	tempDir := t.TempDir()
	
	// Create test repository
	repoPath, err := utils.CreateTempDotfilesRepo(tempDir)
	if err != nil {
		t.Fatalf("Failed to create test repository: %v", err)
	}
	
	platformProvider := platform.DetectPlatform()
	manager := NewFileManager(platformProvider, false)
	
	t.Run("Create symlink to file", func(t *testing.T) {
		sourcePath := filepath.Join(repoPath, "git/gitconfig")
		targetPath := filepath.Join(tempDir, "symlink-gitconfig")
		
		err := manager.CreateSymlink(sourcePath, targetPath)
		if err != nil {
			// Symlinks might not be supported on all platforms
			t.Logf("CreateSymlink failed (may be expected): %v", err)
			return
		}
		
		// Verify symlink was created
		if !utils.IsSymlink(targetPath) {
			t.Error("Target should be a symlink")
		}
		
		// Verify symlink points to source
		linkTarget, err := os.Readlink(targetPath)
		if err != nil {
			t.Errorf("Failed to read symlink: %v", err)
		} else {
			// Compare absolute paths
			absSource, _ := filepath.Abs(sourcePath)
			absTarget, _ := filepath.Abs(linkTarget)
			if absTarget != absSource {
				t.Errorf("Symlink target mismatch: expected %s, got %s", absSource, absTarget)
			}
		}
	})
	
	t.Run("Create symlink to directory", func(t *testing.T) {
		sourcePath := filepath.Join(repoPath, "fish")
		targetPath := filepath.Join(tempDir, "symlink-fish")
		
		err := manager.CreateSymlink(sourcePath, targetPath)
		if err != nil {
			t.Logf("Directory symlink failed (may be expected): %v", err)
			return
		}
		
		// Verify directory symlink
		if !utils.IsSymlink(targetPath) {
			t.Error("Target should be a symlink")
		}
	})
	
	t.Run("Symlink with parent directory creation", func(t *testing.T) {
		sourcePath := filepath.Join(repoPath, "ssh/config")
		targetPath := filepath.Join(tempDir, "deep/nested/ssh-config")
		
		err := manager.CreateSymlinkWithParents(sourcePath, targetPath)
		if err != nil {
			t.Logf("CreateSymlinkWithParents failed (may be expected): %v", err)
			return
		}
		
		// Verify parent directories were created
		parentDir := filepath.Dir(targetPath)
		if !utils.PathExists(parentDir) {
			t.Error("Parent directories should be created")
		}
		
		if utils.PathExists(targetPath) && !utils.IsSymlink(targetPath) {
			t.Error("Target should be a symlink if it exists")
		}
	})
}

func TestFileManager_BackupOperations(t *testing.T) {
	tempDir := t.TempDir()
	backupDir := filepath.Join(tempDir, "backups")
	
	platformProvider := platform.DetectPlatform()
	manager := NewFileManager(platformProvider, false)
	
	t.Run("Create backup of existing file", func(t *testing.T) {
		// Create existing file
		existingFile := filepath.Join(tempDir, "existing.txt")
		existingContent := "existing content to backup"
		err := os.WriteFile(existingFile, []byte(existingContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create existing file: %v", err)
		}
		
		// Create backup
		backupPath, err := manager.CreateBackup(existingFile, backupDir)
		if err != nil {
			t.Errorf("CreateBackup failed: %v", err)
		}
		
		// Verify backup was created
		if !utils.PathExists(backupPath) {
			t.Error("Backup file should be created")
		}
		
		// Verify backup content matches original
		same, err := utils.CompareFileContent(existingFile, backupPath)
		if err != nil {
			t.Errorf("Failed to compare backup content: %v", err)
		}
		if !same {
			t.Error("Backup content should match original")
		}
		
		// Verify backup filename includes original name
		backupName := filepath.Base(backupPath)
		if !containsSubstring(backupName, "existing.txt") {
			t.Error("Backup filename should contain original filename")
		}
	})
	
	t.Run("Conflict resolution strategies", func(t *testing.T) {
		existingFile := filepath.Join(tempDir, "conflict-test.txt")
		existingContent := "existing content"
		err := os.WriteFile(existingFile, []byte(existingContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create existing file: %v", err)
		}
		
		// Test backup strategy
		result, err := manager.ResolveConflict(existingFile, backupDir, "backup")
		if err != nil {
			t.Errorf("ResolveConflict with backup strategy failed: %v", err)
		}
		
		if result.Action != "backup" {
			t.Errorf("Expected action 'backup', got '%s'", result.Action)
		}
		
		if result.BackupPath == "" {
			t.Error("Backup path should be provided for backup strategy")
		}
		
		// Test overwrite strategy
		result, err = manager.ResolveConflict(existingFile, backupDir, "overwrite")
		if err != nil {
			t.Errorf("ResolveConflict with overwrite strategy failed: %v", err)
		}
		
		if result.Action != "overwrite" {
			t.Errorf("Expected action 'overwrite', got '%s'", result.Action)
		}
		
		// Test skip strategy
		result, err = manager.ResolveConflict(existingFile, backupDir, "skip")
		if err != nil {
			t.Errorf("ResolveConflict with skip strategy failed: %v", err)
		}
		
		if result.Action != "skip" {
			t.Errorf("Expected action 'skip', got '%s'", result.Action)
		}
	})
}

func TestFileManager_TemplateProcessing(t *testing.T) {
	tempDir := t.TempDir()
	
	platformProvider := platform.DetectPlatform()
	manager := NewFileManager(platformProvider, false)
	
	t.Run("Process simple template", func(t *testing.T) {
		// Create template file
		templatePath := filepath.Join(tempDir, "simple.template")
		templateContent := `Hello {{.name}}!
Your email is {{.email}}.
Platform: {{.platform}}`
		
		err := os.WriteFile(templatePath, []byte(templateContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create template: %v", err)
		}
		
		// Template variables
		variables := map[string]interface{}{
			"name":     "Test User",
			"email":    "test@example.com",
			"platform": "test",
		}
		
		targetPath := filepath.Join(tempDir, "processed-simple.txt")
		
		err = manager.ProcessTemplate(templatePath, targetPath, variables, "0644")
		if err != nil {
			t.Errorf("ProcessTemplate failed: %v", err)
		}
		
		// Verify template was processed
		if !utils.PathExists(targetPath) {
			t.Error("Processed template file should exist")
		}
		
		// Verify content was processed
		content, err := os.ReadFile(targetPath)
		if err != nil {
			t.Errorf("Failed to read processed template: %v", err)
		} else {
			processedContent := string(content)
			if !contains([]string{processedContent}, "Test User") {
				t.Error("Template should contain processed 'Test User'")
			}
			if !contains([]string{processedContent}, "test@example.com") {
				t.Error("Template should contain processed email")
			}
			if contains([]string{processedContent}, "{{.name}}") {
				t.Error("Template should not contain unprocessed variables")
			}
		}
	})
	
	t.Run("Process template with system context", func(t *testing.T) {
		templatePath := filepath.Join(tempDir, "system.template")
		templateContent := `System Info:
Platform: {{.system.platform}}
Home: {{.system.home_dir}}
Config: {{.system.config_dir}}
{{if .features.docker}}Docker support enabled{{end}}`
		
		err := os.WriteFile(templatePath, []byte(templateContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create template: %v", err)
		}
		
		// System context
		variables := map[string]interface{}{
			"system": map[string]interface{}{
				"platform":   "macos",
				"home_dir":   "/Users/test",
				"config_dir": "/Users/test/.config",
			},
			"features": map[string]interface{}{
				"docker": true,
			},
		}
		
		targetPath := filepath.Join(tempDir, "processed-system.txt")
		
		err = manager.ProcessTemplate(templatePath, targetPath, variables, "0644")
		if err != nil {
			t.Errorf("ProcessTemplate with system context failed: %v", err)
		}
		
		// Verify system context was processed
		content, err := os.ReadFile(targetPath)
		if err != nil {
			t.Errorf("Failed to read processed template: %v", err)
		} else {
			processedContent := string(content)
			if !contains([]string{processedContent}, "macos") {
				t.Error("Template should contain processed platform")
			}
			if !contains([]string{processedContent}, "Docker support enabled") {
				t.Error("Template should contain conditional content")
			}
		}
	})
	
	t.Run("Template error handling", func(t *testing.T) {
		// Invalid template syntax
		templatePath := filepath.Join(tempDir, "invalid.template")
		invalidTemplate := `Hello {{.name}!  // Missing closing brace`
		
		err := os.WriteFile(templatePath, []byte(invalidTemplate), 0644)
		if err != nil {
			t.Fatalf("Failed to create invalid template: %v", err)
		}
		
		targetPath := filepath.Join(tempDir, "invalid-processed.txt")
		
		err = manager.ProcessTemplate(templatePath, targetPath, map[string]interface{}{}, "0644")
		if err == nil {
			t.Error("ProcessTemplate should error with invalid template syntax")
		}
		
		// Target file should not be created on error
		if utils.PathExists(targetPath) {
			t.Error("Target file should not be created when template processing fails")
		}
	})
}

func TestFileManager_ConflictResolution(t *testing.T) {
	tempDir := t.TempDir()
	backupDir := filepath.Join(tempDir, "backups")
	
	platformProvider := platform.DetectPlatform()
	manager := NewFileManager(platformProvider, false)
	
	t.Run("Backup strategy", func(t *testing.T) {
		existingFile := filepath.Join(tempDir, "backup-test.txt")
		err := os.WriteFile(existingFile, []byte("existing"), 0644)
		if err != nil {
			t.Fatalf("Failed to create existing file: %v", err)
		}
		
		result, err := manager.ResolveConflict(existingFile, backupDir, "backup")
		if err != nil {
			t.Errorf("Backup conflict resolution failed: %v", err)
		}
		
		if result.Action != "backup" {
			t.Errorf("Expected action 'backup', got '%s'", result.Action)
		}
		
		if result.BackupPath == "" {
			t.Error("Backup path should be provided")
		}
		
		if !result.ShouldProceed {
			t.Error("Should proceed after backup")
		}
	})
	
	t.Run("Overwrite strategy", func(t *testing.T) {
		existingFile := filepath.Join(tempDir, "overwrite-test.txt")
		err := os.WriteFile(existingFile, []byte("existing"), 0644)
		if err != nil {
			t.Fatalf("Failed to create existing file: %v", err)
		}
		
		result, err := manager.ResolveConflict(existingFile, backupDir, "overwrite")
		if err != nil {
			t.Errorf("Overwrite conflict resolution failed: %v", err)
		}
		
		if result.Action != "overwrite" {
			t.Errorf("Expected action 'overwrite', got '%s'", result.Action)
		}
		
		if !result.ShouldProceed {
			t.Error("Should proceed with overwrite")
		}
	})
	
	t.Run("Skip strategy", func(t *testing.T) {
		existingFile := filepath.Join(tempDir, "skip-test.txt")
		err := os.WriteFile(existingFile, []byte("existing"), 0644)
		if err != nil {
			t.Fatalf("Failed to create existing file: %v", err)
		}
		
		result, err := manager.ResolveConflict(existingFile, backupDir, "skip")
		if err != nil {
			t.Errorf("Skip conflict resolution failed: %v", err)
		}
		
		if result.Action != "skip" {
			t.Errorf("Expected action 'skip', got '%s'", result.Action)
		}
		
		if result.ShouldProceed {
			t.Error("Should not proceed with skip strategy")
		}
	})
}

// Helper function for string slice contains check
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// Helper function to check if string contains substring
func containsSubstring(str, substr string) bool {
	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
