// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &SystemDataSource{}

func NewSystemDataSource() datasource.DataSource {
	return &SystemDataSource{}
}

type SystemDataSource struct {
	client *DotfilesClient
}

type SystemDataSourceModel struct {
	ID           types.String `tfsdk:"id"`
	Platform     types.String `tfsdk:"platform"`
	Architecture types.String `tfsdk:"architecture"`
	HomeDir      types.String `tfsdk:"home_dir"`
	ConfigDir    types.String `tfsdk:"config_dir"`
}

func (d *SystemDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_system"
}

func (d *SystemDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "System information data source",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Data source identifier",
			},
			"platform": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Operating system platform",
			},
			"architecture": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "System architecture",
			},
			"home_dir": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "User home directory",
			},
			"config_dir": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "User config directory",
			},
		},
	}
}

func (d *SystemDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *SystemDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data SystemDataSourceModel

	data.ID = types.StringValue("system")
	data.Platform = types.StringValue(d.client.Platform)
	data.Architecture = types.StringValue(d.client.Architecture)
	data.HomeDir = types.StringValue(d.client.HomeDir)
	data.ConfigDir = types.StringValue(d.client.ConfigDir)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
