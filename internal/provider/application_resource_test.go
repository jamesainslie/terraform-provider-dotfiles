// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestApplicationResource tests the new dotfiles_application resource.
func TestApplicationResource(t *testing.T) {
	t.Run("NewApplicationResource", func(t *testing.T) {
		r := NewApplicationResource()
		if r == nil {
			t.Fatal("NewApplicationResource() returned nil")
		}

		// ApplicationResource should implement the Resource interface
		if r == nil {
			t.Error("ApplicationResource should not be nil")
		}
	})

	t.Run("Metadata", func(t *testing.T) {
		r := NewApplicationResource()
		ctx := context.Background()

		req := resource.MetadataRequest{
			ProviderTypeName: "dotfiles",
		}
		resp := &resource.MetadataResponse{}

		r.Metadata(ctx, req, resp)

		expectedTypeName := "dotfiles_application"
		if resp.TypeName != expectedTypeName {
			t.Errorf("Expected TypeName %s, got %s", expectedTypeName, resp.TypeName)
		}
	})

	t.Run("Schema", func(t *testing.T) {
		r := NewApplicationResource()
		ctx := context.Background()

		req := resource.SchemaRequest{}
		resp := &resource.SchemaResponse{}

		r.Schema(ctx, req, resp)

		if resp.Diagnostics.HasError() {
			t.Errorf("Schema validation failed: %v", resp.Diagnostics)
		}

		schema := resp.Schema

		// Check required attributes
		requiredAttrs := []string{"repository", "application", "source_path"}
		for _, attr := range requiredAttrs {
			if _, exists := schema.Attributes[attr]; !exists {
				t.Errorf("Required attribute %s not found in schema", attr)
			}
		}

		// Check application detection attributes
		detectionAttrs := []string{"detect_installation", "skip_if_not_installed", "warn_if_not_installed"}
		for _, attr := range detectionAttrs {
			if _, exists := schema.Attributes[attr]; !exists {
				t.Errorf("Detection attribute %s not found in schema", attr)
			}
		}

		// Check version compatibility attributes
		versionAttrs := []string{"min_version", "max_version"}
		for _, attr := range versionAttrs {
			if _, exists := schema.Attributes[attr]; !exists {
				t.Errorf("Version attribute %s not found in schema", attr)
			}
		}

		// Check for detection_methods block
		if _, exists := schema.Blocks["detection_methods"]; !exists {
			t.Error("detection_methods block should be defined in application resource schema")
		}

		// Check for config_mappings block
		if _, exists := schema.Blocks["config_mappings"]; !exists {
			t.Error("config_mappings block should be defined in application resource schema")
		}

		// Check computed attributes
		computedAttrs := []string{"id", "installed", "version", "installation_path"}
		for _, attr := range computedAttrs {
			if _, exists := schema.Attributes[attr]; !exists {
				t.Errorf("Computed attribute %s not found in schema", attr)
			}
		}
	})
}

// TestApplicationResourceModel tests the application resource data model.
func TestApplicationResourceModel(t *testing.T) {
	// Test the application resource model with comprehensive configuration
	model := ApplicationResourceModel{
		ID:                 types.StringValue("cursor-app"),
		Repository:         types.StringValue("test-repo"),
		Application:        types.StringValue("cursor"),
		SourcePath:         types.StringValue("tools/cursor"),
		DetectInstallation: types.BoolValue(true),
		SkipIfNotInstalled: types.BoolValue(true),
		WarnIfNotInstalled: types.BoolValue(false),
		MinVersion:         types.StringValue("0.17.0"),
		MaxVersion:         types.StringValue("1.0.0"),

		// Computed attributes
		Installed:        types.BoolValue(true),
		Version:          types.StringValue("0.19.3"),
		InstallationPath: types.StringValue("/Applications/Cursor.app"),
	}

	// Verify all fields can be accessed
	if model.ID.ValueString() != "cursor-app" {
		t.Error("ID field not working correctly")
	}
	if model.Application.ValueString() != "cursor" {
		t.Error("Application field not working correctly")
	}
	if !model.DetectInstallation.ValueBool() {
		t.Error("DetectInstallation field not working correctly")
	}
	if !model.SkipIfNotInstalled.ValueBool() {
		t.Error("SkipIfNotInstalled field not working correctly")
	}
	if model.MinVersion.ValueString() != "0.17.0" {
		t.Error("MinVersion field not working correctly")
	}
	if !model.Installed.ValueBool() {
		t.Error("Installed computed field not working correctly")
	}
	if model.Version.ValueString() != "0.19.3" {
		t.Error("Version computed field not working correctly")
	}
}

// TestDetectionMethodsModel tests the detection methods configuration.
func TestDetectionMethodsModel(t *testing.T) {
	// Test detection methods configuration
	methods := []DetectionMethodModel{
		{
			Type: types.StringValue("command"),
			Test: types.StringValue("command -v cursor"),
		},
		{
			Type: types.StringValue("file"),
			Path: types.StringValue("/Applications/Cursor.app"),
		},
		{
			Type: types.StringValue("brew_cask"),
			Name: types.StringValue("cursor"),
		},
		{
			Type:    types.StringValue("package_manager"),
			Name:    types.StringValue("cursor"),
			Manager: types.StringValue("brew"),
		},
	}

	// Verify detection methods
	if len(methods) != 4 {
		t.Errorf("Expected 4 detection methods, got %d", len(methods))
	}

	// Test command detection
	if methods[0].Type.ValueString() != "command" {
		t.Error("First method should be command type")
	}
	if methods[0].Test.ValueString() != "command -v cursor" {
		t.Error("Command test should be set correctly")
	}

	// Test file detection
	if methods[1].Type.ValueString() != "file" {
		t.Error("Second method should be file type")
	}
	if methods[1].Path.ValueString() != "/Applications/Cursor.app" {
		t.Error("File path should be set correctly")
	}

	// Test brew cask detection
	if methods[2].Type.ValueString() != "brew_cask" {
		t.Error("Third method should be brew_cask type")
	}
	if methods[2].Name.ValueString() != "cursor" {
		t.Error("Brew cask name should be set correctly")
	}

	// Test package manager detection
	if methods[3].Type.ValueString() != "package_manager" {
		t.Error("Fourth method should be package_manager type")
	}
	if methods[3].Manager.ValueString() != "brew" {
		t.Error("Package manager should be set correctly")
	}
}

// TestConfigMappingsModel tests the configuration mappings model.
func TestConfigMappingsModel(t *testing.T) {
	// Test config mappings
	mappings := map[string]ConfigMappingModel{
		"cli-config.json": {
			TargetPath:    types.StringValue("~/.cursor/cli-config.json"),
			Required:      types.BoolValue(false),
			MergeStrategy: types.StringValue("replace"),
		},
		"user/*.json": {
			TargetPathTemplate: types.StringValue("{{.app_support_dir}}/Cursor/User/{filename}"),
			MergeStrategy:      types.StringValue("deep_merge"),
			Required:           types.BoolValue(true),
		},
	}

	// Verify mappings
	if len(mappings) != 2 {
		t.Errorf("Expected 2 config mappings, got %d", len(mappings))
	}

	// Test simple mapping
	cliMapping := mappings["cli-config.json"]
	if cliMapping.TargetPath.ValueString() != "~/.cursor/cli-config.json" {
		t.Error("CLI config target path should be set correctly")
	}
	if cliMapping.Required.ValueBool() {
		t.Error("CLI config should not be required")
	}

	// Test template mapping
	userMapping := mappings["user/*.json"]
	if !strings.Contains(userMapping.TargetPathTemplate.ValueString(), "{{.app_support_dir}}") {
		t.Error("User config should use target path template")
	}
	if userMapping.MergeStrategy.ValueString() != "deep_merge" {
		t.Error("User config should use deep merge strategy")
	}
}
