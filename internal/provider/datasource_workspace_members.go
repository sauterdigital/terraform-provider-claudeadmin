package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/sauterdigital/terraform-provider-anthropic/internal/anthropic"
)

var (
	_ datasource.DataSource              = &WorkspaceMembersDataSource{}
	_ datasource.DataSourceWithConfigure = &WorkspaceMembersDataSource{}
)

func NewWorkspaceMembersDataSource() datasource.DataSource { return &WorkspaceMembersDataSource{} }

type WorkspaceMembersDataSource struct{ client *anthropic.Client }

type WorkspaceMembersDataSourceModel struct {
	WorkspaceID types.String                   `tfsdk:"workspace_id"`
	Members     []WorkspaceMemberResourceModel `tfsdk:"members"`
}

func (d *WorkspaceMembersDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workspace_members"
}

func (d *WorkspaceMembersDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all members of a workspace.",
		Attributes: map[string]schema.Attribute{
			"workspace_id": schema.StringAttribute{Required: true},
			"members": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":             schema.StringAttribute{Computed: true},
						"workspace_id":   schema.StringAttribute{Computed: true},
						"user_id":        schema.StringAttribute{Computed: true},
						"workspace_role": schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *WorkspaceMembersDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, diags := clientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	d.client = c
}

func (d *WorkspaceMembersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data WorkspaceMembersDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	list, err := d.client.ListWorkspaceMembers(ctx, data.WorkspaceID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to list workspace members", err.Error())
		return
	}
	out := make([]WorkspaceMemberResourceModel, 0, len(list))
	for i := range list {
		var m WorkspaceMemberResourceModel
		workspaceMemberToModel(&list[i], &m)
		out = append(out, m)
	}
	data.Members = out
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

var _ = anthropic.WorkspaceMember{}
