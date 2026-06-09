resource "anthropic_workspace_member" "platform_dev" {
  workspace_id   = anthropic_workspace.example.id
  user_id        = data.anthropic_organization_member.alice.id
  workspace_role = "workspace_developer"
}
