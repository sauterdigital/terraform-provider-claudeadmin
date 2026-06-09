package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"

	"github.com/sauterdigital/terraform-provider-anthropic/internal/anthropic"
)

var (
	_ datasource.DataSource              = &ExternalKeysDataSource{}
	_ datasource.DataSourceWithConfigure = &ExternalKeysDataSource{}
)

func NewExternalKeysDataSource() datasource.DataSource { return &ExternalKeysDataSource{} }

type ExternalKeysDataSource struct{ client *anthropic.Client }

type ExternalKeysDataSourceModel struct {
	ExternalKeys []ExternalKeyModel `tfsdk:"external_keys"`
}

func (d *ExternalKeysDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_external_keys"
}

func (d *ExternalKeysDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all CMEK external keys in the organization.",
		Attributes: map[string]schema.Attribute{
			"external_keys": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":           schema.StringAttribute{Computed: true},
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
				},
			},
		},
	}
}

func (d *ExternalKeysDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, diags := clientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	d.client = c
}

func (d *ExternalKeysDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ExternalKeysDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	list, err := d.client.ListExternalKeys(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list external keys", err.Error())
		return
	}
	out := make([]ExternalKeyModel, 0, len(list))
	for i := range list {
		var m ExternalKeyModel
		externalKeyToModel(&list[i], &m)
		out = append(out, m)
	}
	data.ExternalKeys = out
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}
