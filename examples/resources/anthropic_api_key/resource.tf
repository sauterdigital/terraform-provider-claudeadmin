# API keys cannot be created via the Admin API. Create the key in the Anthropic
# Console first, then reference its ID here to manage name/status declaratively.
resource "anthropic_api_key" "ci" {
  id     = "apikey_01Rj2N8SVvo6BePZj99NhmiT"
  name   = "ci-runner"
  status = "active"
}
