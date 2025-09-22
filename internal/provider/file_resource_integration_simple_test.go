// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jamesainslie/terraform-provider-dotfiles/internal/fileops"
	"github.com/jamesainslie/terraform-provider-dotfiles/internal/platform"
	"github.com/jamesainslie/terraform-provider-dotfiles/internal/template"
	"github.com/jamesainslie/terraform-provider-dotfiles/internal/utils"
)

// TestFileResourceImplementation tests the file resource implementation directly
func TestFileResourceImplementation(t *testing.T) {
	tempDir := t.TempDir()
	
	// Create test repository
	repoPath, err := utils.CreateTempDotfilesRepo(tempDir)
	if err != nil {
		t.Fatalf("Failed to create test repository: %v", err)
	}
	
	t.Run("Test file operations integration", func(t *testing.T) {
		platformProvider := platform.DetectPlatform()
		fileManager := fileops.NewFileManager(platformProvider, false)
		
		// Test file copy
		sourcePath := filepath.Join(repoPath, "git/gitconfig")
		targetPath := filepath.Join(tempDir, "test-gitconfig")
		
		err := fileManager.CopyFile(sourcePath, targetPath, "0644")
		if err != nil {
			t.Errorf("File copy failed: %v", err)
		}
		
		// Verify file exists
		if !utils.PathExists(targetPath) {
			t.Error("Target file should exist after copy")
		}
		
		// Test template processing
		templatePath := filepath.Join(tempDir, "test.template")
		templateContent := `Hello {{.name}}!
Your platform is {{.system.platform}}.`
		
		err = os.WriteFile(templatePath, []byte(templateContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create template: %v", err)
		}
		
		templateOutputPath := filepath.Join(tempDir, "processed-template.txt")
		templateVars := map[string]interface{}{
			"name": "Test User",
		}
		systemInfo := map[string]interface{}{
			"platform": "test",
		}
		context := template.CreateTemplateContext(systemInfo, templateVars)
		
		err = fileManager.ProcessTemplate(templatePath, templateOutputPath, context, "0644")
		if err != nil {
			t.Errorf("Template processing failed: %v", err)
		}
		
		// Verify template was processed
		if !utils.PathExists(templateOutputPath) {
			t.Error("Template output should exist")
		} else {
			content, err := os.ReadFile(templateOutputPath)
			if err != nil {
				t.Errorf("Failed to read template output: %v", err)
			} else {
				processedContent := string(content)
				if !containsString(processedContent, "Test User") {
					t.Errorf("Template should contain processed name, got: %s", processedContent)
				}
				if !containsString(processedContent, "test") {
					t.Errorf("Template should contain processed platform, got: %s", processedContent)
				}
			}
		}
	})
	
	t.Run("Test backup system", func(t *testing.T) {
		platformProvider := platform.DetectPlatform()
		fileManager := fileops.NewFileManager(platformProvider, false)
		
		// Create existing file
		existingFile := filepath.Join(tempDir, "backup-test.txt")
		originalContent := "original content"
		err := os.WriteFile(existingFile, []byte(originalContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create existing file: %v", err)
		}
		
		// Test backup creation
		backupDir := filepath.Join(tempDir, "backups")
		backupPath, err := fileManager.CreateBackup(existingFile, backupDir)
		if err != nil {
			t.Errorf("Backup creation failed: %v", err)
		}
		
		// Verify backup exists
		if !utils.PathExists(backupPath) {
			t.Error("Backup file should exist")
		}
		
		// Verify backup content
		same, err := utils.CompareFileContent(existingFile, backupPath)
		if err != nil {
			t.Errorf("Failed to compare backup content: %v", err)
		} else if !same {
			t.Error("Backup content should match original")
		}
		
		// Test conflict resolution
		result, err := fileManager.ResolveConflict(existingFile, backupDir, "backup")
		if err != nil {
			t.Errorf("Conflict resolution failed: %v", err)
		}
		
		if result.Action != "backup" {
			t.Errorf("Expected backup action, got %s", result.Action)
		}
		
		if !result.ShouldProceed {
			t.Error("Should proceed after backup conflict resolution")
		}
	})
}

// Helper function to check if string contains substring
func containsString(str, substr string) bool {
	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
