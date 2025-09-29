// Copyright (c) HashCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

package provider

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/jamesainslie/terraform-provider-dotfiles/internal/utils"
)

func TestRepositoryResource(t *testing.T) {
	t.Run("NewRepositoryResource", func(t *testing.T) {
		testNewRepositoryResource(t)
	})

	t.Run("Metadata", func(t *testing.T) {
		testRepositoryResourceMetadata(t)
	})

	t.Run("Schema", func(t *testing.T) {
		testRepositoryResourceSchema(t)
	})

	t.Run("Configure", func(t *testing.T) {
		testRepositoryResourceConfigure(t)
	})

	t.Run("BuildAuthConfig", func(t *testing.T) {
		testRepositoryResourceBuildAuthConfig(t)
	})
}

// testNewRepositoryResource tests NewRepositoryResource creation
func testNewRepositoryResource(t *testing.T) {
	r := NewRepositoryResource()
	if r == nil {
		t.Fatal("NewRepositoryResource() returned nil")
	}

	// RepositoryResource should implement the Resource interface
	if r == nil {
		t.Error("RepositoryResource should not be nil")
	}
}

// testRepositoryResourceMetadata tests repository resource metadata
func testRepositoryResourceMetadata(t *testing.T) {
	r := NewRepositoryResource()
	ctx := context.Background()

	req := resource.MetadataRequest{
		ProviderTypeName: "dotfiles",
	}
	resp := &resource.MetadataResponse{}

	r.Metadata(ctx, req, resp)

	expectedTypeName := "dotfiles_repository"
	if resp.TypeName != expectedTypeName {
		t.Errorf("Expected TypeName %s, got %s", expectedTypeName, resp.TypeName)
	}
}

// testRepositoryResourceSchema tests repository resource schema
func testRepositoryResourceSchema(t *testing.T) {
	r := NewRepositoryResource()
	ctx := context.Background()

	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}

	r.Schema(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("Schema validation failed: %v", resp.Diagnostics)
	}

	validateRepositoryResourceSchemaAttributes(t, resp.Schema)
}

// validateRepositoryResourceSchemaAttributes validates schema attributes
func validateRepositoryResourceSchemaAttributes(t *testing.T, schema schema.Schema) {
	validateRepositoryRequiredAttributes(t, schema)
	validateRepositoryOptionalAttributes(t, schema)
	validateRepositoryComputedAttributes(t, schema)
	validateRepositorySensitiveAttributes(t, schema)
}

// validateRepositoryRequiredAttributes validates required schema attributes
func validateRepositoryRequiredAttributes(t *testing.T, schema schema.Schema) {
	requiredAttrs := []string{"id", "name", "source_path"}
	for _, attr := range requiredAttrs {
		if _, exists := schema.Attributes[attr]; !exists {
			t.Errorf("Required attribute %s not found in schema", attr)
		}
	}
}

// validateRepositoryOptionalAttributes validates optional schema attributes
func validateRepositoryOptionalAttributes(t *testing.T, schema schema.Schema) {
	optionalAttrs := []string{
		"description", "default_backup_enabled", "default_file_mode", "default_dir_mode",
		"git_branch", "git_personal_access_token", "git_username", "git_ssh_private_key_path",
		"git_ssh_passphrase", "git_update_interval",
	}
	for _, attr := range optionalAttrs {
		if _, exists := schema.Attributes[attr]; !exists {
			t.Errorf("Optional attribute %s not found in schema", attr)
		}
	}
}

// validateRepositoryComputedAttributes validates computed schema attributes
func validateRepositoryComputedAttributes(t *testing.T, schema schema.Schema) {
	computedAttrs := []string{"local_path", "last_commit", "last_update"}
	for _, attr := range computedAttrs {
		if _, exists := schema.Attributes[attr]; !exists {
			t.Errorf("Computed attribute %s not found in schema", attr)
		}
	}
}

// validateRepositorySensitiveAttributes validates sensitive schema attributes
func validateRepositorySensitiveAttributes(t *testing.T, schema schema.Schema) {
	sensitiveAttrs := []string{"git_personal_access_token", "git_ssh_passphrase"}
	for _, attr := range sensitiveAttrs {
		if schemaAttr, exists := schema.Attributes[attr]; exists {
			if stringAttr, ok := schemaAttr.(interface{ GetSensitive() bool }); ok {
				if !stringAttr.GetSensitive() {
					t.Errorf("Attribute %s should be marked as sensitive", attr)
				}
			}
		}
	}
}

// testRepositoryResourceConfigure tests repository resource configuration
func testRepositoryResourceConfigure(t *testing.T) {
	r := NewRepositoryResource()
	repoResource, ok := r.(*RepositoryResource)
	if !ok {
		t.Fatal("NewRepositoryResource() did not return *RepositoryResource")
	}

	ctx := context.Background()

	testRepositoryConfigureWithNilProvider(t, ctx, repoResource)
	testRepositoryConfigureWithValidClient(t, ctx, repoResource)
	testRepositoryConfigureWithInvalidProvider(t, ctx, repoResource)
}

// testRepositoryConfigureWithNilProvider tests configure with nil provider
func testRepositoryConfigureWithNilProvider(t *testing.T, ctx context.Context, repoResource *RepositoryResource) {
	req := resource.ConfigureRequest{
		ProviderData: nil,
	}
	resp := &resource.ConfigureResponse{}

	repoResource.Configure(ctx, req, resp)

	// Should not error with nil provider data
	if resp.Diagnostics.HasError() {
		t.Errorf("Configure with nil provider data should not error: %v", resp.Diagnostics)
	}
}

// testRepositoryConfigureWithValidClient tests configure with valid client
func testRepositoryConfigureWithValidClient(t *testing.T, ctx context.Context, repoResource *RepositoryResource) {
	client := &DotfilesClient{
		Platform:     "test",
		Architecture: "test",
		HomeDir:      "/tmp",
		ConfigDir:    "/tmp/.config",
	}

	req := resource.ConfigureRequest{
		ProviderData: client,
	}
	resp := &resource.ConfigureResponse{}

	repoResource.Configure(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("Configure with valid client failed: %v", resp.Diagnostics)
	}
}

// testRepositoryConfigureWithInvalidProvider tests configure with invalid provider
func testRepositoryConfigureWithInvalidProvider(t *testing.T, ctx context.Context, repoResource *RepositoryResource) {
	req := resource.ConfigureRequest{
		ProviderData: "invalid",
	}
	resp := &resource.ConfigureResponse{}

	repoResource.Configure(ctx, req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("Configure with invalid provider data should error")
	}
}

// testRepositoryResourceBuildAuthConfig tests build auth config functionality
func testRepositoryResourceBuildAuthConfig(t *testing.T) {
	r := &RepositoryResource{}

	testBuildAuthConfigWithPAT(t, r)
	testBuildAuthConfigWithSSH(t, r)
	testBuildAuthConfigWithNullValues(t, r)
}

// testBuildAuthConfigWithPAT tests build auth config with Personal Access Token
func testBuildAuthConfigWithPAT(t *testing.T, r *RepositoryResource) {
	data := &RepositoryResourceModel{
		GitPersonalAccessToken: types.StringValue("ghp_test_token"),
		GitUsername:            types.StringValue("testuser"),
	}

	authConfig := r.buildAuthConfig(data)

	if authConfig.PersonalAccessToken != "ghp_test_token" {
		t.Errorf("Expected PAT 'ghp_test_token', got '%s'", authConfig.PersonalAccessToken)
	}
	if authConfig.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", authConfig.Username)
	}
}

// testBuildAuthConfigWithSSH tests build auth config with SSH
func testBuildAuthConfigWithSSH(t *testing.T, r *RepositoryResource) {
	data := &RepositoryResourceModel{
		GitSSHPrivateKeyPath: types.StringValue("/path/to/key"),
		GitSSHPassphrase:     types.StringValue("passphrase"),
	}

	authConfig := r.buildAuthConfig(data)

	if authConfig.SSHPrivateKeyPath != "/path/to/key" {
		t.Errorf("Expected SSH key path '/path/to/key', got '%s'", authConfig.SSHPrivateKeyPath)
	}
	if authConfig.SSHPassphrase != "passphrase" {
		t.Errorf("Expected SSH passphrase 'passphrase', got '%s'", authConfig.SSHPassphrase)
	}
}

// testBuildAuthConfigWithNullValues tests build auth config with null values
func testBuildAuthConfigWithNullValues(t *testing.T, r *RepositoryResource) {
	data := &RepositoryResourceModel{
		GitPersonalAccessToken: types.StringNull(),
		GitUsername:            types.StringNull(),
	}

	authConfig := r.buildAuthConfig(data)

	if authConfig.PersonalAccessToken != "" {
		t.Errorf("Expected empty PAT for null value, got: %q", authConfig.PersonalAccessToken)
	}
	if authConfig.Username != "" {
		t.Errorf("Expected empty username for null value, got: %q", authConfig.Username)
	}
}

func TestRepositoryResourceModel(t *testing.T) {
	// Test the resource model with various values
	model := RepositoryResourceModel{
		ID:                     types.StringValue("test-repo"),
		Name:                   types.StringValue("test-repository"),
		SourcePath:             types.StringValue("https://github.com/user/repo.git"),
		Description:            types.StringValue("Test repository"),
		DefaultBackupEnabled:   types.BoolValue(true),
		DefaultFileMode:        types.StringValue("0644"),
		DefaultDirMode:         types.StringValue("0755"),
		GitBranch:              types.StringValue("main"),
		GitPersonalAccessToken: types.StringValue("ghp_token"),
		GitUsername:            types.StringValue("user"),
		LocalPath:              types.StringValue("/tmp/cache"),
		LastCommit:             types.StringValue("abc123"),
		LastUpdate:             types.StringValue("2024-01-01T00:00:00Z"),
	}

	// Verify all fields can be accessed
	if model.ID.ValueString() != "test-repo" {
		t.Error("ID field not working correctly")
	}
	if model.Name.ValueString() != "test-repository" {
		t.Error("Name field not working correctly")
	}
	if model.SourcePath.ValueString() != "https://github.com/user/repo.git" {
		t.Error("SourcePath field not working correctly")
	}
	if !model.DefaultBackupEnabled.ValueBool() {
		t.Error("DefaultBackupEnabled field not working correctly")
	}
}

func TestRepositoryResourceLocalOperations(t *testing.T) {
	// Create temporary directory for testing
	tempDir := t.TempDir()

	// Create test dotfiles repository
	repoPath, err := utils.CreateTempDotfilesRepo(tempDir)
	if err != nil {
		t.Fatalf("Failed to create test repository: %v", err)
	}

	client := &DotfilesClient{
		Config: &DotfilesConfig{
			DotfilesRoot: repoPath,
		},
		Platform:     "test",
		Architecture: "test",
		HomeDir:      tempDir,
		ConfigDir:    filepath.Join(tempDir, ".config"),
	}

	r := &RepositoryResource{client: client}
	ctx := context.Background()

	t.Run("setupLocalRepository valid path", func(t *testing.T) {
		data := &RepositoryResourceModel{
			SourcePath: types.StringValue(repoPath),
		}

		err := r.setupLocalRepository(ctx, data)
		if err != nil {
			t.Errorf("setupLocalRepository failed for valid path: %v", err)
		}

		// Verify source path was updated to absolute path
		if data.SourcePath.ValueString() != repoPath {
			t.Errorf("Source path should be absolute: expected %s, got %s", repoPath, data.SourcePath.ValueString())
		}
	})

	t.Run("setupLocalRepository with tilde expansion", func(t *testing.T) {
		// Create a test directory in the temp dir
		testRepoDir := filepath.Join(tempDir, "tilde-test")
		err := os.MkdirAll(testRepoDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create test repo dir: %v", err)
		}

		// Use relative path from client home dir
		relPath := "tilde-test"

		data := &RepositoryResourceModel{
			SourcePath: types.StringValue("~/" + relPath),
		}

		err = r.setupLocalRepository(ctx, data)
		if err != nil {
			t.Errorf("setupLocalRepository failed for tilde path: %v", err)
		}

		expectedPath := filepath.Join(client.HomeDir, relPath)
		if data.SourcePath.ValueString() != expectedPath {
			t.Errorf("Tilde expansion failed: expected %s, got %s", expectedPath, data.SourcePath.ValueString())
		}
	})

	t.Run("setupLocalRepository non-existent path", func(t *testing.T) {
		nonExistentPath := filepath.Join(tempDir, "nonexistent")

		data := &RepositoryResourceModel{
			SourcePath: types.StringValue(nonExistentPath),
		}

		err := r.setupLocalRepository(ctx, data)
		if err != nil {
			t.Errorf("setupLocalRepository should succeed and create non-existent path: %v", err)
		}

		// Verify the directory was created
		if !utils.PathExists(nonExistentPath) {
			t.Error("setupLocalRepository should have created the non-existent directory")
		}

		// Verify it's a directory
		info, err := os.Stat(nonExistentPath)
		if err != nil {
			t.Errorf("Failed to stat created directory: %v", err)
		}
		if !info.IsDir() {
			t.Error("Created path should be a directory")
		}
	})

	t.Run("setupLocalRepository file instead of directory", func(t *testing.T) {
		// Create a file instead of directory
		testFile := filepath.Join(tempDir, "not-a-directory.txt")
		err := os.WriteFile(testFile, []byte("test"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		data := &RepositoryResourceModel{
			SourcePath: types.StringValue(testFile),
		}

		err = r.setupLocalRepository(ctx, data)
		if err == nil {
			t.Error("setupLocalRepository should fail when source is a file, not directory")
		}
	})
}
