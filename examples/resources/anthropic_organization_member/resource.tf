# Users join an organization by accepting an `anthropic_invite`. Once the user
# exists, manage their org-level role declaratively here. Destroying this
# resource removes the user from the organization.
resource "anthropic_organization_member" "alice" {
  id   = "user_01WCz1FkmYMm4gnmykNKUu3Q"
  role = "developer"
}
