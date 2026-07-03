package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/sauterdigital/terraform-provider-claude-admin/internal/anthropic"
)

var (
	_ datasource.DataSource              = &ComplianceActivitiesDataSource{}
	_ datasource.DataSourceWithConfigure = &ComplianceActivitiesDataSource{}
)

func NewComplianceActivitiesDataSource() datasource.DataSource {
	return &ComplianceActivitiesDataSource{}
}

type ComplianceActivitiesDataSource struct{ client *anthropic.Client }

type complianceActivityModel struct {
	ID           types.String `tfsdk:"id"`
	Timestamp    types.String `tfsdk:"timestamp"`
	ActorID      types.String `tfsdk:"actor_id"`
	ActorType    types.String `tfsdk:"actor_type"`
	ActorEmail   types.String `tfsdk:"actor_email"`
	Action       types.String `tfsdk:"action"`
	ResourceID   types.String `tfsdk:"resource_id"`
	ResourceType types.String `tfsdk:"resource_type"`
	Outcome      types.String `tfsdk:"outcome"`
}

type ComplianceActivitiesModel struct {
	StartingAt   types.String              `tfsdk:"starting_at"`
	EndingAt     types.String              `tfsdk:"ending_at"`
	ActorID      types.String              `tfsdk:"actor_id"`
	Action       types.String              `tfsdk:"action"`
	ResourceType types.String              `tfsdk:"resource_type"`
	Limit        types.Int64               `tfsdk:"limit"`
	Activities   []complianceActivityModel `tfsdk:"activities"`
}

func (d *ComplianceActivitiesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_compliance_activities"
}

func (d *ComplianceActivitiesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads the Compliance Activity Feed. Enterprise + Compliance API key (scope `read:compliance_activities`). All filters are optional; unset filters return the full window. Provider paginates until exhausted — the entire result set is materialized in state, so bound the window via `starting_at`/`ending_at` for busy orgs.",
		Attributes: map[string]schema.Attribute{
			"starting_at":   schema.StringAttribute{Optional: true, Description: "RFC3339 lower bound (inclusive)."},
			"ending_at":     schema.StringAttribute{Optional: true, Description: "RFC3339 upper bound (exclusive)."},
			"actor_id":      schema.StringAttribute{Optional: true, Description: "Filter by actor (user_id / service_account_id)."},
			"action":        schema.StringAttribute{Optional: true, Description: "Filter by action string (e.g. `workspace.update`)."},
			"resource_type": schema.StringAttribute{Optional: true, Description: "Filter by resource type."},
			"limit":         schema.Int64Attribute{Optional: true, Description: "Per-page size (default 100, max per Anthropic). Total result set is not capped by this."},
			"activities": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":            schema.StringAttribute{Computed: true},
						"timestamp":     schema.StringAttribute{Computed: true},
						"actor_id":      schema.StringAttribute{Computed: true},
						"actor_type":    schema.StringAttribute{Computed: true},
						"actor_email":   schema.StringAttribute{Computed: true},
						"action":        schema.StringAttribute{Computed: true},
						"resource_id":   schema.StringAttribute{Computed: true},
						"resource_type": schema.StringAttribute{Computed: true},
						"outcome":       schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *ComplianceActivitiesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, diags := clientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	d.client = c
}

func (d *ComplianceActivitiesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var cfg ComplianceActivitiesModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}
	params := anthropic.ListComplianceActivitiesParams{
		StartingAt:   cfg.StartingAt.ValueString(),
		EndingAt:     cfg.EndingAt.ValueString(),
		ActorID:      cfg.ActorID.ValueString(),
		Action:       cfg.Action.ValueString(),
		ResourceType: cfg.ResourceType.ValueString(),
		Limit:        int(cfg.Limit.ValueInt64()),
	}
	acts, err := d.client.ListComplianceActivities(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list compliance activities", err.Error())
		return
	}
	out := make([]complianceActivityModel, 0, len(acts))
	for _, a := range acts {
		out = append(out, complianceActivityModel{
			ID:           types.StringValue(a.ID),
			Timestamp:    types.StringValue(a.Timestamp),
			ActorID:      optionalStringValue(a.ActorID),
			ActorType:    optionalStringValue(a.ActorType),
			ActorEmail:   optionalStringValue(a.ActorEmail),
			Action:       types.StringValue(a.Action),
			ResourceID:   optionalStringValue(a.ResourceID),
			ResourceType: optionalStringValue(a.ResourceType),
			Outcome:      optionalStringValue(a.Outcome),
		})
	}
	cfg.Activities = out
	resp.Diagnostics.Append(resp.State.Set(ctx, cfg)...)
}
