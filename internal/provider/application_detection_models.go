// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// DetectionMethodModel describes a detection method configuration.
type DetectionMethodModel struct {
	Type    types.String `tfsdk:"type"`
	Test    types.String `tfsdk:"test"`
	Path    types.String `tfsdk:"path"`
	Name    types.String `tfsdk:"name"`
	Manager types.String `tfsdk:"manager"`
}

// ConfigMappingModel describes a configuration mapping.
type ConfigMappingModel struct {
	TargetPath         types.String `tfsdk:"target_path"`
	TargetPathTemplate types.String `tfsdk:"target_path_template"`
	MergeStrategy      types.String `tfsdk:"merge_strategy"`
	Required           types.Bool   `tfsdk:"required"`
}

// DetectionMethodsModel represents detection methods configuration.
type DetectionMethodsModel struct {
	Methods []DetectionMethodModel `tfsdk:"methods"`
}

// GetDetectionMethodsSchemaBlock returns the schema block for detection methods.
func GetDetectionMethodsSchemaBlock() schema.ListNestedBlock {
	return schema.ListNestedBlock{
		MarkdownDescription: "Application detection methods configuration",
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"type": schema.StringAttribute{
					Required:            true,
					MarkdownDescription: "Detection method type: command, file, brew_cask, package_manager",
				},
				"test": schema.StringAttribute{
					Optional:            true,
					MarkdownDescription: "Command to test for command-type detection",
				},
				"path": schema.StringAttribute{
					Optional:            true,
					MarkdownDescription: "File/directory path to check for file-type detection",
				},
				"name": schema.StringAttribute{
					Optional:            true,
					MarkdownDescription: "Package/cask name for package manager detection",
				},
				"manager": schema.StringAttribute{
					Optional:            true,
					MarkdownDescription: "Package manager to use: brew, apt, yum, pacman",
				},
			},
		},
	}
}

// ConfigMappingsModel represents configuration mappings.
type ConfigMappingsModel struct {
	Mappings map[string]ConfigMappingModel `tfsdk:"mappings"`
}

// GetConfigMappingsSchemaBlock returns the schema block for configuration mappings.
func GetConfigMappingsSchemaBlock() schema.SingleNestedBlock {
	return schema.SingleNestedBlock{
		MarkdownDescription: "Configuration file mappings and strategies",
		Attributes: map[string]schema.Attribute{
			"target_path": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Target path for configuration file",
			},
			"target_path_template": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Template for generating target path ({{.app_support_dir}}, {filename})",
			},
			"merge_strategy": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("replace"),
				MarkdownDescription: "Merge strategy: replace, shallow_merge, deep_merge",
			},
			"required": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Whether this configuration file is required",
			},
		},
	}
}

// GetApplicationDetectionAttributes returns attributes for application detection in existing resources.
func GetApplicationDetectionAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"require_application": schema.StringAttribute{
			Optional:            true,
			MarkdownDescription: "Require this application to be installed before configuring",
		},
		"application_version_min": schema.StringAttribute{
			Optional:            true,
			MarkdownDescription: "Minimum required version of the application",
		},
		"application_version_max": schema.StringAttribute{
			Optional:            true,
			MarkdownDescription: "Maximum supported version of the application",
		},
		"skip_if_app_missing": schema.BoolAttribute{
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(false),
			MarkdownDescription: "Skip this resource if required application is missing",
		},
	}
}
