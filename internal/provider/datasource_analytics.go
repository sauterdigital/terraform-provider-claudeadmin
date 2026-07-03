package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/sauterdigital/terraform-provider-claude-admin/internal/anthropic"
)

// ============================================================
// Activity Summaries — daily org-wide metrics
// ============================================================

var (
	_ datasource.DataSource              = &ActivitySummariesDataSource{}
	_ datasource.DataSourceWithConfigure = &ActivitySummariesDataSource{}
)

func NewActivitySummariesDataSource() datasource.DataSource {
	return &ActivitySummariesDataSource{}
}

type ActivitySummariesDataSource struct{ client *anthropic.Client }

type ActivitySummariesModel struct {
	StartingDate types.String           `tfsdk:"starting_date"`
	EndingDate   types.String           `tfsdk:"ending_date"`
	Summaries    []ActivitySummaryModel `tfsdk:"summaries"`
}

type ActivitySummaryModel struct {
	StartingAt                   types.String  `tfsdk:"starting_at"`
	EndingAt                     types.String  `tfsdk:"ending_at"`
	AssignedSeatCount            types.Int64   `tfsdk:"assigned_seat_count"`
	DailyActiveUserCount         types.Int64   `tfsdk:"daily_active_user_count"`
	WeeklyActiveUserCount        types.Int64   `tfsdk:"weekly_active_user_count"`
	MonthlyActiveUserCount       types.Int64   `tfsdk:"monthly_active_user_count"`
	DailyAdoptionRate            types.Float64 `tfsdk:"daily_adoption_rate"`
	WeeklyAdoptionRate           types.Float64 `tfsdk:"weekly_adoption_rate"`
	MonthlyAdoptionRate          types.Float64 `tfsdk:"monthly_adoption_rate"`
	CoworkDailyActiveUserCount   types.Int64   `tfsdk:"cowork_daily_active_user_count"`
	CoworkWeeklyActiveUserCount  types.Int64   `tfsdk:"cowork_weekly_active_user_count"`
	CoworkMonthlyActiveUserCount types.Int64   `tfsdk:"cowork_monthly_active_user_count"`
	PendingInviteCount           types.Int64   `tfsdk:"pending_invite_count"`
}

func (d *ActivitySummariesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_activity_summaries"
}

func (d *ActivitySummariesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Org-wide activity summaries per day. Requires Claude Enterprise plan + API key with `read:analytics` scope. Data is finalized with a 3-day lag (so `starting_date` must be ≥ 3 days ago).",
		Attributes: map[string]schema.Attribute{
			"starting_date": schema.StringAttribute{Required: true, Description: "UTC YYYY-MM-DD, ≥3 days in the past, ≥ 2026-01-01."},
			"ending_date":   schema.StringAttribute{Optional: true, Description: "UTC YYYY-MM-DD exclusive. Defaults to 2 days before today."},
			"summaries": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"starting_at":                      schema.StringAttribute{Computed: true},
						"ending_at":                        schema.StringAttribute{Computed: true},
						"assigned_seat_count":              schema.Int64Attribute{Computed: true},
						"daily_active_user_count":          schema.Int64Attribute{Computed: true},
						"weekly_active_user_count":         schema.Int64Attribute{Computed: true},
						"monthly_active_user_count":        schema.Int64Attribute{Computed: true},
						"daily_adoption_rate":              schema.Float64Attribute{Computed: true},
						"weekly_adoption_rate":             schema.Float64Attribute{Computed: true},
						"monthly_adoption_rate":            schema.Float64Attribute{Computed: true},
						"cowork_daily_active_user_count":   schema.Int64Attribute{Computed: true},
						"cowork_weekly_active_user_count":  schema.Int64Attribute{Computed: true},
						"cowork_monthly_active_user_count": schema.Int64Attribute{Computed: true},
						"pending_invite_count":             schema.Int64Attribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *ActivitySummariesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, diags := clientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	d.client = c
}

func (d *ActivitySummariesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ActivitySummariesModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r, err := d.client.GetActivitySummaries(ctx, anthropic.GetActivitySummariesParams{
		StartingDate: data.StartingDate.ValueString(),
		EndingDate:   data.EndingDate.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to fetch activity summaries", err.Error())
		return
	}
	out := make([]ActivitySummaryModel, 0, len(r.Summaries))
	for _, s := range r.Summaries {
		out = append(out, ActivitySummaryModel{
			StartingAt:                   types.StringValue(s.StartingAt),
			EndingAt:                     types.StringValue(s.EndingAt),
			AssignedSeatCount:            int64PtrToTF(s.AssignedSeatCount),
			DailyActiveUserCount:         types.Int64Value(s.DailyActiveUserCount),
			WeeklyActiveUserCount:        types.Int64Value(s.WeeklyActiveUserCount),
			MonthlyActiveUserCount:       types.Int64Value(s.MonthlyActiveUserCount),
			DailyAdoptionRate:            float64PtrToTF(s.DailyAdoptionRate),
			WeeklyAdoptionRate:           float64PtrToTF(s.WeeklyAdoptionRate),
			MonthlyAdoptionRate:          float64PtrToTF(s.MonthlyAdoptionRate),
			CoworkDailyActiveUserCount:   types.Int64Value(s.CoworkDailyActiveUserCount),
			CoworkWeeklyActiveUserCount:  types.Int64Value(s.CoworkWeeklyActiveUserCount),
			CoworkMonthlyActiveUserCount: types.Int64Value(s.CoworkMonthlyActiveUserCount),
			PendingInviteCount:           int64PtrToTF(s.PendingInviteCount),
		})
	}
	data.Summaries = out
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func int64PtrToTF(v *int64) types.Int64 {
	if v == nil {
		return types.Int64Null()
	}
	return types.Int64Value(*v)
}

func float64PtrToTF(v *float64) types.Float64 {
	if v == nil {
		return types.Float64Null()
	}
	return types.Float64Value(*v)
}
