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

const (
	envAdminAPIKey = "ANTHROPIC_ADMIN_API_KEY"
	envOAuthToken  = "ANTHROPIC_OAUTH_TOKEN"
)

type AnthropicProvider struct {
	version string
}

type AnthropicProviderModel struct {
	AdminAPIKey types.String `tfsdk:"admin_api_key"`
	OAuthToken  types.String `tfsdk:"oauth_token"`
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
		Description: "Manages Anthropic (Claude) platform resources via the Admin API. Most endpoints use the Admin API key (`admin_api_key`). A handful of newer surfaces — Service Accounts, Federation Issuers/Rules, MCP Tunnels — require OAuth Bearer auth (`oauth_token`) and reject the Admin API key. Configure both if you intend to manage those.",
		Attributes: map[string]schema.Attribute{
			"admin_api_key": schema.StringAttribute{
				Description: "Anthropic Admin API key (`sk-ant-admin-...`). May also be set via the ANTHROPIC_ADMIN_API_KEY environment variable. Used as `x-api-key` header.",
				Optional:    true,
				Sensitive:   true,
			},
			"oauth_token": schema.StringAttribute{
				Description: "OAuth Bearer token (user OAuth or WIF-minted service account token). May also be set via ANTHROPIC_OAUTH_TOKEN. When set, Bearer auth is used for ALL requests (the API's modern preferred pattern). Required for endpoints that reject Admin API keys: Service Accounts (Create/Update/Archive), SA Workspace Members, Federation Issuers, Federation Rules, MCP Tunnels.",
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

	oauthToken := os.Getenv(envOAuthToken)
	if !data.OAuthToken.IsNull() && !data.OAuthToken.IsUnknown() {
		oauthToken = data.OAuthToken.ValueString()
	}

	if apiKey == "" && oauthToken == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("admin_api_key"),
			"Missing Anthropic credentials",
			"Set either `admin_api_key` / "+envAdminAPIKey+" (for most Admin API endpoints) or `oauth_token` / "+envOAuthToken+" (for Service Accounts, Federation, MCP Tunnels).",
		)
		return
	}

	baseURL := "https://api.anthropic.com"
	if !data.BaseURL.IsNull() && !data.BaseURL.IsUnknown() {
		baseURL = data.BaseURL.ValueString()
	}

	client := anthropic.NewClient(baseURL, apiKey, p.version)
	if oauthToken != "" {
		client.SetOAuthToken(oauthToken)
	}
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
		NewSpendLimitResource,
		NewSpendLimitIncreaseDecisionResource,
		NewServiceAccountResource,
		NewServiceAccountWorkspaceResource,
		NewFederationIssuerResource,
		NewFederationRuleResource,
		NewFederationRuleWorkspaceResource,
		NewTunnelCertificateResource,
	}
}

func (p *AnthropicProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewOrganizationDataSource,
		NewWorkspaceDataSource,
		NewWorkspacesDataSource,
		NewWorkspaceRateLimitsDataSource,
		NewOrganizationRateLimitsDataSource,
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
		NewEffectiveSpendLimitsDataSource,
		NewSpendLimitIncreaseRequestDataSource,
		NewSpendLimitIncreaseRequestsDataSource,
		NewActivitySummariesDataSource,
		NewTokenUsageOverTimeDataSource,
		NewPerUserTokenUsageDataSource,
		NewCostOverTimeDataSource,
		NewPerUserCostDataSource,
		NewUserActivityDataSource,
		NewSkillsUsageDataSource,
		NewConnectorsUsageDataSource,
		NewChatProjectsUsageDataSource,
		NewServiceAccountDataSource,
		NewServiceAccountsDataSource,
		NewServiceAccountWorkspacesDataSource,
		NewWorkspaceServiceAccountsDataSource,
		NewTunnelDataSource,
		NewTunnelsDataSource,
		NewTunnelCertificatesDataSource,
		NewTunnelTokenDataSource,
	}
}
