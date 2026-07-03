data "anthropic_compliance_group_members" "engineering" {
  group_id = "group_01XYZ..."
}

output "engineering_emails" {
  value = [for m in data.anthropic_compliance_group_members.engineering.members : m.email]
}
