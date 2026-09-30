# Design: Testing Matrix and Documentation

## Golden Samples (test/contract/golden/)

```
test/contract/golden/
├── register.json       # Digest handshake: 401 → Digest response → 200
├── persons_post.json   # POST /VIID/Persons → 200 with ItemCount
├── subscribes.json     # Subscribe CRUD → 200
└── catalog.json        # GET /VIID/APEs → 200 with non-empty APEObject
```

Each JSON file is a self-contained HTTP interaction:

```json
{
  "description": "System/Register with Digest auth",
  "steps": [
    {
      "direction": "outbound",
      "method": "POST",
      "path": "/VIID/System/Register",
      "headers": {"Content-Type": "application/VIID+JSON", "Authorization": "Digest ..."},
      "body": {"RegisterObject": {"DeviceID": "41000000005030312222"}}
    },
    {
      "direction": "inbound",
      "status": 401,
      "headers": {"WWW-Authenticate": "Digest realm=\"viid\"..."}
    }
  ]
}
```

The golden test runner:
1. Loads `*.json` files
2. Starts a real protocol server in-process
3. Replays the steps
4. Compares response status codes and body structure

Discrepancies are reported as failures.

## E2E Tests (test/e2e/)

```
test/e2e/
├── protocol_e2e_test.go   # Dual-process: start binary + client
└── capture_e2e_test.go    # Capture middleware end-to-end
```

```go
// protocol_e2e_test.go
func TestProtocolE2E_RegisterAndPush(t *testing.T) {
    // 1. Start a fresh protocol server (httptest.Server or real TCP listener)
    // 2. Create a wire.Client targeting that server
    // 3. Client.Register() → expect 200
    // 4. Client.PostJSON("/VIID/Persons", person) → expect 200
    // 5. Query server's ResourceStore → expect 1 person
}
```

E2E tests use `net.Listen` (real TCP) to test the full HTTP stack including middleware, header parsing, and body reading.

## Documentation

All documentation lives in `docs/`:

| File | Content |
|---|---|
| `ARCHITECTURE.md` | System overview, component diagram, data flow |
| `PROTOCOL.md` | GAT 1400.4 protocol reference: routes, request/response shapes, error codes |
| `USER_GUIDE.md` | Quick start, config file format, YAML scenario syntax |
| `OPERATIONS.md` | Deployment, TLS, monitoring, logging, upgrade |
| `TESTING.md` | How to run tests, golden samples, e2e, CI |
| `CHANGELOG.md` | Per-version release notes (mirrors openspec/CHANGELOG.md) |

The `docs/CHANGELOG.md` is auto-generated from `openspec/CHANGELOG.md` by a `make sync-changelog` target.