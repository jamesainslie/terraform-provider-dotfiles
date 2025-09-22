// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

package provider

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// PermissionsModel defines the permissions configuration block.
type PermissionsModel struct {
	Directory types.String `tfsdk:"directory"`
	Files     types.String `tfsdk:"files"`
	Recursive types.Bool   `tfsdk:"recursive"`
}

// EnhancedFileResourceModel extends FileResourceModel with permission management.
type EnhancedFileResourceModel struct {
	FileResourceModel
	Permissions     *PermissionsModel `tfsdk:"permissions"`
	PermissionRules types.Map         `tfsdk:"permission_rules"`

	// Post-creation hooks (Priority 2 feature)
	PostCreateCommands types.List `tfsdk:"post_create_commands"`
	PostUpdateCommands types.List `tfsdk:"post_update_commands"`
	PreDestroyCommands types.List `tfsdk:"pre_destroy_commands"`
}

// EnhancedSymlinkResourceModel extends SymlinkResourceModel with permission management.
type EnhancedSymlinkResourceModel struct {
	SymlinkResourceModel
	Permissions     *PermissionsModel `tfsdk:"permissions"`
	PermissionRules types.Map         `tfsdk:"permission_rules"`

	// Post-creation hooks (Priority 2 feature)
	PostCreateCommands types.List `tfsdk:"post_create_commands"`
	PostUpdateCommands types.List `tfsdk:"post_update_commands"`
	PreDestroyCommands types.List `tfsdk:"pre_destroy_commands"`
}

// GetPermissionsSchemaBlock returns the schema block for permissions.
func GetPermissionsSchemaBlock() schema.SingleNestedBlock {
	return schema.SingleNestedBlock{
		MarkdownDescription: "Permission management for files and directories",
		Attributes: map[string]schema.Attribute{
			"directory": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("0755"),
				MarkdownDescription: "Directory permission mode (e.g., '0755')",
			},
			"files": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("0644"),
				MarkdownDescription: "File permission mode (e.g., '0644')",
			},
			"recursive": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				MarkdownDescription: "Apply permissions recursively to subdirectories and files",
			},
		},
	}
}

// GetPermissionRulesAttribute returns the schema attribute for permission rules.
func GetPermissionRulesAttribute() schema.MapAttribute {
	return schema.MapAttribute{
		Optional:            true,
		ElementType:         types.StringType,
		MarkdownDescription: "Pattern-based permission rules (e.g., 'id_*' = '0600')",
	}
}

// GetPostHooksAttributes returns the schema attributes for post-creation hooks.
func GetPostHooksAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"post_create_commands": schema.ListAttribute{
			Optional:            true,
			ElementType:         types.StringType,
			MarkdownDescription: "Commands to execute after resource creation",
		},
		"post_update_commands": schema.ListAttribute{
			Optional:            true,
			ElementType:         types.StringType,
			MarkdownDescription: "Commands to execute after resource update",
		},
		"pre_destroy_commands": schema.ListAttribute{
			Optional:            true,
			ElementType:         types.StringType,
			MarkdownDescription: "Commands to execute before resource destruction",
		},
	}
}

// parsePermission parses a permission string (e.g., "0644") to uint32.
func parsePermission(perm string) (uint32, error) {
	if perm == "" {
		return 0, fmt.Errorf("permission cannot be empty")
	}

	// Remove leading zeros for parsing, but preserve them for validation
	trimmed := strings.TrimLeft(perm, "0")
	if trimmed == "" {
		trimmed = "0"
	}

	// Parse as octal
	parsed, err := strconv.ParseUint(trimmed, 8, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid permission format %q: %w", perm, err)
	}

	// Validate permission range (0-777)
	if parsed > 0777 {
		return 0, fmt.Errorf("permission %q is out of valid range (0-777)", perm)
	}

	return uint32(parsed), nil
}

// matchesPermissionPattern checks if a filename matches a permission rule pattern.
func matchesPermissionPattern(pattern, filename string) bool {
	// Simple glob matching - can be enhanced with more sophisticated patterns
	matched, err := filepath.Match(pattern, filename)
	if err != nil {
		return false
	}
	return matched
}

// isMoreSpecific determines if pattern1 is more specific than pattern2.
func isMoreSpecific(pattern1, pattern2 string) bool {
	// Count wildcards - fewer wildcards means more specific
	wildcards1 := strings.Count(pattern1, "*") + strings.Count(pattern1, "?")
	wildcards2 := strings.Count(pattern2, "*") + strings.Count(pattern2, "?")

	if wildcards1 != wildcards2 {
		return wildcards1 < wildcards2
	}

	// If same number of wildcards, prefer exact character matches
	exactChars1 := len(pattern1) - wildcards1
	exactChars2 := len(pattern2) - wildcards2

	return exactChars1 > exactChars2
}

// ApplyPermissionRules applies permission rules to a file based on patterns.
func ApplyPermissionRules(filename string, rules types.Map, defaultPerm string) (string, error) {
	if rules.IsNull() || rules.IsUnknown() {
		return defaultPerm, nil
	}

	elements := rules.Elements()
	bestMatch := ""
	bestPerm := ""

	// Find the most specific pattern match
	for pattern, permValue := range elements {
		if strPerm, ok := permValue.(types.String); ok {
			if matchesPermissionPattern(pattern, filename) {
				// Prefer more specific patterns (patterns with more characters are generally more specific)
				if bestMatch == "" || len(pattern) > len(bestMatch) || isMoreSpecific(pattern, bestMatch) {
					// Validate the permission
					if _, err := parsePermission(strPerm.ValueString()); err != nil {
						return defaultPerm, fmt.Errorf("invalid permission in rule %s: %w", pattern, err)
					}
					bestMatch = pattern
					bestPerm = strPerm.ValueString()
				}
			}
		}
	}

	if bestMatch != "" {
		return bestPerm, nil
	}

	return defaultPerm, nil
}

// ValidatePermissionsModel validates the permissions configuration.
func ValidatePermissionsModel(permissions *PermissionsModel) error {
	if permissions == nil {
		return nil
	}

	// Validate directory permission
	if !permissions.Directory.IsNull() {
		if _, err := parsePermission(permissions.Directory.ValueString()); err != nil {
			return fmt.Errorf("invalid directory permission: %w", err)
		}
	}

	// Validate file permission
	if !permissions.Files.IsNull() {
		if _, err := parsePermission(permissions.Files.ValueString()); err != nil {
			return fmt.Errorf("invalid files permission: %w", err)
		}
	}

	return nil
}

// ValidatePermissionRules validates permission rules map.
func ValidatePermissionRules(rules types.Map) error {
	if rules.IsNull() || rules.IsUnknown() {
		return nil
	}

	elements := rules.Elements()
	for pattern, permValue := range elements {
		if strPerm, ok := permValue.(types.String); ok {
			if _, err := parsePermission(strPerm.ValueString()); err != nil {
				return fmt.Errorf("invalid permission in rule %s: %w", pattern, err)
			}
		} else {
			return fmt.Errorf("permission rule %s has invalid type", pattern)
		}
	}

	return nil
}
