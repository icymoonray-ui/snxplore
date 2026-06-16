# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project status

Active build. The read-first **v1 surface is implemented and tested** (a Go CLI named `snxplore`). [BRIEF.md](BRIEF.md) holds the original design intent; the full research + implementation plan lives at `~/.claude/plans/golden-gathering-corbato.md` (4 deep-research rounds). Read both before structural changes.

What exists: `cmd/` (cobra commands) + `internal/{auth,snclient,introspect,store,output,config}`. Mutation is **not** built (deliberate later phase). The name `snxplore` and module path `github.com/icymoonray/snxplore` are placeholders pending final confirmation.

## Common commands

```bash
go build -o snxplore .              # build (CGO-free; pure-Go SQLite)
CGO_ENABLED=0 go build ./...        # verify the static-binary path
go test ./...                       # all tests (fixtures/httptest — no instance needed)
go test ./internal/introspect/ -run TestResolveSchemaInheritance   # single test
go vet ./...
```

There is **no live ServiceNow instance during development** — tests use `httptest` mocks. Live verification happens against the user's instance (see the live-verify items below).

## Architecture

The Table API is the spine: **one generic introspective client** (`internal/snclient`), and everything else is built on top by querying self-describing metadata tables.

- `internal/snclient` — `Client.Get(table, GetOptions{Query,Fields,Limit,Offset,DisplayValue})` over `/api/now/table/{table}`. `Record.Str()` extracts values (handles `{value,link}` reference objects).
- `internal/introspect` — the metadata layer: `ResolveSchema` (**recursive `super_class` walk + merge** — the Table API does NOT resolve inherited fields), `ResolveLogic`, `ResolveAccess` (ACL→role graph, **degrades gracefully** if ACLs need `security_admin`), `ResolveFlows`, and `ResolveTable` (the compound report).
- `internal/auth` — pluggable `HTTPClient(ctx, baseURL, Credentials)`: `basic` (default, POC), `client_credentials`, `password`. OAuth uses `x/oauth2` against `/oauth_token.do`.
- `internal/store` — pure-Go SQLite (`zombiezen.com/go/sqlite`) + FTS5 offline cache/search.
- `internal/output` — `Renderer.Emit()` (json or table via the `Tabular` interface), stable exit codes, errors→stderr. JSON is the agent surface.
- `cmd/` — thin cobra commands; `clientForProfile` builds the client from profile/env.

Stack: Go 1.26+, cobra, koanf, `zombiezen.com/go/sqlite`, `golang.org/x/oauth2`. MIT.

## Foundational decisions (change on purpose, not by drift)

1. **Documented REST API only** (`/api/now/table/...`); never internal `/api/now/ui/...`.
2. **Generic introspective Table client**, not N hand-written endpoints. The instance is self-describing — query metadata tables (`sys_db_object`, `sys_dictionary`, `sys_script`, `sys_security_acl`, `sys_hub_flow`, `wf_workflow`, …); never hardcode instance-specific config.
3. **Auth is pluggable.** POC defaults to **HTTP Basic** (the dev instance has no SSO). OAuth 2.0 (both grants, `/oauth_token.do`) is built and retained; SSO is a future requirement.
4. **Read-first v1.** Inspect/query only. Mutation carries real risk — a deliberate later phase; confirm scope before building it.
5. **Machine-readable output is first-class** (an agent layer will consume the CLI).

## Live-verify items (resolve against a real instance)

- Does reading `sys_security_acl*` over REST require **persistent `security_admin`** (vs plain `admin`)? Gates `access`.
- **Flow Designer → table binding**: not yet reliably resolved via the Table API (polymorphic trigger link; differs in Flow Engine V2 / Washington DC+). `flows` lists legacy workflows and notes this.
- Confirm community-sourced field names by introspecting `sys_dictionary` at runtime (the tool self-validates).

## Open questions

- Final published name + GitHub owner → real module path (currently placeholders).
- v1 scope line stays read-only; when to start the guarded mutate surface.
- Where the agent layer lives (this repo vs a separate consumer). Note: `cli-printing-press` (the scaffolder) emits an MCP surface — a possible starting point.

## Useful per-instance references

- REST API Explorer: `https://<instance>.service-now.com/$restapi.do`.
- OAuth setup (only if using OAuth, not Basic): System OAuth → Application Registry → New → "Create an OAuth API endpoint for external clients".
