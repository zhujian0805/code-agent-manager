.PHONY: help install clean test test-race build build-cli build-sidecar node-deps stop-desktop frontend app dev start sidecar desktop-build fmt fmt-check vet check release

SHELL := pwsh.exe
.SHELLFLAGS := -NoProfile -Command

VERSION ?= dev
GOFLAGS ?=
FRONTEND_HOST ?= 127.0.0.1
FRONTEND_PORT ?= 5173
SIDECAR_HOST ?= 127.0.0.1
SIDECAR_PORT ?= 0
TAURI_MANIFEST ?= src-tauri/Cargo.toml
TAURI_CONFIG ?= src-tauri/tauri.conf.json
TAURI_CLI ?= ./frontend/node_modules/.bin/tauri.cmd
SIDECAR_TARGET ?= src-tauri/binaries/cam-sidecar-x86_64-pc-windows-msvc.exe

help:
	@Write-Output "Available commands:"
	@Write-Output "  make start         - Start the full Tauri desktop app (same as make app)"
	@Write-Output "  make app           - Start the Tauri desktop app (alias: make dev)"
	@Write-Output "  make frontend      - Start browser-only Vite frontend at $(FRONTEND_HOST):$(FRONTEND_PORT)"
	@Write-Output "  make sidecar       - Start Go sidecar API at $(SIDECAR_HOST):$(SIDECAR_PORT)"
	@Write-Output "  make desktop-build - Build frontend, Go sidecar, and cargo-check Tauri shell"
	@Write-Output "  make build         - Build Go CLI binaries and sidecar into dist/"
	@Write-Output "  make install       - Build and install cam/code-agent-manager"
	@Write-Output "  make test          - Run Go test suite"
	@Write-Output "  make test-race     - Run Go tests with race detector"
	@Write-Output "  make fmt           - Format Go code"
	@Write-Output "  make vet           - Run go vet"
	@Write-Output "  make check         - Run fmt check, vet, tests, frontend tests, and sidecar build"
	@Write-Output "  make clean         - Remove build artifacts"

install:
	bash ./install.sh install

clean:
	Remove-Item -Recurse -Force dist, frontend/dist, src-tauri/target, src-tauri/binaries -ErrorAction SilentlyContinue
	Get-ChildItem -Recurse -Filter *.test -ErrorAction SilentlyContinue | Remove-Item -Force -ErrorAction SilentlyContinue
	Get-ChildItem -Recurse -Filter coverage.out -ErrorAction SilentlyContinue | Remove-Item -Force -ErrorAction SilentlyContinue

build: build-cli build-sidecar

node-deps:
	npm --prefix frontend install

build-cli:
	New-Item -ItemType Directory -Force dist | Out-Null
	go build $(GOFLAGS) -ldflags "-X main.version=$(VERSION)" -o dist/cam ./cmd/cam
	go build $(GOFLAGS) -ldflags "-X main.version=$(VERSION)" -o dist/code-agent-manager ./cmd/code-agent-manager

stop-desktop:
	$$processes = Get-Process cam-sidecar, cam-desktop -ErrorAction SilentlyContinue; if ($$processes) { $$processes | Stop-Process -Force -ErrorAction SilentlyContinue }; exit 0

build-sidecar: stop-desktop
	New-Item -ItemType Directory -Force dist, src-tauri/binaries | Out-Null
	go build $(GOFLAGS) -ldflags "-X main.version=$(VERSION)" -o dist/cam-sidecar ./cmd/cam-sidecar
	go build $(GOFLAGS) -ldflags "-X main.version=$(VERSION)" -o $(SIDECAR_TARGET) ./cmd/cam-sidecar

frontend:
	npm --prefix frontend run dev -- --host $(FRONTEND_HOST) --port $(FRONTEND_PORT) --strictPort

sidecar:
	go run ./cmd/cam-sidecar --host $(SIDECAR_HOST) --port $(SIDECAR_PORT)

app dev start: node-deps build-sidecar
	$$env:CARGO_HTTP_CHECK_REVOKE='false'; $(TAURI_CLI) dev --config $(TAURI_CONFIG)

desktop-build: build-sidecar
	npm --prefix frontend run build
	$$env:CARGO_HTTP_CHECK_REVOKE='false'; cargo check --manifest-path $(TAURI_MANIFEST)

test:
	go test $(GOFLAGS) ./...

test-race:
	go test $(GOFLAGS) -race ./...

fmt:
	gofmt -s -w cmd internal

fmt-check:
	$$files = gofmt -s -l cmd internal; if ($$files) { $$files; exit 1 }

vet:
	go vet ./...

check: fmt-check vet test build-sidecar
	npm --prefix frontend test -- --run
	npm --prefix frontend run build
	$$env:CARGO_HTTP_CHECK_REVOKE='false'; cargo check --manifest-path $(TAURI_MANIFEST)
	Write-Output "All checks passed!"

release: clean check build
	Write-Output "Release build completed successfully!"
	Get-ChildItem dist
