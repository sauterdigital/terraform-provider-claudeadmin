package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/sauterdigital/terraform-provider-anthropic/internal/anthropic"
)

const envAdminAPIKey = "ANTHROPIC_ADMIN_API_KEY"

type AnthropicProvider struct {
	version string
}

type AnthropicProviderModel struct {
	AdminAPIKey types.String `tfsdk:"admin_api_key"`
	BaseURL     types.String `tfsdk:"base_url"`
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &AnthropicProvider{version: version}
	}
}

func (p *AnthropicProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "anthropic"
	resp.Version = p.version
}

func (p *AnthropicProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages Anthropic (Claude) platform resources via the Admin API.",
		Attributes: map[string]schema.Attribute{
			"admin_api_key": schema.StringAttribute{
				Description: "Anthropic Admin API key. May also be set via the ANTHROPIC_ADMIN_API_KEY environment variable.",
				Optional:    true,
				Sensitive:   true,
			},
			"base_url": schema.StringAttribute{
				Description: "Override the API base URL. Defaults to https://api.anthropic.com.",
				Optional:    true,
			},
		},
	}
}

func (p *AnthropicProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data AnthropicProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiKey := os.Getenv(envAdminAPIKey)
	if !data.AdminAPIKey.IsNull() && !data.AdminAPIKey.IsUnknown() {
		apiKey = data.AdminAPIKey.ValueString()
	}
	if apiKey == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("admin_api_key"),
			"Missing Anthropic Admin API key",
			"Set the `admin_api_key` provider attribute or the "+envAdminAPIKey+" environment variable.",
		)
		return
	}

	baseURL := "https://api.anthropic.com"
	if !data.BaseURL.IsNull() && !data.BaseURL.IsUnknown() {
		baseURL = data.BaseURL.ValueString()
	}

	client := anthropic.NewClient(baseURL, apiKey, p.version)
	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *AnthropicProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewWorkspaceResource,
		NewAPIKeyResource,
		NewWorkspaceMemberResource,
		NewInviteResource,
		NewOrganizationMemberResource,
		NewExternalKeyResource,
	}
}

func (p *AnthropicProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewOrganizationDataSource,
		NewWorkspaceDataSource,
		NewWorkspacesDataSource,
		NewWorkspaceRateLimitsDataSource,
		NewAPIKeyDataSource,
		NewAPIKeysDataSource,
		NewWorkspaceMemberDataSource,
		NewWorkspaceMembersDataSource,
		NewInviteDataSource,
		NewInvitesDataSource,
		NewOrganizationMemberDataSource,
		NewOrganizationMembersDataSource,
		NewExternalKeyDataSource,
		NewExternalKeysDataSource,
		NewUsageReportDataSource,
		NewClaudeCodeUsageReportDataSource,
		NewCostReportDataSource,
	}
}
