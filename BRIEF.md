# ServiceNow CLI — Project Brief

> Starter context for a publishable, agent-native CLI for managing ServiceNow instances.
> Written 2026-06-15. This is the foundation; refine as the design proves out.

## Vision

A **general-purpose, publishable** CLI that any ServiceNow admin can pick up to **understand and
manage their instance and workflows**. An agentic layer sits *on top* of this CLI — the CLI is the
clean, deterministic substrate; the agent orchestrates it. If the tool can let an admin understand
and manage the instance, it can support end-user workflows easily on the same foundation.

Audience is anyone running a ServiceNow instance — not one company's setup. **Generality and
portability across instances/releases are hard requirements**, not nice-to-haves.

## Foundational architecture decisions

These were reasoned through before any code. Change them deliberately, not by drift.

1. **Build on the documented Now Platform REST API — NOT internal UI APIs.**
   - The `/api/now/ui/...` family is internal, undocumented, and varies by release. Fine for a
     personal script; a bad foundation for a tool published for arbitrary instances.
   - The documented REST API (`/api/now/table/...` etc.) is present on every instance and stable
     across releases. That stability is what makes the tool publishable.
   - UI APIs are at most a rare, clearly-labeled escape hatch — never the basis.

2. **The Table API is the spine; design around a generic, introspective core.**
   - `/api/now/table/{table}` reads/writes *any* table — OOB, custom, all of it. You wrap **one
     generic table interface**, not N hand-written endpoints.
   - The instance is **self-describing through the same API**. "Understanding the instance" falls
     out of querying metadata tables:
     - `sys_db_object` — tables
     - `sys_dictionary` — fields/columns
     - `sys_hub_flow` — Flow Designer flows (workflows)
     - `sys_script` — business rules
     - `sys_user_role`, `sys_security_acl` (ACLs)
     - `sys_update_set` — update sets
     - scheduled jobs, etc.
   - Workflows and configuration are just records. A generic, metadata-aware table client already
     covers most admin understanding/management without hardcoding anything instance-specific.

3. **Auth: OAuth 2.0.**
   - Register an OAuth endpoint in the instance: **System OAuth → Application Registry → New →
     "Create an OAuth API endpoint for external clients"** → client ID + secret.
   - Token endpoint: `https://<instance>.service-now.com/oauth_token.do`.
   - Expect to implement token fetch/refresh ourselves (generators usually scaffold generic
     bearer-header auth, not a full OAuth flow). Small, normal work.

4. **Read-first v1.** (Confirm scope.)
   - Read/inspect surface (query flows, business rules, ACLs, dependencies) is mostly the generic
     Table API + introspection — safe, agent-friendly, the natural first release.
   - Mutate surface (create/update flows, push update sets, edit scripts) carries the real risk and
     per-feature design — a deliberate later phase.

## Tooling

- **`cli-printing-press`** (https://github.com/mvanhorn/cli-printing-press, MIT) — a Go,
  agent-native CLI *generator*. Use it for **scaffolding**: the agent-native shape, local SQLite
  persistence, offline search, compound insight commands. All genuinely wanted.
- **Caveat:** it's a *spec-driven, per-endpoint* generator. ServiceNow's surface is better modeled
  as one generic table interface + introspection, which cuts against the per-endpoint grain. So:
  bootstrap structure with the generator, then build the generic core deliberately. Don't force a
  giant per-endpoint spec.
- Install (global tool): `curl -fsSL https://raw.githubusercontent.com/mvanhorn/cli-printing-press/main/scripts/install.sh | bash`
  (requires Go 1.26.4+; `brew install go`). Run `/printing-press <name>` inside a Claude Code
  session rooted in *this* directory.

## Stack

- **Language:** Go (1.26.4+). Matches the generator and produces a single distributable binary.
- **Module path:** set to where it'll be published, e.g. `go mod init github.com/<you>/servicenow-cli`.
- **License:** add MIT from day one.
- **Local store:** SQLite (from the generator's scaffolding) for cache/offline search.

## First experiment — prove the foundation before building outward

The single highest-value early test: **one authenticated Table API round-trip.**

1. OAuth: client creds → token from `/oauth_token.do`.
2. `GET /api/now/table/sys_db_object` → list the instance's tables.

If that works against a real instance, the entire "understand the instance generically" thesis is
proven, and everything else builds outward from it. Do this before investing in scaffolding.

## Open questions to resolve

- Exact published name (avoid ServiceNow trademark confusion — e.g. `snow-cli`, `now-cli`, or a
  distinct brand).
- v1 scope line: read-only, or include a guarded mutate surface?
- Where the agent layer lives — inside this repo or a separate consumer of the CLI.

## Pointers

- ServiceNow REST API Explorer (per instance): `https://<instance>.service-now.com/$restapi.do`
  — lists every REST API the instance exposes; source for endpoint shapes.
- Community OpenAPI specs for the Table API exist — verify against the target release before trusting.
