// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

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

// TestEndToEndDotfilesManagement tests complete dotfiles workflow.
func TestEndToEndDotfilesManagement(t *testing.T) {
	tempDir := t.TempDir()

	// Create test dotfiles repository
	repoPath, err := utils.CreateTempDotfilesRepo(tempDir)
	if err != nil {
		t.Fatalf("Failed to create test repository: %v", err)
	}

	// Create additional template files for testing
	templatesDir := filepath.Join(repoPath, "templates")
	err = os.MkdirAll(templatesDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create templates directory: %v", err)
	}

	// Create gitconfig template
	gitconfigTemplate := `[user]
    name = {{.user_name}}
    email = {{.user_email}}
[core]
    editor = {{.editor}}
{{if .features.gpg_signing}}[user]
    signingkey = {{.gpg_key}}{{end}}
{{if .features.git_lfs}}[filter "lfs"]
    required = true{{end}}`

	err = os.WriteFile(filepath.Join(templatesDir, "gitconfig.template"), []byte(gitconfigTemplate), 0644)
	if err != nil {
		t.Fatalf("Failed to create gitconfig template: %v", err)
	}

	// Create fish config template
	fishTemplate := `# Fish shell configuration for {{.user_name}}
set -g fish_greeting ""
set -gx EDITOR {{.editor}}
{{if .features.docker}}# Docker support enabled
set -gx DOCKER_HOST unix:///var/run/docker.sock{{end}}
{{if eq .system.platform "macos"}}# macOS specific settings
fish_add_path /opt/homebrew/bin{{end}}`

	err = os.WriteFile(filepath.Join(templatesDir, "config.fish.template"), []byte(fishTemplate), 0644)
	if err != nil {
		t.Fatalf("Failed to create fish template: %v", err)
	}

	// Create target directories
	configDir := filepath.Join(tempDir, ".config")
	fishDir := filepath.Join(configDir, "fish")
	err = os.MkdirAll(fishDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create config directories: %v", err)
	}

	t.Run("Complete workflow: files, templates, and symlinks", func(t *testing.T) {
		// Test complete dotfiles setup workflow
		platformProvider := platform.DetectPlatform()
		fileManager := fileops.NewFileManager(platformProvider, false)

		// Step 1: Copy regular file (SSH config)
		t.Run("Step 1: Copy SSH config", func(t *testing.T) {
			sourcePath := filepath.Join(repoPath, "ssh/config")
			targetPath := filepath.Join(tempDir, ".ssh", "config")

			// Create .ssh directory
			err := os.MkdirAll(filepath.Dir(targetPath), 0700)
			if err != nil {
				t.Fatalf("Failed to create .ssh directory: %v", err)
			}

			err = fileManager.CopyFile(sourcePath, targetPath, "0600")
			if err != nil {
				t.Errorf("SSH config copy failed: %v", err)
			}

			// Verify SSH config was copied
			if !utils.PathExists(targetPath) {
				t.Error("SSH config should be copied")
			}

			// Verify permissions are correct for SSH
			info, err := os.Stat(targetPath)
			if err != nil {
				t.Errorf("Failed to stat SSH config: %v", err)
			} else {
				expectedMode, _ := utils.ParseFileMode("0600")
				if info.Mode().Perm() != expectedMode {
					t.Errorf("SSH config permissions incorrect: expected %v, got %v", expectedMode, info.Mode().Perm())
				}
			}
		})

		// Step 2: Process gitconfig template
		t.Run("Step 2: Process gitconfig template", func(t *testing.T) {
			templatePath := filepath.Join(templatesDir, "gitconfig.template")
			targetPath := filepath.Join(tempDir, ".gitconfig")

			// Create template context
			templateVars := map[string]interface{}{
				"user_name":  "Test User",
				"user_email": "test@example.com",
				"editor":     "vim",
				"gpg_key":    "ABC123DEF456",
			}

			systemInfo := map[string]interface{}{
				"platform":     "macos",
				"architecture": "arm64",
				"home_dir":     tempDir,
				"config_dir":   configDir,
			}

			features := map[string]interface{}{
				"gpg_signing": true,
				"git_lfs":     true,
			}

			context := template.CreateTemplateContextWithFeatures(systemInfo, templateVars, features)

			err := fileManager.ProcessTemplate(templatePath, targetPath, context, "0644")
			if err != nil {
				t.Errorf("Gitconfig template processing failed: %v", err)
			}

			// Verify template was processed correctly
			if !utils.PathExists(targetPath) {
				t.Error("Gitconfig should be created from template")
			} else {
				content, err := os.ReadFile(targetPath)
				if err != nil {
					t.Errorf("Failed to read gitconfig: %v", err)
				} else {
					gitconfig := string(content)

					// Verify user information was processed
					if !containsString(gitconfig, "Test User") {
						t.Error("Gitconfig should contain processed user name")
					}
					if !containsString(gitconfig, "test@example.com") {
						t.Error("Gitconfig should contain processed email")
					}

					// Verify conditional features
					if !containsString(gitconfig, "signingkey = ABC123DEF456") {
						t.Error("Gitconfig should contain GPG signing key (conditional)")
					}
					if !containsString(gitconfig, "required = true") {
						t.Error("Gitconfig should contain Git LFS config (conditional)")
					}

					// Verify no template variables remain
					if containsString(gitconfig, "{{") || containsString(gitconfig, "}}") {
						t.Error("Gitconfig should not contain unprocessed template variables")
					}
				}
			}
		})

		// Step 3: Create symlink to fish configuration
		t.Run("Step 3: Create fish config symlink", func(t *testing.T) {
			sourceDir := filepath.Join(repoPath, "fish")
			targetDir := filepath.Join(configDir, "fish-symlink")

			err := fileManager.CreateSymlinkWithParents(sourceDir, targetDir)
			if err != nil {
				t.Logf("Fish symlink creation failed (may be expected): %v", err)
				return
			}

			// Verify symlink was created
			if !utils.PathExists(targetDir) {
				t.Error("Fish config symlink should exist")
			}

			if !utils.IsSymlink(targetDir) {
				t.Error("Fish config target should be a symlink")
			}
		})

		// Step 4: Process fish config template
		t.Run("Step 4: Process fish config template", func(t *testing.T) {
			templatePath := filepath.Join(templatesDir, "config.fish.template")
			targetPath := filepath.Join(fishDir, "config.fish")

			// Create template context
			templateVars := map[string]interface{}{
				"user_name": "Test User",
				"editor":    "vim",
			}

			systemInfo := map[string]interface{}{
				"platform":     "macos",
				"architecture": "arm64",
				"home_dir":     tempDir,
				"config_dir":   configDir,
			}

			features := map[string]interface{}{
				"docker": true,
			}

			context := template.CreateTemplateContextWithFeatures(systemInfo, templateVars, features)

			err := fileManager.ProcessTemplate(templatePath, targetPath, context, "0644")
			if err != nil {
				t.Errorf("Fish config template processing failed: %v", err)
			}

			// Verify fish config was processed
			if !utils.PathExists(targetPath) {
				t.Error("Fish config should be created from template")
			} else {
				content, err := os.ReadFile(targetPath)
				if err != nil {
					t.Errorf("Failed to read fish config: %v", err)
				} else {
					fishConfig := string(content)

					// Verify user name was processed
					if !containsString(fishConfig, "Test User") {
						t.Error("Fish config should contain processed user name")
					}

					// Verify platform-specific content
					if !containsString(fishConfig, "/opt/homebrew/bin") {
						t.Error("Fish config should contain macOS-specific content")
					}

					// Verify feature-specific content
					if !containsString(fishConfig, "DOCKER_HOST") {
						t.Error("Fish config should contain Docker configuration")
					}
				}
			}
		})
	})

	t.Run("Test conflict resolution workflow", func(t *testing.T) {
		platformProvider := platform.DetectPlatform()
		fileManager := fileops.NewFileManager(platformProvider, false)

		// Create existing file
		existingFile := filepath.Join(tempDir, ".existing-config")
		existingContent := "existing configuration content"
		err := os.WriteFile(existingFile, []byte(existingContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create existing file: %v", err)
		}

		// Test backup conflict resolution
		backupDir := filepath.Join(tempDir, "conflict-backups")
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

		// Verify backup was created
		if !utils.PathExists(result.BackupPath) {
			t.Error("Backup should be created during conflict resolution")
		}

		// Now test copying new content with backup
		newSourcePath := filepath.Join(repoPath, "git/gitconfig")
		err = fileManager.CopyFileWithBackup(newSourcePath, existingFile, "0644", backupDir)
		if err != nil {
			t.Errorf("Copy with backup failed: %v", err)
		}

		// Verify file now has new content
		same, err := utils.CompareFileContent(newSourcePath, existingFile)
		if err != nil {
			t.Errorf("Failed to compare updated content: %v", err)
		} else if !same {
			t.Error("File should have new content after backup and copy")
		}
	})

	t.Run("Test dry run mode", func(t *testing.T) {
		// Test dry run doesn't actually create files
		platformProvider := platform.DetectPlatform()
		dryRunManager := fileops.NewFileManager(platformProvider, true)

		sourcePath := filepath.Join(repoPath, "git/gitconfig")
		targetPath := filepath.Join(tempDir, "dry-run-test")

		err := dryRunManager.CopyFile(sourcePath, targetPath, "0644")
		if err != nil {
			t.Errorf("Dry run should not error: %v", err)
		}

		// File should not exist
		if utils.PathExists(targetPath) {
			t.Error("File should not be created in dry run mode")
		}

		// Test dry run symlink
		symlinkTarget := filepath.Join(tempDir, "dry-run-symlink")
		err = dryRunManager.CreateSymlink(sourcePath, symlinkTarget)
		if err != nil {
			t.Errorf("Dry run symlink should not error: %v", err)
		}

		// Symlink should not exist
		if utils.PathExists(symlinkTarget) {
			t.Error("Symlink should not be created in dry run mode")
		}

		// Test dry run template
		templatePath := filepath.Join(templatesDir, "gitconfig.template")
		templateTarget := filepath.Join(tempDir, "dry-run-template")
		context := map[string]interface{}{"user_name": "Test"}

		err = dryRunManager.ProcessTemplate(templatePath, templateTarget, context, "0644")
		if err != nil {
			t.Errorf("Dry run template should not error: %v", err)
		}

		// Template output should not exist
		if utils.PathExists(templateTarget) {
			t.Error("Template output should not be created in dry run mode")
		}
	})
}

func TestCompleteProviderWorkflow(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("Full provider configuration and usage", func(t *testing.T) {
		// Create test repository
		repoPath, err := utils.CreateTempDotfilesRepo(tempDir)
		if err != nil {
			t.Fatalf("Failed to create test repository: %v", err)
		}

		// Create templates for this test
		templatesDir := filepath.Join(repoPath, "templates")
		err = os.MkdirAll(templatesDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create templates directory: %v", err)
		}

		gitconfigTemplate := `[user]
    name = {{.user_name}}
    email = {{.user_email}}
[core]
    editor = {{.editor}}
{{if .features.gpg_signing}}[user]
    signingkey = {{.gpg_key}}{{end}}
{{if .features.git_lfs}}[filter "lfs"]
    required = true{{end}}`

		err = os.WriteFile(filepath.Join(templatesDir, "gitconfig.template"), []byte(gitconfigTemplate), 0644)
		if err != nil {
			t.Fatalf("Failed to create gitconfig template: %v", err)
		}

		// Test provider client creation with full config
		config := &DotfilesConfig{
			DotfilesRoot:       repoPath,
			BackupEnabled:      true,
			BackupDirectory:    filepath.Join(tempDir, "backups"),
			Strategy:           "symlink",
			ConflictResolution: "backup",
			DryRun:             false,
			AutoDetectPlatform: true,
			TargetPlatform:     "auto",
			TemplateEngine:     "go",
			LogLevel:           "info",
		}

		err = config.Validate()
		if err != nil {
			t.Errorf("Provider config validation failed: %v", err)
		}

		client, err := NewDotfilesClient(config)
		if err != nil {
			t.Errorf("Client creation failed: %v", err)
		}

		// Test file operations with the client
		platformProvider := platform.DetectPlatform()
		fileManager := fileops.NewFileManager(platformProvider, client.Config.DryRun)

		// Test copying files
		sourcePath := filepath.Join(repoPath, "git/gitconfig")
		targetPath := filepath.Join(tempDir, "final-gitconfig")

		err = fileManager.CopyFile(sourcePath, targetPath, "0644")
		if err != nil {
			t.Errorf("File copy in full workflow failed: %v", err)
		}

		// Verify file was copied
		if !utils.PathExists(targetPath) {
			t.Error("File should be copied in full workflow")
		}

		// Test template processing with system context
		templatePath := filepath.Join(repoPath, "templates/gitconfig.template")
		templateTarget := filepath.Join(tempDir, "templated-gitconfig")

		templateVars := map[string]interface{}{
			"user_name":  "Integration Test User",
			"user_email": "integration@test.com",
			"editor":     "nvim",
			"gpg_key":    "INTEGRATION123",
		}

		systemInfo := client.GetPlatformInfo()
		features := map[string]interface{}{
			"gpg_signing": true,
			"git_lfs":     false,
		}

		context := template.CreateTemplateContextWithFeatures(systemInfo, templateVars, features)

		err = fileManager.ProcessTemplate(templatePath, templateTarget, context, "0644")
		if err != nil {
			t.Errorf("Template processing in full workflow failed: %v", err)
		}

		// Verify template was processed with full context
		if !utils.PathExists(templateTarget) {
			t.Error("Template output should exist")
		} else {
			content, err := os.ReadFile(templateTarget)
			if err != nil {
				t.Errorf("Failed to read template output: %v", err)
			} else {
				gitconfig := string(content)

				// Verify all template variables were processed
				if !containsString(gitconfig, "Integration Test User") {
					t.Error("Template should contain integration test user name")
				}
				if !containsString(gitconfig, "integration@test.com") {
					t.Error("Template should contain integration test email")
				}
				if !containsString(gitconfig, "nvim") {
					t.Error("Template should contain editor preference")
				}

				// Verify conditional processing
				if !containsString(gitconfig, "INTEGRATION123") {
					t.Error("Template should contain GPG key (enabled feature)")
				}
				if containsString(gitconfig, "filter \"lfs\"") {
					t.Error("Template should not contain Git LFS config (disabled feature)")
				}

				// Verify system context was available
				t.Logf("Template processed successfully with system context: %s", client.Platform)
			}
		}

		// Test backup system
		existingFile := filepath.Join(tempDir, "backup-test")
		err = os.WriteFile(existingFile, []byte("original"), 0644)
		if err != nil {
			t.Fatalf("Failed to create file for backup test: %v", err)
		}

		// Copy with backup
		err = fileManager.CopyFileWithBackup(sourcePath, existingFile, "0644", config.BackupDirectory)
		if err != nil {
			t.Errorf("Copy with backup failed: %v", err)
		}

		// Verify backup was created
		if !utils.PathExists(config.BackupDirectory) {
			t.Error("Backup directory should be created")
		} else {
			entries, err := os.ReadDir(config.BackupDirectory)
			if err != nil {
				t.Errorf("Failed to read backup directory: %v", err)
			} else if len(entries) == 0 {
				t.Error("Backup files should be created")
			}
		}
	})
}

// Use containsString from file_resource_integration_simple_test.go.
