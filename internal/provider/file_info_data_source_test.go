// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestFileInfoDataSource(t *testing.T) {
	t.Run("NewFileInfoDataSource", func(t *testing.T) {
		d := NewFileInfoDataSource()
		if d == nil {
			t.Fatal("NewFileInfoDataSource() returned nil")
		}

		// FileInfoDataSource should implement the DataSource interface
		if d == nil {
			t.Error("FileInfoDataSource should not be nil")
		}
	})

	t.Run("Metadata", func(t *testing.T) {
		d := NewFileInfoDataSource()
		ctx := context.Background()

		req := datasource.MetadataRequest{
			ProviderTypeName: "dotfiles",
		}
		resp := &datasource.MetadataResponse{}

		d.Metadata(ctx, req, resp)

		expectedTypeName := "dotfiles_file_info"
		if resp.TypeName != expectedTypeName {
			t.Errorf("Expected TypeName %s, got %s", expectedTypeName, resp.TypeName)
		}
	})

	t.Run("Schema", func(t *testing.T) {
		d := NewFileInfoDataSource()
		ctx := context.Background()

		req := datasource.SchemaRequest{}
		resp := &datasource.SchemaResponse{}

		d.Schema(ctx, req, resp)

		if resp.Diagnostics.HasError() {
			t.Errorf("Schema validation failed: %v", resp.Diagnostics)
		}

		schema := resp.Schema

		// Check required attributes
		requiredAttrs := []string{"path"}
		for _, attr := range requiredAttrs {
			if _, exists := schema.Attributes[attr]; !exists {
				t.Errorf("Required attribute %s not found in schema", attr)
			}
		}

		// Check computed attributes
		computedAttrs := []string{"id", "exists", "is_symlink", "permissions", "size"}
		for _, attr := range computedAttrs {
			if _, exists := schema.Attributes[attr]; !exists {
				t.Errorf("Computed attribute %s not found in schema", attr)
			}
		}
	})

	t.Run("Configure", func(t *testing.T) {
		d := NewFileInfoDataSource()
		fileInfoDS, ok := d.(*FileInfoDataSource)
		if !ok {
			t.Fatal("NewFileInfoDataSource() did not return *FileInfoDataSource")
		}

		ctx := context.Background()

		// Test with nil provider data
		req := datasource.ConfigureRequest{
			ProviderData: nil,
		}
		resp := &datasource.ConfigureResponse{}

		fileInfoDS.Configure(ctx, req, resp)

		// Should not error with nil provider data
		if resp.Diagnostics.HasError() {
			t.Errorf("Configure with nil provider data should not error: %v", resp.Diagnostics)
		}

		// Test with valid client
		client := &DotfilesClient{
			Platform:     "macos",
			Architecture: "arm64",
			HomeDir:      "/Users/test",
			ConfigDir:    "/Users/test/.config",
		}

		req.ProviderData = client
		resp = &datasource.ConfigureResponse{}

		fileInfoDS.Configure(ctx, req, resp)

		if resp.Diagnostics.HasError() {
			t.Errorf("Configure with valid client failed: %v", resp.Diagnostics)
		}

		// Test with invalid provider data type
		req.ProviderData = "invalid"
		resp = &datasource.ConfigureResponse{}

		fileInfoDS.Configure(ctx, req, resp)

		if !resp.Diagnostics.HasError() {
			t.Error("Configure with invalid provider data should error")
		}
	})
}

func TestFileInfoDataSourceModel(t *testing.T) {
	// Test the data source model with various values
	model := FileInfoDataSourceModel{
		ID:          types.StringValue("/tmp/test.txt"),
		Path:        types.StringValue("/tmp/test.txt"),
		Exists:      types.BoolValue(true),
		IsSymlink:   types.BoolValue(false),
		Permissions: types.StringValue("0644"),
		Size:        types.Int64Value(1024),
	}

	// Verify all fields can be accessed
	if model.ID.ValueString() != "/tmp/test.txt" {
		t.Error("ID field not working correctly")
	}
	if model.Path.ValueString() != "/tmp/test.txt" {
		t.Error("Path field not working correctly")
	}
	if !model.Exists.ValueBool() {
		t.Error("Exists field not working correctly")
	}
	if model.IsSymlink.ValueBool() {
		t.Error("IsSymlink field not working correctly")
	}
	if model.Permissions.ValueString() != "0644" {
		t.Error("Permissions field not working correctly")
	}
	if model.Size.ValueInt64() != 1024 {
		t.Error("Size field not working correctly")
	}
}
