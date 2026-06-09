package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"

	"github.com/sauterdigital/terraform-provider-anthropic/internal/anthropic"
)

var (
	_ datasource.DataSource              = &ExternalKeyDataSource{}
	_ datasource.DataSourceWithConfigure = &ExternalKeyDataSource{}
)

func NewExternalKeyDataSource() datasource.DataSource { return &ExternalKeyDataSource{} }

type ExternalKeyDataSource struct{ client *anthropic.Client }

func (d *ExternalKeyDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_external_key"
}

func (d *ExternalKeyDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches a single CMEK external key by ID.",
		Attributes: map[string]schema.Attribute{
			"id":           schema.StringAttribute{Required: true},
			"display_name": schema.StringAttribute{Computed: true},
			"geo":          schema.StringAttribute{Computed: true},
			"created_at":   schema.StringAttribute{Computed: true},
			"updated_at":   schema.StringAttribute{Computed: true},
			"provider_config": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"type":      schema.StringAttribute{Computed: true},
					"kms_arn":   schema.StringAttribute{Computed: true},
					"role_arn":  schema.StringAttribute{Computed: true},
					"region":    schema.StringAttribute{Computed: true},
					"key_name":  schema.StringAttribute{Computed: true},
					"tenant_id": schema.StringAttribute{Computed: true},
					"vault_uri": schema.StringAttribute{Computed: true},
					"client_id": schema.StringAttribute{Computed: true},
				},
			},
		},
	}
}

func (d *ExternalKeyDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, diags := clientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	d.client = c
}

func (d *ExternalKeyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ExternalKeyModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	k, err := d.client.GetExternalKey(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read external key", err.Error())
		return
	}
	externalKeyToModel(k, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}
