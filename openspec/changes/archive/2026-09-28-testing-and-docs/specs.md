## ADDED Requirements

### Requirement: Golden samples MUST cover all four route families

The golden sample set MUST include at least one interaction for each of:
- System routes (Register, Keepalive, Time)
- Collection routes (at least Person POST + GET)
- Cascade routes (at least Subscribe POST + GET)
- Catalog routes (at least APE GET)

#### Scenario: Missing APE catalog golden sample
WHEN the test suite runs with `--golden` flag
THEN every `test/contract/golden/*.json` file with a catalog step MUST pass
AND a missing catalog sample MUST NOT cause the test to be skipped silently.

### Requirement: E2E tests MUST use real TCP sockets

E2E tests MUST NOT mock the HTTP layer. The server MUST be bound to a real port via `net.Listen` or `httptest.Server`. The client MUST connect over actual TCP.

#### Scenario: Real socket round-trip
WHEN `TestProtocolE2E_RegisterAndPush` runs
THEN the client MUST send a TCP SYN to the server
AND the response MUST come back over the same TCP connection
AND no mock or fake HTTP transport MUST be used.

### Requirement: E2E tests MUST verify end-to-end correctness

Each E2E test MUST assert at least one observable effect beyond HTTP status:
- Server's in-memory state (ResourceStore, NodeService) reflects the client's request
- CaptureStore contains a record for each request
- NodeService reports the node as online after Register

### Requirement: Golden sample runner MUST report mismatches as failures

When a golden sample is loaded and replayed, any discrepancy in response status code or key body fields MUST cause `t.Fatal`.

#### Scenario: Status code mismatch
WHEN the golden sample expects status 200 but the server returns 401
THEN the test MUST fail with a message describing the mismatch.

### Requirement: All internal packages MUST have ≥90% test coverage

Each package under `internal/` MUST achieve at least 90% line coverage as reported by `go test -cover`. Coverage reports MUST be published in CI.

### Requirement: Documentation MUST be kept in sync with code

When a change introduces a new route, new config key, or new YAML field, the corresponding documentation MUST be updated in the same change.

---

## ADDED Architecture Decisions

### Decision: Golden samples are version-controlled JSON

Golden samples are plain JSON files committed to the repo. This makes them easy to review, diff, and update when protocol semantics change.

### Decision: E2E tests live in test/e2e/

E2E tests are in a top-level `test/` directory, not inside `internal/`, because they exercise multiple layers and should not be gated by the `internal/` package boundary rules.

### Decision: Documentation lives in docs/ not in the source tree

Docs are generated from OpenSpec specs and stored in `docs/`, which is served as part of the project README. Implementation details remain in source comments.