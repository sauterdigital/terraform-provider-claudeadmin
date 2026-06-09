package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/sauterdigital/terraform-provider-anthropic/internal/anthropic"
)

var (
	_ datasource.DataSource              = &ClaudeCodeUsageReportDataSource{}
	_ datasource.DataSourceWithConfigure = &ClaudeCodeUsageReportDataSource{}
)

func NewClaudeCodeUsageReportDataSource() datasource.DataSource {
	return &ClaudeCodeUsageReportDataSource{}
}

type ClaudeCodeUsageReportDataSource struct{ client *anthropic.Client }

type ClaudeCodeUsageReportModel struct {
	StartingAt types.String                `tfsdk:"starting_at"`
	Limit      types.Int64                 `tfsdk:"limit"`
	Page       types.String                `tfsdk:"page"`
	HasMore    types.Bool                  `tfsdk:"has_more"`
	NextPage   types.String                `tfsdk:"next_page"`
	Entries    []ClaudeCodeUsageEntryModel `tfsdk:"entries"`
}

type ClaudeCodeUsageEntryModel struct {
	Date             types.String                      `tfsdk:"date"`
	OrganizationID   types.String                      `tfsdk:"organization_id"`
	CustomerType     types.String                      `tfsdk:"customer_type"`
	SubscriptionType types.String                      `tfsdk:"subscription_type"`
	TerminalType     types.String                      `tfsdk:"terminal_type"`
	ActorType        types.String                      `tfsdk:"actor_type"`
	ActorEmail       types.String                      `tfsdk:"actor_email"`
	ActorAPIKeyName  types.String                      `tfsdk:"actor_api_key_name"`
	CoreMetrics      ClaudeCodeCoreMetricsModel        `tfsdk:"core_metrics"`
	ModelBreakdown   []ClaudeCodeModelBreakdownModel   `tfsdk:"model_breakdown"`
	ToolActions      map[string]ClaudeCodeToolActModel `tfsdk:"tool_actions"`
}

type ClaudeCodeCoreMetricsModel struct {
	CommitsByClaudeCode      types.Int64               `tfsdk:"commits_by_claude_code"`
	PullRequestsByClaudeCode types.Int64               `tfsdk:"pull_requests_by_claude_code"`
	NumSessions              types.Int64               `tfsdk:"num_sessions"`
	LinesOfCode              ClaudeCodeLineCountsModel `tfsdk:"lines_of_code"`
}

type ClaudeCodeLineCountsModel struct {
	Added   types.Int64 `tfsdk:"added"`
	Removed types.Int64 `tfsdk:"removed"`
}

type ClaudeCodeModelBreakdownModel struct {
	Model             types.String  `tfsdk:"model"`
	EstimatedAmount   types.Float64 `tfsdk:"estimated_amount"`
	EstimatedCurrency types.String  `tfsdk:"estimated_currency"`
	InputTokens       types.Int64   `tfsdk:"input_tokens"`
	OutputTokens      types.Int64   `tfsdk:"output_tokens"`
	CacheCreation     types.Int64   `tfsdk:"cache_creation_tokens"`
	CacheRead         types.Int64   `tfsdk:"cache_read_tokens"`
}

type ClaudeCodeToolActModel struct {
	Accepted types.Int64 `tfsdk:"accepted"`
	Rejected types.Int64 `tfsdk:"rejected"`
}

func (d *ClaudeCodeUsageReportDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_claude_code_usage_report"
}

func (d *ClaudeCodeUsageReportDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches per-day Claude Code usage metrics from the Admin API. `starting_at` is a single date (YYYY-MM-DD) — the response covers only that day. Polymorphic actor is flattened: exactly one of `actor_email` (user) or `actor_api_key_name` (api) is set, indicated by `actor_type`.",
		Attributes: map[string]schema.Attribute{
			"starting_at": schema.StringAttribute{Required: true, Description: "YYYY-MM-DD date."},
			"limit":       schema.Int64Attribute{Optional: true},
			"page":        schema.StringAttribute{Optional: true},

			"has_more":  schema.BoolAttribute{Computed: true},
			"next_page": schema.StringAttribute{Computed: true},
			"entries": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"date":               schema.StringAttribute{Computed: true},
						"organization_id":    schema.StringAttribute{Computed: true},
						"customer_type":      schema.StringAttribute{Computed: true},
						"subscription_type":  schema.StringAttribute{Computed: true},
						"terminal_type":      schema.StringAttribute{Computed: true},
						"actor_type":         schema.StringAttribute{Computed: true},
						"actor_email":        schema.StringAttribute{Computed: true},
						"actor_api_key_name": schema.StringAttribute{Computed: true},
						"core_metrics": schema.SingleNestedAttribute{
							Computed: true,
							Attributes: map[string]schema.Attribute{
								"commits_by_claude_code":       schema.Int64Attribute{Computed: true},
								"pull_requests_by_claude_code": schema.Int64Attribute{Computed: true},
								"num_sessions":                 schema.Int64Attribute{Computed: true},
								"lines_of_code": schema.SingleNestedAttribute{
									Computed: true,
									Attributes: map[string]schema.Attribute{
										"added":   schema.Int64Attribute{Computed: true},
										"removed": schema.Int64Attribute{Computed: true},
									},
								},
							},
						},
						"model_breakdown": schema.ListNestedAttribute{
							Computed: true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"model":                 schema.StringAttribute{Computed: true},
									"estimated_amount":      schema.Float64Attribute{Computed: true, Description: "Cost in minor units (cents for USD)."},
									"estimated_currency":    schema.StringAttribute{Computed: true},
									"input_tokens":          schema.Int64Attribute{Computed: true},
									"output_tokens":         schema.Int64Attribute{Computed: true},
									"cache_creation_tokens": schema.Int64Attribute{Computed: true},
									"cache_read_tokens":     schema.Int64Attribute{Computed: true},
								},
							},
						},
						"tool_actions": schema.MapNestedAttribute{
							Computed:    true,
							Description: "Tool-name to acceptance counts. Keys observed include edit_tool, multi_edit_tool, notebook_edit_tool, write_tool.",
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"accepted": schema.Int64Attribute{Computed: true},
									"rejected": schema.Int64Attribute{Computed: true},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (d *ClaudeCodeUsageReportDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, diags := clientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	d.client = c
}

func (d *ClaudeCodeUsageReportDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ClaudeCodeUsageReportModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	report, err := d.client.GetClaudeCodeUsageReport(ctx, anthropic.ClaudeCodeUsageReportParams{
		StartingAt: data.StartingAt.ValueString(),
		Limit:      int(data.Limit.ValueInt64()),
		Page:       data.Page.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to fetch Claude Code usage report", err.Error())
		return
	}

	data.HasMore = types.BoolValue(report.HasMore)
	if report.NextPage != nil {
		data.NextPage = types.StringValue(*report.NextPage)
	} else {
		data.NextPage = types.StringNull()
	}

	entries := make([]ClaudeCodeUsageEntryModel, 0, len(report.Data))
	for _, e := range report.Data {
		entry := ClaudeCodeUsageEntryModel{
			Date:            types.StringValue(e.Date),
			OrganizationID:  types.StringValue(e.OrganizationID),
			CustomerType:    types.StringValue(e.CustomerType),
			TerminalType:    types.StringValue(e.TerminalType),
			ActorType:       types.StringValue(e.Actor.Type),
			ActorEmail:      stringValueOrEmpty(e.Actor.EmailAddress),
			ActorAPIKeyName: stringValueOrEmpty(e.Actor.APIKeyName),
			CoreMetrics: ClaudeCodeCoreMetricsModel{
				CommitsByClaudeCode:      types.Int64Value(e.CoreMetrics.CommitsByClaudeCode),
				PullRequestsByClaudeCode: types.Int64Value(e.CoreMetrics.PullRequestsByClaudeCode),
				NumSessions:              types.Int64Value(e.CoreMetrics.NumSessions),
				LinesOfCode: ClaudeCodeLineCountsModel{
					Added:   types.Int64Value(e.CoreMetrics.LinesOfCode.Added),
					Removed: types.Int64Value(e.CoreMetrics.LinesOfCode.Removed),
				},
			},
		}
		if e.SubscriptionType != nil {
			entry.SubscriptionType = types.StringValue(*e.SubscriptionType)
		} else {
			entry.SubscriptionType = types.StringNull()
		}

		breakdown := make([]ClaudeCodeModelBreakdownModel, 0, len(e.ModelBreakdown))
		for _, m := range e.ModelBreakdown {
			breakdown = append(breakdown, ClaudeCodeModelBreakdownModel{
				Model:             types.StringValue(m.Model),
				EstimatedAmount:   types.Float64Value(m.EstimatedCost.Amount),
				EstimatedCurrency: types.StringValue(m.EstimatedCost.Currency),
				InputTokens:       types.Int64Value(m.Tokens.Input),
				OutputTokens:      types.Int64Value(m.Tokens.Output),
				CacheCreation:     types.Int64Value(m.Tokens.CacheCreation),
				CacheRead:         types.Int64Value(m.Tokens.CacheRead),
			})
		}
		entry.ModelBreakdown = breakdown

		tools := make(map[string]ClaudeCodeToolActModel, len(e.ToolActions))
		for name, a := range e.ToolActions {
			tools[name] = ClaudeCodeToolActModel{
				Accepted: types.Int64Value(a.Accepted),
				Rejected: types.Int64Value(a.Rejected),
			}
		}
		entry.ToolActions = tools

		entries = append(entries, entry)
	}
	data.Entries = entries

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}
