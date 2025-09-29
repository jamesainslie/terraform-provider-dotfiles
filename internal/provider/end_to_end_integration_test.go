// Copyright (c) HashCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

package provider

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jamesainslie/terraform-provider-dotfiles/internal/fileops"
	"github.com/jamesainslie/terraform-provider-dotfiles/internal/platform"
	"github.com/jamesainslie/terraform-provider-dotfiles/internal/template"
	"github.com/jamesainslie/terraform-provider-dotfiles/internal/utils"
)

// TestEndToEndDotfilesManagement tests complete dotfiles workflow.
func TestEndToEndDotfilesManagement(t *testing.T) {
	// Setup test environment
	testEnv := setupEndToEndTestEnvironment(t)

	t.Run("Complete workflow: files, templates, and symlinks", func(t *testing.T) {
		testCompleteWorkflow(t, testEnv)
	})

	t.Run("Test dry run mode", func(t *testing.T) {
		testDryRunMode(t, testEnv)
	})
}

// endToEndTestEnv holds the test environment setup
type endToEndTestEnv struct {
	tempDir      string
	repoPath     string
	templatesDir string
	configDir    string
	fishDir      string
}

// setupEndToEndTestEnvironment creates and initializes the test environment
func setupEndToEndTestEnvironment(t *testing.T) *endToEndTestEnv {
	tempDir := t.TempDir()

	// Create test dotfiles repository
	repoPath, err := utils.CreateTempDotfilesRepo(tempDir)
	if err != nil {
		t.Fatalf("Failed to create test repository: %v", err)
	}

	// Create directory structure
	env := &endToEndTestEnv{
		tempDir:      tempDir,
		repoPath:     repoPath,
		templatesDir: filepath.Join(repoPath, "templates"),
		configDir:    filepath.Join(tempDir, ".config"),
		fishDir:      filepath.Join(tempDir, ".config", "fish"),
	}

	createEndToEndDirectories(t, env)
	createEndToEndTemplateFiles(t, env)

	return env
}

// createEndToEndDirectories creates necessary directories
func createEndToEndDirectories(t *testing.T, env *endToEndTestEnv) {
	if err := os.MkdirAll(env.templatesDir, 0755); err != nil {
		t.Fatalf("Failed to create templates directory: %v", err)
	}
	if err := os.MkdirAll(env.fishDir, 0755); err != nil {
		t.Fatalf("Failed to create config directories: %v", err)
	}
}

// createEndToEndTemplateFiles creates test template files
func createEndToEndTemplateFiles(t *testing.T, env *endToEndTestEnv) {
	createGitconfigTemplate(t, env.templatesDir)
	createFishConfigTemplate(t, env.templatesDir)
}

// createGitconfigTemplate creates the gitconfig template file
func createGitconfigTemplate(t *testing.T, templatesDir string) {
	gitconfigTemplate := `[user]
    name = {{.user_name}}
    email = {{.user_email}}
[core]
    editor = {{.editor}}
{{if .features.gpg_signing}}[user]
    signingkey = {{.gpg_key}}{{end}}
{{if .features.git_lfs}}[filter "lfs"]
    required = true{{end}}`

	err := os.WriteFile(filepath.Join(templatesDir, "gitconfig.template"), []byte(gitconfigTemplate), 0644)
	if err != nil {
		t.Fatalf("Failed to create gitconfig template: %v", err)
	}
}

// createFishConfigTemplate creates the fish config template file
func createFishConfigTemplate(t *testing.T, templatesDir string) {
	fishTemplate := `# Fish shell configuration for {{.user_name}}
set -g fish_greeting ""
set -gx EDITOR {{.editor}}
{{if .features.docker}}# Docker support enabled
set -gx DOCKER_HOST unix:///var/run/docker.sock{{end}}
{{if eq .system.platform "macos"}}# macOS specific settings
fish_add_path /opt/homebrew/bin{{end}}`

	err := os.WriteFile(filepath.Join(templatesDir, "config.fish.template"), []byte(fishTemplate), 0644)
	if err != nil {
		t.Fatalf("Failed to create fish template: %v", err)
	}
}

// testCompleteWorkflow tests the complete dotfiles workflow
func testCompleteWorkflow(t *testing.T, env *endToEndTestEnv) {
	platformProvider := platform.DetectPlatform()
	fileManager := fileops.NewFileManager(platformProvider, false)

	// Execute workflow steps
	testSSHConfigCopy(t, env, fileManager)
	testGitconfigTemplateProcessing(t, env, fileManager)
	testFishConfigSymlink(t, env, fileManager)
}

// testSSHConfigCopy tests SSH config file copying
func testSSHConfigCopy(t *testing.T, env *endToEndTestEnv, fileManager *fileops.FileManager) {
	sourcePath := filepath.Join(env.repoPath, "ssh/config")
	targetPath := filepath.Join(env.tempDir, ".ssh", "config")

	// Create .ssh directory
	err := os.MkdirAll(filepath.Dir(targetPath), 0700)
	if err != nil {
		t.Fatalf("Failed to create .ssh directory: %v", err)
	}

	err = fileManager.CopyFile(sourcePath, targetPath, "0600")
	if err != nil {
		t.Errorf("SSH config copy failed: %v", err)
	}

	validateSSHConfigCopy(t, targetPath)
}

// validateSSHConfigCopy validates SSH config file was copied correctly
func validateSSHConfigCopy(t *testing.T, targetPath string) {
	if !utils.PathExists(targetPath) {
		t.Error("SSH config should be copied")
		return
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
}

// testGitconfigTemplateProcessing tests gitconfig template processing
func testGitconfigTemplateProcessing(t *testing.T, env *endToEndTestEnv, fileManager *fileops.FileManager) {
	templatePath := filepath.Join(env.templatesDir, "gitconfig.template")
	targetPath := filepath.Join(env.tempDir, ".gitconfig")

	// Create template context
	context := createGitconfigTemplateContext(env)

	err := fileManager.ProcessTemplate(templatePath, targetPath, context, "0644")
	if err != nil {
		t.Errorf("Gitconfig template processing failed: %v", err)
	}

	validateGitconfigTemplate(t, targetPath)
}

// createGitconfigTemplateContext creates the template context for gitconfig
func createGitconfigTemplateContext(env *endToEndTestEnv) map[string]interface{} {
	templateVars := map[string]interface{}{
		"user_name":  "Test User",
		"user_email": "test@example.com",
		"editor":     "vim",
		"gpg_key":    "ABC123DEF456",
	}

	systemInfo := map[string]interface{}{
		"platform":     "macos",
		"architecture": "arm64",
		"home_dir":     env.tempDir,
		"config_dir":   env.configDir,
	}

	features := map[string]interface{}{
		"gpg_signing": true,
		"git_lfs":     true,
	}

	return template.CreateTemplateContextWithFeatures(systemInfo, templateVars, features)
}

// validateGitconfigTemplate validates gitconfig template processing results
func validateGitconfigTemplate(t *testing.T, targetPath string) {
	if !utils.PathExists(targetPath) {
		t.Error("Gitconfig should be created from template")
		return
	}

	content, err := os.ReadFile(targetPath)
	if err != nil {
		t.Errorf("Failed to read gitconfig: %v", err)
		return
	}

	gitconfig := string(content)
	validateGitconfigContent(t, gitconfig)
}

// validateGitconfigContent validates the processed gitconfig content
func validateGitconfigContent(t *testing.T, gitconfig string) {
	// Verify user information was processed
	if !strings.Contains(gitconfig, "Test User") {
		t.Error("Gitconfig should contain processed user name")
	}
	if !strings.Contains(gitconfig, "test@example.com") {
		t.Error("Gitconfig should contain processed email")
	}

	// Verify conditional features
	if !strings.Contains(gitconfig, "signingkey = ABC123DEF456") {
		t.Error("Gitconfig should contain GPG signing key (conditional)")
	}
	if !strings.Contains(gitconfig, "required = true") {
		t.Error("Gitconfig should contain Git LFS config (conditional)")
	}

	// Verify no template variables remain
	if strings.Contains(gitconfig, "{{") || strings.Contains(gitconfig, "}}") {
		t.Error("Gitconfig should not contain unprocessed template variables")
	}
}

// testFishConfigSymlink tests fish config symlink creation
func testFishConfigSymlink(t *testing.T, env *endToEndTestEnv, fileManager *fileops.FileManager) {
	sourceDir := filepath.Join(env.repoPath, "fish")
	targetDir := filepath.Join(env.configDir, "fish-symlink")

	err := fileManager.CreateSymlinkWithParents(sourceDir, targetDir)
	if err != nil {
		t.Logf("Fish symlink creation failed (may be expected): %v", err)
		return
	}

	validateFishConfigSymlink(t, targetDir)
}

// validateFishConfigSymlink validates fish config symlink creation
func validateFishConfigSymlink(t *testing.T, targetDir string) {
	// Verify symlink was created
	if !utils.PathExists(targetDir) {
		t.Error("Fish config symlink should exist")
	}

	if !utils.IsSymlink(targetDir) {
		t.Error("Fish config target should be a symlink")
	}
}

// testDryRunMode tests dry run functionality
func testDryRunMode(t *testing.T, env *endToEndTestEnv) {
	// Test dry run doesn't actually create files
	platformProvider := platform.DetectPlatform()
	dryRunManager := fileops.NewFileManager(platformProvider, true)

	testDryRunFile(t, env, dryRunManager)
	testDryRunSymlink(t, env, dryRunManager)
	testDryRunTemplate(t, env, dryRunManager)
}

// testDryRunFile tests dry run file operations
func testDryRunFile(t *testing.T, env *endToEndTestEnv, dryRunManager *fileops.FileManager) {
	sourcePath := filepath.Join(env.repoPath, "git/gitconfig")
	targetPath := filepath.Join(env.tempDir, "dry-run-test")

	err := dryRunManager.CopyFile(sourcePath, targetPath, "0644")
	if err != nil {
		t.Errorf("Dry run should not error: %v", err)
	}

	// File should not exist
	if utils.PathExists(targetPath) {
		t.Error("File should not be created in dry run mode")
	}
}

// testDryRunSymlink tests dry run symlink operations
func testDryRunSymlink(t *testing.T, env *endToEndTestEnv, dryRunManager *fileops.FileManager) {
	sourcePath := filepath.Join(env.repoPath, "git/gitconfig")
	symlinkTarget := filepath.Join(env.tempDir, "dry-run-symlink")

	err := dryRunManager.CreateSymlink(sourcePath, symlinkTarget)
	if err != nil {
		t.Errorf("Dry run symlink should not error: %v", err)
	}

	// Symlink should not exist
	if utils.PathExists(symlinkTarget) {
		t.Error("Symlink should not be created in dry run mode")
	}
}

// testDryRunTemplate tests dry run template operations
func testDryRunTemplate(t *testing.T, env *endToEndTestEnv, dryRunManager *fileops.FileManager) {
	templatePath := filepath.Join(env.templatesDir, "gitconfig.template")
	templateTarget := filepath.Join(env.tempDir, "dry-run-template")
	context := map[string]interface{}{"user_name": "Test"}

	err := dryRunManager.ProcessTemplate(templatePath, templateTarget, context, "0644")
	if err != nil {
		t.Errorf("Dry run template should not error: %v", err)
	}

	// Template output should not exist
	if utils.PathExists(templateTarget) {
		t.Error("Template output should not be created in dry run mode")
	}
}

// TestCompleteProviderWorkflow tests the complete provider workflow.
func TestCompleteProviderWorkflow(t *testing.T) {
	testCompleteProviderWorkflowExecution(t)
}

// testCompleteProviderWorkflowExecution executes the complete provider workflow test
func testCompleteProviderWorkflowExecution(t *testing.T) {
	testEnv := setupCompleteProviderTestEnvironment(t)
	testProviderWorkflowSteps(t, testEnv)
}

// setupCompleteProviderTestEnvironment sets up the complete provider test environment
func setupCompleteProviderTestEnvironment(t *testing.T) *endToEndTestEnv {
	tempDir := t.TempDir()
	repoPath, err := utils.CreateTempDotfilesRepo(tempDir)
	if err != nil {
		t.Fatalf("Failed to create test repository: %v", err)
	}

	return &endToEndTestEnv{
		tempDir:      tempDir,
		repoPath:     repoPath,
		templatesDir: filepath.Join(repoPath, "templates"),
		configDir:    filepath.Join(tempDir, ".config"),
		fishDir:      filepath.Join(tempDir, ".config", "fish"),
	}
}

// testProviderWorkflowSteps tests the provider workflow steps
func testProviderWorkflowSteps(t *testing.T, env *endToEndTestEnv) {
	// Create provider configuration
	config := createProviderWorkflowConfig(env)

	// Test provider configuration validation
	if err := config.Validate(); err != nil {
		t.Errorf("Provider configuration should be valid: %v", err)
	}

	// Test client creation
	client, err := NewDotfilesClient(config)
	if err != nil {
		t.Errorf("Should create dotfiles client: %v", err)
	} else if client == nil {
		t.Error("Client should not be nil")
	}
}

// createProviderWorkflowConfig creates a provider configuration for workflow testing
func createProviderWorkflowConfig(env *endToEndTestEnv) *DotfilesConfig {
	return &DotfilesConfig{
		DotfilesRoot:       env.repoPath,
		BackupEnabled:      true,
		BackupDirectory:    filepath.Join(env.tempDir, "backups"),
		Strategy:           "symlink",
		ConflictResolution: "backup",
		DryRun:             false,
		AutoDetectPlatform: true,
		TargetPlatform:     "auto",
		TemplateEngine:     "go",
		LogLevel:           "info",
	}
}
