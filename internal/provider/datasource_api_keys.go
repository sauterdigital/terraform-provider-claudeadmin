package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/sauterdigital/terraform-provider-anthropic/internal/anthropic"
)

var (
	_ datasource.DataSource              = &APIKeysDataSource{}
	_ datasource.DataSourceWithConfigure = &APIKeysDataSource{}
)

func NewAPIKeysDataSource() datasource.DataSource { return &APIKeysDataSource{} }

type APIKeysDataSource struct{ client *anthropic.Client }

type APIKeysDataSourceModel struct {
	Status      types.String          `tfsdk:"status"`
	WorkspaceID types.String          `tfsdk:"workspace_id"`
	CreatedByID types.String          `tfsdk:"created_by_user_id"`
	APIKeys     []APIKeyResourceModel `tfsdk:"api_keys"`
}

func (d *APIKeysDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_keys"
}

func (d *APIKeysDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists Anthropic API keys with optional filters.",
		Attributes: map[string]schema.Attribute{
			"status":             schema.StringAttribute{Optional: true, Description: "Filter by status: active, inactive, archived, expired."},
			"workspace_id":       schema.StringAttribute{Optional: true, Description: "Filter by workspace ID."},
			"created_by_user_id": schema.StringAttribute{Optional: true, Description: "Filter by creating user's ID."},
			"api_keys": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":               schema.StringAttribute{Computed: true},
						"name":             schema.StringAttribute{Computed: true},
						"status":           schema.StringAttribute{Computed: true},
						"created_at":       schema.StringAttribute{Computed: true},
						"expires_at":       schema.StringAttribute{Computed: true},
						"partial_key_hint": schema.StringAttribute{Computed: true},
						"workspace_id":     schema.StringAttribute{Computed: true},
						"created_by_id":    schema.StringAttribute{Computed: true},
						"created_by_type":  schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *APIKeysDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, diags := clientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	d.client = c
}

func (d *APIKeysDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data APIKeysDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	list, err := d.client.ListAPIKeys(ctx, anthropic.ListAPIKeysParams{
		Status:          data.Status.ValueString(),
		WorkspaceID:     data.WorkspaceID.ValueString(),
		CreatedByUserID: data.CreatedByID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to list API keys", err.Error())
		return
	}
	out := make([]APIKeyResourceModel, 0, len(list))
	for i := range list {
		var m APIKeyResourceModel
		apiKeyToModel(&list[i], &m)
		out = append(out, m)
	}
	data.APIKeys = out
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}
