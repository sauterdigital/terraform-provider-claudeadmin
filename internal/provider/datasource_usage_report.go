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
	_ datasource.DataSource              = &UsageReportDataSource{}
	_ datasource.DataSourceWithConfigure = &UsageReportDataSource{}
)

func NewUsageReportDataSource() datasource.DataSource { return &UsageReportDataSource{} }

type UsageReportDataSource struct{ client *anthropic.Client }

type UsageReportDataSourceModel struct {
	StartingAt        types.String   `tfsdk:"starting_at"`
	EndingAt          types.String   `tfsdk:"ending_at"`
	BucketWidth       types.String   `tfsdk:"bucket_width"`
	Limit             types.Int64    `tfsdk:"limit"`
	Page              types.String   `tfsdk:"page"`
	GroupBy           []types.String `tfsdk:"group_by"`
	WorkspaceIDs      []types.String `tfsdk:"workspace_ids"`
	APIKeyIDs         []types.String `tfsdk:"api_key_ids"`
	AccountIDs        []types.String `tfsdk:"account_ids"`
	ServiceAccountIDs []types.String `tfsdk:"service_account_ids"`
	Models            []types.String `tfsdk:"models"`
	ServiceTiers      []types.String `tfsdk:"service_tiers"`
	ContextWindow     []types.String `tfsdk:"context_window"`
	InferenceGeos     []types.String `tfsdk:"inference_geos"`
	Speeds            []types.String `tfsdk:"speeds"`

	HasMore  types.Bool         `tfsdk:"has_more"`
	NextPage types.String       `tfsdk:"next_page"`
	Buckets  []UsageBucketModel `tfsdk:"buckets"`
}

type UsageBucketModel struct {
	StartingAt types.String       `tfsdk:"starting_at"`
	EndingAt   types.String       `tfsdk:"ending_at"`
	Results    []UsageResultModel `tfsdk:"results"`
}

type UsageResultModel struct {
	AccountID            types.String `tfsdk:"account_id"`
	APIKeyID             types.String `tfsdk:"api_key_id"`
	ServiceAccountID     types.String `tfsdk:"service_account_id"`
	WorkspaceID          types.String `tfsdk:"workspace_id"`
	Model                types.String `tfsdk:"model"`
	ContextWindow        types.String `tfsdk:"context_window"`
	InferenceGeo         types.String `tfsdk:"inference_geo"`
	ServiceTier          types.String `tfsdk:"service_tier"`
	UncachedInputTokens  types.Int64  `tfsdk:"uncached_input_tokens"`
	CacheReadInputTokens types.Int64  `tfsdk:"cache_read_input_tokens"`
	OutputTokens         types.Int64  `tfsdk:"output_tokens"`
	CacheCreation1H      types.Int64  `tfsdk:"cache_creation_1h_input_tokens"`
	CacheCreation5M      types.Int64  `tfsdk:"cache_creation_5m_input_tokens"`
	WebSearchRequests    types.Int64  `tfsdk:"web_search_requests"`
}

func (d *UsageReportDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_usage_report"
}

func (d *UsageReportDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches the Messages usage report from the Anthropic Admin API. Returns one or more time buckets with token, cache, and tool-use counts. Suitable as input to FinOps/observability pipelines.",
		Attributes: map[string]schema.Attribute{
			"starting_at": schema.StringAttribute{Required: true, Description: "RFC 3339 timestamp; buckets starting at or after this are returned."},
			"ending_at":   schema.StringAttribute{Optional: true},
			"bucket_width": schema.StringAttribute{
				Optional:    true,
				Description: "1m, 1h, or 1d (default 1d).",
				Validators: []validator.String{
					stringvalidator.OneOf("1m", "1h", "1d"),
				},
			},
			"limit": schema.Int64Attribute{Optional: true},
			"page":  schema.StringAttribute{Optional: true, Description: "next_page token from a prior response."},
			"group_by": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Validators: []validator.List{
					listvalidator.ValueStringsAre(stringvalidator.OneOf(
						"api_key_id", "workspace_id", "model", "service_tier",
						"context_window", "inference_geo", "speed",
						"account_id", "service_account_id",
					)),
				},
			},
			"workspace_ids":       schema.ListAttribute{Optional: true, ElementType: types.StringType},
			"api_key_ids":         schema.ListAttribute{Optional: true, ElementType: types.StringType},
			"account_ids":         schema.ListAttribute{Optional: true, ElementType: types.StringType},
			"service_account_ids": schema.ListAttribute{Optional: true, ElementType: types.StringType},
			"models":              schema.ListAttribute{Optional: true, ElementType: types.StringType},
			"service_tiers": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Validators: []validator.List{
					listvalidator.ValueStringsAre(stringvalidator.OneOf("standard", "batch", "priority", "priority_on_demand", "flex", "flex_discount")),
				},
			},
			"context_window": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Validators: []validator.List{
					listvalidator.ValueStringsAre(stringvalidator.OneOf("0-200k", "200k-1M")),
				},
			},
			"inference_geos": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Validators: []validator.List{
					listvalidator.ValueStringsAre(stringvalidator.OneOf("global", "us", "not_available")),
				},
			},
			"speeds": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Validators: []validator.List{
					listvalidator.ValueStringsAre(stringvalidator.OneOf("standard", "fast")),
				},
			},

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
									"account_id":                     schema.StringAttribute{Computed: true},
									"api_key_id":                     schema.StringAttribute{Computed: true},
									"service_account_id":             schema.StringAttribute{Computed: true},
									"workspace_id":                   schema.StringAttribute{Computed: true},
									"model":                          schema.StringAttribute{Computed: true},
									"context_window":                 schema.StringAttribute{Computed: true},
									"inference_geo":                  schema.StringAttribute{Computed: true},
									"service_tier":                   schema.StringAttribute{Computed: true},
									"uncached_input_tokens":          schema.Int64Attribute{Computed: true},
									"cache_read_input_tokens":        schema.Int64Attribute{Computed: true},
									"output_tokens":                  schema.Int64Attribute{Computed: true},
									"cache_creation_1h_input_tokens": schema.Int64Attribute{Computed: true},
									"cache_creation_5m_input_tokens": schema.Int64Attribute{Computed: true},
									"web_search_requests":            schema.Int64Attribute{Computed: true},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (d *UsageReportDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, diags := clientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	d.client = c
}

func (d *UsageReportDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data UsageReportDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := anthropic.UsageReportParams{
		StartingAt:        data.StartingAt.ValueString(),
		EndingAt:          data.EndingAt.ValueString(),
		BucketWidth:       data.BucketWidth.ValueString(),
		Limit:             int(data.Limit.ValueInt64()),
		Page:              data.Page.ValueString(),
		GroupBy:           stringSliceFromTF(data.GroupBy),
		WorkspaceIDs:      stringSliceFromTF(data.WorkspaceIDs),
		APIKeyIDs:         stringSliceFromTF(data.APIKeyIDs),
		AccountIDs:        stringSliceFromTF(data.AccountIDs),
		ServiceAccountIDs: stringSliceFromTF(data.ServiceAccountIDs),
		Models:            stringSliceFromTF(data.Models),
		ServiceTiers:      stringSliceFromTF(data.ServiceTiers),
		ContextWindow:     stringSliceFromTF(data.ContextWindow),
		InferenceGeos:     stringSliceFromTF(data.InferenceGeos),
		Speeds:            stringSliceFromTF(data.Speeds),
	}

	report, err := d.client.GetUsageReport(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError("Failed to fetch usage report", err.Error())
		return
	}

	data.HasMore = types.BoolValue(report.HasMore)
	data.NextPage = stringValueOrEmpty(report.NextPage)

	buckets := make([]UsageBucketModel, 0, len(report.Data))
	for _, b := range report.Data {
		results := make([]UsageResultModel, 0, len(b.Results))
		for _, r := range b.Results {
			results = append(results, UsageResultModel{
				AccountID:            optionalStringValue(r.AccountID),
				APIKeyID:             optionalStringValue(r.APIKeyID),
				ServiceAccountID:     optionalStringValue(r.ServiceAccountID),
				WorkspaceID:          optionalStringValue(r.WorkspaceID),
				Model:                optionalStringValue(r.Model),
				ContextWindow:        optionalStringValue(r.ContextWindow),
				InferenceGeo:         optionalStringValue(r.InferenceGeo),
				ServiceTier:          optionalStringValue(r.ServiceTier),
				UncachedInputTokens:  types.Int64Value(r.UncachedInputTokens),
				CacheReadInputTokens: types.Int64Value(r.CacheReadInputTokens),
				OutputTokens:         types.Int64Value(r.OutputTokens),
				CacheCreation1H:      types.Int64Value(r.CacheCreation.Ephemeral1HInputTokens),
				CacheCreation5M:      types.Int64Value(r.CacheCreation.Ephemeral5MInputTokens),
				WebSearchRequests:    types.Int64Value(r.ServerToolUse.WebSearchRequests),
			})
		}
		buckets = append(buckets, UsageBucketModel{
			StartingAt: types.StringValue(b.StartingAt),
			EndingAt:   types.StringValue(b.EndingAt),
			Results:    results,
		})
	}
	data.Buckets = buckets

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func stringSliceFromTF(in []types.String) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, 0, len(in))
	for _, v := range in {
		if !v.IsNull() && !v.IsUnknown() {
			out = append(out, v.ValueString())
		}
	}
	return out
}
