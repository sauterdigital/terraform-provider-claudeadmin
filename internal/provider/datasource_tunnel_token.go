package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/sauterdigital/terraform-provider-anthropic/internal/anthropic"
)

var (
	_ datasource.DataSource              = &TunnelTokenDataSource{}
	_ datasource.DataSourceWithConfigure = &TunnelTokenDataSource{}
)

func NewTunnelTokenDataSource() datasource.DataSource { return &TunnelTokenDataSource{} }

type TunnelTokenDataSource struct{ client *anthropic.Client }

type TunnelTokenModel struct {
	TunnelID    types.String `tfsdk:"tunnel_id"`
	ID          types.String `tfsdk:"id"`
	TunnelToken types.String `tfsdk:"tunnel_token"`
}

func (d *TunnelTokenDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tunnel_token"
}

func (d *TunnelTokenDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reveals the current connection token for an MCP tunnel. **Sensitive** — `tunnel_token` is the actual secret that MCP servers use to connect. Pipe it into a Kubernetes Secret, Vault, or similar via output. To rotate the token declaratively, use the `anthropic_tunnel_token_rotation` resource (change its `rotation_id` to trigger). Beta — requires OAuth Bearer auth.",
		Attributes: map[string]schema.Attribute{
			"tunnel_id":    schema.StringAttribute{Required: true},
			"id":           schema.StringAttribute{Computed: true, Description: "Stable identifier for the current token value. Changes only on rotation."},
			"tunnel_token": schema.StringAttribute{Computed: true, Sensitive: true},
		},
	}
}

func (d *TunnelTokenDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, diags := clientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	d.client = c
}

func (d *TunnelTokenDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data TunnelTokenModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	t, err := d.client.RevealTunnelToken(ctx, data.TunnelID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to reveal tunnel token", err.Error())
		return
	}
	data.ID = types.StringValue(t.ID)
	data.TunnelToken = types.StringValue(t.TunnelToken)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}
