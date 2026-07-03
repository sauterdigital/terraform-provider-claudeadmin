package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"

	"github.com/sauterdigital/terraform-provider-claude-admin/internal/anthropic"
)

var (
	_ datasource.DataSource              = &WorkspaceMemberDataSource{}
	_ datasource.DataSourceWithConfigure = &WorkspaceMemberDataSource{}
)

func NewWorkspaceMemberDataSource() datasource.DataSource { return &WorkspaceMemberDataSource{} }

type WorkspaceMemberDataSource struct{ client *anthropic.Client }

func (d *WorkspaceMemberDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workspace_member"
}

func (d *WorkspaceMemberDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches a single workspace membership.",
		Attributes: map[string]schema.Attribute{
			"id":             schema.StringAttribute{Computed: true},
			"workspace_id":   schema.StringAttribute{Required: true},
			"user_id":        schema.StringAttribute{Required: true},
			"workspace_role": schema.StringAttribute{Computed: true},
		},
	}
}

func (d *WorkspaceMemberDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, diags := clientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	d.client = c
}

func (d *WorkspaceMemberDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data WorkspaceMemberResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	m, err := d.client.GetWorkspaceMember(ctx, data.WorkspaceID.ValueString(), data.UserID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read workspace member", err.Error())
		return
	}
	workspaceMemberToModel(m, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}
