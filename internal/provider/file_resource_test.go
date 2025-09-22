// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestFileResource(t *testing.T) {
	t.Run("NewFileResource", func(t *testing.T) {
		r := NewFileResource()
		if r == nil {
			t.Fatal("NewFileResource() returned nil")
		}

		// Verify it implements the Resource interface
		_, ok := r.(resource.Resource)
		if !ok {
			t.Error("FileResource does not implement resource.Resource interface")
		}
	})

	t.Run("Metadata", func(t *testing.T) {
		r := NewFileResource()
		ctx := context.Background()

		req := resource.MetadataRequest{
			ProviderTypeName: "dotfiles",
		}
		resp := &resource.MetadataResponse{}

		r.Metadata(ctx, req, resp)

		expectedTypeName := "dotfiles_file"
		if resp.TypeName != expectedTypeName {
			t.Errorf("Expected TypeName %s, got %s", expectedTypeName, resp.TypeName)
		}
	})

	t.Run("Schema", func(t *testing.T) {
		r := NewFileResource()
		ctx := context.Background()

		req := resource.SchemaRequest{}
		resp := &resource.SchemaResponse{}

		r.Schema(ctx, req, resp)

		if resp.Diagnostics.HasError() {
			t.Errorf("Schema validation failed: %v", resp.Diagnostics)
		}

		schema := resp.Schema

		// Check required attributes
		requiredAttrs := []string{"repository", "name", "source_path", "target_path"}
		for _, attr := range requiredAttrs {
			if _, exists := schema.Attributes[attr]; !exists {
				t.Errorf("Required attribute %s not found in schema", attr)
			}
		}

		// Check optional attributes
		optionalAttrs := []string{"is_template", "file_mode", "backup_enabled"}
		for _, attr := range optionalAttrs {
			if _, exists := schema.Attributes[attr]; !exists {
				t.Errorf("Optional attribute %s not found in schema", attr)
			}
		}

		// Check computed attributes
		computedAttrs := []string{"id"}
		for _, attr := range computedAttrs {
			if _, exists := schema.Attributes[attr]; !exists {
				t.Errorf("Computed attribute %s not found in schema", attr)
			}
		}
	})

	t.Run("Configure", func(t *testing.T) {
		r := NewFileResource()
		fileResource, ok := r.(*FileResource)
		if !ok {
			t.Fatal("NewFileResource() did not return *FileResource")
		}

		ctx := context.Background()

		// Test with nil provider data
		req := resource.ConfigureRequest{
			ProviderData: nil,
		}
		resp := &resource.ConfigureResponse{}

		fileResource.Configure(ctx, req, resp)

		// Should not error with nil provider data
		if resp.Diagnostics.HasError() {
			t.Errorf("Configure with nil provider data should not error: %v", resp.Diagnostics)
		}

		// Test with valid client
		client := &DotfilesClient{
			Platform:     "test",
			Architecture: "test",
			HomeDir:      "/tmp",
			ConfigDir:    "/tmp/.config",
		}

		req.ProviderData = client
		resp = &resource.ConfigureResponse{}

		fileResource.Configure(ctx, req, resp)

		if resp.Diagnostics.HasError() {
			t.Errorf("Configure with valid client failed: %v", resp.Diagnostics)
		}

		// Test with invalid provider data type
		req.ProviderData = "invalid"
		resp = &resource.ConfigureResponse{}

		fileResource.Configure(ctx, req, resp)

		if !resp.Diagnostics.HasError() {
			t.Error("Configure with invalid provider data should error")
		}
	})
}

func TestFileResourceModel(t *testing.T) {
	// Test the resource model with various values
	model := FileResourceModel{
		ID:            types.StringValue("test-file"),
		Repository:    types.StringValue("test-repo"),
		Name:          types.StringValue("test-file-name"),
		SourcePath:    types.StringValue("git/gitconfig"),
		TargetPath:    types.StringValue("~/.gitconfig"),
		IsTemplate:    types.BoolValue(true),
		FileMode:      types.StringValue("0644"),
		BackupEnabled: types.BoolValue(true),
	}

	// Verify all fields can be accessed
	if model.ID.ValueString() != "test-file" {
		t.Error("ID field not working correctly")
	}
	if model.Repository.ValueString() != "test-repo" {
		t.Error("Repository field not working correctly")
	}
	if model.Name.ValueString() != "test-file-name" {
		t.Error("Name field not working correctly")
	}
	if model.SourcePath.ValueString() != "git/gitconfig" {
		t.Error("SourcePath field not working correctly")
	}
	if model.TargetPath.ValueString() != "~/.gitconfig" {
		t.Error("TargetPath field not working correctly")
	}
	if !model.IsTemplate.ValueBool() {
		t.Error("IsTemplate field not working correctly")
	}
	if model.FileMode.ValueString() != "0644" {
		t.Error("FileMode field not working correctly")
	}
	if !model.BackupEnabled.ValueBool() {
		t.Error("BackupEnabled field not working correctly")
	}
}

// TestFileResourceCRUD is planned for when file operations are implemented
// Currently the resource methods are stubs, so we focus on testing
// the schema, metadata, and configuration which are fully functional
