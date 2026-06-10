# All rate-limit groups active on the organization (Messages API).
data "anthropic_organization_rate_limits" "all" {}

# Filter to a single group — e.g. only model_group entries.
data "anthropic_organization_rate_limits" "models_only" {
  group_type = "model_group"
}

# Filter to the entry containing a specific model. Returns 404 if absent.
data "anthropic_organization_rate_limits" "opus" {
  model = "claude-opus-4-7"
}

output "org_rpm_per_model" {
  value = {
    for g in data.anthropic_organization_rate_limits.all.groups :
    join(",", g.models) => {
      for l in g.limits : l.type => l.value
    } if g.group_type == "model_group"
  }
}
