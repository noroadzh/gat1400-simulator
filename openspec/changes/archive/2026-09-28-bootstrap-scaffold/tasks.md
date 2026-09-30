## Tasks

### Task: Initialize Go module

- [x] Run `go mod init github.com/noroadzh/gat1400-simulator`
- [x] Set `go 1.21` in `go.mod`
- [x] Verify `go build ./cmd/...` succeeds with zero source files

### Task: Create directory structure

- [x] Create `cmd/simulator/`, `internal/`, `configs/`, `data/`, `web/`, `test/`, `openspec/`, `docs/`
- [x] Add `.gitkeep` files for empty directories tracked by Git

### Task: Create main entry point

- [x] Write `cmd/simulator/main.go` with a minimal `main()` that logs "hello world"
- [x] Verify `go run ./cmd/simulator/` prints the greeting and exits 0

### Task: Add Makefile

- [x] Define `build`, `test`, `lint`, `run`, `clean` targets
- [x] Define `BINARY_NAME`, `GO_LDFLAGS` variables
- [x] Verify `make build` produces `bin/gat1400-simulator`
- [x] Verify `make test` runs `go test ./...` with `-race`

### Task: Add golangci-lint configuration

- [x] Create `.golangci.yml` with linters: `vet`, `staticcheck`, `gofmt`, `goimports`, `revive`
- [x] Configure `issues.exclude-rules` for test files
- [x] Verify `golangci-lint run ./...` reports zero errors on the minimal source

### Task: Add GitHub Actions CI workflow

- [x] Create `.github/workflows/ci.yml`
- [x] Define jobs: lint → test (with race) → build (linux/amd64, darwin/arm64)
- [x] Verify workflow syntax with `act` or push to a test branch

### Task: Add .gitignore

- [x] Ignore `bin/`, `*.db`, `*.pcap`, `.vscode/`, `.idea/`, `vendor/`
- [x] Include `go.sum` (do NOT ignore)

### Verification

All tasks are complete when:
- `go build ./cmd/...` → exit 0, binary in `bin/`
- `go test ./...` → exit 0
- `golangci-lint run ./...` → 0 errors
- `make build test lint` → all green
