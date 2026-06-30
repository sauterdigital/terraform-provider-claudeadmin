# Reveals the active MCP tunnel token. Treat as a secret.
data "anthropic_tunnel_token" "primary" {
  tunnel_id = "tunnel_01ABC..."
}

# Typical pattern — push the revealed token into a downstream secret store.
output "tunnel_token" {
  value     = data.anthropic_tunnel_token.primary.tunnel_token
  sensitive = true
}
