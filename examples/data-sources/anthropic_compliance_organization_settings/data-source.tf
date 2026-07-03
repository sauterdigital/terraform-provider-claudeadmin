data "anthropic_compliance_organization_settings" "main" {
  organization_id = "org_01ABC..."
}

output "security_posture" {
  value = {
    sso_enforced       = data.anthropic_compliance_organization_settings.main.sso_enforced
    mfa_enforced       = data.anthropic_compliance_organization_settings.main.mfa_enforced
    scim_enabled       = data.anthropic_compliance_organization_settings.main.scim_enabled
    audit_retention    = data.anthropic_compliance_organization_settings.main.audit_log_retention_days
    network_acl_active = data.anthropic_compliance_organization_settings.main.network_acl_enabled
    data_residency     = data.anthropic_compliance_organization_settings.main.data_residency
  }
}
