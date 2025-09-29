// Copyright (c) HashCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

package provider

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

func TestProvider(t *testing.T) {
	ctx := context.Background()

	// Create a new provider instance
	p := New("test")()

	// Test provider metadata
	metaReq := provider.MetadataRequest{}
	metaResp := &provider.MetadataResponse{}
	p.Metadata(ctx, metaReq, metaResp)

	if metaResp.TypeName != "dotfiles" {
		t.Errorf("expected TypeName to be 'dotfiles', got %s", metaResp.TypeName)
	}

	if metaResp.Version != "test" {
		t.Errorf("expected Version to be 'test', got %s", metaResp.Version)
	}
}

func TestProviderSchema(t *testing.T) {
	ctx := context.Background()

	// Create a new provider instance
	p := New("test")()

	// Test provider schema
	schemaReq := provider.SchemaRequest{}
	schemaResp := &provider.SchemaResponse{}
	p.Schema(ctx, schemaReq, schemaResp)

	if schemaResp.Diagnostics.HasError() {
		t.Errorf("schema validation failed: %v", schemaResp.Diagnostics)
	}

	// Check that required attributes exist
	schema := schemaResp.Schema
	if _, exists := schema.Attributes["dotfiles_root"]; !exists {
		t.Error("dotfiles_root attribute not found in schema")
	}

	if _, exists := schema.Attributes["backup_enabled"]; !exists {
		t.Error("backup_enabled attribute not found in schema")
	}

	if _, exists := schema.Attributes["strategy"]; !exists {
		t.Error("strategy attribute not found in schema")
	}
}

func TestProviderConfigure(t *testing.T) {
	t.Run("DotfilesClient creation and validation", func(t *testing.T) {
		testDotfilesClientCreation(t)
	})

	t.Run("DotfilesClient with default values", func(t *testing.T) {
		testDotfilesClientDefaults(t)
	})

	t.Run("Configuration validation tests", func(t *testing.T) {
		testConfigurationValidation(t)
	})
}

// testDotfilesClientCreation tests DotfilesClient creation and validation
func testDotfilesClientCreation(t *testing.T) {
	_, dotfilesDir, backupDir := setupProviderTestDirs(t)

	// Create and test configuration
	config := createValidTestConfig(dotfilesDir, backupDir)
	testProviderConfigValidation(t, config)

	// Create and test client
	client := testClientCreation(t, config)
	testClientProperties(t, client, config)
	testClientPlatformInfo(t, client)
}

// setupProviderTestDirs creates test directories for provider testing
func setupProviderTestDirs(t *testing.T) (string, string, string) {
	tmpDir := t.TempDir()
	dotfilesDir := filepath.Join(tmpDir, "dotfiles")
	backupDir := filepath.Join(tmpDir, "backups")

	if err := os.MkdirAll(dotfilesDir, 0755); err != nil {
		t.Fatalf("Failed to create dotfiles directory: %v", err)
	}
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatalf("Failed to create backup directory: %v", err)
	}

	return tmpDir, dotfilesDir, backupDir
}

// createValidTestConfig creates a valid test configuration
func createValidTestConfig(dotfilesDir, backupDir string) *DotfilesConfig {
	return &DotfilesConfig{
		DotfilesRoot:       dotfilesDir,
		BackupEnabled:      true,
		BackupDirectory:    backupDir,
		Strategy:           "symlink",
		ConflictResolution: "backup",
		DryRun:             false,
		AutoDetectPlatform: true,
		TargetPlatform:     "auto",
		TemplateEngine:     "go",
		LogLevel:           "info",
	}
}

// testConfigValidation tests configuration validation
func testProviderConfigValidation(t *testing.T, config *DotfilesConfig) {
	if err := config.Validate(); err != nil {
		t.Errorf("Valid configuration failed validation: %v", err)
	}
}

// testClientCreation creates and tests client creation
func testClientCreation(t *testing.T, config *DotfilesConfig) *DotfilesClient {
	client, err := NewDotfilesClient(config)
	if err != nil {
		t.Errorf("Failed to create DotfilesClient: %v", err)
		return nil
	}
	return client
}

// testClientProperties tests client property validation
func testClientProperties(t *testing.T, client *DotfilesClient, config *DotfilesConfig) {
	if client == nil {
		return
	}

	if client.Config != config {
		t.Error("Client config not set correctly")
	}
	if client.Platform == "" {
		t.Error("Platform should be detected")
	}
	if client.Architecture == "" {
		t.Error("Architecture should be detected")
	}
	if client.HomeDir == "" {
		t.Error("HomeDir should be set")
	}
	if client.ConfigDir == "" {
		t.Error("ConfigDir should be set")
	}
}

// testClientPlatformInfo tests client platform information
func testClientPlatformInfo(t *testing.T, client *DotfilesClient) {
	if client == nil {
		return
	}

	// Verify platform info method
	platformInfo := client.GetPlatformInfo()
	if platformInfo["platform"] != client.Platform {
		t.Error("Platform info doesn't match client platform")
	}

	// Verify platform is one of the expected values
	knownPlatforms := []string{"macos", "linux", "windows"}
	found := false
	for _, platform := range knownPlatforms {
		if client.Platform == platform {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Unknown platform detected: %s, expected one of %v", client.Platform, knownPlatforms)
	}
}

// testDotfilesClientDefaults tests client creation with default values
func testDotfilesClientDefaults(t *testing.T) {
	tmpDir := t.TempDir()

	// Test client creation with minimal config (to test defaults)
	config := &DotfilesConfig{
		DotfilesRoot: tmpDir,
		// Leave other values empty to test defaults
	}

	// Set and validate defaults
	testDefaultsSetup(t, config)
	testDefaultValues(t, config)
	testClientWithDefaults(t, config)
}

// testDefaultsSetup sets up and validates defaults
func testDefaultsSetup(t *testing.T, config *DotfilesConfig) {
	if err := config.SetDefaults(); err != nil {
		t.Errorf("Setting defaults failed: %v", err)
	}
	if err := config.Validate(); err != nil {
		t.Errorf("Configuration validation failed: %v", err)
	}
}

// testDefaultValues verifies default values were set correctly
func testDefaultValues(t *testing.T, config *DotfilesConfig) {
	expectations := map[string]string{
		"Strategy":           "symlink",
		"ConflictResolution": "backup",
		"TargetPlatform":     "auto",
		"TemplateEngine":     "go",
		"LogLevel":           "info",
	}

	if config.Strategy != expectations["Strategy"] {
		t.Errorf("Expected default strategy '%s', got '%s'", expectations["Strategy"], config.Strategy)
	}
	if config.ConflictResolution != expectations["ConflictResolution"] {
		t.Errorf("Expected default conflict resolution '%s', got '%s'", expectations["ConflictResolution"], config.ConflictResolution)
	}
	if config.TargetPlatform != expectations["TargetPlatform"] {
		t.Errorf("Expected default target platform '%s', got '%s'", expectations["TargetPlatform"], config.TargetPlatform)
	}
	if config.TemplateEngine != expectations["TemplateEngine"] {
		t.Errorf("Expected default template engine '%s', got '%s'", expectations["TemplateEngine"], config.TemplateEngine)
	}
	if config.LogLevel != expectations["LogLevel"] {
		t.Errorf("Expected default log level '%s', got '%s'", expectations["LogLevel"], config.LogLevel)
	}
}

// testClientWithDefaults tests client creation with default values
func testClientWithDefaults(t *testing.T, config *DotfilesConfig) {
	client, err := NewDotfilesClient(config)
	if err != nil {
		t.Errorf("Failed to create DotfilesClient with defaults: %v", err)
	} else if client.Platform == "" {
		// Verify client was created with platform detection
		t.Error("Platform detection should work with defaults")
	}
}

// testConfigurationValidation tests configuration validation with various scenarios
func testConfigurationValidation(t *testing.T) {
	tmpDir := t.TempDir()
	testCases := createValidationTestCases(tmpDir)

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectErr {
				if err == nil {
					t.Error("Expected validation error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected validation error: %v", err)
				}
			}
		})
	}
}

// validationTestCase represents a validation test case
type validationTestCase struct {
	name      string
	config    *DotfilesConfig
	expectErr bool
}

// createValidationTestCases creates test cases for configuration validation
func createValidationTestCases(tmpDir string) []validationTestCase {
	return []validationTestCase{
		{
			name: "Valid configuration",
			config: &DotfilesConfig{
				DotfilesRoot:       tmpDir,
				Strategy:           "symlink",
				ConflictResolution: "backup",
				TargetPlatform:     "auto",
				TemplateEngine:     "go",
				LogLevel:           "info",
				BackupEnabled:      true,
				AutoDetectPlatform: true,
				DryRun:             false,
			},
			expectErr: false,
		},
		{
			name: "Invalid strategy",
			config: &DotfilesConfig{
				DotfilesRoot: tmpDir,
				Strategy:     "invalid",
			},
			expectErr: true,
		},
		{
			name: "Invalid conflict resolution",
			config: &DotfilesConfig{
				DotfilesRoot:       tmpDir,
				ConflictResolution: "invalid",
			},
			expectErr: true,
		},
		{
			name: "Invalid target platform",
			config: &DotfilesConfig{
				DotfilesRoot:   tmpDir,
				TargetPlatform: "invalid",
			},
			expectErr: true,
		},
		{
			name: "Invalid template engine",
			config: &DotfilesConfig{
				DotfilesRoot:   tmpDir,
				TemplateEngine: "invalid",
			},
			expectErr: true,
		},
		{
			name: "Invalid log level",
			config: &DotfilesConfig{
				DotfilesRoot: tmpDir,
				LogLevel:     "invalid",
			},
			expectErr: true,
		},
	}
}

func TestProviderResources(t *testing.T) {
	ctx := context.Background()

	// Create a new provider instance
	p := New("test")()

	// Get resources
	resources := p.Resources(ctx)

	if len(resources) == 0 {
		t.Error("no resources returned")
	}

	expectedResources := 6 // repository, file, symlink, directory, application, file_permissions
	if len(resources) != expectedResources {
		t.Errorf("expected %d resources, got %d", expectedResources, len(resources))
	}
}

func TestProviderDataSources(t *testing.T) {
	ctx := context.Background()

	// Create a new provider instance
	p := New("test")()

	// Get data sources
	dataSources := p.DataSources(ctx)

	if len(dataSources) == 0 {
		t.Error("no data sources returned")
	}

	expectedDataSources := 2 // system, file_info
	if len(dataSources) != expectedDataSources {
		t.Errorf("expected %d data sources, got %d", expectedDataSources, len(dataSources))
	}
}

func TestDotfilesConfig(t *testing.T) {
	// Create temporary directory for testing
	tmpDir := t.TempDir()

	// Test default configuration
	config := &DotfilesConfig{
		DotfilesRoot:       tmpDir,
		Strategy:           "symlink",
		ConflictResolution: "backup",
		TargetPlatform:     "auto",
		TemplateEngine:     "go",
		LogLevel:           "info",
		BackupEnabled:      true,
		AutoDetectPlatform: true,
		DryRun:             false,
	}

	if err := config.Validate(); err != nil {
		t.Errorf("valid configuration failed validation: %v", err)
	}
}

func TestDotfilesConfigInvalid(t *testing.T) {
	// Test invalid configuration
	config := &DotfilesConfig{
		Strategy:           "invalid",
		ConflictResolution: "backup",
		TargetPlatform:     "auto",
		TemplateEngine:     "go",
		LogLevel:           "info",
	}

	if err := config.Validate(); err == nil {
		t.Error("invalid configuration passed validation")
	}
}

func TestProtoV6ProviderServer(t *testing.T) {
	// Create a provider server for protocol version 6
	providerFactory := providerserver.NewProtocol6(New("test")())

	// Verify we can create the provider server
	if providerFactory == nil {
		t.Error("failed to create provider server factory")
	}
}
