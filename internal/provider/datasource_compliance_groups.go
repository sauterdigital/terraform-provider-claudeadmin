package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/sauterdigital/terraform-provider-claude-admin/internal/anthropic"
)

var (
	_ datasource.DataSource              = &ComplianceGroupsDataSource{}
	_ datasource.DataSourceWithConfigure = &ComplianceGroupsDataSource{}
)

func NewComplianceGroupsDataSource() datasource.DataSource { return &ComplianceGroupsDataSource{} }

type ComplianceGroupsDataSource struct{ client *anthropic.Client }

type complianceGroupModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	DisplayName types.String `tfsdk:"display_name"`
	SourceType  types.String `tfsdk:"source_type"`
	ExternalID  types.String `tfsdk:"external_id"`
	MemberCount types.Int64  `tfsdk:"member_count"`
}

type ComplianceGroupsModel struct {
	Groups []complianceGroupModel `tfsdk:"groups"`
}

func (d *ComplianceGroupsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_compliance_groups"
}

func (d *ComplianceGroupsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all groups visible to the Compliance API — includes SCIM-provisioned groups (with `source_type = scim` + `external_id` populated) and native Anthropic groups. Enterprise + Compliance API key with scope `read:compliance_org_data`.",
		Attributes: map[string]schema.Attribute{
			"groups": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":           schema.StringAttribute{Computed: true},
						"name":         schema.StringAttribute{Computed: true},
						"display_name": schema.StringAttribute{Computed: true},
						"source_type":  schema.StringAttribute{Computed: true},
						"external_id":  schema.StringAttribute{Computed: true},
						"member_count": schema.Int64Attribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *ComplianceGroupsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, diags := clientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	d.client = c
}

func (d *ComplianceGroupsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	groups, err := d.client.ListComplianceGroups(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list compliance groups", err.Error())
		return
	}
	out := make([]complianceGroupModel, 0, len(groups))
	for _, g := range groups {
		mc := types.Int64Null()
		if g.MemberCount != nil {
			mc = types.Int64Value(*g.MemberCount)
		}
		out = append(out, complianceGroupModel{
			ID:          types.StringValue(g.ID),
			Name:        types.StringValue(g.Name),
			DisplayName: optionalStringValue(g.DisplayName),
			SourceType:  optionalStringValue(g.SourceType),
			ExternalID:  optionalStringValue(g.ExternalID),
			MemberCount: mc,
		})
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, ComplianceGroupsModel{Groups: out})...)
}
