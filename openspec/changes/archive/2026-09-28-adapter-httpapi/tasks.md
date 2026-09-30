## Tasks

### Task: Define Server struct

- [x] Create `internal/adapter/httpapi/server.go`
- [x] Define `Server` with `e *echo.Echo`, deps (NodeService, ScenarioService, Recorder, NonceStore, IDGenerator)
- [x] Implement `NewServer(log, nodeSvc, scenSvc, rec, nonce, idGen, cfg)`
- [x] Implement `Start(addr) error` and `Shutdown(ctx) error`

### Task: Implement System routes

- [x] `POST /VIID/System/Register` — Digest + body `RegisterObject.DeviceID`, calls `nodeSvc.MarkSeen`
- [x] `POST /VIID/System/UnRegister` — Digest, marks node as stopped
- [x] `POST /VIID/System/Keepalive` — User-Identify, calls `MarkSeen`
- [x] `GET /VIID/System/Time` — returns RFC3339 timestamp

### Task: Implement Collection routes

- [x] POST/GET/PUT/DELETE for each Kind: Person, Face, Vehicle, Plate, NonMotorVehicle, Image, Object
- [x] Resource store with kind-bucketed maps
- [x] Subroutes `/Info`, `/Data` for metadata blob

### Task: Implement Cascade routes

- [x] `/VIID/Subscribes` CRUD
- [x] `/VIID/SubscribeNotifications` POST + GET with `subscribeId` query
- [x] `/VIID/Dispositions` CRUD

### Task: Implement Catalog routes

- [x] `/VIID/APEs` GET → canned list
- [x] `/VIID/APSs`, `/VIID/Tollgates`, `/VIID/Lanes` GET → canned lists

### Task: Implement Digest middleware

- [x] `middleware.go::DigestAuth(realm, username, password, qop, nonceStore)`
- [x] Parse `Authorization` header
- [x] Compute expected response, compare with constant-time
- [x] Issue 401 + WWW-Authenticate when missing/invalid

### Task: Implement UserIdentify middleware

- [x] `middleware.go::UserIdentify(nodeSvc)`
- [x] Extract `User-Identify` header
- [x] Call `MarkSeen`, return 404 if unknown

### Task: Implement Capture middleware

- [x] `middleware.go::CaptureMiddleware(recorder, defaultNodeID)`
- [x] Buffer request body, then `Next()`, then record capture
- [x] Fall back to header-based NodeID if no default

### Task: Custom Binder

- [x] `binder.go::NewVIBinder` that handles `application/VIID+JSON` and `application/json`

### Task: Response helpers

- [x] `response.go::OK`, `Invalid`, `NotFound`, `Unauthorized`, `ServerError`

### Task: Integration tests

- [x] `internal/adapter/httpapi/server_test.go`
- [x] Cover all four route families
- [x] Verify Digest challenge on missing auth
- [x] Verify CRUD round-trip for collection/cascade
- [x] Verify catalog returns at least one entry per family

### Verification

- `go test ./internal/adapter/httpapi/...` → exit 0
- All routes return `application/VIID+JSON` Content-Type
- Digest middleware rejects requests with reused nonces