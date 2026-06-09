terraform {
  required_providers {
    anthropic = {
      source  = "sauterdigital/anthropic"
      version = "~> 0.1"
    }
  }
}

provider "anthropic" {
  # admin_api_key = "sk-ant-admin-..." # or set ANTHROPIC_ADMIN_API_KEY
}
