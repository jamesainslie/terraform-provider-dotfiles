// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestSystemDataSource(t *testing.T) {
	t.Run("NewSystemDataSource", func(t *testing.T) {
		d := NewSystemDataSource()
		if d == nil {
			t.Fatal("NewSystemDataSource() returned nil")
		}

		// SystemDataSource should implement the DataSource interface
		if d == nil {
			t.Error("SystemDataSource should not be nil")
		}
	})

	t.Run("Metadata", func(t *testing.T) {
		d := NewSystemDataSource()
		ctx := context.Background()

		req := datasource.MetadataRequest{
			ProviderTypeName: "dotfiles",
		}
		resp := &datasource.MetadataResponse{}

		d.Metadata(ctx, req, resp)

		expectedTypeName := "dotfiles_system"
		if resp.TypeName != expectedTypeName {
			t.Errorf("Expected TypeName %s, got %s", expectedTypeName, resp.TypeName)
		}
	})

	t.Run("Schema", func(t *testing.T) {
		d := NewSystemDataSource()
		ctx := context.Background()

		req := datasource.SchemaRequest{}
		resp := &datasource.SchemaResponse{}

		d.Schema(ctx, req, resp)

		if resp.Diagnostics.HasError() {
			t.Errorf("Schema validation failed: %v", resp.Diagnostics)
		}

		schema := resp.Schema

		// Check computed attributes (all should be computed for a data source)
		expectedAttrs := []string{"id", "platform", "architecture", "home_dir", "config_dir"}
		for _, attr := range expectedAttrs {
			if _, exists := schema.Attributes[attr]; !exists {
				t.Errorf("Expected attribute %s not found in schema", attr)
			}
		}
	})

	t.Run("Configure", func(t *testing.T) {
		d := NewSystemDataSource()
		systemDS, ok := d.(*SystemDataSource)
		if !ok {
			t.Fatal("NewSystemDataSource() did not return *SystemDataSource")
		}

		ctx := context.Background()

		// Test with nil provider data
		req := datasource.ConfigureRequest{
			ProviderData: nil,
		}
		resp := &datasource.ConfigureResponse{}

		systemDS.Configure(ctx, req, resp)

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

		systemDS.Configure(ctx, req, resp)

		if resp.Diagnostics.HasError() {
			t.Errorf("Configure with valid client failed: %v", resp.Diagnostics)
		}

		// Test with invalid provider data type
		req.ProviderData = "invalid"
		resp = &datasource.ConfigureResponse{}

		systemDS.Configure(ctx, req, resp)

		if !resp.Diagnostics.HasError() {
			t.Error("Configure with invalid provider data should error")
		}
	})

	t.Run("Read", func(t *testing.T) {
		client := &DotfilesClient{
			Platform:     "macos",
			Architecture: "arm64",
			HomeDir:      "/Users/test",
			ConfigDir:    "/Users/test/.config",
		}

		d := &SystemDataSource{client: client}
		_ = d // Test that we can create the data source with a client

		// TODO: Add actual Read testing when we implement proper request/response mocking
		// The current implementation requires a properly structured request which is complex to mock
		t.Log("SystemDataSource Read method will be tested with integration tests")
	})
}

func TestSystemDataSourceModel(t *testing.T) {
	// Test the data source model with various values
	model := SystemDataSourceModel{
		ID:           types.StringValue("system"),
		Platform:     types.StringValue("macos"),
		Architecture: types.StringValue("arm64"),
		HomeDir:      types.StringValue("/Users/test"),
		ConfigDir:    types.StringValue("/Users/test/.config"),
	}

	// Verify all fields can be accessed
	if model.ID.ValueString() != "system" {
		t.Error("ID field not working correctly")
	}
	if model.Platform.ValueString() != "macos" {
		t.Error("Platform field not working correctly")
	}
	if model.Architecture.ValueString() != "arm64" {
		t.Error("Architecture field not working correctly")
	}
	if model.HomeDir.ValueString() != "/Users/test" {
		t.Error("HomeDir field not working correctly")
	}
	if model.ConfigDir.ValueString() != "/Users/test/.config" {
		t.Error("ConfigDir field not working correctly")
	}
}
