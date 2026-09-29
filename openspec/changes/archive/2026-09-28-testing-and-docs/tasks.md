## Tasks

### Task: Golden sample files

- [ ] `test/contract/golden/register.json` — Digest handshake 401 → 200
- [ ] `test/contract/golden/persons_post.json` — POST + GET + DELETE
- [ ] `test/contract/golden/subscribes.json` — Subscribe CRUD
- [ ] `test/contract/golden/catalog.json` — APE/APS/Tollgate/Lane GET
- [ ] `test/contract/golden/keepalive.json` — User-Identify keepalive

### Task: Golden sample runner

- [ ] `test/contract/golden_test.go` — load `*.json`, replay, compare
- [ ] Flag: `--golden` to enable (default: skip if no golden files)
- [ ] Report diff on mismatch

### Task: E2E test: protocol_e2e_test.go

- [ ] `net.Listen` a real TCP port
- [ ] Start `NewServer` on that port
- [ ] Create `wire.NewClient` targeting that port
- [ ] Register → 200, PostJSON → 200, verify ResourceStore
- [ ] `defer listener.Close()` cleanup

### Task: E2E test: capture_e2e_test.go

- [ ] Start protocol server with CaptureMiddleware
- [ ] Send a POST with body
- [ ] Query CaptureStore → verify entry with request body

### Task: Makefile targets

- [ ] `make test-golden` — run golden sample tests
- [ ] `make test-e2e` — run e2e tests
- [ ] `make test-all` — unit + golden + e2e

### Task: Documentation files

- [ ] `docs/ARCHITECTURE.md` — system overview, component diagram (ASCII)
- [ ] `docs/PROTOCOL.md` — route table, request/response shapes, error codes
- [ ] `docs/USER_GUIDE.md` — quick start, config YAML, scenario YAML syntax
- [ ] `docs/OPERATIONS.md` — deployment, TLS, monitoring, logging
- [ ] `docs/TESTING.md` — running tests, golden samples, e2e, CI
- [ ] `docs/CHANGELOG.md` — release notes (mirror of openspec/CHANGELOG.md)

### Task: OpenSpec delta specs for all 7 changes

- [x] bootstrap-scaffold
- [x] domain-models
- [x] adapter-httpapi
- [x] adapter-wire
- [x] scenario-engine
- [x] web-control-plane
- [x] testing-and-docs (proposal/design/specs written; tasks and docs pending)

### Task: Main specs sync

- [ ] `openspec/specs/domain.md` — merged domain-model requirements
- [ ] `openspec/specs/adapter-httpapi.md` — merged protocol requirements
- [ ] `openspec/specs/adapter-wire.md` — merged client requirements
- [ ] `openspec/specs/scenario.md` — merged engine requirements
- [ ] `openspec/specs/web-bff.md` — merged BFF requirements
- [ ] `openspec/specs/testing.md` — merged testing requirements

### Verification

- `go test ./test/...` → exit 0
- `go test -cover ./internal/...` → all packages ≥ 90%
- All `docs/*.md` files present and non-empty
- `openspec/specs/*.md` files present and non-empty