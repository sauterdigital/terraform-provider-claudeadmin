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

// Shared model + schema fragments for the 4 analytics endpoints
// (token usage / per-user token / cost / per-user cost).

type AnalyticsTimeFiltersModel struct {
	StartingAt     types.String   `tfsdk:"starting_at"`
	EndingAt       types.String   `tfsdk:"ending_at"`
	BucketWidth    types.String   `tfsdk:"bucket_width"`
	GroupBy        []types.String `tfsdk:"group_by"`
	Products       []types.String `tfsdk:"products"`
	Models         []types.String `tfsdk:"models"`
	ContextWindows []types.String `tfsdk:"context_windows"`
	InferenceGeos  []types.String `tfsdk:"inference_geos"`
	Speeds         []types.String `tfsdk:"speeds"`
	UserIDs        []types.String `tfsdk:"user_ids"`
}

type TokenUsageBucketModel struct {
	StartingAt types.String            `tfsdk:"starting_at"`
	EndingAt   types.String            `tfsdk:"ending_at"`
	Results    []TokenUsageResultModel `tfsdk:"results"`
}

type TokenUsageResultModel struct {
	UncachedInputTokens  types.Int64  `tfsdk:"uncached_input_tokens"`
	CacheReadInputTokens types.Int64  `tfsdk:"cache_read_input_tokens"`
	OutputTokens         types.Int64  `tfsdk:"output_tokens"`
	CacheCreation1H      types.Int64  `tfsdk:"cache_creation_1h_input_tokens"`
	CacheCreation5M      types.Int64  `tfsdk:"cache_creation_5m_input_tokens"`
	WebSearchRequests    types.Int64  `tfsdk:"web_search_requests"`
	Requests             types.Int64  `tfsdk:"requests"`
	Product              types.String `tfsdk:"product"`
	Model                types.String `tfsdk:"model"`
	ContextWindow        types.String `tfsdk:"context_window"`
	InferenceGeo         types.String `tfsdk:"inference_geo"`
	Speed                types.String `tfsdk:"speed"`
	UserID               types.String `tfsdk:"user_id"`
}

type CostBucketV2Model struct {
	StartingAt types.String        `tfsdk:"starting_at"`
	EndingAt   types.String        `tfsdk:"ending_at"`
	Results    []CostResultV2Model `tfsdk:"results"`
}

type CostResultV2Model struct {
	Amount        types.String `tfsdk:"amount"`
	Currency      types.String `tfsdk:"currency"`
	Product       types.String `tfsdk:"product"`
	Model         types.String `tfsdk:"model"`
	ContextWindow types.String `tfsdk:"context_window"`
	InferenceGeo  types.String `tfsdk:"inference_geo"`
	Speed         types.String `tfsdk:"speed"`
	UserID        types.String `tfsdk:"user_id"`
	TokenType     types.String `tfsdk:"token_type"`
}

func analyticsTimeAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"starting_at":  schema.StringAttribute{Required: true, Description: "RFC 3339 timestamp, ≥ 2026-01-01T00:00:00Z, within last 365 days."},
		"ending_at":    schema.StringAttribute{Optional: true, Description: "Defaults to min(now, starting_at + 31d). Range ≤ 31 days."},
		"bucket_width": schema.StringAttribute{Optional: true, Validators: []validator.String{stringvalidator.OneOf("1m", "1h", "1d")}},
		"group_by": schema.ListAttribute{
			Optional: true, ElementType: types.StringType,
			Validators: []validator.List{listvalidator.ValueStringsAre(stringvalidator.OneOf("product", "model", "context_window", "inference_geo", "speed"))},
		},
		"products":        schema.ListAttribute{Optional: true, ElementType: types.StringType},
		"models":          schema.ListAttribute{Optional: true, ElementType: types.StringType},
		"context_windows": schema.ListAttribute{Optional: true, ElementType: types.StringType, Validators: []validator.List{listvalidator.ValueStringsAre(stringvalidator.OneOf("0-200k", "200k-1M"))}},
		"inference_geos":  schema.ListAttribute{Optional: true, ElementType: types.StringType, Validators: []validator.List{listvalidator.ValueStringsAre(stringvalidator.OneOf("global", "us", "not_available"))}},
		"speeds":          schema.ListAttribute{Optional: true, ElementType: types.StringType, Validators: []validator.List{listvalidator.ValueStringsAre(stringvalidator.OneOf("fast", "standard"))}},
		"user_ids":        schema.ListAttribute{Optional: true, ElementType: types.StringType},
	}
}

func tokenUsageBucketsSchema() schema.ListNestedAttribute {
	return schema.ListNestedAttribute{
		Computed: true,
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"starting_at": schema.StringAttribute{Computed: true},
				"ending_at":   schema.StringAttribute{Computed: true},
				"results": schema.ListNestedAttribute{
					Computed: true,
					NestedObject: schema.NestedAttributeObject{
						Attributes: map[string]schema.Attribute{
							"uncached_input_tokens":          schema.Int64Attribute{Computed: true},
							"cache_read_input_tokens":        schema.Int64Attribute{Computed: true},
							"output_tokens":                  schema.Int64Attribute{Computed: true},
							"cache_creation_1h_input_tokens": schema.Int64Attribute{Computed: true},
							"cache_creation_5m_input_tokens": schema.Int64Attribute{Computed: true},
							"web_search_requests":            schema.Int64Attribute{Computed: true},
							"requests":                       schema.Int64Attribute{Computed: true},
							"product":                        schema.StringAttribute{Computed: true},
							"model":                          schema.StringAttribute{Computed: true},
							"context_window":                 schema.StringAttribute{Computed: true},
							"inference_geo":                  schema.StringAttribute{Computed: true},
							"speed":                          schema.StringAttribute{Computed: true},
							"user_id":                        schema.StringAttribute{Computed: true},
						},
					},
				},
			},
		},
	}
}

func costBucketsV2Schema() schema.ListNestedAttribute {
	return schema.ListNestedAttribute{
		Computed: true,
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"starting_at": schema.StringAttribute{Computed: true},
				"ending_at":   schema.StringAttribute{Computed: true},
				"results": schema.ListNestedAttribute{
					Computed: true,
					NestedObject: schema.NestedAttributeObject{
						Attributes: map[string]schema.Attribute{
							"amount":         schema.StringAttribute{Computed: true},
							"currency":       schema.StringAttribute{Computed: true},
							"product":        schema.StringAttribute{Computed: true},
							"model":          schema.StringAttribute{Computed: true},
							"context_window": schema.StringAttribute{Computed: true},
							"inference_geo":  schema.StringAttribute{Computed: true},
							"speed":          schema.StringAttribute{Computed: true},
							"user_id":        schema.StringAttribute{Computed: true},
							"token_type":     schema.StringAttribute{Computed: true},
						},
					},
				},
			},
		},
	}
}

func paramsFromModel(m AnalyticsTimeFiltersModel) anthropic.AnalyticsParams {
	return anthropic.AnalyticsParams{
		StartingAt:     m.StartingAt.ValueString(),
		EndingAt:       m.EndingAt.ValueString(),
		BucketWidth:    m.BucketWidth.ValueString(),
		GroupBy:        stringSliceFromTF(m.GroupBy),
		Products:       stringSliceFromTF(m.Products),
		Models:         stringSliceFromTF(m.Models),
		ContextWindows: stringSliceFromTF(m.ContextWindows),
		InferenceGeos:  stringSliceFromTF(m.InferenceGeos),
		Speeds:         stringSliceFromTF(m.Speeds),
		UserIDs:        stringSliceFromTF(m.UserIDs),
	}
}

func tokenBucketsToModel(buckets []anthropic.TokenUsageBucket) []TokenUsageBucketModel {
	out := make([]TokenUsageBucketModel, 0, len(buckets))
	for _, b := range buckets {
		results := make([]TokenUsageResultModel, 0, len(b.Results))
		for _, r := range b.Results {
			results = append(results, TokenUsageResultModel{
				UncachedInputTokens:  types.Int64Value(r.UncachedInputTokens),
				CacheReadInputTokens: types.Int64Value(r.CacheReadInputTokens),
				OutputTokens:         types.Int64Value(r.OutputTokens),
				CacheCreation1H:      types.Int64Value(r.CacheCreation.Ephemeral1HInputTokens),
				CacheCreation5M:      types.Int64Value(r.CacheCreation.Ephemeral5MInputTokens),
				WebSearchRequests:    types.Int64Value(r.ServerToolUse.WebSearchRequests),
				Requests:             types.Int64Value(r.Requests),
				Product:              optionalStringValue(r.Product),
				Model:                optionalStringValue(r.Model),
				ContextWindow:        optionalStringValue(r.ContextWindow),
				InferenceGeo:         optionalStringValue(r.InferenceGeo),
				Speed:                optionalStringValue(r.Speed),
				UserID:               optionalStringValue(r.UserID),
			})
		}
		out = append(out, TokenUsageBucketModel{
			StartingAt: types.StringValue(b.StartingAt),
			EndingAt:   types.StringValue(b.EndingAt),
			Results:    results,
		})
	}
	return out
}

func costBucketsToModel(buckets []anthropic.CostBucketV2) []CostBucketV2Model {
	out := make([]CostBucketV2Model, 0, len(buckets))
	for _, b := range buckets {
		results := make([]CostResultV2Model, 0, len(b.Results))
		for _, r := range b.Results {
			results = append(results, CostResultV2Model{
				Amount:        types.StringValue(r.Amount),
				Currency:      types.StringValue(r.Currency),
				Product:       optionalStringValue(r.Product),
				Model:         optionalStringValue(r.Model),
				ContextWindow: optionalStringValue(r.ContextWindow),
				InferenceGeo:  optionalStringValue(r.InferenceGeo),
				Speed:         optionalStringValue(r.Speed),
				UserID:        optionalStringValue(r.UserID),
				TokenType:     optionalStringValue(r.TokenType),
			})
		}
		out = append(out, CostBucketV2Model{
			StartingAt: types.StringValue(b.StartingAt),
			EndingAt:   types.StringValue(b.EndingAt),
			Results:    results,
		})
	}
	return out
}

// ---------- Token Usage Over Time ----------

var (
	_ datasource.DataSource              = &TokenUsageOverTimeDataSource{}
	_ datasource.DataSourceWithConfigure = &TokenUsageOverTimeDataSource{}
)

func NewTokenUsageOverTimeDataSource() datasource.DataSource { return &TokenUsageOverTimeDataSource{} }

type TokenUsageOverTimeDataSource struct{ client *anthropic.Client }

type TokenUsageWithFiltersModel struct {
	AnalyticsTimeFiltersModel
	Buckets []TokenUsageBucketModel `tfsdk:"buckets"`
}

func (d *TokenUsageOverTimeDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_token_usage_over_time"
}

func (d *TokenUsageOverTimeDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attrs := analyticsTimeAttributes()
	attrs["buckets"] = tokenUsageBucketsSchema()
	resp.Schema = schema.Schema{
		Description: "Token usage over time (analytics v2). Enterprise plan + read:analytics scope.",
		Attributes:  attrs,
	}
}

func (d *TokenUsageOverTimeDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, diags := clientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	d.client = c
}

func (d *TokenUsageOverTimeDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data TokenUsageWithFiltersModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r, err := d.client.GetTokenUsageOverTime(ctx, paramsFromModel(data.AnalyticsTimeFiltersModel))
	if err != nil {
		resp.Diagnostics.AddError("Failed to fetch token usage over time", err.Error())
		return
	}
	data.Buckets = tokenBucketsToModel(r.Data)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

// ---------- Per-User Token Usage ----------

var (
	_ datasource.DataSource              = &PerUserTokenUsageDataSource{}
	_ datasource.DataSourceWithConfigure = &PerUserTokenUsageDataSource{}
)

func NewPerUserTokenUsageDataSource() datasource.DataSource { return &PerUserTokenUsageDataSource{} }

type PerUserTokenUsageDataSource struct{ client *anthropic.Client }

func (d *PerUserTokenUsageDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_per_user_token_usage"
}

func (d *PerUserTokenUsageDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attrs := analyticsTimeAttributes()
	attrs["buckets"] = tokenUsageBucketsSchema()
	resp.Schema = schema.Schema{
		Description: "Per-user token usage breakdown (analytics v2). Each result row carries the user_id it applies to. Enterprise plan + read:analytics scope.",
		Attributes:  attrs,
	}
}

func (d *PerUserTokenUsageDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, diags := clientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	d.client = c
}

func (d *PerUserTokenUsageDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data TokenUsageWithFiltersModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r, err := d.client.GetPerUserTokenUsage(ctx, paramsFromModel(data.AnalyticsTimeFiltersModel))
	if err != nil {
		resp.Diagnostics.AddError("Failed to fetch per-user token usage", err.Error())
		return
	}
	data.Buckets = tokenBucketsToModel(r.Data)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

// ---------- Cost Over Time ----------

var (
	_ datasource.DataSource              = &CostOverTimeDataSource{}
	_ datasource.DataSourceWithConfigure = &CostOverTimeDataSource{}
)

func NewCostOverTimeDataSource() datasource.DataSource { return &CostOverTimeDataSource{} }

type CostOverTimeDataSource struct{ client *anthropic.Client }

type CostWithFiltersModel struct {
	AnalyticsTimeFiltersModel
	Buckets []CostBucketV2Model `tfsdk:"buckets"`
}

func (d *CostOverTimeDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cost_over_time"
}

func (d *CostOverTimeDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attrs := analyticsTimeAttributes()
	attrs["buckets"] = costBucketsV2Schema()
	resp.Schema = schema.Schema{
		Description: "Cost over time (analytics v2). Enterprise plan + read:analytics scope.",
		Attributes:  attrs,
	}
}

func (d *CostOverTimeDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, diags := clientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	d.client = c
}

func (d *CostOverTimeDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data CostWithFiltersModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r, err := d.client.GetCostOverTime(ctx, paramsFromModel(data.AnalyticsTimeFiltersModel))
	if err != nil {
		resp.Diagnostics.AddError("Failed to fetch cost over time", err.Error())
		return
	}
	data.Buckets = costBucketsToModel(r.Data)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

// ---------- Per-User Cost ----------

var (
	_ datasource.DataSource              = &PerUserCostDataSource{}
	_ datasource.DataSourceWithConfigure = &PerUserCostDataSource{}
)

func NewPerUserCostDataSource() datasource.DataSource { return &PerUserCostDataSource{} }

type PerUserCostDataSource struct{ client *anthropic.Client }

func (d *PerUserCostDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_per_user_cost"
}

func (d *PerUserCostDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attrs := analyticsTimeAttributes()
	attrs["buckets"] = costBucketsV2Schema()
	resp.Schema = schema.Schema{
		Description: "Per-user cost breakdown (analytics v2). Enterprise plan + read:analytics scope.",
		Attributes:  attrs,
	}
}

func (d *PerUserCostDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, diags := clientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	d.client = c
}

func (d *PerUserCostDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data CostWithFiltersModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r, err := d.client.GetPerUserCost(ctx, paramsFromModel(data.AnalyticsTimeFiltersModel))
	if err != nil {
		resp.Diagnostics.AddError("Failed to fetch per-user cost", err.Error())
		return
	}
	data.Buckets = costBucketsToModel(r.Data)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}
