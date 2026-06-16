# Tauri Go Sidecar Migration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the Wails native runtime with a Tauri desktop shell that launches a Go sidecar serving the shared app API over localhost.

**Architecture:** Keep `frontend/` as the React/Vite UI. Add `src-tauri/` as a thin Rust/Tauri shell that manages the desktop window and sidecar lifecycle. Add `cmd/cam-sidecar` and `internal/sidecar` as the Go HTTP sidecar over `internal/appapi`; CLI and sidecar share `appapi`, while frontend talks to sidecar HTTP in Tauri/browser modes.

**Tech Stack:** Go, net/http, React/Vite/TypeScript, Rust, Tauri v2 config scaffold, Vitest, Go tests.

---

## Files

### Create
- `cmd/cam-sidecar/main.go` — Go sidecar executable entrypoint.
- `internal/sidecar/server.go` — HTTP router/server over `internal/appapi` and desktop-compatible services.
- `internal/sidecar/server_test.go` — sidecar API tests.
- `src-tauri/Cargo.toml` — Tauri Rust crate manifest.
- `src-tauri/src/main.rs` — thin Tauri shell entrypoint.
- `src-tauri/tauri.conf.json` — Tauri app config.
- `src-tauri/capabilities/default.json` — Tauri permission config.

### Modify
- `frontend/src/services/api.ts` — replace Wails binding transport with sidecar HTTP transport plus mock fallback.
- `frontend/src/services/api.test.ts` — add sidecar HTTP transport tests.
- `frontend/package.json` — add Tauri scripts and dependencies if needed.
- `install.sh` — build/install sidecar and prefer Tauri build when available.
- `.gitignore` — ignore Tauri generated outputs.
- `go.mod` / `go.sum` — remove Wails dependency if no Wails code remains.
- `cmd/cam-desktop/*` / `wails.json` — retire Wails runtime files after Tauri sidecar is present.

---

## Task 1: Finish shared provider appapi baseline

- [ ] Run `go test ./internal/appapi ./internal/desktop ./internal/cli -run 'TestProvider|TestProviderService' -v`.
- [ ] Fix compile or behavior issues before starting Tauri migration.

## Task 2: Add Go sidecar HTTP server

- [ ] Create `internal/sidecar/server.go` with a `Server` type that accepts `Version`, `ProvidersPath`, and `Token`.
- [ ] Implement auth middleware requiring `Authorization: Bearer <token>` when token is non-empty.
- [ ] Add routes:
  - `GET /api/app/version`
  - `GET /api/providers`
  - `GET /api/providers/{name}`
  - `POST /api/providers`
  - `PATCH /api/providers/{name}`
  - `DELETE /api/providers/{name}`
  - `POST /api/providers/{name}/enable`
  - `POST /api/providers/{name}/disable`
  - `POST /api/providers/{name}/rename`
  - `GET /api/providers/{name}/models`
  - read-only/list routes for tools, MCP clients/servers, entities, config files, doctor checks, and launch dry-run so all current UI pages can call sidecar rather than Wails.
- [ ] Create `internal/sidecar/server_test.go` using `httptest` for auth, provider CRUD, and app version.
- [ ] Run `go test ./internal/sidecar -v`.

## Task 3: Add sidecar executable

- [ ] Create `cmd/cam-sidecar/main.go`.
- [ ] Add flags: `--host`, `--port`, `--token`, `--providers`, `--version-json`.
- [ ] If `--port 0`, bind random localhost port and print one JSON startup line to stdout: `{"port":12345,"token":"..."}`.
- [ ] Run `go test ./cmd/cam-sidecar ./internal/sidecar`.
- [ ] Run `go run ./cmd/cam-sidecar --version-json` and verify JSON output.

## Task 4: Replace frontend Wails adapter with sidecar transport

- [ ] Update `frontend/src/services/api.ts` so it detects API base URL from:
  1. `window.__CAM_SIDECAR__ = { baseUrl, token }`
  2. `import.meta.env.VITE_CAM_API_BASE_URL`
  3. mock fallback.
- [ ] Implement HTTP helpers for providers, tools, MCP, entities, config, doctor, and launch dry-run.
- [ ] Keep mock fallback when no sidecar base URL exists.
- [ ] Update `frontend/src/services/api.test.ts` to test mock fallback and sidecar calls through mocked `fetch`.
- [ ] Run `npm --prefix frontend test -- --run src/services/api.test.ts`.

## Task 5: Add Tauri shell scaffold

- [ ] Create `src-tauri/Cargo.toml` with Tauri v2 dependencies.
- [ ] Create `src-tauri/src/main.rs` with a thin Tauri app that opens the frontend window.
- [ ] Create `src-tauri/tauri.conf.json` pointing `beforeDevCommand` to `npm --prefix frontend run dev -- --host 127.0.0.1`, `devUrl` to `http://127.0.0.1:5173`, and `frontendDist` to `../cmd/cam-desktop/webui/dist` or `../frontend/dist` depending on final Vite output.
- [ ] Create `src-tauri/capabilities/default.json` with minimum shell/window permissions.
- [ ] Run `cargo check --manifest-path src-tauri/Cargo.toml` if Cargo/Tauri dependencies are available; otherwise run JSON/schema sanity checks and report environment limitation.

## Task 6: Retire Wails runtime and update build scripts

- [ ] Remove Wails imports and `wails.json` runtime usage.
- [ ] Replace `cmd/cam-desktop` with either a compatibility launcher message or remove it from desktop build path.
- [ ] Remove Wails dependency from `go.mod` if no Go package imports it.
- [ ] Update `install.sh --desktop` to build `cmd/cam-sidecar` and run `cargo tauri build` when available.
- [ ] Keep CLI install behavior unchanged.
- [ ] Run `go mod tidy`.

## Task 7: Full verification

- [ ] Run `go test ./...`.
- [ ] Run `go vet ./...`.
- [ ] Run `npm --prefix frontend test -- --run`.
- [ ] Run `npm --prefix frontend run build`.
- [ ] Run `go build ./cmd/cam-sidecar`.
- [ ] Run `go run ./cmd/cam-sidecar --version-json`.
- [ ] Run `bash -n install.sh`.
- [ ] Run Tauri build/check commands if `cargo` and Tauri dependencies are available.

## Self-Review

- Scope: covers Tauri scaffold, Go sidecar, frontend transport, Wails retirement, install support, and tests.
- Placeholder scan: no TBD/TODO placeholders.
- Type consistency: sidecar JSON responses map to existing frontend `types.ts` and appapi provider types.
