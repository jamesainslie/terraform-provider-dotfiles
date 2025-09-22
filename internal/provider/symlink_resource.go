// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &SymlinkResource{}

func NewSymlinkResource() resource.Resource {
	return &SymlinkResource{}
}

type SymlinkResource struct {
	client *DotfilesClient
}

type SymlinkResourceModel struct {
	ID            types.String `tfsdk:"id"`
	Repository    types.String `tfsdk:"repository"`
	Name          types.String `tfsdk:"name"`
	SourcePath    types.String `tfsdk:"source_path"`
	TargetPath    types.String `tfsdk:"target_path"`
	ForceUpdate   types.Bool   `tfsdk:"force_update"`
	CreateParents types.Bool   `tfsdk:"create_parents"`
}

func (r *SymlinkResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_symlink"
}

func (r *SymlinkResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages symbolic links to dotfiles",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Symlink identifier",
			},
			"repository": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Repository ID this symlink belongs to",
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Symlink name/identifier",
			},
			"source_path": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Path to source in repository",
			},
			"target_path": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Target symlink path",
			},
			"force_update": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Force update existing symlinks",
			},
			"create_parents": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Create parent directories",
			},
		},
	}
}

func (r *SymlinkResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*DotfilesClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			"Expected *DotfilesClient, got something else.",
		)
		return
	}

	r.client = client
}

func (r *SymlinkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data SymlinkResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.ID = data.Name
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SymlinkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data SymlinkResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SymlinkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data SymlinkResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SymlinkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// TODO: Implement symlink deletion logic
}
