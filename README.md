# snxplore

[![CI](https://github.com/icymoonray-ui/snxplore/actions/workflows/ci.yml/badge.svg)](https://github.com/icymoonray-ui/snxplore/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

> A read-first, agent-native CLI for understanding any ServiceNow instance — built only on the documented Now Platform Table API.

`snxplore` (sn + *explore*) lets a ServiceNow admin point at an arbitrary instance and ask **"what is this table, how does it work, and who can touch it?"** — schema, forms-era metadata, business logic, automation, and access control — without needing to be a platform expert. It emits clean JSON for agents and readable tables for humans.

It is deliberately the **inverse of ServiceNow's official tooling**: where the Now CLI / SDK / IDE are for *building* scoped applications, `snxplore` is for *understanding and operating* an existing instance. It wraps **one generic, introspective Table API client** rather than N hand-written endpoints, so it works against arbitrary tables (OOB or custom) and stays stable across releases.

**Status:** read-first **proof-of-concept**. The read/inspect surface below is complete and tested; mutation (create/update) is a deliberate later phase. Names and module path are not yet final (see [Project status](#project-status)).

---

## Install

The binary is a single, statically-linked, **CGO-free** executable (pure-Go SQLite), so it cross-compiles cleanly.

Prebuilt binaries for Linux/macOS/Windows (amd64 + arm64) are attached to each [release](https://github.com/icymoonray-ui/snxplore/releases). Or, with **Go 1.26+**:

```bash
# install the latest tagged version
go install github.com/icymoonray-ui/snxplore@latest

# …or build from source
git clone https://github.com/icymoonray-ui/snxplore && cd snxplore
go build -o snxplore .

# cross-compile, e.g. for Linux:
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o snxplore .
```

## Authentication

`snxplore` authenticates per request via a pluggable method. **Basic auth is the default** (simplest; works on any instance without SSO). OAuth is built in and one flag away.

Configure via environment variables (a config file with named profiles is also supported):

```bash
export SNXPLORE_INSTANCE="dev12345"          # short id or full https URL
export SNXPLORE_USERNAME="admin"
export SNXPLORE_PASSWORD="••••••••"
# Basic is the default. For OAuth instead:
#   export SNXPLORE_AUTH="client_credentials"   # or "password"
#   export SNXPLORE_CLIENT_ID=...  SNXPLORE_CLIENT_SECRET=...
```

| Method | `SNXPLORE_AUTH` | Needs |
|---|---|---|
| HTTP Basic *(default)* | `basic` | username + password |
| OAuth client credentials | `client_credentials` | client id + secret (Application Registry app) |
| OAuth password (ROPC) | `password` | client id + secret + username + password |

> Secrets are read from the environment in this POC; OS-keyring storage is planned.

## Usage

```bash
# Prove the connection — list the instance's tables
snxplore table list

# ⭐ Explain a table: schema + logic + access + automation, in one report
snxplore table incident

# Schema with inherited fields resolved (walks the super_class chain)
snxplore schema incident

# Raw generic read of any table (one page; warns on stderr if results are truncated)
snxplore query incident -q "active=true^priority=1" --fields number,short_description --limit 20

# Page through every matching record (--all overrides --limit; tune with --page-size)
snxplore query incident -q "active=true" --fields number,short_description --all

# What logic runs on a table (business rules + client scripts, in order)
snxplore logic incident

# Who can read/write/create/delete a table (ACLs → roles)
snxplore access incident

# Automation bound to a table (legacy workflows; Flow Designer noted)
snxplore flows incident

# Offline full-text search over everything you've explored
snxplore search "assignment group"
```

### Output for agents

Every command takes `--output json` for machine consumption. Data goes to **stdout**, errors go to **stderr** as a structured envelope, and exit codes are stable:

```bash
snxplore table incident --output json | jq '.schema.fields[].element'
```

```
0 ok · 1 error · 2 usage · 3 auth · 4 not-found · 5 api-error
```

Global flags: `--output json|table`, `--profile <name>`, `--timeout <dur>` (default `30s`, `0` to disable), `--verbose`.

## What it reads

`snxplore` treats the self-describing platform as the source of truth, querying metadata tables:

| Area | Tables |
|---|---|
| Schema | `sys_db_object` (extension via `super_class`), `sys_dictionary` |
| Logic | `sys_script` (business rules), `sys_script_client` (client scripts) |
| Access | `sys_security_acl`, `sys_security_acl_role`, `sys_user_role`, group/role grants |
| Automation | `wf_workflow` (legacy), `sys_hub_flow` (Flow Designer) |

**Inheritance is resolved by the tool**, not the API: `sys_dictionary` returns only a table's own columns, so `snxplore` walks the `super_class` chain and merges (each field shows its `origin`).

## Known limitations (POC)

- **ACLs may require `security_admin`.** Reading `sys_security_acl*` over REST can need the `security_admin` role granted *persistently* to the account (REST has no per-session elevation). When unavailable, `access` degrades with a note instead of failing.
- **Flow Designer → table binding** is not yet reliably resolved via the Table API (the trigger link is polymorphic and changed shape in Flow Engine V2 / Washington DC+). Legacy workflows are listed; flows are noted pending live verification.
- Secrets live in environment variables (keyring planned).

## Architecture

```
cmd/                 cobra commands (thin; delegate down)
internal/
  auth/              pluggable auth: Basic + OAuth (client_credentials, password)
  snclient/          the generic Table API client — the spine
  introspect/        schema (super_class walk), logic, access, flows, report
  store/             SQLite (pure-Go) + FTS5 offline cache/search
  output/            json|table renderer, stable exit codes
  config/            named instance profiles (koanf)
```

## Project status

This is an early POC. The published name (`snxplore`) and Go module path (`github.com/<owner>/snxplore`) are **not final**. Auth is read-only by design for v1. An agentic layer is intended to sit on top of this CLI later; the CLI stays a clean, deterministic substrate.

## License

[MIT](LICENSE).
