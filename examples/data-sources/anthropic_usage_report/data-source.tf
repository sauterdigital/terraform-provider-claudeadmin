# Daily token usage for the last 7 days, broken down by workspace and model.
data "anthropic_usage_report" "last_week" {
  starting_at  = formatdate("YYYY-MM-DD'T'00:00:00'Z'", timeadd(timestamp(), "-168h"))
  bucket_width = "1d"
  group_by     = ["workspace_id", "model"]
}

output "total_output_tokens" {
  value = sum(flatten([
    for b in data.anthropic_usage_report.last_week.buckets : [
      for r in b.results : r.output_tokens
    ]
  ]))
}
