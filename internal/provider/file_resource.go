// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &FileResource{}

func NewFileResource() resource.Resource {
	return &FileResource{}
}

// FileResource defines the resource implementation.
type FileResource struct {
	client *DotfilesClient
}

// FileResourceModel describes the resource data model.
type FileResourceModel struct {
	ID            types.String `tfsdk:"id"`
	Repository    types.String `tfsdk:"repository"`
	Name          types.String `tfsdk:"name"`
	SourcePath    types.String `tfsdk:"source_path"`
	TargetPath    types.String `tfsdk:"target_path"`
	IsTemplate    types.Bool   `tfsdk:"is_template"`
	FileMode      types.String `tfsdk:"file_mode"`
	BackupEnabled types.Bool   `tfsdk:"backup_enabled"`
}

func (r *FileResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_file"
}

func (r *FileResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages individual dotfiles",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "File identifier",
			},
			"repository": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Repository ID this file belongs to",
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "File name/identifier",
			},
			"source_path": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Path to source file in repository",
			},
			"target_path": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Target path where file should be placed",
			},
			"is_template": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Whether the file should be processed as a template",
			},
			"file_mode": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "File permissions (e.g., '0644')",
			},
			"backup_enabled": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Whether to backup existing files",
			},
		},
	}
}

func (r *FileResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*DotfilesClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			"Expected *DotfilesClient, got something else. Please report this issue to the provider developers.",
		)
		return
	}

	r.client = client
}

func (r *FileResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data FileResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// TODO: Implement actual file creation logic
	data.ID = data.Name

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *FileResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data FileResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// TODO: Implement actual file read logic
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *FileResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data FileResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// TODO: Implement actual file update logic
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *FileResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// TODO: Implement actual file deletion logic
}
