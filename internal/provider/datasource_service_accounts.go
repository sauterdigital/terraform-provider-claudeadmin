package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/sauterdigital/terraform-provider-claude-admin/internal/anthropic"
)

// Single SA

var (
	_ datasource.DataSource              = &ServiceAccountDataSource{}
	_ datasource.DataSourceWithConfigure = &ServiceAccountDataSource{}
)

func NewServiceAccountDataSource() datasource.DataSource { return &ServiceAccountDataSource{} }

type ServiceAccountDataSource struct{ client *anthropic.Client }

func (d *ServiceAccountDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_account"
}

func (d *ServiceAccountDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches a single service account by ID. Requires OAuth Bearer auth.",
		Attributes: map[string]schema.Attribute{
			"id":                schema.StringAttribute{Required: true},
			"name":              schema.StringAttribute{Computed: true},
			"description":       schema.StringAttribute{Computed: true},
			"organization_role": schema.StringAttribute{Computed: true},
			"created_at":        schema.StringAttribute{Computed: true},
			"updated_at":        schema.StringAttribute{Computed: true},
			"archived_at":       schema.StringAttribute{Computed: true},
		},
	}
}

func (d *ServiceAccountDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, diags := clientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	d.client = c
}

func (d *ServiceAccountDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ServiceAccountModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	sa, err := d.client.GetServiceAccount(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read service account", err.Error())
		return
	}
	serviceAccountToModel(sa, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

// List SAs

var (
	_ datasource.DataSource              = &ServiceAccountsDataSource{}
	_ datasource.DataSourceWithConfigure = &ServiceAccountsDataSource{}
)

func NewServiceAccountsDataSource() datasource.DataSource { return &ServiceAccountsDataSource{} }

type ServiceAccountsDataSource struct{ client *anthropic.Client }

type ServiceAccountsListModel struct {
	IncludeArchived  types.Bool            `tfsdk:"include_archived"`
	OrganizationRole types.String          `tfsdk:"organization_role"`
	ServiceAccounts  []ServiceAccountModel `tfsdk:"service_accounts"`
}

func (d *ServiceAccountsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_accounts"
}

func (d *ServiceAccountsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists service accounts in the org. Requires OAuth Bearer auth.",
		Attributes: map[string]schema.Attribute{
			"include_archived":  schema.BoolAttribute{Optional: true},
			"organization_role": schema.StringAttribute{Optional: true, Validators: []validator.String{stringvalidator.OneOf("developer", "admin")}},
			"service_accounts": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":                schema.StringAttribute{Computed: true},
						"name":              schema.StringAttribute{Computed: true},
						"description":       schema.StringAttribute{Computed: true},
						"organization_role": schema.StringAttribute{Computed: true},
						"created_at":        schema.StringAttribute{Computed: true},
						"updated_at":        schema.StringAttribute{Computed: true},
						"archived_at":       schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *ServiceAccountsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, diags := clientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	d.client = c
}

func (d *ServiceAccountsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ServiceAccountsListModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	list, err := d.client.ListServiceAccounts(ctx, anthropic.ListServiceAccountsParams{
		IncludeArchived:  data.IncludeArchived.ValueBool(),
		OrganizationRole: data.OrganizationRole.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to list service accounts", err.Error())
		return
	}
	out := make([]ServiceAccountModel, 0, len(list))
	for i := range list {
		var m ServiceAccountModel
		serviceAccountToModel(&list[i], &m)
		out = append(out, m)
	}
	data.ServiceAccounts = out
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

// List SA workspaces (for a SA)

var (
	_ datasource.DataSource              = &ServiceAccountWorkspacesDataSource{}
	_ datasource.DataSourceWithConfigure = &ServiceAccountWorkspacesDataSource{}
)

func NewServiceAccountWorkspacesDataSource() datasource.DataSource {
	return &ServiceAccountWorkspacesDataSource{}
}

type ServiceAccountWorkspacesDataSource struct{ client *anthropic.Client }

type ServiceAccountWorkspacesListModel struct {
	ServiceAccountID types.String                   `tfsdk:"service_account_id"`
	Workspaces       []ServiceAccountWorkspaceModel `tfsdk:"workspaces"`
}

func (d *ServiceAccountWorkspacesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_account_workspaces"
}

func (d *ServiceAccountWorkspacesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all workspace memberships for a service account.",
		Attributes: map[string]schema.Attribute{
			"service_account_id": schema.StringAttribute{Required: true},
			"workspaces": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":                 schema.StringAttribute{Computed: true},
						"service_account_id": schema.StringAttribute{Computed: true},
						"workspace_id":       schema.StringAttribute{Computed: true},
						"workspace_role":     schema.StringAttribute{Computed: true},
						"implicit":           schema.BoolAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *ServiceAccountWorkspacesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, diags := clientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	d.client = c
}

func (d *ServiceAccountWorkspacesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ServiceAccountWorkspacesListModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	list, err := d.client.ListServiceAccountWorkspaces(ctx, data.ServiceAccountID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to list SA workspaces", err.Error())
		return
	}
	out := make([]ServiceAccountWorkspaceModel, 0, len(list))
	for i := range list {
		var m ServiceAccountWorkspaceModel
		saWorkspaceToModel(&list[i], &m)
		out = append(out, m)
	}
	data.Workspaces = out
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

// List SAs in a workspace

var (
	_ datasource.DataSource              = &WorkspaceServiceAccountsDataSource{}
	_ datasource.DataSourceWithConfigure = &WorkspaceServiceAccountsDataSource{}
)

func NewWorkspaceServiceAccountsDataSource() datasource.DataSource {
	return &WorkspaceServiceAccountsDataSource{}
}

type WorkspaceServiceAccountsDataSource struct{ client *anthropic.Client }

type WorkspaceServiceAccountsListModel struct {
	WorkspaceID     types.String                   `tfsdk:"workspace_id"`
	ServiceAccounts []ServiceAccountWorkspaceModel `tfsdk:"service_accounts"`
}

func (d *WorkspaceServiceAccountsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workspace_service_accounts"
}

func (d *WorkspaceServiceAccountsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all service accounts that are members of a workspace.",
		Attributes: map[string]schema.Attribute{
			"workspace_id": schema.StringAttribute{Required: true},
			"service_accounts": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":                 schema.StringAttribute{Computed: true},
						"service_account_id": schema.StringAttribute{Computed: true},
						"workspace_id":       schema.StringAttribute{Computed: true},
						"workspace_role":     schema.StringAttribute{Computed: true},
						"implicit":           schema.BoolAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *WorkspaceServiceAccountsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, diags := clientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	d.client = c
}

func (d *WorkspaceServiceAccountsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data WorkspaceServiceAccountsListModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	list, err := d.client.ListWorkspaceServiceAccountMembers(ctx, data.WorkspaceID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to list workspace service accounts", err.Error())
		return
	}
	out := make([]ServiceAccountWorkspaceModel, 0, len(list))
	for i := range list {
		var m ServiceAccountWorkspaceModel
		saWorkspaceToModel(&list[i], &m)
		out = append(out, m)
	}
	data.ServiceAccounts = out
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}
