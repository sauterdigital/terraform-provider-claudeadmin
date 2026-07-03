package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/sauterdigital/terraform-provider-claude-admin/internal/anthropic"
)

// MCP Tunnels can only be CREATED via Console/CLI; the Admin API exposes
// read + token management + archive. So this is a data source, not a resource.

type TunnelModel struct {
	ID          types.String `tfsdk:"id"`
	DisplayName types.String `tfsdk:"display_name"`
	Domain      types.String `tfsdk:"domain"`
	WorkspaceID types.String `tfsdk:"workspace_id"`
	CreatedAt   types.String `tfsdk:"created_at"`
	ArchivedAt  types.String `tfsdk:"archived_at"`
}

// Single tunnel

var (
	_ datasource.DataSource              = &TunnelDataSource{}
	_ datasource.DataSourceWithConfigure = &TunnelDataSource{}
)

func NewTunnelDataSource() datasource.DataSource { return &TunnelDataSource{} }

type TunnelDataSource struct{ client *anthropic.Client }

func (d *TunnelDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tunnel"
}

func (d *TunnelDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches a single MCP tunnel by ID. Beta — requires OAuth Bearer auth.",
		Attributes: map[string]schema.Attribute{
			"id":           schema.StringAttribute{Required: true},
			"display_name": schema.StringAttribute{Computed: true},
			"domain":       schema.StringAttribute{Computed: true},
			"workspace_id": schema.StringAttribute{Computed: true},
			"created_at":   schema.StringAttribute{Computed: true},
			"archived_at":  schema.StringAttribute{Computed: true},
		},
	}
}

func (d *TunnelDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, diags := clientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	d.client = c
}

func (d *TunnelDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data TunnelModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	t, err := d.client.GetTunnel(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read tunnel", err.Error())
		return
	}
	tunnelToModel(t, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

// List tunnels

var (
	_ datasource.DataSource              = &TunnelsDataSource{}
	_ datasource.DataSourceWithConfigure = &TunnelsDataSource{}
)

func NewTunnelsDataSource() datasource.DataSource { return &TunnelsDataSource{} }

type TunnelsDataSource struct{ client *anthropic.Client }

type TunnelsListModel struct {
	IncludeArchived types.Bool    `tfsdk:"include_archived"`
	WorkspaceID     types.String  `tfsdk:"workspace_id"`
	Tunnels         []TunnelModel `tfsdk:"tunnels"`
}

func (d *TunnelsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tunnels"
}

func (d *TunnelsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists MCP tunnels. Beta — requires OAuth Bearer auth.",
		Attributes: map[string]schema.Attribute{
			"include_archived": schema.BoolAttribute{Optional: true},
			"workspace_id":     schema.StringAttribute{Optional: true},
			"tunnels": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":           schema.StringAttribute{Computed: true},
						"display_name": schema.StringAttribute{Computed: true},
						"domain":       schema.StringAttribute{Computed: true},
						"workspace_id": schema.StringAttribute{Computed: true},
						"created_at":   schema.StringAttribute{Computed: true},
						"archived_at":  schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *TunnelsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, diags := clientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	d.client = c
}

func (d *TunnelsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data TunnelsListModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	list, err := d.client.ListTunnels(ctx, anthropic.ListTunnelsParams{
		IncludeArchived: data.IncludeArchived.ValueBool(),
		WorkspaceID:     data.WorkspaceID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to list tunnels", err.Error())
		return
	}
	out := make([]TunnelModel, 0, len(list))
	for i := range list {
		var m TunnelModel
		tunnelToModel(&list[i], &m)
		out = append(out, m)
	}
	data.Tunnels = out
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

// List certificates for a tunnel

var (
	_ datasource.DataSource              = &TunnelCertificatesDataSource{}
	_ datasource.DataSourceWithConfigure = &TunnelCertificatesDataSource{}
)

func NewTunnelCertificatesDataSource() datasource.DataSource {
	return &TunnelCertificatesDataSource{}
}

type TunnelCertificatesDataSource struct{ client *anthropic.Client }

type TunnelCertificatesListModel struct {
	TunnelID        types.String             `tfsdk:"tunnel_id"`
	IncludeArchived types.Bool               `tfsdk:"include_archived"`
	Certificates    []TunnelCertificateModel `tfsdk:"certificates"`
}

func (d *TunnelCertificatesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tunnel_certificates"
}

func (d *TunnelCertificatesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists certificates registered against an MCP tunnel.",
		Attributes: map[string]schema.Attribute{
			"tunnel_id":        schema.StringAttribute{Required: true},
			"include_archived": schema.BoolAttribute{Optional: true},
			"certificates": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":                 schema.StringAttribute{Computed: true},
						"tunnel_id":          schema.StringAttribute{Computed: true},
						"ca_certificate_pem": schema.StringAttribute{Computed: true, Sensitive: true},
						"fingerprint":        schema.StringAttribute{Computed: true},
						"created_at":         schema.StringAttribute{Computed: true},
						"expires_at":         schema.StringAttribute{Computed: true},
						"archived_at":        schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *TunnelCertificatesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, diags := clientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	d.client = c
}

func (d *TunnelCertificatesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data TunnelCertificatesListModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	list, err := d.client.ListTunnelCertificates(ctx, data.TunnelID.ValueString(), data.IncludeArchived.ValueBool())
	if err != nil {
		resp.Diagnostics.AddError("Failed to list tunnel certificates", err.Error())
		return
	}
	out := make([]TunnelCertificateModel, 0, len(list))
	for i := range list {
		var m TunnelCertificateModel
		tunnelCertToModel(&list[i], &m)
		// The PEM is write-only — API doesn't echo it back in reads
		m.CACertificatePEM = types.StringNull()
		out = append(out, m)
	}
	data.Certificates = out
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func tunnelToModel(t *anthropic.Tunnel, m *TunnelModel) {
	m.ID = types.StringValue(t.ID)
	m.DisplayName = optionalStringValue(t.DisplayName)
	m.Domain = types.StringValue(t.Domain)
	m.WorkspaceID = optionalStringValue(t.WorkspaceID)
	m.CreatedAt = types.StringValue(t.CreatedAt)
	m.ArchivedAt = optionalStringValue(t.ArchivedAt)
}
