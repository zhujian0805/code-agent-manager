# Makefile App Workflow Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `make` the documented entrypoint for starting the Tauri desktop app, browser frontend, and Go sidecar.

**Architecture:** Keep existing Go CLI targets while adding Tauri/sidecar targets. `make app` starts the Tauri desktop shell, `make frontend` starts browser-only Vite, and `make sidecar` starts the Go sidecar for manual API testing.

**Tech Stack:** Make, Go, npm/Vite, Cargo/Tauri, README markdown.

---

## Files

- Modify: `Makefile` — add app/dev/sidecar/frontend/desktop-build targets and update help.
- Modify: `README.md` — document Make-based app startup and Tauri + Go sidecar architecture.

## Task 1: Update Makefile

- [ ] Add variables for frontend host/port, sidecar host/port, and cargo revocation workaround.
- [ ] Add `app` and `dev` aliases that run `cargo tauri dev --manifest-path src-tauri/Cargo.toml` with `CARGO_HTTP_CHECK_REVOKE=false`.
- [ ] Add `frontend` target for browser-only Vite.
- [ ] Add `sidecar` target for local sidecar API startup.
- [ ] Add `desktop-build` target that runs frontend build, builds `cmd/cam-sidecar`, and runs `cargo check` for Tauri.
- [ ] Keep existing `build`, `test`, `vet`, and `check` targets.

## Task 2: Update README

- [ ] Add a Desktop App section after Quick Start.
- [ ] Document `make app`, `make frontend`, `make sidecar`, and `make desktop-build`.
- [ ] Explain that the desktop app is Tauri + Go sidecar and browser mode remains available.

## Task 3: Verify

- [ ] Run `make help`.
- [ ] Run `make desktop-build`.
- [ ] Run `go test ./...`.
- [ ] Run `npm --prefix frontend test -- --run`.
- [ ] Run `git status --short`.

## Self-Review

- No placeholders.
- Targets match current Tauri sidecar architecture.
- README commands match Makefile target names.
