data "anthropic_organization_members" "by_email" {
  email = "alice@example.com"
}

data "anthropic_organization_members" "all" {}
