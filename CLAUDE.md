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

### Resources (6)

| Resource | Notes |
|---|---|
| `anthropic_workspace` | Create / Read / Update / Delete-archives. `tags` mutable, `external_key_id` write-once, `data_residency` currently read-only. |
| `anthropic_api_key` | Update-only — Admin API does not create keys. `Create` requires an existing `id`, applies name/status, `Delete` sets status=`archived`. |
| `anthropic_workspace_member` | Composite id `<workspace_id>:<user_id>`. Role mutable; workspace_id/user_id RequireReplace. |
| `anthropic_invite` | Create + Delete only — invites are immutable; any change to email/role forces replacement. `admin` role not allowed. |
| `anthropic_organization_member` | Update-only — users join via accepted invite. `Create` requires existing user id, sets role. `Delete` removes the user from the org. |
| `anthropic_external_key` | Full CRUD + validate for CMEK configurations. Polymorphic `provider_config` (aws/gcp/azure) modeled as a flat block with `type` discriminator. Delete fails if any workspace still references it. |

### Data sources (17)

Identity / membership: `anthropic_organization`, `anthropic_workspace`, `anthropic_workspaces`, `anthropic_workspace_member`, `anthropic_workspace_members`, `anthropic_organization_member`, `anthropic_organization_members`, `anthropic_invite`, `anthropic_invites`.

Keys / CMEK: `anthropic_api_key`, `anthropic_api_keys`, `anthropic_external_key`, `anthropic_external_keys`.

Operational: `anthropic_workspace_rate_limits` (overrides only; absence = inherit, not no-limit), `anthropic_usage_report` (messages), `anthropic_claude_code_usage_report`, `anthropic_cost_report`.

The three reports + external_keys are the **headline differentiation** vs `terraform-mars/terraform-provider-anthropic`, which covers only workspaces, api_keys, workspace_members, and invites.

### Coverage vs Admin API docs

We cover every endpoint group documented at https://platform.claude.com/docs/en/api/admin **except MCP Tunnels** — a beta surface that uses a different auth model (Bearer/WIF instead of `x-api-key`) and requires the `anthropic-beta: mcp-tunnels-2026-05-19` header. Adding it would require a second auth code path in `internal/anthropic.Client`. Deliberately out of scope for v0.1; track as a future addition if customers need it. Service Accounts and Audit Logs are NOT in the Admin API (confirmed via sitemap — the breadcrumb mention was misleading).

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

- **Authentication**: provider accepts `anthropic_admin_api_key` (env fallback `ANTHROPIC_ADMIN_API_KEY`) and an optional `base_url` (defaults to `https://api.anthropic.com`).
- **Client**: a single `internal/anthropic.Client` is constructed in `provider.Configure` and passed via `resp.ResourceData` / `resp.DataSourceData`. Every resource/data source unwraps it through `clientFromProviderData` in `internal/provider/common.go` — do not instantiate HTTP clients inside resource methods.
- **Headers**: every Admin API request sets `x-api-key`, `anthropic-version: 2023-06-01`, `content-type: application/json`, and a `user-agent` carrying the provider version. The version header is required even for Admin endpoints.
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
