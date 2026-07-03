package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"

	"github.com/sauterdigital/terraform-provider-claude-admin/internal/anthropic"
)

var (
	_ datasource.DataSource              = &InvitesDataSource{}
	_ datasource.DataSourceWithConfigure = &InvitesDataSource{}
)

func NewInvitesDataSource() datasource.DataSource { return &InvitesDataSource{} }

type InvitesDataSource struct{ client *anthropic.Client }

type InvitesDataSourceModel struct {
	Invites []InviteResourceModel `tfsdk:"invites"`
}

func (d *InvitesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_invites"
}

func (d *InvitesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all Anthropic invites in the organization.",
		Attributes: map[string]schema.Attribute{
			"invites": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":         schema.StringAttribute{Computed: true},
						"email":      schema.StringAttribute{Computed: true},
						"role":       schema.StringAttribute{Computed: true},
						"status":     schema.StringAttribute{Computed: true},
						"invited_at": schema.StringAttribute{Computed: true},
						"expires_at": schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *InvitesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, diags := clientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	d.client = c
}

func (d *InvitesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data InvitesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	list, err := d.client.ListInvites(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list invites", err.Error())
		return
	}
	out := make([]InviteResourceModel, 0, len(list))
	for i := range list {
		var m InviteResourceModel
		inviteToModel(&list[i], &m)
		out = append(out, m)
	}
	data.Invites = out
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

var _ = anthropic.Invite{}
