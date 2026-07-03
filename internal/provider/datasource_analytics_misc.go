package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/sauterdigital/terraform-provider-claude-admin/internal/anthropic"
)

// ---------- User Activity (analytics/users) ----------

var (
	_ datasource.DataSource              = &UserActivityDataSource{}
	_ datasource.DataSourceWithConfigure = &UserActivityDataSource{}
)

func NewUserActivityDataSource() datasource.DataSource { return &UserActivityDataSource{} }

type UserActivityDataSource struct{ client *anthropic.Client }

type UserActivityModel struct {
	StartingDate types.String      `tfsdk:"starting_date"`
	EndingDate   types.String      `tfsdk:"ending_date"`
	UserIDs      []types.String    `tfsdk:"user_ids"`
	Users        []UserActivityRow `tfsdk:"users"`
}

type UserActivityRow struct {
	UserID            types.String `tfsdk:"user_id"`
	EmailAddress      types.String `tfsdk:"email_address"`
	Name              types.String `tfsdk:"name"`
	SeatTier          types.String `tfsdk:"seat_tier"`
	LastActiveAt      types.String `tfsdk:"last_active_at"`
	TotalRequests     types.Int64  `tfsdk:"total_requests"`
	TotalInputTokens  types.Int64  `tfsdk:"total_input_tokens"`
	TotalOutputTokens types.Int64  `tfsdk:"total_output_tokens"`
	TotalCostAmount   types.String `tfsdk:"total_cost_amount"`
	Currency          types.String `tfsdk:"currency"`
}

func (d *UserActivityDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user_activity"
}

func (d *UserActivityDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Per-user activity snapshot over a date range. Enterprise plan + read:analytics scope.",
		Attributes: map[string]schema.Attribute{
			"starting_date": schema.StringAttribute{Optional: true, Description: "UTC YYYY-MM-DD."},
			"ending_date":   schema.StringAttribute{Optional: true},
			"user_ids":      schema.ListAttribute{Optional: true, ElementType: types.StringType},
			"users": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"user_id":             schema.StringAttribute{Computed: true},
						"email_address":       schema.StringAttribute{Computed: true},
						"name":                schema.StringAttribute{Computed: true},
						"seat_tier":           schema.StringAttribute{Computed: true},
						"last_active_at":      schema.StringAttribute{Computed: true},
						"total_requests":      schema.Int64Attribute{Computed: true},
						"total_input_tokens":  schema.Int64Attribute{Computed: true},
						"total_output_tokens": schema.Int64Attribute{Computed: true},
						"total_cost_amount":   schema.StringAttribute{Computed: true},
						"currency":            schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *UserActivityDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, diags := clientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	d.client = c
}

func (d *UserActivityDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data UserActivityModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r, err := d.client.ListUserActivity(ctx, anthropic.ListUserActivityParams{
		StartingDate: data.StartingDate.ValueString(),
		EndingDate:   data.EndingDate.ValueString(),
		UserIDs:      stringSliceFromTF(data.UserIDs),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to fetch user activity", err.Error())
		return
	}
	out := make([]UserActivityRow, 0, len(r.Data))
	for _, u := range r.Data {
		out = append(out, UserActivityRow{
			UserID:            types.StringValue(u.UserID),
			EmailAddress:      types.StringValue(u.EmailAddress),
			Name:              types.StringValue(u.Name),
			SeatTier:          optionalStringValue(u.SeatTier),
			LastActiveAt:      optionalStringValue(u.LastActiveAt),
			TotalRequests:     types.Int64Value(u.TotalRequests),
			TotalInputTokens:  types.Int64Value(u.TotalInputTokens),
			TotalOutputTokens: types.Int64Value(u.TotalOutputTokens),
			TotalCostAmount:   optionalStringValue(u.TotalCostAmount),
			Currency:          optionalStringValue(u.Currency),
		})
	}
	data.Users = out
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

// ---------- Skills Usage ----------

var (
	_ datasource.DataSource              = &SkillsUsageDataSource{}
	_ datasource.DataSourceWithConfigure = &SkillsUsageDataSource{}
)

func NewSkillsUsageDataSource() datasource.DataSource { return &SkillsUsageDataSource{} }

type SkillsUsageDataSource struct{ client *anthropic.Client }

type SkillsUsageModel struct {
	StartingDate types.String      `tfsdk:"starting_date"`
	EndingDate   types.String      `tfsdk:"ending_date"`
	Skills       []SkillUsageEntry `tfsdk:"skills"`
}

type SkillUsageEntry struct {
	SkillName       types.String `tfsdk:"skill_name"`
	InvocationCount types.Int64  `tfsdk:"invocation_count"`
	SuccessCount    types.Int64  `tfsdk:"success_count"`
	FailureCount    types.Int64  `tfsdk:"failure_count"`
}

func (d *SkillsUsageDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_skills_usage"
}

func (d *SkillsUsageDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Skills invocation counts (success/failure breakdown) for the date range.",
		Attributes: map[string]schema.Attribute{
			"starting_date": schema.StringAttribute{Optional: true},
			"ending_date":   schema.StringAttribute{Optional: true},
			"skills": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"skill_name":       schema.StringAttribute{Computed: true},
						"invocation_count": schema.Int64Attribute{Computed: true},
						"success_count":    schema.Int64Attribute{Computed: true},
						"failure_count":    schema.Int64Attribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *SkillsUsageDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, diags := clientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	d.client = c
}

func (d *SkillsUsageDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data SkillsUsageModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r, err := d.client.GetSkillsUsage(ctx, anthropic.SimpleAnalyticsParams{
		StartingDate: data.StartingDate.ValueString(),
		EndingDate:   data.EndingDate.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to fetch skills usage", err.Error())
		return
	}
	out := make([]SkillUsageEntry, 0, len(r.Data))
	for _, s := range r.Data {
		out = append(out, SkillUsageEntry{
			SkillName:       types.StringValue(s.SkillName),
			InvocationCount: types.Int64Value(s.InvocationCount),
			SuccessCount:    types.Int64Value(s.SuccessCount),
			FailureCount:    types.Int64Value(s.FailureCount),
		})
	}
	data.Skills = out
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

// ---------- Connectors Usage ----------

var (
	_ datasource.DataSource              = &ConnectorsUsageDataSource{}
	_ datasource.DataSourceWithConfigure = &ConnectorsUsageDataSource{}
)

func NewConnectorsUsageDataSource() datasource.DataSource { return &ConnectorsUsageDataSource{} }

type ConnectorsUsageDataSource struct{ client *anthropic.Client }

type ConnectorsUsageModel struct {
	StartingDate types.String          `tfsdk:"starting_date"`
	EndingDate   types.String          `tfsdk:"ending_date"`
	Connectors   []ConnectorUsageEntry `tfsdk:"connectors"`
}

type ConnectorUsageEntry struct {
	ConnectorName   types.String `tfsdk:"connector_name"`
	InvocationCount types.Int64  `tfsdk:"invocation_count"`
	UniqueUsers     types.Int64  `tfsdk:"unique_users"`
}

func (d *ConnectorsUsageDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_connectors_usage"
}

func (d *ConnectorsUsageDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Connectors invocation + unique-user counts for the date range.",
		Attributes: map[string]schema.Attribute{
			"starting_date": schema.StringAttribute{Optional: true},
			"ending_date":   schema.StringAttribute{Optional: true},
			"connectors": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"connector_name":   schema.StringAttribute{Computed: true},
						"invocation_count": schema.Int64Attribute{Computed: true},
						"unique_users":     schema.Int64Attribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *ConnectorsUsageDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, diags := clientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	d.client = c
}

func (d *ConnectorsUsageDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ConnectorsUsageModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r, err := d.client.GetConnectorsUsage(ctx, anthropic.SimpleAnalyticsParams{
		StartingDate: data.StartingDate.ValueString(),
		EndingDate:   data.EndingDate.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to fetch connectors usage", err.Error())
		return
	}
	out := make([]ConnectorUsageEntry, 0, len(r.Data))
	for _, c := range r.Data {
		out = append(out, ConnectorUsageEntry{
			ConnectorName:   types.StringValue(c.ConnectorName),
			InvocationCount: types.Int64Value(c.InvocationCount),
			UniqueUsers:     types.Int64Value(c.UniqueUsers),
		})
	}
	data.Connectors = out
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

// ---------- Chat Projects Usage ----------

var (
	_ datasource.DataSource              = &ChatProjectsUsageDataSource{}
	_ datasource.DataSourceWithConfigure = &ChatProjectsUsageDataSource{}
)

func NewChatProjectsUsageDataSource() datasource.DataSource { return &ChatProjectsUsageDataSource{} }

type ChatProjectsUsageDataSource struct{ client *anthropic.Client }

type ChatProjectsUsageModel struct {
	StartingDate types.String            `tfsdk:"starting_date"`
	EndingDate   types.String            `tfsdk:"ending_date"`
	Projects     []ChatProjectUsageEntry `tfsdk:"projects"`
}

type ChatProjectUsageEntry struct {
	ProjectID    types.String `tfsdk:"project_id"`
	ProjectName  types.String `tfsdk:"project_name"`
	MessageCount types.Int64  `tfsdk:"message_count"`
	UniqueUsers  types.Int64  `tfsdk:"unique_users"`
}

func (d *ChatProjectsUsageDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_chat_projects_usage"
}

func (d *ChatProjectsUsageDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Per-project chat usage (message counts + unique users) for the date range.",
		Attributes: map[string]schema.Attribute{
			"starting_date": schema.StringAttribute{Optional: true},
			"ending_date":   schema.StringAttribute{Optional: true},
			"projects": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"project_id":    schema.StringAttribute{Computed: true},
						"project_name":  schema.StringAttribute{Computed: true},
						"message_count": schema.Int64Attribute{Computed: true},
						"unique_users":  schema.Int64Attribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *ChatProjectsUsageDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, diags := clientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	d.client = c
}

func (d *ChatProjectsUsageDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ChatProjectsUsageModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r, err := d.client.GetChatProjectsUsage(ctx, anthropic.SimpleAnalyticsParams{
		StartingDate: data.StartingDate.ValueString(),
		EndingDate:   data.EndingDate.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to fetch chat projects usage", err.Error())
		return
	}
	out := make([]ChatProjectUsageEntry, 0, len(r.Data))
	for _, p := range r.Data {
		out = append(out, ChatProjectUsageEntry{
			ProjectID:    types.StringValue(p.ProjectID),
			ProjectName:  types.StringValue(p.ProjectName),
			MessageCount: types.Int64Value(p.MessageCount),
			UniqueUsers:  types.Int64Value(p.UniqueUsers),
		})
	}
	data.Projects = out
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}
