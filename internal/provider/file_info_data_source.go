// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &FileInfoDataSource{}

func NewFileInfoDataSource() datasource.DataSource {
	return &FileInfoDataSource{}
}

type FileInfoDataSource struct {
	client *DotfilesClient
}

type FileInfoDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Path        types.String `tfsdk:"path"`
	Exists      types.Bool   `tfsdk:"exists"`
	IsSymlink   types.Bool   `tfsdk:"is_symlink"`
	Permissions types.String `tfsdk:"permissions"`
	Size        types.Int64  `tfsdk:"size"`
}

func (d *FileInfoDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_file_info"
}

func (d *FileInfoDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "File information data source",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Data source identifier",
			},
			"path": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Path to the file to examine",
			},
			"exists": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the file exists",
			},
			"is_symlink": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the file is a symlink",
			},
			"permissions": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "File permissions",
			},
			"size": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "File size in bytes",
			},
		},
	}
}

func (d *FileInfoDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*DotfilesClient)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Data Source Configure Type", "Expected *DotfilesClient")
		return
	}
	d.client = client
}

func (d *FileInfoDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data FileInfoDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// TODO: Implement actual file info reading logic
	data.ID = data.Path
	data.Exists = types.BoolValue(false)
	data.IsSymlink = types.BoolValue(false)
	data.Permissions = types.StringValue("0644")
	data.Size = types.Int64Value(0)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
