// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &DirectoryResource{}

func NewDirectoryResource() resource.Resource {
	return &DirectoryResource{}
}

type DirectoryResource struct {
	client *DotfilesClient
}

type DirectoryResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	Repository          types.String `tfsdk:"repository"`
	Name                types.String `tfsdk:"name"`
	SourcePath          types.String `tfsdk:"source_path"`
	TargetPath          types.String `tfsdk:"target_path"`
	Recursive           types.Bool   `tfsdk:"recursive"`
	PreservePermissions types.Bool   `tfsdk:"preserve_permissions"`
}

func (r *DirectoryResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_directory"
}

func (r *DirectoryResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages directory structures and their contents",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Directory identifier",
			},
			"repository": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Repository ID this directory belongs to",
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Directory name/identifier",
			},
			"source_path": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Path to source directory in repository",
			},
			"target_path": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Target directory path",
			},
			"recursive": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Process directory recursively",
			},
			"preserve_permissions": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Preserve file permissions",
			},
		},
	}
}

func (r *DirectoryResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*DotfilesClient)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", "Expected *DotfilesClient")
		return
	}
	r.client = client
}

func (r *DirectoryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data DirectoryResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.ID = data.Name
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DirectoryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data DirectoryResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DirectoryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data DirectoryResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set ID and save state
	data.ID = data.Name
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DirectoryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// TODO: Implement directory deletion logic
}
