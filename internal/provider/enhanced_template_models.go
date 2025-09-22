// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// EnhancedFileResourceModelWithTemplate extends EnhancedFileResourceModelWithBackup with template features.
type EnhancedFileResourceModelWithTemplate struct {
	EnhancedFileResourceModelWithBackup
	TemplateEngine       types.String `tfsdk:"template_engine"`
	PlatformTemplateVars types.Map    `tfsdk:"platform_template_vars"`
	TemplateFunctions    types.Map    `tfsdk:"template_functions"`
}

// EnhancedSymlinkResourceModelWithTemplate extends EnhancedSymlinkResourceModelWithBackup with template features.
type EnhancedSymlinkResourceModelWithTemplate struct {
	EnhancedSymlinkResourceModelWithBackup
	TemplateEngine       types.String `tfsdk:"template_engine"`
	PlatformTemplateVars types.Map    `tfsdk:"platform_template_vars"`
	TemplateFunctions    types.Map    `tfsdk:"template_functions"`
}

// EnhancedFileResourceModelWithApplicationDetection extends template model with app detection.
type EnhancedFileResourceModelWithApplicationDetection struct {
	EnhancedFileResourceModelWithTemplate
	RequireApplication    types.String `tfsdk:"require_application"`
	ApplicationVersionMin types.String `tfsdk:"application_version_min"`
	ApplicationVersionMax types.String `tfsdk:"application_version_max"`
	SkipIfAppMissing      types.Bool   `tfsdk:"skip_if_app_missing"`
}

// ApplicationDetectionConfig represents application detection configuration.
type ApplicationDetectionConfig struct {
	RequiredApplication string
	MinVersion          string
	MaxVersion          string
	SkipIfMissing       bool
}

// GetEnhancedTemplateAttributes returns template-related schema attributes.
func GetEnhancedTemplateAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"template_engine": schema.StringAttribute{
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString("go"),
			MarkdownDescription: "Template engine to use: go (default), handlebars, or mustache",
		},
		"platform_template_vars": schema.MapAttribute{
			Optional: true,
			ElementType: types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"credential_helper": types.StringType,
					"diff_tool":         types.StringType,
					"homebrew_path":     types.StringType,
					"config_dir":        types.StringType,
					"shell":             types.StringType,
				},
			},
			MarkdownDescription: "Platform-specific template variables (macos, linux, windows)",
		},
		"template_functions": schema.MapAttribute{
			Optional:            true,
			ElementType:         types.StringType,
			MarkdownDescription: "Custom template functions (name -> value mappings)",
		},
	}
}
