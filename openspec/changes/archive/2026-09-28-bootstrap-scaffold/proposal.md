# Proposal: Bootstrap Scaffold

## Status
Archived — implemented as the project foundation.

## Motivation

We need a minimal, production-grade Go project skeleton that:
- Compiles out of the box with `go build`
- Has deterministic, reproducible CI
- Establishes directory conventions for a hexagonal/ports-and-adapters architecture
- Supports multi-platform builds (Linux amd64, macOS arm64)

## Goals

- `go.mod` with pinned dependencies
- Standard layout: `cmd/`, `internal/`, `configs/`, `data/`, `web/`, `test/`
- `Makefile` with `build`, `test`, `lint`, `run`, `clean`
- GitHub Actions CI: lint → test (race) → build (multi-platform)
- `.golangci.yml` with essential linters (vet, staticcheck, gofmt, goimports, revive)
- Pure-Go SQLite via `modernc.org/sqlite` (no CGO requirement)

## Non-Goals

- No actual GAT 1400 protocol implementation (deferred to domain-models)
- No database migrations or ORM (deferred to adapter-storage)
- No Docker/Kubernetes manifests (deferred to operations docs)

## Open Questions

None — all resolved in design.md.
