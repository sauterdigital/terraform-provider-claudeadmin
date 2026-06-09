package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"

	"github.com/sauterdigital/terraform-provider-anthropic/internal/anthropic"
)

var (
	_ datasource.DataSource              = &APIKeyDataSource{}
	_ datasource.DataSourceWithConfigure = &APIKeyDataSource{}
)

func NewAPIKeyDataSource() datasource.DataSource { return &APIKeyDataSource{} }

type APIKeyDataSource struct{ client *anthropic.Client }

func (d *APIKeyDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_key"
}

func (d *APIKeyDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches a single Anthropic API key by ID.",
		Attributes: map[string]schema.Attribute{
			"id":               schema.StringAttribute{Required: true},
			"name":             schema.StringAttribute{Computed: true},
			"status":           schema.StringAttribute{Computed: true},
			"created_at":       schema.StringAttribute{Computed: true},
			"expires_at":       schema.StringAttribute{Computed: true},
			"partial_key_hint": schema.StringAttribute{Computed: true},
			"workspace_id":     schema.StringAttribute{Computed: true},
			"created_by_id":    schema.StringAttribute{Computed: true},
			"created_by_type":  schema.StringAttribute{Computed: true},
		},
	}
}

func (d *APIKeyDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, diags := clientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	d.client = c
}

func (d *APIKeyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data APIKeyResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	k, err := d.client.GetAPIKey(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read API key", err.Error())
		return
	}
	apiKeyToModel(k, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}
