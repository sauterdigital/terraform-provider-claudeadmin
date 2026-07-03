package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/sauterdigital/terraform-provider-claude-admin/internal/anthropic"
)

var (
	_ datasource.DataSource              = &ComplianceOrganizationsDataSource{}
	_ datasource.DataSourceWithConfigure = &ComplianceOrganizationsDataSource{}
)

func NewComplianceOrganizationsDataSource() datasource.DataSource {
	return &ComplianceOrganizationsDataSource{}
}

type ComplianceOrganizationsDataSource struct{ client *anthropic.Client }

type complianceOrgModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	DisplayName types.String `tfsdk:"display_name"`
	Plan        types.String `tfsdk:"plan"`
	CreatedAt   types.String `tfsdk:"created_at"`
}

type ComplianceOrganizationsModel struct {
	Organizations []complianceOrgModel `tfsdk:"organizations"`
}

func (d *ComplianceOrganizationsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_compliance_organizations"
}

func (d *ComplianceOrganizationsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all organizations visible to the Compliance API key. Enterprise + Compliance API key with scope `read:compliance_org_data`.",
		Attributes: map[string]schema.Attribute{
			"organizations": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":           schema.StringAttribute{Computed: true},
						"name":         schema.StringAttribute{Computed: true},
						"display_name": schema.StringAttribute{Computed: true},
						"plan":         schema.StringAttribute{Computed: true},
						"created_at":   schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *ComplianceOrganizationsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, diags := clientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	d.client = c
}

func (d *ComplianceOrganizationsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	orgs, err := d.client.ListComplianceOrganizations(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list compliance organizations", err.Error())
		return
	}
	out := make([]complianceOrgModel, 0, len(orgs))
	for _, o := range orgs {
		out = append(out, complianceOrgModel{
			ID:          types.StringValue(o.ID),
			Name:        types.StringValue(o.Name),
			DisplayName: types.StringValue(o.DisplayName),
			Plan:        types.StringValue(o.Plan),
			CreatedAt:   types.StringValue(o.CreatedAt),
		})
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, ComplianceOrganizationsModel{Organizations: out})...)
}
