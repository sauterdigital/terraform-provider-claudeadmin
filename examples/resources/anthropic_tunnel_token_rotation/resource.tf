# Rotate the MCP tunnel token declaratively. Change `rotation_id` to trigger
# a new rotation — the resource replaces (RequiresReplace on all inputs) and a
# fresh `tunnel_token` is produced.

resource "anthropic_tunnel_token_rotation" "prod_q3" {
  tunnel_id   = "tunnel_01ABC..."
  rotation_id = "2026-Q3"
  reason      = "quarterly rotation"
}

output "prod_tunnel_token" {
  value     = anthropic_tunnel_token_rotation.prod_q3.tunnel_token
  sensitive = true
}
