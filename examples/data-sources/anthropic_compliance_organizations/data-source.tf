data "anthropic_compliance_organizations" "all" {}

output "org_ids" {
  value = [for o in data.anthropic_compliance_organizations.all.organizations : o.id]
}
