# Design: GAT 1400 HTTP API Adapter

## Server Layout

```
internal/adapter/httpapi/
├── server.go         # Server struct, route registration, lifecycle
├── system.go         # /VIID/System/* handlers
├── collection.go     # /VIID/Persons, Faces, Vehicles, etc.
├── cascade.go        # /VIID/Subscribes, Notifications, Dispositions
├── catalog.go        # /VIID/APEs, APSs, Tollgates, Lanes
├── middleware.go     # DigestAuth, UserIdentify, Capture
├── binder.go         # Custom binder for VIID+JSON and JSON
├── config.go         # AuthConfig, route paths
└── response.go       # Helpers: OKResponse(), NotFound(), etc.
```

## Route Table

| Method | Path | Handler | Auth |
|---|---|---|---|
| POST | `/VIID/System/Register` | System/Register | Digest |
| POST | `/VIID/System/UnRegister` | System/UnRegister | Digest |
| POST | `/VIID/System/Keepalive` | System/Keepalive | User-Identify |
| GET  | `/VIID/System/Time` | System/Time | none |
| POST/GET/PUT/DELETE | `/VIID/Persons` `/VIID/Persons/:id` | Collection | User-Identify |
| POST/GET/PUT/DELETE | `/VIID/...` for Face, Vehicle, Plate, Image, etc. | Collection | User-Identify |
| POST/GET/PUT/DELETE | `/VIID/Subscribes` `/VIID/Subscribes/:id` | Cascade | User-Identify |
| POST/GET | `/VIID/SubscribeNotifications` | Cascade | User-Identify |
| POST/GET/PUT/DELETE | `/VIID/Dispositions` `/VIID/Dispositions/:id` | Cascade | User-Identify |
| GET | `/VIID/APEs` `/VIID/APSs` `/VIID/Tollgates` `/VIID/Lanes` | Catalog | none |

## Middleware Chain

Order matters. The chain applied to all routes is:

1. CaptureMiddleware — log before and after, before any other logic
2. UserIdentifyMiddleware — extracts `User-Identify` header, calls `MarkSeen`
3. DigestAuthMiddleware — only on protected System routes (Register/UnRegister)
4. (route handler)
5. Response serialization

The capture middleware uses an in-memory `bytes.Buffer` to record both request and response bodies (only the body is buffered, headers are copied before consumption).

## Digest Auth Details

```
Server:  Digest realm="<r>", nonce="<n>", qop="auth", opaque=""
Client:  Authorization: Digest username="u", realm="r", nonce="n", uri="...",
         qop=auth, nc=00000001, cnonce="<c>", response="<h>", algorithm=MD5
```

The expected response is:
```
MD5(MD5(u:n:r) + ":" + n + ":" + nc + ":" + c + ":" + qop + ":" + MD5(method + ":" + uri))
```

The server stores nonce → timestamp in SQLite (`storage.NewNonceStore`). Nonces older than 30s are rejected. Duplicate nonces with the same nc are detected and rejected (replay protection).

## Binder

Echo's default binder handles `application/json` but ignores `application/VIID+JSON`. We register a custom binder that delegates to the default binder for both content types. No transformation is needed — VIID+JSON is just JSON with a different MIME marker.

## Capture Middleware Details

Records one Capture per request:

```go
type Capture struct {
    NodeID    string
    Direction string // "inbound" or "outbound"
    Method    string
    Path      string
    URL       string
    Remote    string
    Status    int
    Header    http.Header
    Request   io.Reader
    Response  io.Reader
}
```

The `Recorder` persists to `ports.CaptureStore` (SQLite). Querying is exposed via the BFF (`internal/ui`) but not directly through the protocol server.

## Response Envelope

All protocol responses include a top-level `ResponseStatus` object:

```json
{
  "ResponseStatus": {
    "StatusCode": 0,
    "StatusString": "OK"
  },
  ...domain-specific keys...
}
```

The Server uses helper functions:

```go
func OK(c echo.Context, body any) error
func Invalid(c echo.Context, msg string) error  // 400
func NotFound(c echo.Context, id string) error  // 404
func Unauthorized(c echo.Context, realm string) error // 401 + WWW-Authenticate
func ServerError(c echo.Context, err error) error   // 500
```