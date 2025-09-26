// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

package provider

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestApplicationResourceUnit(t *testing.T) {
	// Unit tests for the application resource methods
	ctx := context.Background()
	
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	
	// Create mock client
	client := &DotfilesClient{
		Config: &DotfilesConfig{
			DotfilesRoot: tempDir,
		},
	}
	
	// Create test resource
	appResource := &ApplicationResource{
		client: client,
	}
	
	t.Run("expandTargetPathTemplate", func(t *testing.T) {
		testCases := []struct {
			name         string
			targetPath   string
			appName      string
			expectError  bool
			expectPrefix string
		}{
			{
				name:         "simple path",
				targetPath:   "/simple/path",
				appName:      "testapp",
				expectError:  false,
				expectPrefix: "/simple/path",
			},
			{
				name:         "home directory template",
				targetPath:   "{{.home_dir}}/config",
				appName:      "testapp",
				expectError:  false,
				expectPrefix: "", // We'll check it contains the home dir
			},
			{
				name:         "application template",
				targetPath:   "/config/{{.application}}/settings.json",
				appName:      "testapp",
				expectError:  false,
				expectPrefix: "/config/testapp/settings.json",
			},
			{
				name:         "tilde expansion",
				targetPath:   "~/config/app",
				appName:      "testapp",
				expectError:  false,
				expectPrefix: "", // We'll check it contains the home dir
			},
		}
		
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result, err := appResource.expandTargetPathTemplate(tc.targetPath, tc.appName)
				
				if tc.expectError && err == nil {
					t.Error("Expected error but got none")
				}
				if !tc.expectError && err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				
				if !tc.expectError && tc.expectPrefix != "" && result != tc.expectPrefix {
					t.Errorf("Expected result to be %s, got %s", tc.expectPrefix, result)
				}
				
				// Special cases for paths that should contain home directory
				if !tc.expectError && (tc.targetPath == "{{.home_dir}}/config" || tc.targetPath == "~/config/app") {
					homeDir, _ := os.UserHomeDir()
					if !filepath.IsAbs(result) || !filepath.HasPrefix(result, homeDir) {
						t.Errorf("Expected result to be absolute path under home directory, got %s", result)
					}
				}
			})
		}
	})
	
	t.Run("createSymlinkForConfig", func(t *testing.T) {
		// Create source file
		sourceFile := filepath.Join(tempDir, "source.txt")
		err := os.WriteFile(sourceFile, []byte("test content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create source file: %v", err)
		}
		
		// Test symlink creation
		targetFile := filepath.Join(tempDir, "subdir", "target.txt")
		err = appResource.createSymlinkForConfig(ctx, sourceFile, targetFile)
		if err != nil {
			t.Errorf("Failed to create symlink: %v", err)
		}
		
		// Verify symlink exists and points to source
		linkTarget, err := os.Readlink(targetFile)
		if err != nil {
			t.Errorf("Failed to read symlink: %v", err)
		}
		if linkTarget != sourceFile {
			t.Errorf("Expected symlink to point to %s, got %s", sourceFile, linkTarget)
		}
	})
	
	t.Run("copyConfigFile", func(t *testing.T) {
		// Create source file
		sourceContent := "test configuration content"
		sourceFile := filepath.Join(tempDir, "source_copy.txt")
		err := os.WriteFile(sourceFile, []byte(sourceContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create source file: %v", err)
		}
		
		// Test file copying
		targetFile := filepath.Join(tempDir, "copy_subdir", "target_copy.txt")
		err = appResource.copyConfigFile(ctx, sourceFile, targetFile)
		if err != nil {
			t.Errorf("Failed to copy file: %v", err)
		}
		
		// Verify copied file exists and has correct content
		copiedContent, err := os.ReadFile(targetFile)
		if err != nil {
			t.Errorf("Failed to read copied file: %v", err)
		}
		if string(copiedContent) != sourceContent {
			t.Errorf("Expected copied content to be %s, got %s", sourceContent, string(copiedContent))
		}
	})
}

func TestApplicationResourceSchema(t *testing.T) {
	ctx := context.Background()
	
	// Create resource
	appResource := NewApplicationResource()
	
	// Test schema
	schemaReq := resource.SchemaRequest{}
	schemaResp := &resource.SchemaResponse{}
	
	appResource.Schema(ctx, schemaReq, schemaResp)
	
	if schemaResp.Diagnostics.HasError() {
		t.Errorf("Schema validation failed: %v", schemaResp.Diagnostics)
	}
	
	// Verify required attributes exist
	schema := schemaResp.Schema
	if _, ok := schema.Attributes["application_name"]; !ok {
		t.Error("Expected application_name attribute in schema")
	}
	if _, ok := schema.Attributes["config_mappings"]; !ok {
		t.Error("Expected config_mappings attribute in schema")
	}
	if _, ok := schema.Attributes["configured_files"]; !ok {
		t.Error("Expected configured_files computed attribute in schema")
	}
	if _, ok := schema.Attributes["last_updated"]; !ok {
		t.Error("Expected last_updated computed attribute in schema")
	}
}

func TestApplicationResourceConfigure(t *testing.T) {
	ctx := context.Background()
	
	// Create resource
	appResource := NewApplicationResource()
	
	// Test configuration with valid client
	client := &DotfilesClient{
		Config: &DotfilesConfig{
			DotfilesRoot: "/test",
		},
	}
	
	configReq := resource.ConfigureRequest{
		ProviderData: client,
	}
	configResp := &resource.ConfigureResponse{}
	
	appResource.(*ApplicationResource).Configure(ctx, configReq, configResp)
	
	if configResp.Diagnostics.HasError() {
		t.Errorf("Configure failed: %v", configResp.Diagnostics)
	}
	
	if appResource.(*ApplicationResource).client != client {
		t.Error("Expected client to be set correctly")
	}
	
	// Test configuration with invalid client type
	appResource2 := NewApplicationResource()
	configReq2 := resource.ConfigureRequest{
		ProviderData: "invalid",
	}
	configResp2 := &resource.ConfigureResponse{}
	
	appResource2.(*ApplicationResource).Configure(ctx, configReq2, configResp2)
	
	if !configResp2.Diagnostics.HasError() {
		t.Error("Expected error with invalid provider data type")
	}
}