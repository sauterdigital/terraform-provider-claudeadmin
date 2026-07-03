package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/sauterdigital/terraform-provider-claude-admin/internal/anthropic"
)

var (
	_ datasource.DataSource              = &ComplianceGroupMembersDataSource{}
	_ datasource.DataSourceWithConfigure = &ComplianceGroupMembersDataSource{}
)

func NewComplianceGroupMembersDataSource() datasource.DataSource {
	return &ComplianceGroupMembersDataSource{}
}

type ComplianceGroupMembersDataSource struct{ client *anthropic.Client }

type complianceGroupMemberModel struct {
	UserID     types.String `tfsdk:"user_id"`
	Email      types.String `tfsdk:"email"`
	Name       types.String `tfsdk:"name"`
	JoinedAt   types.String `tfsdk:"joined_at"`
	SourceType types.String `tfsdk:"source_type"`
}

type ComplianceGroupMembersModel struct {
	GroupID types.String                 `tfsdk:"group_id"`
	Members []complianceGroupMemberModel `tfsdk:"members"`
}

func (d *ComplianceGroupMembersDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_compliance_group_members"
}

func (d *ComplianceGroupMembersDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists members of a specific group. Enterprise + Compliance API key with scope `read:compliance_user_data`.",
		Attributes: map[string]schema.Attribute{
			"group_id": schema.StringAttribute{Required: true},
			"members": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"user_id":     schema.StringAttribute{Computed: true},
						"email":       schema.StringAttribute{Computed: true},
						"name":        schema.StringAttribute{Computed: true},
						"joined_at":   schema.StringAttribute{Computed: true},
						"source_type": schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *ComplianceGroupMembersDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, diags := clientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	d.client = c
}

func (d *ComplianceGroupMembersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var cfg ComplianceGroupMembersModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}
	members, err := d.client.ListComplianceGroupMembers(ctx, cfg.GroupID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to list compliance group members", err.Error())
		return
	}
	out := make([]complianceGroupMemberModel, 0, len(members))
	for _, m := range members {
		out = append(out, complianceGroupMemberModel{
			UserID:     types.StringValue(m.UserID),
			Email:      optionalStringValue(m.Email),
			Name:       optionalStringValue(m.Name),
			JoinedAt:   optionalStringValue(m.JoinedAt),
			SourceType: optionalStringValue(m.SourceType),
		})
	}
	cfg.Members = out
	resp.Diagnostics.Append(resp.State.Set(ctx, cfg)...)
}
