package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/sauterdigital/terraform-provider-claude-admin/internal/anthropic"
)

var (
	_ datasource.DataSource              = &ComplianceOrganizationUsersDataSource{}
	_ datasource.DataSourceWithConfigure = &ComplianceOrganizationUsersDataSource{}
)

func NewComplianceOrganizationUsersDataSource() datasource.DataSource {
	return &ComplianceOrganizationUsersDataSource{}
}

type ComplianceOrganizationUsersDataSource struct{ client *anthropic.Client }

type complianceUserModel struct {
	ID           types.String `tfsdk:"id"`
	Email        types.String `tfsdk:"email"`
	Name         types.String `tfsdk:"name"`
	Role         types.String `tfsdk:"role"`
	Status       types.String `tfsdk:"status"`
	CreatedAt    types.String `tfsdk:"created_at"`
	LastActiveAt types.String `tfsdk:"last_active_at"`
	SourceType   types.String `tfsdk:"source_type"`
	ExternalID   types.String `tfsdk:"external_id"`
}

type ComplianceOrganizationUsersModel struct {
	OrganizationID types.String          `tfsdk:"organization_id"`
	Users          []complianceUserModel `tfsdk:"users"`
}

func (d *ComplianceOrganizationUsersDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_compliance_organization_users"
}

func (d *ComplianceOrganizationUsersDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all members of a given organization from the Compliance API. Includes SCIM-sourced users (with `external_id` populated). Enterprise + Compliance API key with scope `read:compliance_user_data`.",
		Attributes: map[string]schema.Attribute{
			"organization_id": schema.StringAttribute{Required: true},
			"users": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":             schema.StringAttribute{Computed: true},
						"email":          schema.StringAttribute{Computed: true},
						"name":           schema.StringAttribute{Computed: true},
						"role":           schema.StringAttribute{Computed: true},
						"status":         schema.StringAttribute{Computed: true},
						"created_at":     schema.StringAttribute{Computed: true},
						"last_active_at": schema.StringAttribute{Computed: true},
						"source_type":    schema.StringAttribute{Computed: true},
						"external_id":    schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *ComplianceOrganizationUsersDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, diags := clientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	d.client = c
}

func (d *ComplianceOrganizationUsersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var cfg ComplianceOrganizationUsersModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}
	users, err := d.client.ListComplianceOrganizationUsers(ctx, cfg.OrganizationID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to list compliance organization users", err.Error())
		return
	}
	out := make([]complianceUserModel, 0, len(users))
	for _, u := range users {
		out = append(out, complianceUserModel{
			ID:           types.StringValue(u.ID),
			Email:        types.StringValue(u.Email),
			Name:         types.StringValue(u.Name),
			Role:         types.StringValue(u.Role),
			Status:       types.StringValue(u.Status),
			CreatedAt:    types.StringValue(u.CreatedAt),
			LastActiveAt: optionalStringValue(u.LastActiveAt),
			SourceType:   optionalStringValue(u.SourceType),
			ExternalID:   optionalStringValue(u.ExternalID),
		})
	}
	cfg.Users = out
	resp.Diagnostics.Append(resp.State.Set(ctx, cfg)...)
}
