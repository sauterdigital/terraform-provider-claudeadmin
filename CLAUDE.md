# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project goal

Build a Terraform provider that manages Anthropic (Claude) platform resources through the public API at https://platform.claude.com/docs/en/api/overview.

Scope is the **Admin API** surface: https://platform.claude.com/docs/en/api/admin — primarily:

- Organization members (list / get / update role / remove)
- Organization invites (create / list / get / delete)
- Workspaces (create / list / get / update / archive)
- Workspace members (add / list / get / update / remove)
- API keys (list / get / update — note: keys are **not created** by the Admin API, so the provider will likely expose them as data sources plus an update-only resource)

The Admin API requires an **admin API key** (`x-api-key`) distinct from regular Claude API keys.

## Status

Feature-complete v0.1: full Admin API surface, validators on every enum, retry/backoff for 429, generated docs, unit tests, acceptance test scaffolding, and CI. `go build ./... && go vet ./... && gofmt -l . && go test -race ./...` all green; `terraform fmt -check -recursive examples/` clean. Not yet validated against the live Admin API — acceptance tests require `TF_ACC=1` + `ANTHROPIC_ADMIN_API_KEY` and have not been executed.

### Resources (15)

Admin API key compatible:

| Resource | Notes |
|---|---|
| `anthropic_workspace` | Create / Read / Update / Delete-archives. `tags` mutable, `external_key_id` write-once, `data_residency` configurable with RequiresReplace on change. |
| `anthropic_api_key` | Update-only — Admin API does not create keys. `Create` requires an existing `id`, applies name/status, `Delete` sets status=`archived`. |
| `anthropic_workspace_member` | Composite id `<workspace_id>:<user_id>`. Role mutable. |
| `anthropic_invite` | Create + Delete only — invites are immutable. `admin` role not allowed. |
| `anthropic_organization_member` | Update-only — users join via accepted invite. `Create` requires existing user id. |
| `anthropic_external_key` | Full CRUD + validate for CMEK. Polymorphic `provider_config` (aws/gcp/azure). |
| `anthropic_spend_limit` | Per-user spend limit override. Only `scope.type=user` is API-writable; group/seat-tier/org limits are Console-only. |
| `anthropic_spend_limit_increase_decision` | Approve or deny a pending request. One-way (no Update). |

**Require OAuth Bearer (`oauth_token`)** — Admin API keys rejected by the API:

| Resource | Notes |
|---|---|
| `anthropic_service_account` | Named non-human identity for federation. `admin`-role create needs interactive credential. |
| `anthropic_service_account_workspace` | SA → workspace assignment. Composite id `<sa_id>:<workspace_id>`. |
| `anthropic_federation_issuer` | OIDC issuer. Polymorphic JWKS source (discovery / explicit_url / inline). |
| `anthropic_federation_rule` | Binds OIDC claims to a service account. Match supports audience, claims map, CEL condition, subject_prefix. |
| `anthropic_federation_rule_workspace` | Extends a rule to an additional workspace. Composite id. |
| `anthropic_tunnel_certificate` | MCP tunnel CA cert (beta, `anthropic-beta: mcp-tunnels-2026-06-22` added automatically). Uses `/v1/tunnels` (public API). |
| `anthropic_tunnel_token_rotation` | Declarative token rotation. Change `rotation_id` to trigger a new rotation; fresh `tunnel_token` becomes the sensitive output. |

### Data sources (44)

Identity & membership: `anthropic_organization`, `anthropic_workspace[s]`, `anthropic_workspace_member[s]`, `anthropic_organization_member[s]`, `anthropic_invite[s]`.

Keys / CMEK: `anthropic_api_key[s]`, `anthropic_external_key[s]`.

Rate limits: `anthropic_organization_rate_limits` (org baseline), `anthropic_workspace_rate_limits` (workspace overrides).

FinOps reports (legacy v1): `anthropic_usage_report`, `anthropic_claude_code_usage_report`, `anthropic_cost_report`.

FinOps automation: `anthropic_effective_spend_limits`, `anthropic_spend_limit_increase_request[s]`.

Analytics v2 (Enterprise + `read:analytics` scope): `anthropic_activity_summaries`, `anthropic_token_usage_over_time`, `anthropic_per_user_token_usage`, `anthropic_cost_over_time`, `anthropic_per_user_cost`, `anthropic_user_activity`, `anthropic_skills_usage`, `anthropic_connectors_usage`, `anthropic_chat_projects_usage`.

Service accounts (Bearer auth): `anthropic_service_account[s]`, `anthropic_service_account_workspaces`, `anthropic_workspace_service_accounts`.

MCP Tunnels (Bearer + beta): `anthropic_tunnel[s]`, `anthropic_tunnel_certificates`, `anthropic_tunnel_token`.

Compliance API (dedicated `compliance_api_key`, Enterprise): `anthropic_compliance_activities`, `anthropic_compliance_organizations`, `anthropic_compliance_organization_users`, `anthropic_compliance_organization_roles`, `anthropic_compliance_groups`, `anthropic_compliance_group_members`, `anthropic_compliance_organization_settings`.

The FinOps reports + analytics v2 + CMEK + spend limits + service accounts + federation are the **headline differentiation** vs `terraform-mars/terraform-provider-anthropic`, which covers only workspaces, api_keys, workspace_members, and invites.

### Coverage vs Admin API docs

We cover **every** endpoint group documented at https://platform.claude.com/docs/en/api/admin, plus Compliance API (`/v1/compliance/*`) as of v0.3.0. The client supports three auth modes (x-api-key admin, Bearer OAuth, x-api-key compliance); endpoints that reject Admin API keys (Service Accounts, Federation, MCP Tunnels) check `Client.HasOAuth()` upfront and return `ErrOAuthRequired`. Compliance endpoints check `Client.HasCompliance()` and return `ErrComplianceRequired`. MCP Tunnels endpoints automatically attach the `anthropic-beta: mcp-tunnels-2026-06-22` header via `WithBetaHeaders`. Audit Logs are NOT in the Admin API (confirmed via doc audit on 2026-06-10).

Doc-noted limitations we accept:
- `anthropic_spend_limit` only accepts `scope.type=user` for writes; seat-tier / rbac_group / organization-service / organization-level limits remain Console-only.
- `anthropic_tunnel_certificate` uses the Admin API path (`/v1/organizations/tunnels/...`) which is being deprecated in favor of `/v1/tunnels` on the public API. Track migration when the new path stabilizes.
- Federation Issuer / Rule mutations have additional scope restrictions (Console session required when granting `org:admin` scope or managing certain non-`workspace:developer` / `workspace:inference` scopes). Provider passes the call through; the API will reject if scope mismatches.

## Competitive context

A community provider exists at `github.com/terraform-mars/terraform-provider-anthropic` (MIT, v0.3.0 as of Jan 2026) covering 4 resources + 4 data sources (workspaces, api_keys, workspace_members, invites + workspace/api_key list data sources). We chose greenfield over forking; the justification is **differentiation, not parity**.

This repo now ships everything terraform-mars has plus: organization members, external keys (CMEK), workspace rate limits, three usage/cost reports (messages, claude_code, cost), organization metadata, and data sources for every resource we expose (terraform-mars omits invite/member data sources). When adding new functionality, keep this stance: parity is table stakes; the goal is broader Admin API coverage with first-class FinOps signal.

## Layout

```
main.go                                # provider entrypoint
internal/provider/provider.go          # provider config + resource/data-source registry
internal/provider/common.go            # shared client-unwrap helper + small types-utils
internal/provider/workspace_shared.go  # data_residency object plumbing + tags helpers
internal/provider/resource_*.go        # one file per resource (5 total)
internal/provider/datasource_*.go      # one file per data source (12 total)
internal/anthropic/                    # HTTP client, one file per Admin API resource group
  client.go               # shared do() with x-api-key + anthropic-version
  organization.go         # GET /v1/organizations/me
  workspaces.go
  rate_limits.go          # workspace rate-limit overrides (read-only)
  api_keys.go
  workspace_members.go
  invites.go
  users.go                # org members
  external_keys.go        # CMEK configs (CRUD + validate)
  reports.go              # messages usage, claude_code usage, cost report
examples/                              # consumed by tfplugindocs
  provider/provider.tf
  resources/<name>/resource.tf
  resources/<name>/import.sh (where importable)
  data-sources/<name>/data-source.tf
docs/                                  # generated — do not hand-edit
```

Provider address is `registry.terraform.io/sauterdigital/anthropic` (placeholder — update `main.go` and `examples/provider/provider.tf` if the namespace changes). Module path is `github.com/sauterdigital/terraform-provider-anthropic`.

Stack: Go 1.25, `terraform-plugin-framework` v1.19+, plus `terraform-plugin-framework-validators` for input validation and `terraform-plugin-testing` for acceptance tests when they are added.

## Common commands

```bash
make build                                            # go build -o terraform-provider-anthropic
make install                                          # build + place under ~/.terraform.d/plugins/...
make test                                             # unit tests (fast, no API key)
go test -race ./...                                   # same with race detector
make testacc                                          # TF_ACC=1 — hits real API, costs $, mutates state
make fmt vet                                          # gofmt -s -w + go vet
make docs                                             # tfplugindocs generate (binary must be installed)

go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@latest  # one-time
go test ./internal/provider/ -v -run TestAccWorkspace_basic                    # single acceptance test

terraform fmt -check -recursive examples/             # CI also runs this
```

For local end-to-end testing without publishing, point `~/.terraformrc` `dev_overrides` at the built binary, then `terraform plan` against `examples/`.

## CI

`.github/workflows/ci.yml` runs on every push/PR: build, vet, gofmt, race-enabled unit tests, terraform fmt on examples, and a docs-drift check (regenerates `docs/` and fails if anything changed — keep `docs/` committed in sync).

`.github/workflows/acceptance.yml` is `workflow_dispatch` only — it never runs on push because it mutates the real Anthropic org and costs money. Requires `ANTHROPIC_ADMIN_API_KEY` in repo secrets.

## Architecture notes

- **Authentication**: provider accepts `admin_api_key` (env `ANTHROPIC_ADMIN_API_KEY`, x-api-key header) and/or `oauth_token` (env `ANTHROPIC_OAUTH_TOKEN`, Bearer header). When both are set Bearer takes precedence — that's the doc's modern preferred pattern, and a handful of newer endpoints (Service Accounts, Federation, MCP Tunnels) reject x-api-key outright. The client exposes `HasOAuth()` so resource layer can fail-fast with `ErrOAuthRequired` when an endpoint needs Bearer.
- **Beta headers**: `WithBetaHeaders(ctx, "mcp-tunnels-2026-05-19")` adds `anthropic-beta` headers via context. MCP Tunnel client functions do this automatically in `tunnels.go`.
- **Client**: a single `internal/anthropic.Client` is constructed in `provider.Configure` and passed via `resp.ResourceData` / `resp.DataSourceData`. Every resource/data source unwraps it through `clientFromProviderData` in `internal/provider/common.go` — do not instantiate HTTP clients inside resource methods.
- **Headers**: every request sets the chosen auth header, `anthropic-version: 2023-06-01`, `content-type: application/json`, and a `user-agent` carrying the provider version.
- **Retries**: the client retries HTTP 429 up to `maxRetries` (5) times with exponential backoff (base 500ms, capped at 30s, plus small jitter). Honors `Retry-After` header when the API sends one. After exhaustion, returns an `APIError` with `Type: "rate_limit"` and a "max retries exceeded" message. The retry path is testable via the `sleeper` hook on `Client` — see `TestClient_RetriesOn429UntilSuccess`.
- **Pagination**: list endpoints use cursor pagination (`after_id` / `before_id` + `limit`). The client's `List*` methods follow `has_more` + `last_id` until exhausted and return the full slice — data sources don't reimplement paging.
- **Validators**: every enumerated string field (roles, statuses, bucket_width, group_by, provider_config.type, geo, etc.) has a `stringvalidator.OneOf` / `listvalidator.ValueStringsAre(...)` so invalid values fail at plan time with a clear message instead of bouncing off the API.
- **ID semantics**: Anthropic resources use prefixed string IDs (`user_…`, `wrkspc_…`, `apikey_…`, `invite_…`). These flow straight through as Terraform IDs. The only synthesized composite ID is `anthropic_workspace_member.id` = `<workspace_id>:<user_id>`, because the API has no single id for that pairing — `ImportState` parses the colon form.
- **Drift on roles/permissions**: every `Read` hits the API and replaces state with the response, so out-of-band role changes are detected on refresh.
- **Polymorphic `allowed_inference_geos`**: the Admin API returns either the literal string `"unrestricted"` or an array of geo strings. The client (`internal/anthropic/workspaces.go`) normalizes both onto `[]string`, with `["unrestricted"]` denoting unrestricted; the Terraform schema sees only a list.
- **API-key Create**: Anthropic forbids creating keys via the Admin API. `anthropic_api_key` therefore implements `Create` as "fetch by supplied id + apply Update". If the id doesn't exist, Create errors with a clear message. Same pattern applies to `anthropic_organization_member` — users only arrive via accepted invites.
- **Delete-by-archive**: `anthropic_workspace.Delete` calls `POST .../archive` and `anthropic_api_key.Delete` sets status=`archived` — neither is hard-deletion. Document this in resource docstrings rather than papering over it.

## Open questions / TODOs

- **Acceptance tests need to actually run** — scaffolded for workspace (`TestAccWorkspace_basic`, `TestAccWorkspace_withTags`) but never executed. Trigger via the `acceptance` GitHub workflow (manual dispatch) once an admin API key is in repo secrets, or run locally with `make testacc`. Once workspace passes, extend the pattern to every resource.
- **Workspace unarchive** — the Admin API may support unarchive (not documented in the pages this provider was authored from); if confirmed, `Delete` semantics may want to re-activate rather than force destroy.
- **MCP Tunnels** — only Admin API surface left out (beta, Bearer/WIF auth + `anthropic-beta` header). Requires a second auth code path in the client. See "Coverage vs Admin API docs" in Status section.

## Adding a new Admin API endpoint

1. Add the endpoint to the appropriate file under `internal/anthropic/` (or create a new one if it's a new resource group). Match the response shape from the live docs at https://platform.claude.com/docs/en/api/admin — don't guess.
2. Add a resource file (`internal/provider/resource_<name>.go`) and/or a data source file (`internal/provider/datasource_<name>.go`). Follow the pattern in the existing files: `Configure` uses `clientFromProviderData`; `Read` removes from state on 404 via `anthropic.IsNotFound(err)`; `ImportState` uses `ImportStatePassthroughID` unless the ID is composite.
3. Register the constructor in `internal/provider/provider.go` (`Resources` / `DataSources`).
4. Add an example under `examples/resources/<name>/resource.tf` or `examples/data-sources/<name>/data-source.tf` — `tfplugindocs` uses these verbatim.
5. Run `go build ./... && go vet ./... && gofmt -s -w .` before committing.
