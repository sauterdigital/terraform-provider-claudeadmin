package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/sauterdigital/terraform-provider-claude-admin/internal/anthropic"
)

var (
	_ datasource.DataSource              = &ComplianceOrganizationSettingsDataSource{}
	_ datasource.DataSourceWithConfigure = &ComplianceOrganizationSettingsDataSource{}
)

func NewComplianceOrganizationSettingsDataSource() datasource.DataSource {
	return &ComplianceOrganizationSettingsDataSource{}
}

type ComplianceOrganizationSettingsDataSource struct{ client *anthropic.Client }

type ComplianceOrganizationSettingsModel struct {
	OrganizationID        types.String `tfsdk:"organization_id"`
	SSOEnforced           types.Bool   `tfsdk:"sso_enforced"`
	MFAEnforced           types.Bool   `tfsdk:"mfa_enforced"`
	SCIMEnabled           types.Bool   `tfsdk:"scim_enabled"`
	AuditLogRetentionDays types.Int64  `tfsdk:"audit_log_retention_days"`
	NetworkACLEnabled     types.Bool   `tfsdk:"network_acl_enabled"`
	DataResidency         types.String `tfsdk:"data_residency"`
}

func (d *ComplianceOrganizationSettingsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_compliance_organization_settings"
}

func (d *ComplianceOrganizationSettingsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches the effective security settings for an organization (SSO enforcement, MFA enforcement, SCIM state, audit retention, Network ACL, data residency). Enterprise + Compliance API key with scope `read:compliance_org_data`. NOTE (2026-06-30): the previously-required `read:compliance_org_settings` scope was retired in favor of `read:compliance_org_data` — rotate keys created before that date.",
		Attributes: map[string]schema.Attribute{
			"organization_id":          schema.StringAttribute{Required: true},
			"sso_enforced":             schema.BoolAttribute{Computed: true},
			"mfa_enforced":             schema.BoolAttribute{Computed: true},
			"scim_enabled":             schema.BoolAttribute{Computed: true},
			"audit_log_retention_days": schema.Int64Attribute{Computed: true},
			"network_acl_enabled":      schema.BoolAttribute{Computed: true},
			"data_residency":           schema.StringAttribute{Computed: true},
		},
	}
}

func (d *ComplianceOrganizationSettingsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, diags := clientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	d.client = c
}

func (d *ComplianceOrganizationSettingsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var cfg ComplianceOrganizationSettingsModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}
	s, err := d.client.GetComplianceOrganizationSettings(ctx, cfg.OrganizationID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read compliance organization settings", err.Error())
		return
	}
	cfg.SSOEnforced = optionalBoolValue(s.SSOEnforced)
	cfg.MFAEnforced = optionalBoolValue(s.MFAEnforced)
	cfg.SCIMEnabled = optionalBoolValue(s.SCIMEnabled)
	cfg.AuditLogRetentionDays = optionalInt64Value(s.AuditLogRetentionDays)
	cfg.NetworkACLEnabled = optionalBoolValue(s.NetworkACLEnabled)
	cfg.DataResidency = optionalStringValue(s.DataResidency)
	resp.Diagnostics.Append(resp.State.Set(ctx, cfg)...)
}

func optionalBoolValue(b *bool) types.Bool {
	if b == nil {
		return types.BoolNull()
	}
	return types.BoolValue(*b)
}

func optionalInt64Value(n *int64) types.Int64 {
	if n == nil {
		return types.Int64Null()
	}
	return types.Int64Value(*n)
}
