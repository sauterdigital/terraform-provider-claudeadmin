# Requires provider.compliance_api_key (sk-ant-api01-...) OR
# ANTHROPIC_COMPLIANCE_API_KEY env var. Enterprise-only.

data "anthropic_compliance_activities" "recent_admin" {
  starting_at   = "2026-07-01T00:00:00Z"
  ending_at     = "2026-07-31T23:59:59Z"
  resource_type = "workspace"
}

output "admin_actions_july" {
  value = data.anthropic_compliance_activities.recent_admin.activities
}
