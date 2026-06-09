resource "anthropic_workspace" "example" {
  name = "engineering"

  tags = {
    env  = "prod"
    team = "platform"
  }
}

output "workspace_id" {
  value = anthropic_workspace.example.id
}
