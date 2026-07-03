package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/sauterdigital/terraform-provider-claude-admin/internal/anthropic"
)

type ExternalKeyModel struct {
	ID             types.String         `tfsdk:"id"`
	DisplayName    types.String         `tfsdk:"display_name"`
	Geo            types.String         `tfsdk:"geo"`
	CreatedAt      types.String         `tfsdk:"created_at"`
	UpdatedAt      types.String         `tfsdk:"updated_at"`
	ProviderConfig *ProviderConfigModel `tfsdk:"provider_config"`
}

type ProviderConfigModel struct {
	Type     types.String `tfsdk:"type"`
	KMSArn   types.String `tfsdk:"kms_arn"`
	RoleArn  types.String `tfsdk:"role_arn"`
	Region   types.String `tfsdk:"region"`
	KeyName  types.String `tfsdk:"key_name"`
	TenantID types.String `tfsdk:"tenant_id"`
	VaultURI types.String `tfsdk:"vault_uri"`
	ClientID types.String `tfsdk:"client_id"`
}

func externalKeyToModel(k *anthropic.ExternalKey, m *ExternalKeyModel) {
	m.ID = types.StringValue(k.ID)
	m.DisplayName = types.StringValue(k.DisplayName)
	m.Geo = types.StringValue(k.Geo)
	m.CreatedAt = types.StringValue(k.CreatedAt)
	m.UpdatedAt = types.StringValue(k.UpdatedAt)
	m.ProviderConfig = &ProviderConfigModel{
		Type:     types.StringValue(k.ProviderConfig.Type),
		KMSArn:   stringValueOrEmpty(k.ProviderConfig.KMSArn),
		RoleArn:  stringValueOrEmpty(k.ProviderConfig.RoleArn),
		Region:   stringValueOrEmpty(k.ProviderConfig.Region),
		KeyName:  stringValueOrEmpty(k.ProviderConfig.KeyName),
		TenantID: stringValueOrEmpty(k.ProviderConfig.TenantID),
		VaultURI: stringValueOrEmpty(k.ProviderConfig.VaultURI),
		ClientID: stringValueOrEmpty(k.ProviderConfig.ClientID),
	}
}

func providerConfigFromModel(m *ProviderConfigModel) anthropic.ProviderConfig {
	if m == nil {
		return anthropic.ProviderConfig{}
	}
	return anthropic.ProviderConfig{
		Type:     m.Type.ValueString(),
		KMSArn:   m.KMSArn.ValueString(),
		RoleArn:  m.RoleArn.ValueString(),
		Region:   m.Region.ValueString(),
		KeyName:  m.KeyName.ValueString(),
		TenantID: m.TenantID.ValueString(),
		VaultURI: m.VaultURI.ValueString(),
		ClientID: m.ClientID.ValueString(),
	}
}
