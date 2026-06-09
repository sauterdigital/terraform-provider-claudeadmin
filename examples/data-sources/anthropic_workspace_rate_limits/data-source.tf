data "anthropic_workspace_rate_limits" "platform" {
  workspace_id = "wrkspc_01JwQvzr7rXLA5AGx3HKfFUJ"
}

# Optional filter — only overrides for the token_count group.
data "anthropic_workspace_rate_limits" "platform_tokens" {
  workspace_id = "wrkspc_01JwQvzr7rXLA5AGx3HKfFUJ"
  group_type   = "token_count"
}
