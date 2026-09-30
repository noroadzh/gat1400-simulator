## Tasks

### Task: Define Node, Role, Capability, Status

- [x] Create `internal/domain/node/node.go`
- [x] Define `Node` struct with all fields
- [x] Define `Role`, `Capability`, `Status` enums
- [x] Implement `Sanity()` returning sentinel errors

### Task: Define Resource, Kind, Metadata

- [x] Create `internal/domain/resource/resource.go`
- [x] Define `Resource` struct
- [x] Define `Kind` enum (7 values)
- [x] Implement `Sanity()`

### Task: Define Subscription, Disposition

- [x] Add to `internal/domain/resource/` (or new package)
- [x] Implement `Sanity()` on both

### Task: Define Scenario, ScheduleSpec, NodeSpec, etc.

- [x] Create `internal/domain/scenario/scenario.go`
- [x] Define struct tags for both `yaml` (scenario loader) and `json` (HTTP)

### Task: Define ResponseStatus, Code

- [x] Create `internal/domain/response/response.go`
- [x] Define `Code` enum (5 values) and `ResponseStatus` struct
- [x] Add helper functions `OK()`, `Invalid()`, etc.

### Task: Implement ID Generator

- [x] Create `internal/domain/ids/ids.go`
- [x] Implement `Generator` with mutex, `NewGenerator(siteCode, industryCode uint32)`
- [x] Implement `DeviceID()`, `UUID()`, `Nonce()`, `SubscribeID()`
- [x] Use `crypto/rand` for Nonce and UUID; monotonic counter for DeviceID sequence

### Task: Unit tests

- [x] `internal/domain/node/node_test.go` — Sanity() coverage, HasCapability()
- [x] `internal/domain/resource/resource_test.go` — Kind exhaustiveness
- [x] `internal/domain/scenario/scenario_test.go` — YAML/JSON round-trip
- [x] `internal/domain/response/response_test.go` — Code mapping
- [x] `internal/domain/ids/ids_test.go` — DeviceID layout, concurrency, Nonce uniqueness

### Verification

- `go test ./internal/domain/...` → exit 0
- Coverage ≥ 90% for domain packages