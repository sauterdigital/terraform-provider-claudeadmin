terraform {
  required_providers {
    anthropic = {
      source  = "sauterdigital/claude-admin"
      version = "~> 0.3"
    }
  }
}

provider "anthropic" {
  # admin_api_key      = "sk-ant-admin-..." # or ANTHROPIC_ADMIN_API_KEY
  # oauth_token        = "..."              # or ANTHROPIC_OAUTH_TOKEN (Service Accounts, Federation, MCP Tunnels)
  # compliance_api_key = "sk-ant-api01-..." # or ANTHROPIC_COMPLIANCE_API_KEY (Compliance data sources)
}
