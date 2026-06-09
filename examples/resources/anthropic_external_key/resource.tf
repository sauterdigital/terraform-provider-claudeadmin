resource "anthropic_external_key" "aws_prod" {
  display_name = "prod-cmek"
  provider_config = {
    type     = "aws"
    kms_arn  = "arn:aws:kms:us-east-1:123456789012:key/abc123"
    role_arn = "arn:aws:iam::123456789012:role/anthropic-cmek"
    region   = "us-east-1"
  }
}

resource "anthropic_external_key" "gcp_eu" {
  display_name = "eu-cmek"
  provider_config = {
    type     = "gcp"
    key_name = "projects/p/locations/global/keyRings/r/cryptoKeys/k"
  }
}
