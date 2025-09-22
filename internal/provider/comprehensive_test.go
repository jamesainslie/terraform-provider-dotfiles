// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

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
			} else {
				if dotfilesProvider.version != version {
					t.Errorf("Expected version %s, got %s", version, dotfilesProvider.version)
				}
			}
		})
	}
}

func TestAllPlatformProviders(t *testing.T) {
	// Test that all platform providers implement the interface correctly
	// Test the platform detection rather than creating providers directly
	detectedPlatform := platform.DetectPlatform()

	testCases := []struct {
		name     string
		provider platform.PlatformProvider
	}{
		{
			name:     "Detected Platform",
			provider: detectedPlatform,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test all interface methods
			platform := tc.provider.GetPlatform()
			if platform == "" {
				t.Error("GetPlatform() returned empty string")
			}

			arch := tc.provider.GetArchitecture()
			if arch == "" {
				t.Error("GetArchitecture() returned empty string")
			}

			separator := tc.provider.GetPathSeparator()
			if separator == "" {
				t.Error("GetPathSeparator() returned empty string")
			}

			// Test directory operations
			homeDir, err := tc.provider.GetHomeDir()
			if err != nil {
				t.Logf("GetHomeDir() error (may be expected): %v", err)
			} else if homeDir == "" {
				t.Error("GetHomeDir() returned empty string")
			}

			configDir, err := tc.provider.GetConfigDir()
			if err != nil {
				t.Logf("GetConfigDir() error (may be expected): %v", err)
			} else if configDir == "" {
				t.Error("GetConfigDir() returned empty string")
			}

			appSupportDir, err := tc.provider.GetAppSupportDir()
			if err != nil {
				t.Logf("GetAppSupportDir() error (may be expected): %v", err)
			} else if appSupportDir == "" {
				t.Error("GetAppSupportDir() returned empty string")
			}

			// Test path operations with safe inputs
			safeInputs := []string{".", "/tmp", "test"}
			for _, input := range safeInputs {
				_, err := tc.provider.ExpandPath(input)
				if err != nil {
					t.Logf("ExpandPath(%q) error (may be expected): %v", input, err)
				}

				_, err = tc.provider.ResolvePath(input)
				if err != nil {
					t.Logf("ResolvePath(%q) error (may be expected): %v", input, err)
				}
			}

			// Test application detection
			testApps := []string{"git", "test-nonexistent-app"}
			for _, app := range testApps {
				info, err := tc.provider.DetectApplication(app)
				if err != nil {
					t.Errorf("DetectApplication(%q) failed: %v", app, err)
				} else if info == nil {
					t.Errorf("DetectApplication(%q) returned nil info", app)
				} else if info.Name != app {
					t.Errorf("DetectApplication(%q) returned wrong name: %s", app, info.Name)
				}
			}

			// Test application paths
			paths, err := tc.provider.GetApplicationPaths("test-app")
			if err != nil {
				t.Errorf("GetApplicationPaths failed: %v", err)
			} else if len(paths) == 0 {
				t.Error("GetApplicationPaths returned no paths")
			}
		})
	}
}

func TestApplicationDetectionComprehensive(t *testing.T) {
	platformProvider := platform.DetectPlatform()

	// Test with applications that should exist on most systems
	commonApps := []string{"ls", "cat", "echo"}

	for _, app := range commonApps {
		t.Run("App: "+app, func(t *testing.T) {
			info, err := platformProvider.DetectApplication(app)
			if err != nil {
				t.Errorf("DetectApplication(%s) failed: %v", app, err)
			}

			if info == nil {
				t.Errorf("DetectApplication(%s) returned nil", app)
			} else {
				if info.Name != app {
					t.Errorf("Expected app name %s, got %s", app, info.Name)
				}

				// These apps likely exist on most systems
				if !info.Installed {
					t.Logf("App %s not found (may be normal on some systems)", app)
				}
			}
		})
	}

	// Test with application that definitely doesn't exist
	t.Run("Nonexistent application", func(t *testing.T) {
		info, err := platformProvider.DetectApplication("definitely-does-not-exist-12345")
		if err != nil {
			t.Errorf("DetectApplication should not error for nonexistent app: %v", err)
		}

		if info == nil {
			t.Error("DetectApplication should return info even for nonexistent apps")
		} else {
			if info.Installed {
				t.Error("Nonexistent app should not be marked as installed")
			}
		}
	})
}
