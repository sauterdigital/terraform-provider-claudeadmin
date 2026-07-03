package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/sauterdigital/terraform-provider-claude-admin/internal/anthropic"
)

var (
	_ datasource.DataSource              = &OrganizationMembersDataSource{}
	_ datasource.DataSourceWithConfigure = &OrganizationMembersDataSource{}
)

func NewOrganizationMembersDataSource() datasource.DataSource {
	return &OrganizationMembersDataSource{}
}

type OrganizationMembersDataSource struct{ client *anthropic.Client }

type OrganizationMembersDataSourceModel struct {
	Email   types.String                      `tfsdk:"email"`
	Members []OrganizationMemberResourceModel `tfsdk:"members"`
}

func (d *OrganizationMembersDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_members"
}

func (d *OrganizationMembersDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists organization members with optional email filter.",
		Attributes: map[string]schema.Attribute{
			"email": schema.StringAttribute{Optional: true, Description: "Filter results by user email (exact match)."},
			"members": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":       schema.StringAttribute{Computed: true},
						"email":    schema.StringAttribute{Computed: true},
						"name":     schema.StringAttribute{Computed: true},
						"role":     schema.StringAttribute{Computed: true},
						"added_at": schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *OrganizationMembersDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, diags := clientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	d.client = c
}

func (d *OrganizationMembersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data OrganizationMembersDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	list, err := d.client.ListUsers(ctx, anthropic.ListUsersParams{Email: data.Email.ValueString()})
	if err != nil {
		resp.Diagnostics.AddError("Failed to list organization members", err.Error())
		return
	}
	out := make([]OrganizationMemberResourceModel, 0, len(list))
	for i := range list {
		var m OrganizationMemberResourceModel
		userToModel(&list[i], &m)
		out = append(out, m)
	}
	data.Members = out
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}
