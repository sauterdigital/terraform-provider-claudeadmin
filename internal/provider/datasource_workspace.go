package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/sauterdigital/terraform-provider-claude-admin/internal/anthropic"
)

var (
	_ datasource.DataSource              = &WorkspaceDataSource{}
	_ datasource.DataSourceWithConfigure = &WorkspaceDataSource{}
)

func NewWorkspaceDataSource() datasource.DataSource { return &WorkspaceDataSource{} }

type WorkspaceDataSource struct{ client *anthropic.Client }

func (d *WorkspaceDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workspace"
}

func (d *WorkspaceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches a single Anthropic workspace by ID.",
		Attributes: map[string]schema.Attribute{
			"id":              schema.StringAttribute{Required: true, Description: "Workspace ID."},
			"name":            schema.StringAttribute{Computed: true},
			"created_at":      schema.StringAttribute{Computed: true},
			"archived_at":     schema.StringAttribute{Computed: true},
			"display_color":   schema.StringAttribute{Computed: true},
			"compartment_id":  schema.StringAttribute{Computed: true},
			"external_key_id": schema.StringAttribute{Computed: true},
			"tags": schema.MapAttribute{
				Computed:    true,
				ElementType: types.StringType,
			},
			"data_residency": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"workspace_geo":         schema.StringAttribute{Computed: true},
					"default_inference_geo": schema.StringAttribute{Computed: true},
					"allowed_inference_geos": schema.ListAttribute{
						Computed:    true,
						ElementType: types.StringType,
					},
				},
			},
		},
	}
}

func (d *WorkspaceDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, diags := clientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	d.client = c
}

func (d *WorkspaceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data WorkspaceResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ws, err := d.client.GetWorkspace(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read workspace", err.Error())
		return
	}
	resp.Diagnostics.Append(workspaceToModel(ctx, ws, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}
