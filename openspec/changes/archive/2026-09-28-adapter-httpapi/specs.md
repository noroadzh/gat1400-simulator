## ADDED Requirements

### Requirement: All protocol routes MUST respond with application/VIID+JSON

Every response from the protocol server MUST carry `Content-Type: application/VIID+JSON`. The Content-Type MUST be set before the body is written to the wire.

#### Scenario: Persons POST
WHEN a client POSTs to `/VIID/Persons`
THEN the response Content-Type MUST equal `application/VIID+JSON; charset=UTF-8`.

### Requirement: Digest authentication MUST be enforced on System Register/UnRegister

Routes `/VIID/System/Register` and `/VIID/System/UnRegister` MUST require HTTP Digest auth. A request without a valid `Authorization: Digest ...` header MUST receive `401 Unauthorized` with `WWW-Authenticate: Digest realm="..."`.

#### Scenario: First POST without Authorization
WHEN a client POSTs to `/VIID/System/Register` without `Authorization`
THEN the response MUST be `401`
AND the `WWW-Authenticate` header MUST contain `Digest realm="<configured realm>"`.

#### Scenario: Valid Digest request
WHEN a client POSTs to `/VIID/System/Register` with a correct Digest response computed per RFC 2617
THEN the response MUST be `200` with `ResponseStatus.StatusCode=0`.

### Requirement: User-Identify header MUST register a heartbeat

When any request carries `User-Identify: <node-id>`, the server MUST call `NodeService.MarkSeen(ctx, node-id)` and update the node's `LastSeenAt` and `Status=online`.

#### Scenario: Known node keepalive
WHEN a known node sends POST `/VIID/System/Keepalive` with `User-Identify: DEV-1`
THEN the node's status MUST be `online` and `LastSeenAt` MUST be within the past 5 seconds.

#### Scenario: Unknown node User-Identify
WHEN a request carries `User-Identify: unknown`
THEN the server MUST return `404 Not Found` (response body MUST contain `StatusCode=2`).

### Requirement: Resource collection CRUD MUST be supported

Each Kind (`Person`, `Face`, `Vehicle`, etc.) MUST support POST (batch insert), GET (list or single), DELETE.

#### Scenario: POST /VIID/Persons with 2 persons
WHEN a client POSTs `{"PersonList": {"PersonObject": [...]}}` with 2 items
THEN the response MUST be `200` with `ItemCount=2`.

#### Scenario: GET /VIID/Persons after insertion
WHEN a client GETs `/VIID/Persons`
THEN the response MUST include both inserted persons in `PersonList.PersonObject`.

#### Scenario: DELETE removes item
WHEN a client DELETEs `/VIID/Persons/p1`
THEN a subsequent GET for `/VIID/Persons/p1` MUST return `404`.

### Requirement: Cascade routes MUST be supported

`/VIID/Subscribes`, `/VIID/SubscribeNotifications`, `/VIID/Dispositions` MUST be exposed with full CRUD.

### Requirement: Catalog routes MUST be static

`/VIID/APEs`, `/VIID/APSs`, `/VIID/Tollgates`, `/VIID/Lanes` MUST return canned catalog data seeded from `internal/adapter/httpapi/catalog.go`. The data MUST contain at least one entry per collection.

### Requirement: All responses MUST include ResponseStatus

Every successful or error response MUST contain a top-level `ResponseStatus` field with `StatusCode` (integer 0–4) and `StatusString` ("OK" / "INVALID" / "NOTFOUND" / "UNAUTHORIZED" / "SERVER_ERROR").

### Requirement: Capture middleware MUST record every request

Every inbound request MUST be captured with NodeID, Method, Path, URL, Remote, Status, Header, RequestBody, ResponseBody. The capture MUST be persisted asynchronously to the CaptureStore (non-blocking).

#### Scenario: PostBody preserved
WHEN a POST contains a 1KB JSON body and the response is 200
THEN the CaptureStore MUST contain an entry with the original body and the response body.

### Requirement: Nonce replay MUST be rejected

When a Digest request uses a nonce that was already consumed with the same `nc`, the server MUST return `401 Unauthorized`.

---

## ADDED Architecture Decisions

### Decision: Use labstack/echo for HTTP routing

Echo provides grouped routes, middleware chain, and custom binder support that map directly onto the four route families.

### Decision: NonceStore persists to SQLite

Nonces are stored in SQLite rather than in-process map. This allows horizontal scaling (multiple server processes can verify the same nonce) and survives restarts.

### Decision: Digest middleware runs before route handler

Digest must be checked before any other logic to prevent denial-of-service via expensive handlers when auth fails.