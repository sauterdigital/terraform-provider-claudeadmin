package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"

	"github.com/sauterdigital/terraform-provider-claude-admin/internal/anthropic"
)

var (
	_ datasource.DataSource              = &OrganizationMemberDataSource{}
	_ datasource.DataSourceWithConfigure = &OrganizationMemberDataSource{}
)

func NewOrganizationMemberDataSource() datasource.DataSource {
	return &OrganizationMemberDataSource{}
}

type OrganizationMemberDataSource struct{ client *anthropic.Client }

func (d *OrganizationMemberDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_member"
}

func (d *OrganizationMemberDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches a single organization member by user ID.",
		Attributes: map[string]schema.Attribute{
			"id":       schema.StringAttribute{Required: true},
			"email":    schema.StringAttribute{Computed: true},
			"name":     schema.StringAttribute{Computed: true},
			"role":     schema.StringAttribute{Computed: true},
			"added_at": schema.StringAttribute{Computed: true},
		},
	}
}

func (d *OrganizationMemberDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, diags := clientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	d.client = c
}

func (d *OrganizationMemberDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data OrganizationMemberResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	u, err := d.client.GetUser(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read organization member", err.Error())
		return
	}
	userToModel(u, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}
