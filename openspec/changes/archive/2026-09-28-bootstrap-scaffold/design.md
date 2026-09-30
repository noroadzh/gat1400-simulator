# Design: Bootstrap Scaffold

## Technology Selection

| Category | Choice | Rationale |
|---|---|---|
| Language | Go 1.21+ | Slog standard library, range-over-func, generics |
| SQLite | modernc.org/sqlite | Pure Go, no CGO, cross-platform builds |
| HTTP framework | labstack/echo/v4 | Mature, well-documented, groups middleware |
| Config | viper | Environment variables, YAML, flag binding |
| Logging | log/slog | Standard library, structured, JSON/text handlers |
| Linting | golangci-lint | Unified CLI for 20+ linters |

## Directory Layout

```
gat1400-simulator/
├── cmd/                  # Application entry points
│   └── simulator/        # Main binary
├── internal/             # Private packages (hexagonal layers)
│   ├── app/              # Application services (orchestration)
│   ├── adapter/          # Infrastructure adapters (HTTP, storage, wire)
│   ├── domain/           # Domain models (pure business logic)
│   └── ui/               # Web BFF (control plane)
├── configs/              # YAML config files (node definitions, scenarios)
├── data/                 # Runtime data (SQLite DB, captured PCAP)
├── web/                  # Embedded frontend assets
├── test/                 # Contract tests, e2e, golden samples
├── openspec/             # OpenSpec change management
├── docs/                 # Protocol references, architecture docs
├── Makefile
├── go.mod / go.sum
├── .golangci.yml
└── .github/workflows/
```

## Build Targets

- `make build` → `bin/gat1400-simulator` (current OS/arch)
- `make test` → `go test -race ./...`
- `make lint` → `golangci-lint run`
- `make run` → `go run ./cmd/simulator/`
- `make clean` → rm bin/

## CI Pipeline

1. golangci-lint
2. go test -race ./...
3. goreleaser cross-build (linux/amd64, darwin/arm64)

## Security Constraints

- No hard-coded credentials in source
- Credentials loaded from config file or environment
- SQLite file permissions restricted to user-only (mode 0600)
