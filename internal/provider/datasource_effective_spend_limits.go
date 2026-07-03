package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/sauterdigital/terraform-provider-claude-admin/internal/anthropic"
)

var (
	_ datasource.DataSource              = &EffectiveSpendLimitsDataSource{}
	_ datasource.DataSourceWithConfigure = &EffectiveSpendLimitsDataSource{}
)

func NewEffectiveSpendLimitsDataSource() datasource.DataSource {
	return &EffectiveSpendLimitsDataSource{}
}

type EffectiveSpendLimitsDataSource struct{ client *anthropic.Client }

type EffectiveSpendLimitsModel struct {
	Period    []types.String      `tfsdk:"period"`
	UserIDs   []types.String      `tfsdk:"user_ids"`
	Summaries []SpendSummaryModel `tfsdk:"summaries"`
}

type SpendSummaryModel struct {
	UserID            types.String `tfsdk:"user_id"`
	UserEmail         types.String `tfsdk:"user_email"`
	UserName          types.String `tfsdk:"user_name"`
	UserDeleted       types.Bool   `tfsdk:"user_deleted"`
	Amount            types.String `tfsdk:"amount"`
	Currency          types.String `tfsdk:"currency"`
	Period            types.String `tfsdk:"period"`
	PeriodToDateSpend types.String `tfsdk:"period_to_date_spend"`
	SourceScopeType   types.String `tfsdk:"source_scope_type"`
}

func (d *EffectiveSpendLimitsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_effective_spend_limits"
}

func (d *EffectiveSpendLimitsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists each member's effective spend limit and period-to-date spend. Source scope shows where the limit came from (user override, seat_tier, rbac_group, organization_service, organization).",
		Attributes: map[string]schema.Attribute{
			"period": schema.ListAttribute{
				Optional: true, ElementType: types.StringType,
				Validators: []validator.List{
					listvalidator.ValueStringsAre(stringvalidator.OneOf("monthly", "daily", "weekly")),
				},
			},
			"user_ids": schema.ListAttribute{Optional: true, ElementType: types.StringType},
			"summaries": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"user_id":              schema.StringAttribute{Computed: true},
						"user_email":           schema.StringAttribute{Computed: true},
						"user_name":            schema.StringAttribute{Computed: true},
						"user_deleted":         schema.BoolAttribute{Computed: true},
						"amount":               schema.StringAttribute{Computed: true},
						"currency":             schema.StringAttribute{Computed: true},
						"period":               schema.StringAttribute{Computed: true},
						"period_to_date_spend": schema.StringAttribute{Computed: true},
						"source_scope_type":    schema.StringAttribute{Computed: true, Description: "Scope where this limit was inherited from."},
					},
				},
			},
		},
	}
}

func (d *EffectiveSpendLimitsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, diags := clientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	d.client = c
}

func (d *EffectiveSpendLimitsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data EffectiveSpendLimitsModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	list, err := d.client.ListEffectiveSpendLimits(ctx, anthropic.ListEffectiveSpendLimitsParams{
		Period:  stringSliceFromTF(data.Period),
		UserIDs: stringSliceFromTF(data.UserIDs),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to list effective spend limits", err.Error())
		return
	}
	out := make([]SpendSummaryModel, 0, len(list))
	for _, s := range list {
		out = append(out, SpendSummaryModel{
			UserID:            types.StringValue(s.Actor.UserID),
			UserEmail:         stringValueOrEmpty(s.Actor.EmailAddress),
			UserName:          stringValueOrEmpty(s.Actor.Name),
			UserDeleted:       types.BoolValue(s.Actor.Deleted),
			Amount:            types.StringValue(s.Amount),
			Currency:          types.StringValue(s.Currency),
			Period:            types.StringValue(s.Period),
			PeriodToDateSpend: types.StringValue(s.PeriodToDateSpend),
			SourceScopeType:   types.StringValue(s.Source.Type),
		})
	}
	data.Summaries = out
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}
