data "anthropic_compliance_organization_roles" "main" {
  organization_id = "org_01ABC..."
}

# Audit custom (non-built-in) roles and their permissions
output "custom_roles" {
  value = [
    for r in data.anthropic_compliance_organization_roles.main.roles : {
      name        = r.name
      permissions = r.permissions
    }
    if !r.is_built_in
  ]
}
