# Daily cost per workspace for the last 30 days.
data "anthropic_cost_report" "last_month" {
  starting_at  = formatdate("YYYY-MM-DD'T'00:00:00'Z'", timeadd(timestamp(), "-720h"))
  bucket_width = "1d"
  group_by     = ["workspace_id"]
}

output "cost_per_workspace" {
  value = {
    for b in data.anthropic_cost_report.last_month.buckets :
    b.starting_at => {
      for r in b.results : coalesce(r.workspace_id, "default") => r.amount
    }
  }
}
