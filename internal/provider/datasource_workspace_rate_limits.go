package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/sauterdigital/terraform-provider-anthropic/internal/anthropic"
)

var (
	_ datasource.DataSource              = &WorkspaceRateLimitsDataSource{}
	_ datasource.DataSourceWithConfigure = &WorkspaceRateLimitsDataSource{}
)

func NewWorkspaceRateLimitsDataSource() datasource.DataSource {
	return &WorkspaceRateLimitsDataSource{}
}

type WorkspaceRateLimitsDataSource struct{ client *anthropic.Client }

type WorkspaceRateLimitsDataSourceModel struct {
	WorkspaceID types.String          `tfsdk:"workspace_id"`
	GroupType   types.String          `tfsdk:"group_type"`
	Groups      []RateLimitGroupModel `tfsdk:"groups"`
}

type RateLimitGroupModel struct {
	GroupType types.String          `tfsdk:"group_type"`
	Models    []types.String        `tfsdk:"models"`
	Limits    []RateLimitValueModel `tfsdk:"limits"`
}

type RateLimitValueModel struct {
	Type     types.String `tfsdk:"type"`
	Value    types.Int64  `tfsdk:"value"`
	OrgLimit types.Int64  `tfsdk:"org_limit"`
}

func (d *WorkspaceRateLimitsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workspace_rate_limits"
}

func (d *WorkspaceRateLimitsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists rate-limit OVERRIDES configured on a workspace. Groups inherited from the organization are NOT returned — absence of a group means inherit, not no-limit.",
		Attributes: map[string]schema.Attribute{
			"workspace_id": schema.StringAttribute{Required: true},
			"group_type": schema.StringAttribute{
				Optional:    true,
				Description: "Optional filter: model_group, batch, token_count, files, skills, web_search.",
				Validators: []validator.String{
					stringvalidator.OneOf("model_group", "batch", "token_count", "files", "skills", "web_search"),
				},
			},
			"groups": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"group_type": schema.StringAttribute{Computed: true},
						"models": schema.ListAttribute{
							Computed:    true,
							ElementType: types.StringType,
						},
						"limits": schema.ListNestedAttribute{
							Computed: true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"type":      schema.StringAttribute{Computed: true},
									"value":     schema.Int64Attribute{Computed: true},
									"org_limit": schema.Int64Attribute{Computed: true},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (d *WorkspaceRateLimitsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, diags := clientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	d.client = c
}

func (d *WorkspaceRateLimitsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data WorkspaceRateLimitsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	groups, err := d.client.ListWorkspaceRateLimits(ctx, data.WorkspaceID.ValueString(), data.GroupType.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to list workspace rate limits", err.Error())
		return
	}

	out := make([]RateLimitGroupModel, 0, len(groups))
	for _, g := range groups {
		models := make([]types.String, 0, len(g.Models))
		for _, m := range g.Models {
			models = append(models, types.StringValue(m))
		}
		limits := make([]RateLimitValueModel, 0, len(g.Limits))
		for _, l := range g.Limits {
			orgLimit := types.Int64Null()
			if l.OrgLimit != nil {
				orgLimit = types.Int64Value(*l.OrgLimit)
			}
			limits = append(limits, RateLimitValueModel{
				Type:     types.StringValue(l.Type),
				Value:    types.Int64Value(l.Value),
				OrgLimit: orgLimit,
			})
		}
		out = append(out, RateLimitGroupModel{
			GroupType: types.StringValue(g.GroupType),
			Models:    models,
			Limits:    limits,
		})
	}
	data.Groups = out
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}
