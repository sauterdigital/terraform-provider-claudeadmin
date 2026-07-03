# terraform-provider-claude-admin

[![ci](https://github.com/sauterdigital/terraform-provider-claude-admin/actions/workflows/ci.yml/badge.svg)](https://github.com/sauterdigital/terraform-provider-claude-admin/actions/workflows/ci.yml)
[![license](https://img.shields.io/badge/license-MPL--2.0-blue.svg)](./LICENSE)

> **Unofficial, community-maintained** Terraform provider for the Anthropic Admin API. Not affiliated with, endorsed by, sponsored by, or supported by Anthropic PBC. "Anthropic", "Claude", and related marks are trademarks of Anthropic PBC and are used here solely to identify compatibility (nominative fair use). For official products and support, see [anthropic.com](https://www.anthropic.com).

Terraform provider for the [Anthropic Admin API](https://platform.claude.com/docs/en/api/admin). Manages workspaces, API keys, organization and workspace members, invites, CMEK external keys, service accounts (OAuth Bearer), federation issuers + rules (workload identity federation), spend limits, MCP tunnel certificates + declarative token rotation (beta), Compliance API data (activity feed, orgs, users, roles, SCIM groups, org security settings), and exposes the full analytics surface (token usage, cost, Claude Code, Skills, Connectors, Chat Projects, per-user breakdowns) as data sources for FinOps pipelines.

Covers **every documented Admin API endpoint** plus Compliance API: 15 resources + 44 data sources spanning ~90 endpoints.

## Quick start

```hcl
terraform {
  required_providers {
    anthropic = {
      source  = "sauterdigital/claude-admin"
      version = "~> 0.3"
    }
  }
}

provider "anthropic" {
  # admin_api_key      = "sk-ant-admin-..."   # or ANTHROPIC_ADMIN_API_KEY
  # oauth_token        = "..."                # or ANTHROPIC_OAUTH_TOKEN (Service Accounts, Federation, MCP Tunnels)
  # compliance_api_key = "sk-ant-api01-..."   # or ANTHROPIC_COMPLIANCE_API_KEY (Compliance data sources)
}

resource "anthropic_workspace" "platform" {
  name = "platform"
  tags = {
    env  = "prod"
    team = "platform"
  }
}

# Daily cost per workspace for the last 30 days — feed into your FinOps stack.
data "anthropic_cost_report" "monthly" {
  starting_at  = formatdate("YYYY-MM-DD'T'00:00:00'Z'", timeadd(timestamp(), "-720h"))
  bucket_width = "1d"
  group_by     = ["workspace_id"]
}
```

The Admin API key is distinct from regular Claude API keys — generate it in the Anthropic Console under organization settings. The Compliance API key (Enterprise) is issued separately for `/v1/compliance/*` endpoints.

## What's included

**15 resources**

Authenticated with `admin_api_key` (x-api-key):

| Resource | Notes |
|---|---|
| `anthropic_workspace` | Full CRUD. Tags mutable, `external_key_id` write-once, `data_residency` triggers replace on change. |
| `anthropic_api_key` | Update-only — the Admin API can't create keys. Supply an existing `id` and the provider manages name/status. |
| `anthropic_workspace_member` | Composite id `<workspace_id>:<user_id>`; role mutable. |
| `anthropic_invite` | Immutable after create — changes to email/role force replacement. |
| `anthropic_organization_member` | Set org role for an existing user (joined via accepted invite). |
| `anthropic_external_key` | CMEK config CRUD + validate, polymorphic across AWS / GCP / Azure. |
| `anthropic_spend_limit` | Per-user spend limit override (org/group/seat-tier limits stay in Console). |
| `anthropic_spend_limit_increase_decision` | Approve or deny a user's request to raise their cap. |

Require `oauth_token` (Bearer auth) — Admin API keys are rejected:

| Resource | Notes |
|---|---|
| `anthropic_service_account` | Named non-human identity for federation. `admin`-role creation needs interactive credential. |
| `anthropic_service_account_workspace` | Assigns an SA to a workspace with a role. |
| `anthropic_federation_issuer` | OIDC issuer registration (GitHub Actions, GitLab, etc). Polymorphic JWKS source. |
| `anthropic_federation_rule` | Workload identity federation rule binding OIDC claims to an SA. |
| `anthropic_federation_rule_workspace` | Extends a rule to an additional workspace. |
| `anthropic_tunnel_certificate` | MCP tunnel CA certificate (beta, `mcp-tunnels-2026-06-22` header added automatically). |
| `anthropic_tunnel_token_rotation` | Declarative MCP tunnel token rotation. Change `rotation_id` to trigger a new rotation; fresh token becomes a sensitive state attribute. |

**44 data sources**

- Identity & membership: `anthropic_organization`, `anthropic_workspace[s]`, `anthropic_workspace_member[s]`, `anthropic_organization_member[s]`, `anthropic_invite[s]`
- Keys / CMEK: `anthropic_api_key[s]`, `anthropic_external_key[s]`
- Operational: `anthropic_organization_rate_limits`, `anthropic_workspace_rate_limits`
- FinOps reports (legacy v1): `anthropic_usage_report`, `anthropic_claude_code_usage_report`, `anthropic_cost_report`
- FinOps automation: `anthropic_effective_spend_limits`, `anthropic_spend_limit_increase_request[s]`
- Analytics v2 (Enterprise + `read:analytics` scope): `anthropic_activity_summaries`, `anthropic_token_usage_over_time`, `anthropic_per_user_token_usage`, `anthropic_cost_over_time`, `anthropic_per_user_cost`, `anthropic_user_activity`, `anthropic_skills_usage`, `anthropic_connectors_usage`, `anthropic_chat_projects_usage`
- Service accounts (Bearer): `anthropic_service_account[s]`, `anthropic_service_account_workspaces`, `anthropic_workspace_service_accounts`
- MCP Tunnels (Bearer + beta): `anthropic_tunnel[s]`, `anthropic_tunnel_certificates`, `anthropic_tunnel_token`
- Compliance API (dedicated `compliance_api_key`, Enterprise): `anthropic_compliance_activities`, `anthropic_compliance_organizations`, `anthropic_compliance_organization_users`, `anthropic_compliance_organization_roles`, `anthropic_compliance_groups`, `anthropic_compliance_group_members`, `anthropic_compliance_organization_settings`

Full schema reference: [`docs/`](./docs).

## Configuration

| Argument | Env var | Description |
|---|---|---|
| `admin_api_key` | `ANTHROPIC_ADMIN_API_KEY` | Admin API key (`sk-ant-admin-...`). Used as `x-api-key` header. Required for most endpoints. |
| `oauth_token` | `ANTHROPIC_OAUTH_TOKEN` | OAuth Bearer token (user OAuth or WIF-minted SA token). **Required** for Service Accounts, Federation, and MCP Tunnels (which reject Admin API keys). When set, Bearer auth is used for ALL non-Compliance requests. |
| `compliance_api_key` | `ANTHROPIC_COMPLIANCE_API_KEY` | Compliance API key (`sk-ant-api01-...`). Used exclusively for `/v1/compliance/*` endpoints — those reject both Admin API keys and OAuth bearer tokens. Required to use any `anthropic_compliance_*` data source. |
| `base_url` | — | Optional. Defaults to `https://api.anthropic.com`. Override for staging or mock servers. |

At least one of `admin_api_key` or `oauth_token` must be set. When both are configured the client uses Bearer (the API's modern preferred pattern). Compliance calls always use `compliance_api_key`. Every request sets `anthropic-version: 2023-06-01` and a provider-versioned `User-Agent`. HTTP 429 responses are retried with exponential backoff (capped at 30s), honoring `Retry-After` when present.

## Development

Requirements: Go 1.25, Terraform ≥ 1.0.

```bash
make build                       # compile the provider binary
make test                        # unit tests (fast, no API access)
make testacc                     # acceptance tests — requires TF_ACC=1 + ANTHROPIC_ADMIN_API_KEY, creates real workspaces
make fmt vet                     # gofmt + go vet
make docs                        # regenerate docs/ (requires tfplugindocs in PATH)

go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@latest   # one-time
```

To use a local build in a real config without publishing, add a `dev_overrides` block to `~/.terraformrc`:

```hcl
provider_installation {
  dev_overrides {
    "sauterdigital/claude-admin" = "/path/to/your/$GOPATH/bin"
  }
  direct {}
}
```

Then `make install` and run `terraform plan` against `examples/`.

## CI

- [`ci.yml`](./.github/workflows/ci.yml) runs on every push/PR: build, vet, gofmt, race-enabled unit tests, `terraform fmt` on examples, and a docs-drift check.
- [`acceptance.yml`](./.github/workflows/acceptance.yml) is `workflow_dispatch` only — acceptance tests mutate the real organization and incur API cost, so they never run automatically. Requires `ANTHROPIC_ADMIN_API_KEY` in repo secrets.
- [`release.yml`](./.github/workflows/release.yml) fires on `v*` tag push: builds signed, multi-arch artifacts via goreleaser, attaches them to a GitHub Release, and includes the `terraform-registry-manifest.json` the Terraform Registry needs to ingest the release. Requires `GPG_PRIVATE_KEY` and `PASSPHRASE` repo secrets.

## Publishing a release

1. Confirm `go test -race ./...` passes locally and `make docs` shows no diff.
2. Bump the `VERSION` in `Makefile`, commit, then tag: `git tag -a v0.X.Y -m "..."` and `git push origin v0.X.Y`.
3. The `release` workflow builds the binaries, signs the checksum file with the GPG key, and creates a GitHub Release.
4. First-time only: register the provider at https://registry.terraform.io — Public Namespaces → sauterdigital → Publish → Provider — and upload the matching GPG public key. Subsequent releases are picked up automatically when the workflow finishes.

## Trademark & disclaimer

This project is not affiliated with Anthropic PBC. All Anthropic product names, logos, and brands are property of their respective owners. References to "Anthropic", "Claude", or related marks appear here strictly to describe API compatibility, which is nominative fair use. For official Anthropic products and enterprise support, contact Anthropic directly.

## License

[Mozilla Public License 2.0](./LICENSE) — the standard for HashiCorp-ecosystem Terraform providers.
