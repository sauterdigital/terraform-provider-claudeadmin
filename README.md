# terraform-provider-claudeadmin

[![ci](https://github.com/sauterdigital/terraform-provider-claudeadmin/actions/workflows/ci.yml/badge.svg)](https://github.com/sauterdigital/terraform-provider-claudeadmin/actions/workflows/ci.yml)
[![license](https://img.shields.io/badge/license-MPL--2.0-blue.svg)](./LICENSE)

> **Unofficial, community-maintained** Terraform provider for the Anthropic Admin API. Not affiliated with, endorsed by, sponsored by, or supported by Anthropic PBC. "Anthropic", "Claude", and related marks are trademarks of Anthropic PBC and are used here solely to identify compatibility (nominative fair use). For official products and support, see [anthropic.com](https://www.anthropic.com).

Terraform provider for the [Anthropic Admin API](https://platform.claude.com/docs/en/api/admin), scoped to **Claude Console (Claude Platform)** organizations. Manages workspaces, API keys, organization and workspace members, invites, CMEK external keys, service accounts (OAuth Bearer), federation issuers + rules (workload identity federation), and MCP tunnel certificates + declarative token rotation (beta).

Covers **every documented Console Admin API endpoint** that fits a declarative resource model: 13 resources + 26 data sources.

> **Breaking changes (2026-08-12):**
> - Spend Limits (`claudeadmin_spend_limit`, `claudeadmin_spend_limit_increase_decision`, `claudeadmin_effective_spend_limits`, `claudeadmin_spend_limit_increase_request[s]`) moved to the sibling [`sauterdigital/claudeenterprise`](https://github.com/sauterdigital/terraform-provider-claudeenterprise) provider — it's a Claude-Enterprise-only surface and never belonged here.
> - **Analytics v2 and the Compliance / Compliance Content APIs were removed outright, not moved.** Both are Claude-Enterprise-only, and — on reflection — neither fits a Terraform resource model well: they're pure reporting/audit reads with no desired-state to converge on, better served by a script or BI pipeline calling [the Analytics API](https://platform.claude.com/docs/en/manage-claude/analytics-api) / [the Compliance API](https://platform.claude.com/docs/en/manage-claude/compliance-api) directly. See "Out of scope" below.
>
> Same schemas for the Spend Limits move: `terraform state rm` + re-`import` under the new provider source. For Analytics/Compliance there's no replacement resource — remove them from your config and call the APIs directly.

## Quick start

```hcl
terraform {
  required_providers {
    anthropic = {
      source  = "sauterdigital/claudeadmin"
      version = "~> 0.3"
    }
  }
}

provider "anthropic" {
  # admin_api_key = "sk-ant-admin-..."   # or ANTHROPIC_ADMIN_API_KEY
  # oauth_token   = "..."                # or ANTHROPIC_OAUTH_TOKEN (Service Accounts, Federation, MCP Tunnels)
}

resource "claudeadmin_workspace" "platform" {
  name = "platform"
  tags = {
    env  = "prod"
    team = "platform"
  }
}

# Daily cost per workspace for the last 30 days — feed into your FinOps stack.
data "claudeadmin_cost_report" "monthly" {
  starting_at  = formatdate("YYYY-MM-DD'T'00:00:00'Z'", timeadd(timestamp(), "-720h"))
  bucket_width = "1d"
  group_by     = ["workspace_id"]
}
```

The Admin API key is distinct from regular Claude API keys — generate it in the Anthropic Console under organization settings.

## What's included

**13 resources**

Authenticated with `admin_api_key` (x-api-key):

| Resource | Notes |
|---|---|
| `claudeadmin_workspace` | Full CRUD. Tags mutable, `external_key_id` write-once, `data_residency` triggers replace on change. |
| `claudeadmin_api_key` | Update-only — the Admin API can't create keys. Supply an existing `id` and the provider manages name/status. |
| `claudeadmin_workspace_member` | Composite id `<workspace_id>:<user_id>`; role mutable. |
| `claudeadmin_invite` | Immutable after create — changes to email/role force replacement. |
| `claudeadmin_organization_member` | Set org role for an existing user (joined via accepted invite). |
| `claudeadmin_external_key` | CMEK config CRUD + validate, polymorphic across AWS / GCP / Azure. |

Require `oauth_token` (Bearer auth) — Admin API keys are rejected:

| Resource | Notes |
|---|---|
| `claudeadmin_service_account` | Named non-human identity for federation. `admin`-role creation needs interactive credential. |
| `claudeadmin_service_account_workspace` | Assigns an SA to a workspace with a role. |
| `claudeadmin_federation_issuer` | OIDC issuer registration (GitHub Actions, GitLab, etc). Polymorphic JWKS source. |
| `claudeadmin_federation_rule` | Workload identity federation rule binding OIDC claims to an SA. |
| `claudeadmin_federation_rule_workspace` | Extends a rule to an additional workspace. |
| `claudeadmin_tunnel_certificate` | MCP tunnel CA certificate (beta, `mcp-tunnels-2026-06-22` header added automatically). |
| `claudeadmin_tunnel_token_rotation` | Declarative MCP tunnel token rotation. Change `rotation_id` to trigger a new rotation; fresh token becomes a sensitive state attribute. |

**26 data sources**

- Identity & membership: `claudeadmin_organization`, `claudeadmin_workspace[s]`, `claudeadmin_workspace_member[s]`, `claudeadmin_organization_member[s]`, `claudeadmin_invite[s]`
- Keys / CMEK: `claudeadmin_api_key[s]`, `claudeadmin_external_key[s]`
- Operational: `claudeadmin_organization_rate_limits`, `claudeadmin_workspace_rate_limits`
- FinOps reports (legacy v1, Console-available): `claudeadmin_usage_report`, `claudeadmin_claude_code_usage_report`, `claudeadmin_cost_report`
- Service accounts (Bearer): `claudeadmin_service_account[s]`, `claudeadmin_service_account_workspaces`, `claudeadmin_workspace_service_accounts`
- MCP Tunnels (Bearer + beta): `claudeadmin_tunnel[s]`, `claudeadmin_tunnel_certificates`, `claudeadmin_tunnel_token`

Full schema reference: [`docs/`](./docs).

## Out of scope, deliberately

- **Claude Enterprise (claude.ai) members with the `managed` role, RBAC groups, custom roles, and Spend Limits** — use [`sauterdigital/claudeenterprise`](https://github.com/sauterdigital/terraform-provider-claudeenterprise). A key from this provider's organization type cannot manage that one.
- **Analytics v2** (per-user/time-bucketed usage, cost, adoption) and **the Compliance / Compliance Content APIs** (audit activity feed, eDiscovery/DLP content export and deletion) are Claude-Enterprise-only, read/report-oriented surfaces with no meaningful desired-state to converge on — they don't fit a Terraform resource model, `.tf` files aren't the natural place to schedule a report or run an eDiscovery export. Call them directly: [Analytics API](https://platform.claude.com/docs/en/manage-claude/analytics-api), [Compliance API](https://platform.claude.com/docs/en/manage-claude/compliance-api). Both were implemented here through v0.5.0 and removed in 0.6.0 — this is a deliberate scope narrowing, not an oversight.

## Configuration

| Argument | Env var | Description |
|---|---|---|
| `admin_api_key` | `ANTHROPIC_ADMIN_API_KEY` | Admin API key (`sk-ant-admin-...`). Used as `x-api-key` header. Required for most endpoints. |
| `oauth_token` | `ANTHROPIC_OAUTH_TOKEN` | OAuth Bearer token (user OAuth or WIF-minted SA token). **Required** for Service Accounts, Federation, and MCP Tunnels (which reject Admin API keys). When set, Bearer auth is used for ALL requests. |
| `base_url` | — | Optional. Defaults to `https://api.anthropic.com`. Override for staging or mock servers. |

At least one of `admin_api_key` or `oauth_token` must be set. When both are configured the client uses Bearer (the API's modern preferred pattern). Every request sets `anthropic-version: 2023-06-01` and a provider-versioned `User-Agent`. HTTP 429 responses are retried with exponential backoff (capped at 30s), honoring `Retry-After` when present.

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
    "sauterdigital/claudeadmin" = "/path/to/your/$GOPATH/bin"
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
