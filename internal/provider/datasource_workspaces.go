package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/sauterdigital/terraform-provider-claude-admin/internal/anthropic"
)

var (
	_ datasource.DataSource              = &WorkspacesDataSource{}
	_ datasource.DataSourceWithConfigure = &WorkspacesDataSource{}
)

func NewWorkspacesDataSource() datasource.DataSource { return &WorkspacesDataSource{} }

type WorkspacesDataSource struct{ client *anthropic.Client }

type WorkspacesDataSourceModel struct {
	IncludeArchived types.Bool               `tfsdk:"include_archived"`
	Workspaces      []WorkspaceResourceModel `tfsdk:"workspaces"`
}

func (d *WorkspacesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workspaces"
}

func (d *WorkspacesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all Anthropic workspaces in the organization.",
		Attributes: map[string]schema.Attribute{
			"include_archived": schema.BoolAttribute{
				Optional:    true,
				Description: "Whether to include archived workspaces in the response.",
			},
			"workspaces": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":              schema.StringAttribute{Computed: true},
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
				},
			},
		},
	}
}

func (d *WorkspacesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, diags := clientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	d.client = c
}

func (d *WorkspacesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data WorkspacesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	list, err := d.client.ListWorkspaces(ctx, anthropic.ListWorkspacesParams{
		IncludeArchived: data.IncludeArchived.ValueBool(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to list workspaces", err.Error())
		return
	}

	out := make([]WorkspaceResourceModel, 0, len(list))
	for i := range list {
		var m WorkspaceResourceModel
		resp.Diagnostics.Append(workspaceToModel(ctx, &list[i], &m)...)
		if resp.Diagnostics.HasError() {
			return
		}
		out = append(out, m)
	}
	data.Workspaces = out
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}
