# SQLite App State Store Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Move provider state from `providers.json` persistence to a SQLite database and establish SQLite as the future home for CLI/UI app state.

**Architecture:** Add `internal/appstate` as the SQLite storage layer. `internal/appapi.ProviderAPI` becomes the provider use-case layer over `appstate`, while `internal/providers` remains the provider model/JSON compatibility package for import and legacy config parsing.

**Tech Stack:** Go `database/sql`, pure-Go `modernc.org/sqlite`, existing provider/appapi DTOs, Go tests.

---

## Files

- Create: `internal/appstate/store.go` — SQLite store, schema migrations, provider CRUD, app_state key/value table.
- Create: `internal/appstate/store_test.go` — SQLite provider CRUD and JSON import tests.
- Modify: `internal/appapi/providers.go` — use `appstate.Store` instead of writing providers JSON.
- Modify: `internal/appapi/providers_test.go` — assert DB-backed behavior and JSON import compatibility.
- Modify: `go.mod`, `go.sum` — add `modernc.org/sqlite`.

## Tasks

1. Add pure-Go SQLite dependency.
2. Implement `internal/appstate.Store` with schema creation.
3. Implement provider import from `providers.json` when DB has no providers.
4. Implement provider CRUD/list/rename/enable over SQLite.
5. Switch `internal/appapi.ProviderAPI` to use `appstate`.
6. Run focused tests, then full Go/frontend/Tauri verification.

## Verification

- `go test ./internal/appstate ./internal/appapi -v`
- `go test ./...`
- `npm --prefix frontend test -- --run`
- `npm --prefix frontend run build`
- `go vet ./...`
- `make desktop-build`
