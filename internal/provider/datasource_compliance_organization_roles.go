package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/sauterdigital/terraform-provider-anthropic/internal/anthropic"
)

var (
	_ datasource.DataSource              = &ComplianceOrganizationRolesDataSource{}
	_ datasource.DataSourceWithConfigure = &ComplianceOrganizationRolesDataSource{}
)

func NewComplianceOrganizationRolesDataSource() datasource.DataSource {
	return &ComplianceOrganizationRolesDataSource{}
}

type ComplianceOrganizationRolesDataSource struct{ client *anthropic.Client }

type compliancePermissionModel struct {
	Action   types.String `tfsdk:"action"`
	Resource types.String `tfsdk:"resource"`
	Effect   types.String `tfsdk:"effect"`
}

type complianceRoleModel struct {
	ID          types.String                `tfsdk:"id"`
	Name        types.String                `tfsdk:"name"`
	Description types.String                `tfsdk:"description"`
	IsBuiltIn   types.Bool                  `tfsdk:"is_built_in"`
	Permissions []compliancePermissionModel `tfsdk:"permissions"`
}

type ComplianceOrganizationRolesModel struct {
	OrganizationID types.String          `tfsdk:"organization_id"`
	Roles          []complianceRoleModel `tfsdk:"roles"`
}

func (d *ComplianceOrganizationRolesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_compliance_organization_roles"
}

func (d *ComplianceOrganizationRolesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists roles defined at the organization level and expands their permission grants. Useful to audit RBAC drift. Enterprise + Compliance API key with scope `read:compliance_org_data`.",
		Attributes: map[string]schema.Attribute{
			"organization_id": schema.StringAttribute{Required: true},
			"roles": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":          schema.StringAttribute{Computed: true},
						"name":        schema.StringAttribute{Computed: true},
						"description": schema.StringAttribute{Computed: true},
						"is_built_in": schema.BoolAttribute{Computed: true},
						"permissions": schema.ListNestedAttribute{
							Computed: true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"action":   schema.StringAttribute{Computed: true},
									"resource": schema.StringAttribute{Computed: true},
									"effect":   schema.StringAttribute{Computed: true},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (d *ComplianceOrganizationRolesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, diags := clientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	d.client = c
}

func (d *ComplianceOrganizationRolesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var cfg ComplianceOrganizationRolesModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}
	roles, err := d.client.ListComplianceOrganizationRoles(ctx, cfg.OrganizationID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to list compliance organization roles", err.Error())
		return
	}
	out := make([]complianceRoleModel, 0, len(roles))
	for _, r := range roles {
		perms := make([]compliancePermissionModel, 0, len(r.Permissions))
		for _, p := range r.Permissions {
			perms = append(perms, compliancePermissionModel{
				Action:   types.StringValue(p.Action),
				Resource: types.StringValue(p.Resource),
				Effect:   types.StringValue(p.Effect),
			})
		}
		out = append(out, complianceRoleModel{
			ID:          types.StringValue(r.ID),
			Name:        types.StringValue(r.Name),
			Description: optionalStringValue(r.Description),
			IsBuiltIn:   types.BoolValue(r.IsBuiltIn),
			Permissions: perms,
		})
	}
	cfg.Roles = out
	resp.Diagnostics.Append(resp.State.Set(ctx, cfg)...)
}
