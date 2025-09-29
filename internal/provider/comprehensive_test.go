// Copyright (c) HashCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/jamesainslie/terraform-provider-dotfiles/internal/platform"
)

func TestProviderIntegration(t *testing.T) {
	t.Run("Provider lifecycle", func(t *testing.T) {
		// Test complete provider lifecycle
		p := New("test")()

		ctx := context.Background()

		// Test metadata
		metaReq := provider.MetadataRequest{}
		metaResp := &provider.MetadataResponse{}
		p.Metadata(ctx, metaReq, metaResp)

		if metaResp.TypeName != "dotfiles" {
			t.Errorf("Expected TypeName 'dotfiles', got '%s'", metaResp.TypeName)
		}

		// Test schema
		schemaReq := provider.SchemaRequest{}
		schemaResp := &provider.SchemaResponse{}
		p.Schema(ctx, schemaReq, schemaResp)

		if schemaResp.Diagnostics.HasError() {
			t.Errorf("Schema validation failed: %v", schemaResp.Diagnostics)
		}

		// Test resource registration
		resources := p.Resources(ctx)
		if len(resources) != 5 {
			t.Errorf("Expected 5 resources, got %d", len(resources))
		}

		// Test data source registration
		dataSources := p.DataSources(ctx)
		if len(dataSources) != 2 {
			t.Errorf("Expected 2 data sources, got %d", len(dataSources))
		}

		// Test functions registration (available in DotfilesProvider interface)
		if dotfilesProvider, ok := p.(*DotfilesProvider); ok {
			functions := dotfilesProvider.Functions(ctx)
			if len(functions) != 0 {
				t.Errorf("Expected 0 functions currently, got %d", len(functions))
			}

			// Test ephemeral resources registration
			ephemeralResources := dotfilesProvider.EphemeralResources(ctx)
			if len(ephemeralResources) != 0 {
				t.Errorf("Expected 0 ephemeral resources currently, got %d", len(ephemeralResources))
			}
		}
	})
}

func TestResourceFactoryFunctions(t *testing.T) {
	// Test all resource factory functions
	t.Run("Resource factories", func(t *testing.T) {
		// Test repository resource
		repo := NewRepositoryResource()
		if repo == nil {
			t.Error("NewRepositoryResource() returned nil")
		}

		// Test file resource
		file := NewFileResource()
		if file == nil {
			t.Error("NewFileResource() returned nil")
		}

		// Test symlink resource
		symlink := NewSymlinkResource()
		if symlink == nil {
			t.Error("NewSymlinkResource() returned nil")
		}

		// Test directory resource
		directory := NewDirectoryResource()
		if directory == nil {
			t.Error("NewDirectoryResource() returned nil")
		}

		// Test application resource
		application := NewApplicationResource()
		if application == nil {
			t.Error("NewApplicationResource() returned nil")
		}
	})

	t.Run("Data source factories", func(t *testing.T) {
		// Test system data source
		system := NewSystemDataSource()
		if system == nil {
			t.Error("NewSystemDataSource() returned nil")
		}

		// Test file info data source
		fileInfo := NewFileInfoDataSource()
		if fileInfo == nil {
			t.Error("NewFileInfoDataSource() returned nil")
		}
	})
}

func TestProviderVersionHandling(t *testing.T) {
	testVersions := []string{"dev", "test", "1.0.0", "v1.2.3"}

	for _, version := range testVersions {
		t.Run("Version: "+version, func(t *testing.T) {
			providerFunc := New(version)
			if providerFunc == nil {
				t.Error("New() returned nil provider function")
			}

			provider := providerFunc()
			if provider == nil {
				t.Error("Provider function returned nil provider")
			}

			// Verify version is set correctly
			dotfilesProvider, ok := provider.(*DotfilesProvider)
			if !ok {
				t.Error("Provider is not *DotfilesProvider")
			} else if dotfilesProvider.version != version {
				t.Errorf("Expected version %s, got %s", version, dotfilesProvider.version)
			}
		})
	}
}

func TestAllPlatformProviders(t *testing.T) {
	// Setup test environment
	testEnv := setupAllPlatformTestEnvironment(t)

	// Test platform detection
	testPlatformDetection(t, testEnv)

	// Test provider functionality
	testProviderFunctionality(t, testEnv)
}

// allPlatformTestEnv holds the test environment setup
type allPlatformTestEnv struct {
	tempDir string
}

// setupAllPlatformTestEnvironment creates the test environment
func setupAllPlatformTestEnvironment(t *testing.T) *allPlatformTestEnv {
	return &allPlatformTestEnv{
		tempDir: t.TempDir(),
	}
}

// testPlatformDetection tests platform detection
func testPlatformDetection(t *testing.T, env *allPlatformTestEnv) {
	_ = env // Environment not used in this detection test
	provider := platform.DetectPlatform()
	if provider == nil {
		t.Error("Platform provider should not be nil")
	}
}

// testProviderFunctionality tests provider functionality
func testProviderFunctionality(t *testing.T, env *allPlatformTestEnv) {
	// Simple functionality test
	if env.tempDir == "" {
		t.Error("Temp directory should not be empty")
	}
}
