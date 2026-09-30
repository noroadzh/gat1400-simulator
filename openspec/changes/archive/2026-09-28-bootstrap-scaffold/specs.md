## ADDED Requirements

### Requirement: The project MUST compile without errors

When `go build ./cmd/...` is executed on a clean checkout with Go 1.21+, the build MUST complete with exit code 0 and produce at least one binary in `bin/`.

### Requirement: The test suite MUST pass

When `go test ./...` is executed, all tests MUST pass with exit code 0. The `-race` flag MUST be used in CI to detect data races.

### Requirement: golangci-lint MUST report no errors

When `golangci-lint run ./...` is executed, there MUST be zero linter errors reported.

### Requirement: All private packages MUST reside under internal/

No package outside of `cmd/` or `test/` MAY import `internal/` packages. This constraint MUST be enforced by CI (golangci-lint `exported` rule checks).

### Requirement: SQLite dependency MUST be pure-Go

The project MUST use `modernc.org/sqlite` as its SQLite driver. Any import of `database/sql` with `github.com/mattn/go-sqlite3` MUST cause a build failure.

### Requirement: Makefile MUST provide standard targets

The Makefile MUST define targets `build`, `test`, `lint`, `run`, and `clean`. Each target MUST execute the corresponding standard Go or toolchain command.

### Requirement: GitHub Actions CI MUST run on every push

A workflow file at `.github/workflows/ci.yml` MUST trigger on `push` and `pull_request` to branches `main` and `release/**`. The workflow MUST run lint, test, and build steps in that order.

### Requirement: Go module version MUST be declared as 1.21+

The `go.mod` file MUST contain `go 1.21` or higher to ensure access to slog, range-over-func, and generics.

---

## ADDED Architecture Decisions

### Decision: Hexagonal Architecture

The codebase MUST be organized into three layers:
- **domain/**: Pure domain models, no external dependencies.
- **app/**: Application services, depends only on domain and ports (interfaces).
- **adapter/**: Infrastructure adapters (HTTP server, HTTP client, SQLite storage), depends on domain, app, and external libraries.

This separation MUST be verified by ensuring `internal/app/` contains no imports from `internal/adapter/`.

### Decision: Pure-Go SQLite

SQLite operations are implemented via `modernc.org/sqlite`, which compiles to pure Go WebAssembly and requires no CGO. This allows macOS arm64 (Apple Silicon) and Linux amd64 binaries to be built without cross-compilation toolchains.

### Decision: Standard Library Logging

All packages MUST use `log/slog` (not third-party loggers) for structured logging. Handlers (JSON for production, text for development) are configured at application startup.
