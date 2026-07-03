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
	_ datasource.DataSource              = &CostReportDataSource{}
	_ datasource.DataSourceWithConfigure = &CostReportDataSource{}
)

func NewCostReportDataSource() datasource.DataSource { return &CostReportDataSource{} }

type CostReportDataSource struct{ client *anthropic.Client }

type CostReportDataSourceModel struct {
	StartingAt   types.String      `tfsdk:"starting_at"`
	EndingAt     types.String      `tfsdk:"ending_at"`
	BucketWidth  types.String      `tfsdk:"bucket_width"`
	Limit        types.Int64       `tfsdk:"limit"`
	Page         types.String      `tfsdk:"page"`
	GroupBy      []types.String    `tfsdk:"group_by"`
	WorkspaceIDs []types.String    `tfsdk:"workspace_ids"`
	HasMore      types.Bool        `tfsdk:"has_more"`
	NextPage     types.String      `tfsdk:"next_page"`
	Buckets      []CostBucketModel `tfsdk:"buckets"`
}

type CostBucketModel struct {
	StartingAt types.String      `tfsdk:"starting_at"`
	EndingAt   types.String      `tfsdk:"ending_at"`
	Results    []CostResultModel `tfsdk:"results"`
}

type CostResultModel struct {
	Amount        types.String `tfsdk:"amount"`
	Currency      types.String `tfsdk:"currency"`
	CostType      types.String `tfsdk:"cost_type"`
	Description   types.String `tfsdk:"description"`
	WorkspaceID   types.String `tfsdk:"workspace_id"`
	Model         types.String `tfsdk:"model"`
	TokenType     types.String `tfsdk:"token_type"`
	ContextWindow types.String `tfsdk:"context_window"`
	InferenceGeo  types.String `tfsdk:"inference_geo"`
	ServiceTier   types.String `tfsdk:"service_tier"`
}

func (d *CostReportDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cost_report"
}

func (d *CostReportDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches the cost report from the Anthropic Admin API. Returns monetary cost amounts per time bucket, optionally grouped by workspace_id and/or description. Currently only `1d` bucket width is supported by the API.",
		Attributes: map[string]schema.Attribute{
			"starting_at": schema.StringAttribute{Required: true},
			"ending_at":   schema.StringAttribute{Optional: true},
			"bucket_width": schema.StringAttribute{
				Optional:    true,
				Description: "Only `1d` is currently supported by the API.",
				Validators: []validator.String{
					stringvalidator.OneOf("1d"),
				},
			},
			"limit": schema.Int64Attribute{Optional: true},
			"page":  schema.StringAttribute{Optional: true},
			"group_by": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Subset of: workspace_id, description.",
				Validators: []validator.List{
					listvalidator.ValueStringsAre(stringvalidator.OneOf("workspace_id", "description")),
				},
			},
			"workspace_ids": schema.ListAttribute{Optional: true, ElementType: types.StringType},

			"has_more":  schema.BoolAttribute{Computed: true},
			"next_page": schema.StringAttribute{Computed: true},
			"buckets": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"starting_at": schema.StringAttribute{Computed: true},
						"ending_at":   schema.StringAttribute{Computed: true},
						"results": schema.ListNestedAttribute{
							Computed: true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"amount":         schema.StringAttribute{Computed: true, Description: "Decimal cost amount as a string (e.g. \"123.45\")."},
									"currency":       schema.StringAttribute{Computed: true},
									"cost_type":      schema.StringAttribute{Computed: true},
									"description":    schema.StringAttribute{Computed: true},
									"workspace_id":   schema.StringAttribute{Computed: true},
									"model":          schema.StringAttribute{Computed: true},
									"token_type":     schema.StringAttribute{Computed: true},
									"context_window": schema.StringAttribute{Computed: true},
									"inference_geo":  schema.StringAttribute{Computed: true},
									"service_tier":   schema.StringAttribute{Computed: true},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (d *CostReportDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, diags := clientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	d.client = c
}

func (d *CostReportDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data CostReportDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	report, err := d.client.GetCostReport(ctx, anthropic.CostReportParams{
		StartingAt:   data.StartingAt.ValueString(),
		EndingAt:     data.EndingAt.ValueString(),
		BucketWidth:  data.BucketWidth.ValueString(),
		Limit:        int(data.Limit.ValueInt64()),
		Page:         data.Page.ValueString(),
		GroupBy:      stringSliceFromTF(data.GroupBy),
		WorkspaceIDs: stringSliceFromTF(data.WorkspaceIDs),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to fetch cost report", err.Error())
		return
	}

	data.HasMore = types.BoolValue(report.HasMore)
	data.NextPage = stringValueOrEmpty(report.NextPage)

	buckets := make([]CostBucketModel, 0, len(report.Data))
	for _, b := range report.Data {
		results := make([]CostResultModel, 0, len(b.Results))
		for _, r := range b.Results {
			results = append(results, CostResultModel{
				Amount:        types.StringValue(r.Amount),
				Currency:      types.StringValue(r.Currency),
				CostType:      optionalStringValue(r.CostType),
				Description:   optionalStringValue(r.Description),
				WorkspaceID:   optionalStringValue(r.WorkspaceID),
				Model:         optionalStringValue(r.Model),
				TokenType:     optionalStringValue(r.TokenType),
				ContextWindow: optionalStringValue(r.ContextWindow),
				InferenceGeo:  optionalStringValue(r.InferenceGeo),
				ServiceTier:   optionalStringValue(r.ServiceTier),
			})
		}
		buckets = append(buckets, CostBucketModel{
			StartingAt: types.StringValue(b.StartingAt),
			EndingAt:   types.StringValue(b.EndingAt),
			Results:    results,
		})
	}
	data.Buckets = buckets

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}
