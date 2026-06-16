# Wails Native Desktop Runtime Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Turn the existing browser-only React desktop UI into a native Wails desktop app, add tests for the native runtime wiring, and verify backend/frontend tests and builds pass.

**Architecture:** Keep the existing CLI and `internal/desktop` services intact. Add a Wails v2 native entrypoint in `cmd/cam-desktop` that binds the existing service structs, embeds the Vite production bundle when available, and falls back to a small embedded diagnostic page when the bundle has not been built. Configure Vite to emit the desktop bundle into the Go entrypoint's embedded asset tree, and configure `wails.json`/`install.sh` to prefer Wails native builds when the Wails CLI is installed while preserving the existing lightweight Go fallback.

**Tech Stack:** Go, Wails v2, React, TypeScript, Vite, Vitest, Go `testing`, PowerShell validation commands.

---

## Files

### Create
- `cmd/cam-desktop/assets.go` — embeds desktop web assets and selects built UI or fallback UI.
- `cmd/cam-desktop/assets_test.go` — verifies the fallback asset filesystem works without a frontend build.
- `cmd/cam-desktop/webui/fallback/index.html` — diagnostic fallback page embedded in the native binary.
- `wails.json` — Wails project configuration for native dev/build commands.

### Modify
- `cmd/cam-desktop/main.go` — run a native Wails app by default, keep `--services` smoke mode.
- `cmd/cam-desktop/main_test.go` — add service-list tests if needed.
- `frontend/vite.config.ts` — emit production assets into `cmd/cam-desktop/webui/dist` for embedding.
- `.gitignore` — ignore generated Wails/Vite asset output, keep fallback assets trackable.
- `go.mod` / `go.sum` — add Wails v2 dependency.
- `install.sh` — prefer `wails build` when available, then `wails3 build`, then Go fallback.

---

## Task 1: Add Wails dependency and native app entrypoint

**Files:**
- Modify: `go.mod`
- Modify: `go.sum`
- Modify: `cmd/cam-desktop/main.go`

- [ ] Add Wails v2 with `go get github.com/wailsapp/wails/v2@latest`.
- [ ] Replace the print-only default path in `cmd/cam-desktop/main.go` with a `runDesktopApp()` function using `app.Run(&options.App{...})`.
- [ ] Bind the existing services: `services.App`, `services.Providers`, `services.MCP`, `services.Entities`, `services.Tools`, `services.Doctor`, `services.Config`, and `services.Launch`.
- [ ] Keep `--services` as a non-GUI smoke command that prints the service names as JSON.
- [ ] Keep `--version` as a non-GUI smoke command that prints the desktop version.
- [ ] Verify compilation with `go test ./cmd/cam-desktop`.

## Task 2: Embed frontend assets with fallback

**Files:**
- Create: `cmd/cam-desktop/assets.go`
- Create: `cmd/cam-desktop/assets_test.go`
- Create: `cmd/cam-desktop/webui/fallback/index.html`
- Modify: `.gitignore`

- [ ] Add `//go:embed all:webui` to embed everything under `cmd/cam-desktop/webui`.
- [ ] Implement `desktopAssets()` so it returns `webui/dist` when `index.html` exists, otherwise `webui/fallback`.
- [ ] Add a fallback page that clearly says the native shell is running and instructs developers to run `npm --prefix frontend run build` for the full React UI.
- [ ] Ignore generated `cmd/cam-desktop/webui/dist/` and Wails build output.
- [ ] Test that `desktopAssets()` can open `index.html` without a frontend build.
- [ ] Verify with `go test ./cmd/cam-desktop -run TestDesktopAssets`.

## Task 3: Connect Vite build output to Wails embed tree

**Files:**
- Modify: `frontend/vite.config.ts`

- [ ] Add `build.outDir = '../cmd/cam-desktop/webui/dist'`.
- [ ] Add `build.emptyOutDir = true`.
- [ ] Keep the existing Vitest settings unchanged.
- [ ] Verify with `npm --prefix frontend run build`.
- [ ] Verify the generated file exists at `cmd/cam-desktop/webui/dist/index.html`.

## Task 4: Add Wails project config and installer support

**Files:**
- Create: `wails.json`
- Modify: `install.sh`

- [ ] Add Wails v2 config with project name `cam-desktop`, output filename `cam-desktop`, frontend build command `npm --prefix frontend run build`, and dev server URL `http://127.0.0.1:5173`.
- [ ] Update `install.sh` desktop build logic to run `wails build` when available and `wails3 build` as a fallback.
- [ ] Preserve the existing `go build ./cmd/cam-desktop` fallback when no Wails CLI exists.
- [ ] Verify Go-side behavior with `go test ./cmd/cam-desktop ./internal/desktop`.

## Task 5: Frontend binding compatibility tests

**Files:**
- Modify: `frontend/src/services/api.test.ts`
- Modify: `frontend/src/services/api.ts` only if tests reveal binding-name incompatibility.

- [ ] Add tests proving the adapter uses Wails-style service bindings when `window.go.desktop` is present.
- [ ] Keep mock fallback behavior for browser-only dev mode.
- [ ] If Wails exposes `AppService`/`ProviderService` names, support both the existing short aliases and service-struct names.
- [ ] Verify with `npm --prefix frontend test -- --run src/services/api.test.ts`.

## Task 6: Full verification

**Files:** all changed files.

- [ ] Run `go test ./cmd/cam-desktop ./internal/desktop`.
- [ ] Run `go test ./...`.
- [ ] Run `npm --prefix frontend test -- --run`.
- [ ] Run `npm --prefix frontend run build`.
- [ ] Run `go build ./cmd/cam-desktop` after the frontend build so embedded React assets are compiled into the native app.
- [ ] Run `go run ./cmd/cam-desktop --services` and verify the JSON service list.
- [ ] If Wails CLI is installed, run `wails build`; if it is not installed, record that native packaging could not be executed in this environment while the Wails app source compiles.

## Self-Review

- Spec coverage: native Wails runtime, frontend embedding, Wails config, installer support, binding compatibility, backend/frontend verification are covered.
- Placeholder scan: no TBD/TODO placeholders remain.
- Type consistency: service names match the existing `internal/desktop.Services` fields and frontend adapter methods.
