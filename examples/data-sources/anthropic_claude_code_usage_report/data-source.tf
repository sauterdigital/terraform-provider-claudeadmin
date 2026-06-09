data "anthropic_claude_code_usage_report" "yesterday" {
  starting_at = formatdate("YYYY-MM-DD", timeadd(timestamp(), "-24h"))
}

output "sessions_by_user" {
  value = {
    for e in data.anthropic_claude_code_usage_report.yesterday.entries :
    coalesce(e.actor_email, e.actor_api_key_name) => e.core_metrics.num_sessions
  }
}
