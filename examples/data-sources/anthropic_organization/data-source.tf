data "anthropic_organization" "current" {}

output "org_id" {
  value = data.anthropic_organization.current.id
}
