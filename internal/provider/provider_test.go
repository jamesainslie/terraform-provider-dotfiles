// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
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
		// Test DotfilesClient creation directly
		config := &DotfilesConfig{
			DotfilesRoot:       "/tmp/test-dotfiles",
			BackupEnabled:      true,
			BackupDirectory:    "/tmp/test-backups",
			Strategy:           "symlink",
			ConflictResolution: "backup",
			DryRun:             false,
			AutoDetectPlatform: true,
			TargetPlatform:     "auto",
			TemplateEngine:     "go",
			LogLevel:           "info",
		}

		// Test configuration validation
		if err := config.Validate(); err != nil {
			t.Errorf("Valid configuration failed validation: %v", err)
		}

		// Test client creation
		client, err := NewDotfilesClient(config)
		if err != nil {
			t.Errorf("Failed to create DotfilesClient: %v", err)
		} else {
			// Verify client properties
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
	})

	t.Run("DotfilesClient with default values", func(t *testing.T) {
		// Test client creation with minimal config (to test defaults)
		config := &DotfilesConfig{
			DotfilesRoot: "/tmp/test-dotfiles",
			// Leave other values empty to test defaults
		}

		// Validate should set defaults
		if err := config.Validate(); err != nil {
			t.Errorf("Configuration validation failed: %v", err)
		}

		// Verify defaults were set
		if config.Strategy != "symlink" {
			t.Errorf("Expected default strategy 'symlink', got '%s'", config.Strategy)
		}
		if config.ConflictResolution != "backup" {
			t.Errorf("Expected default conflict resolution 'backup', got '%s'", config.ConflictResolution)
		}
		if config.TargetPlatform != "auto" {
			t.Errorf("Expected default target platform 'auto', got '%s'", config.TargetPlatform)
		}
		if config.TemplateEngine != "go" {
			t.Errorf("Expected default template engine 'go', got '%s'", config.TemplateEngine)
		}
		if config.LogLevel != "info" {
			t.Errorf("Expected default log level 'info', got '%s'", config.LogLevel)
		}

		// Test client creation
		client, err := NewDotfilesClient(config)
		if err != nil {
			t.Errorf("Failed to create DotfilesClient with defaults: %v", err)
		} else {
			// Verify client was created with platform detection
			if client.Platform == "" {
				t.Error("Platform detection should work with defaults")
			}
		}
	})

	t.Run("Configuration validation tests", func(t *testing.T) {
		tests := []struct {
			name      string
			config    *DotfilesConfig
			expectErr bool
		}{
			{
				name: "Valid configuration",
				config: &DotfilesConfig{
					DotfilesRoot:       "/tmp/test",
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
					DotfilesRoot: "/tmp/test",
					Strategy:     "invalid",
				},
				expectErr: true,
			},
			{
				name: "Invalid conflict resolution",
				config: &DotfilesConfig{
					DotfilesRoot:       "/tmp/test",
					ConflictResolution: "invalid",
				},
				expectErr: true,
			},
			{
				name: "Invalid target platform",
				config: &DotfilesConfig{
					DotfilesRoot:   "/tmp/test",
					TargetPlatform: "invalid",
				},
				expectErr: true,
			},
			{
				name: "Invalid template engine",
				config: &DotfilesConfig{
					DotfilesRoot:   "/tmp/test",
					TemplateEngine: "invalid",
				},
				expectErr: true,
			},
			{
				name: "Invalid log level",
				config: &DotfilesConfig{
					DotfilesRoot: "/tmp/test",
					LogLevel:     "invalid",
				},
				expectErr: true,
			},
		}

		for _, tt := range tests {
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
	})
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

	expectedResources := 4 // repository, file, symlink, directory
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
	// Test default configuration
	config := &DotfilesConfig{
		DotfilesRoot:       "/tmp/test-dotfiles",
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
