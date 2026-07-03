package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"

	"github.com/sauterdigital/terraform-provider-claude-admin/internal/anthropic"
)

var (
	_ datasource.DataSource              = &InviteDataSource{}
	_ datasource.DataSourceWithConfigure = &InviteDataSource{}
)

func NewInviteDataSource() datasource.DataSource { return &InviteDataSource{} }

type InviteDataSource struct{ client *anthropic.Client }

func (d *InviteDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_invite"
}

func (d *InviteDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches a single Anthropic invite by ID.",
		Attributes: map[string]schema.Attribute{
			"id":         schema.StringAttribute{Required: true},
			"email":      schema.StringAttribute{Computed: true},
			"role":       schema.StringAttribute{Computed: true},
			"status":     schema.StringAttribute{Computed: true},
			"invited_at": schema.StringAttribute{Computed: true},
			"expires_at": schema.StringAttribute{Computed: true},
		},
	}
}

func (d *InviteDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, diags := clientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	d.client = c
}

func (d *InviteDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data InviteResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	inv, err := d.client.GetInvite(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read invite", err.Error())
		return
	}
	inviteToModel(inv, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}
