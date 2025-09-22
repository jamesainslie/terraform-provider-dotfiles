// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestPermissionsSchemaExtensions tests that permission management.
// schema attributes are properly defined for resources.
func TestPermissionsSchemaExtensions(t *testing.T) {
	t.Run("FileResource should support permissions", func(t *testing.T) {
		r := NewFileResource()
		ctx := context.Background()

		req := resource.SchemaRequest{}
		resp := &resource.SchemaResponse{}

		r.Schema(ctx, req, resp)

		if resp.Diagnostics.HasError() {
			t.Fatalf("Schema validation failed: %v", resp.Diagnostics)
		}

		// Check for permissions block
		if _, exists := resp.Schema.Blocks["permissions"]; !exists {
			t.Error("permissions block should be defined in file resource schema")
		}

		// Check for permission_rules attribute
		if _, exists := resp.Schema.Attributes["permission_rules"]; !exists {
			t.Error("permission_rules attribute should be defined in file resource schema")
		}
	})

	t.Run("SymlinkResource should support permissions", func(t *testing.T) {
		r := NewSymlinkResource()
		ctx := context.Background()

		req := resource.SchemaRequest{}
		resp := &resource.SchemaResponse{}

		r.Schema(ctx, req, resp)

		if resp.Diagnostics.HasError() {
			t.Fatalf("Schema validation failed: %v", resp.Diagnostics)
		}

		// Check for permissions block
		if _, exists := resp.Schema.Blocks["permissions"]; !exists {
			t.Error("permissions block should be defined in symlink resource schema")
		}

		// Check for permission_rules attribute
		if _, exists := resp.Schema.Attributes["permission_rules"]; !exists {
			t.Error("permission_rules attribute should be defined in symlink resource schema")
		}
	})
}

// TestPermissionParsing tests parsing of permission values.
func TestPermissionParsing(t *testing.T) {
	testCases := []struct {
		name        string
		input       string
		expectError bool
		expected    uint32
	}{
		{
			name:        "valid octal permission",
			input:       "0644",
			expectError: false,
			expected:    0644,
		},
		{
			name:        "valid three digit octal",
			input:       "644",
			expectError: false,
			expected:    0644,
		},
		{
			name:        "valid directory permission",
			input:       "0755",
			expectError: false,
			expected:    0755,
		},
		{
			name:        "strict directory permission",
			input:       "0700",
			expectError: false,
			expected:    0700,
		},
		{
			name:        "invalid permission",
			input:       "999",
			expectError: true,
		},
		{
			name:        "invalid format",
			input:       "abc",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := parsePermission(tc.input)
			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error for input %s, but got none", tc.input)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for input %s: %v", tc.input, err)
				}
				if result != tc.expected {
					t.Errorf("Expected %o, got %o", tc.expected, result)
				}
			}
		})
	}
}

// TestPermissionRuleMatching tests pattern matching for permission rules.
func TestPermissionRuleMatching(t *testing.T) {
	testCases := []struct {
		name     string
		pattern  string
		filename string
		matches  bool
	}{
		{
			name:     "exact match",
			pattern:  "config.fish",
			filename: "config.fish",
			matches:  true,
		},
		{
			name:     "glob star match",
			pattern:  "id_*",
			filename: "id_rsa",
			matches:  true,
		},
		{
			name:     "extension match",
			pattern:  "*.pub",
			filename: "id_rsa.pub",
			matches:  true,
		},
		{
			name:     "no match",
			pattern:  "id_*",
			filename: "config.fish",
			matches:  false,
		},
		{
			name:     "complex pattern with conf",
			pattern:  "*.conf",
			filename: "app.conf",
			matches:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := matchesPermissionPattern(tc.pattern, tc.filename)
			if result != tc.matches {
				t.Errorf("Pattern %s with filename %s: expected %v, got %v",
					tc.pattern, tc.filename, tc.matches, result)
			}
		})
	}
}

// TestPermissionsResourceModel tests the enhanced resource model.
func TestPermissionsResourceModel(t *testing.T) {
	// Test enhanced FileResourceModel with permissions
	model := EnhancedFileResourceModel{
		FileResourceModel: FileResourceModel{
			ID:         types.StringValue("test-file"),
			Repository: types.StringValue("test-repo"),
			Name:       types.StringValue("ssh-config"),
			SourcePath: types.StringValue("ssh/config"),
			TargetPath: types.StringValue("~/.ssh/config"),
		},
		Permissions: &PermissionsModel{
			Directory: types.StringValue("0700"),
			Files:     types.StringValue("0600"),
			Recursive: types.BoolValue(true),
		},
		PermissionRules: func() types.Map {
			rules := map[string]attr.Value{
				"id_*":        types.StringValue("0600"),
				"*.pub":       types.StringValue("0644"),
				"known_hosts": types.StringValue("0600"),
			}
			mapVal, _ := types.MapValue(types.StringType, rules)
			return mapVal
		}(),
	}

	// Verify permissions model
	if model.Permissions.Directory.ValueString() != "0700" {
		t.Error("Directory permission not set correctly")
	}
	if model.Permissions.Files.ValueString() != "0600" {
		t.Error("Files permission not set correctly")
	}
	if !model.Permissions.Recursive.ValueBool() {
		t.Error("Recursive permission not set correctly")
	}

	// Verify permission rules
	rules := model.PermissionRules.Elements()
	if len(rules) != 3 {
		t.Errorf("Expected 3 permission rules, got %d", len(rules))
	}
}
