package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/sauterdigital/terraform-provider-claude-admin/internal/anthropic"
)

var (
	_ datasource.DataSource              = &OrganizationRateLimitsDataSource{}
	_ datasource.DataSourceWithConfigure = &OrganizationRateLimitsDataSource{}
)

func NewOrganizationRateLimitsDataSource() datasource.DataSource {
	return &OrganizationRateLimitsDataSource{}
}

type OrganizationRateLimitsDataSource struct{ client *anthropic.Client }

type OrganizationRateLimitsDataSourceModel struct {
	GroupType types.String             `tfsdk:"group_type"`
	Model     types.String             `tfsdk:"model"`
	Groups    []OrgRateLimitGroupModel `tfsdk:"groups"`
}

type OrgRateLimitGroupModel struct {
	GroupType types.String             `tfsdk:"group_type"`
	Models    []types.String           `tfsdk:"models"`
	Limits    []OrgRateLimitValueModel `tfsdk:"limits"`
}

type OrgRateLimitValueModel struct {
	Type  types.String `tfsdk:"type"`
	Value types.Int64  `tfsdk:"value"`
}

func (d *OrganizationRateLimitsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_rate_limits"
}

func (d *OrganizationRateLimitsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists Messages API rate limits configured at the ORGANIZATION level. Unlike `anthropic_workspace_rate_limits`, this returns every group active on the org (not just overrides) — it's the source of truth for what the org's baseline limits actually are.",
		Attributes: map[string]schema.Attribute{
			"group_type": schema.StringAttribute{
				Optional:    true,
				Description: "Optional filter: model_group, batch, token_count, files, skills, web_search.",
				Validators: []validator.String{
					stringvalidator.OneOf("model_group", "batch", "token_count", "files", "skills", "web_search"),
				},
			},
			"model": schema.StringAttribute{
				Optional:    true,
				Description: "Optional. Filter to the single entry containing this model. Accepts model names and aliases. The API returns 404 if the model has no rate limits in this org.",
			},
			"groups": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"group_type": schema.StringAttribute{Computed: true},
						"models": schema.ListAttribute{
							Computed:    true,
							ElementType: types.StringType,
							Description: "Model names this entry applies to. Null when group_type != model_group.",
						},
						"limits": schema.ListNestedAttribute{
							Computed: true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"type":  schema.StringAttribute{Computed: true, Description: "Limiter type (e.g. requests_per_minute, input_tokens_per_minute)."},
									"value": schema.Int64Attribute{Computed: true},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (d *OrganizationRateLimitsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, diags := clientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	d.client = c
}

func (d *OrganizationRateLimitsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data OrganizationRateLimitsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	groups, err := d.client.ListOrganizationRateLimits(ctx, anthropic.ListOrganizationRateLimitsParams{
		GroupType: data.GroupType.ValueString(),
		Model:     data.Model.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to list organization rate limits", err.Error())
		return
	}

	out := make([]OrgRateLimitGroupModel, 0, len(groups))
	for _, g := range groups {
		models := make([]types.String, 0, len(g.Models))
		for _, m := range g.Models {
			models = append(models, types.StringValue(m))
		}
		limits := make([]OrgRateLimitValueModel, 0, len(g.Limits))
		for _, l := range g.Limits {
			limits = append(limits, OrgRateLimitValueModel{
				Type:  types.StringValue(l.Type),
				Value: types.Int64Value(l.Value),
			})
		}
		out = append(out, OrgRateLimitGroupModel{
			GroupType: types.StringValue(g.GroupType),
			Models:    models,
			Limits:    limits,
		})
	}
	data.Groups = out
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}
