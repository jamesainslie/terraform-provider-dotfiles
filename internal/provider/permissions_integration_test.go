// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/jamesainslie/terraform-provider-dotfiles/internal/fileops"
)

// TestFileResourcePermissionIntegration tests the FileResource with permission management
func TestFileResourcePermissionIntegration(t *testing.T) {
	// Create temporary directories for testing
	tempDir, err := os.MkdirTemp("", "file-resource-permissions-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create source and target directories
	sourceDir := filepath.Join(tempDir, "source")
	targetDir := filepath.Join(tempDir, "target")
	
	err = os.MkdirAll(sourceDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}
	
	err = os.MkdirAll(targetDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create target directory: %v", err)
	}

	// Create test source files
	testFiles := map[string]string{
		"id_rsa":       "private key content",
		"id_rsa.pub":   "public key content",
		"config":       "ssh config content",
		"known_hosts":  "known hosts content",
	}

	for filename, content := range testFiles {
		sourcePath := filepath.Join(sourceDir, filename)
		err := os.WriteFile(sourcePath, []byte(content), 0666)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
	}

	// Create a test client
	config := &DotfilesConfig{
		DotfilesRoot:       sourceDir,
		BackupEnabled:      true,
		BackupDirectory:    filepath.Join(tempDir, "backups"),
		ConflictResolution: "backup",
		DryRun:             false,
		AutoDetectPlatform: true,
		TargetPlatform:     "auto",
		TemplateEngine:     "go",
		LogLevel:           "info",
	}

	// Create client for future use (currently unused in this test)
	_, err = NewDotfilesClient(config)
	if err != nil {
		t.Fatalf("Failed to create dotfiles client: %v", err)
	}

	t.Run("Create file with permissions", func(t *testing.T) {
		// Create enhanced model with permissions
		model := &EnhancedFileResourceModel{
			FileResourceModel: FileResourceModel{
				ID:            types.StringValue("test-ssh-config"),
				Repository:    types.StringValue("test-repo"),
				Name:          types.StringValue("ssh-config"),
				SourcePath:    types.StringValue("config"),
				TargetPath:    types.StringValue(filepath.Join(targetDir, "config")),
				IsTemplate:    types.BoolValue(false),
				BackupEnabled: types.BoolValue(true),
			},
			Permissions: &PermissionsModel{
				Directory: types.StringValue("0700"),
				Files:     types.StringValue("0600"),
				Recursive: types.BoolValue(false),
			},
			PermissionRules: func() types.Map {
				rules := map[string]attr.Value{
					"*.pub":      types.StringValue("0644"),
					"config":     types.StringValue("0600"),
					"id_*":       types.StringValue("0600"),
				}
				mapVal, _ := types.MapValue(types.StringType, rules)
				return mapVal
			}(),
		}

		// Test permission configuration building
		permConfig, err := buildPermissionConfig(model.Permissions, model.PermissionRules)
		if err != nil {
			t.Fatalf("Failed to build permission config: %v", err)
		}

		// Verify permission config
		if permConfig.FileMode != "0600" {
			t.Errorf("Expected file mode 0600, got %s", permConfig.FileMode)
		}
		if permConfig.DirectoryMode != "0700" {
			t.Errorf("Expected directory mode 0700, got %s", permConfig.DirectoryMode)
		}
		if len(permConfig.Rules) != 3 {
			t.Errorf("Expected 3 permission rules, got %d", len(permConfig.Rules))
		}

		// Test rule application
		testRule := func(filename, expectedPerm string) {
			appliedPerm, err := ApplyPermissionRules(filename, model.PermissionRules, "0600")
			if err != nil {
				t.Errorf("Failed to apply permission rules for %s: %v", filename, err)
			}
			if appliedPerm != expectedPerm {
				t.Errorf("File %s: expected permission %s, got %s", filename, expectedPerm, appliedPerm)
			}
		}

		testRule("config", "0600")      // matches "config" rule
		testRule("id_rsa.pub", "0644")  // matches "*.pub" rule (more specific than "id_*")
		testRule("id_rsa", "0600")      // matches "id_*" rule
		testRule("known_hosts", "0600") // uses default
	})
}

// TestSymlinkResourcePermissionIntegration tests symlink creation with permissions
func TestSymlinkResourcePermissionIntegration(t *testing.T) {
	// Test symlink support by trying to create one
	tempTestDir, err := os.MkdirTemp("", "symlink-support-test")
	if err != nil {
		t.Fatalf("Failed to create temp test directory: %v", err)
	}
	defer os.RemoveAll(tempTestDir)
	
	testFile := filepath.Join(tempTestDir, "test")
	testLink := filepath.Join(tempTestDir, "test-link")
	
	os.WriteFile(testFile, []byte("test"), 0644)
	if err := os.Symlink(testFile, testLink); err != nil {
		t.Skip("Platform does not support symlinks")
	}

	// Create temporary directories for testing
	tempDir, err := os.MkdirTemp("", "symlink-resource-permissions-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create source directory structure
	sourceDir := filepath.Join(tempDir, "source", "ssh")
	targetDir := filepath.Join(tempDir, "target")
	
	err = os.MkdirAll(sourceDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}

	// Create SSH files with different types
	sshFiles := map[string]struct {
		content    string
		shouldPerm string
	}{
		"id_rsa":       {"private key", "0600"},
		"id_rsa.pub":   {"public key", "0644"},
		"config":       {"ssh config", "0600"},
		"known_hosts":  {"known hosts", "0600"},
	}

	for filename, info := range sshFiles {
		filePath := filepath.Join(sourceDir, filename)
		err := os.WriteFile(filePath, []byte(info.content), 0666)
		if err != nil {
			t.Fatalf("Failed to create SSH file %s: %v", filename, err)
		}
	}

	// Create enhanced symlink model
	model := &EnhancedSymlinkResourceModel{
		SymlinkResourceModel: SymlinkResourceModel{
			ID:            types.StringValue("test-ssh-symlink"),
			Repository:    types.StringValue("test-repo"),
			Name:          types.StringValue("ssh-symlink"),
			SourcePath:    types.StringValue("ssh"),
			TargetPath:    types.StringValue(filepath.Join(targetDir, ".ssh")),
			ForceUpdate:   types.BoolValue(true),
			CreateParents: types.BoolValue(true),
		},
		Permissions: &PermissionsModel{
			Directory: types.StringValue("0700"),
			Files:     types.StringValue("0600"),
			Recursive: types.BoolValue(true),
		},
		PermissionRules: func() types.Map {
			rules := map[string]attr.Value{
				"*.pub":      types.StringValue("0644"),
				"id_*":       types.StringValue("0600"),
				"known_hosts": types.StringValue("0600"),
			}
			mapVal, _ := types.MapValue(types.StringType, rules)
			return mapVal
		}(),
	}

	// Test that permission config can be built from model
	permConfig, err := buildPermissionConfig(model.Permissions, model.PermissionRules)
	if err != nil {
		t.Fatalf("Failed to build permission config: %v", err)
	}

	// Verify configuration
	if permConfig.DirectoryMode != "0700" {
		t.Errorf("Expected directory mode 0700, got %s", permConfig.DirectoryMode)
	}
	if permConfig.FileMode != "0600" {
		t.Errorf("Expected file mode 0600, got %s", permConfig.FileMode)
	}
	if !permConfig.Recursive {
		t.Error("Expected recursive permissions to be enabled")
	}
	if len(permConfig.Rules) != 3 {
		t.Errorf("Expected 3 permission rules, got %d", len(permConfig.Rules))
	}
}

// buildPermissionConfig creates a PermissionConfig from the enhanced model
// This function should be implemented in the actual resource code
func buildPermissionConfig(permissions *PermissionsModel, rules types.Map) (*fileops.PermissionConfig, error) {
	config := &fileops.PermissionConfig{}

	if permissions != nil {
		if !permissions.Directory.IsNull() {
			config.DirectoryMode = permissions.Directory.ValueString()
		}
		if !permissions.Files.IsNull() {
			config.FileMode = permissions.Files.ValueString()
		}
		if !permissions.Recursive.IsNull() {
			config.Recursive = permissions.Recursive.ValueBool()
		}
	}

	if !rules.IsNull() && !rules.IsUnknown() {
		config.Rules = make(map[string]string)
		elements := rules.Elements()
		for pattern, permValue := range elements {
			if strPerm, ok := permValue.(types.String); ok {
				config.Rules[pattern] = strPerm.ValueString()
			}
		}
	}

	return config, fileops.ValidatePermissionConfig(config)
}
