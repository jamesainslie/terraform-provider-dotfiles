// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestSymlinkResource(t *testing.T) {
	t.Run("NewSymlinkResource", func(t *testing.T) {
		r := NewSymlinkResource()
		if r == nil {
			t.Fatal("NewSymlinkResource() returned nil")
		}

		// SymlinkResource should implement the Resource interface
		if r == nil {
			t.Error("SymlinkResource should not be nil")
		}
	})

	t.Run("Metadata", func(t *testing.T) {
		r := NewSymlinkResource()
		ctx := context.Background()

		req := resource.MetadataRequest{
			ProviderTypeName: "dotfiles",
		}
		resp := &resource.MetadataResponse{}

		r.Metadata(ctx, req, resp)

		expectedTypeName := "dotfiles_symlink"
		if resp.TypeName != expectedTypeName {
			t.Errorf("Expected TypeName %s, got %s", expectedTypeName, resp.TypeName)
		}
	})

	t.Run("Schema", func(t *testing.T) {
		r := NewSymlinkResource()
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
		optionalAttrs := []string{"force_update", "create_parents"}
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
		r := NewSymlinkResource()
		symlinkResource, ok := r.(*SymlinkResource)
		if !ok {
			t.Fatal("NewSymlinkResource() did not return *SymlinkResource")
		}

		ctx := context.Background()

		// Test with nil provider data
		req := resource.ConfigureRequest{
			ProviderData: nil,
		}
		resp := &resource.ConfigureResponse{}

		symlinkResource.Configure(ctx, req, resp)

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

		symlinkResource.Configure(ctx, req, resp)

		if resp.Diagnostics.HasError() {
			t.Errorf("Configure with valid client failed: %v", resp.Diagnostics)
		}

		// Test with invalid provider data type
		req.ProviderData = "invalid"
		resp = &resource.ConfigureResponse{}

		symlinkResource.Configure(ctx, req, resp)

		if !resp.Diagnostics.HasError() {
			t.Error("Configure with invalid provider data should error")
		}
	})
}

func TestSymlinkResourceModel(t *testing.T) {
	// Test the resource model with various values
	model := SymlinkResourceModel{
		ID:            types.StringValue("test-symlink"),
		Repository:    types.StringValue("test-repo"),
		Name:          types.StringValue("test-symlink-name"),
		SourcePath:    types.StringValue("fish"),
		TargetPath:    types.StringValue("~/.config/fish"),
		ForceUpdate:   types.BoolValue(true),
		CreateParents: types.BoolValue(true),
	}

	// Verify all fields can be accessed
	if model.ID.ValueString() != "test-symlink" {
		t.Error("ID field not working correctly")
	}
	if model.Repository.ValueString() != "test-repo" {
		t.Error("Repository field not working correctly")
	}
	if model.Name.ValueString() != "test-symlink-name" {
		t.Error("Name field not working correctly")
	}
	if model.SourcePath.ValueString() != "fish" {
		t.Error("SourcePath field not working correctly")
	}
	if model.TargetPath.ValueString() != "~/.config/fish" {
		t.Error("TargetPath field not working correctly")
	}
	if !model.ForceUpdate.ValueBool() {
		t.Error("ForceUpdate field not working correctly")
	}
	if !model.CreateParents.ValueBool() {
		t.Error("CreateParents field not working correctly")
	}
}
