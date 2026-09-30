## ADDED Requirements

### Requirement: BFF MUST expose control REST API under /api/control/

All control-plane endpoints MUST be under the `/api/control/` prefix. The protocol server (`/VIID/...`) MUST be served from a different port.

#### Scenario: Different ports
WHEN the simulator is started with default config
THEN `:19000` MUST serve the BFF (control plane)
AND `:19001` MUST serve the protocol server (VIID).

### Requirement: BFF MUST serve the embedded Vue3 SPA at /

The BFF MUST serve the embedded `web/dist/` directory at the root path. Any non-API path MUST fall back to `index.html` (SPA routing).

#### Scenario: Unknown frontend path
WHEN a client GETs `/nodes`
THEN the response MUST be the embedded `index.html`
AND the Content-Type MUST be `text/html`.

### Requirement: WebSocket MUST broadcast events to connected clients

When an event is published to the hub, all connected WebSocket clients MUST receive it within 100ms.

#### Scenario: Node status change
WHEN a node's status changes from `online` to `stopped`
THEN all connected WebSocket clients MUST receive an event of type `node.status` with the updated node's `id` and `status`.

### Requirement: BFF MUST return JSON responses

All REST endpoints MUST return `Content-Type: application/json`. Responses MUST be wrapped in `{items: [...]}` for lists or `{...object...}` for single objects.

### Requirement: BFF MUST validate node creation payloads

When a POST to `/api/control/nodes` is missing required fields (`id`, `name`, `role`), the response MUST be `400 Bad Request` with a descriptive error message.

### Requirement: BFF MUST persist captures and serve them on demand

The BFF MUST persist all captures to the `CaptureStore` and expose them via `/api/control/captures`. The query MUST support `limit`, `nodeId`, `method`, `path` filters.

### Requirement: BFF MUST support JSONL and HAR exports

Endpoints `/api/control/captures/export/jsonl` and `/api/control/captures/export/har` MUST return a `200 OK` body with `{path, count}`. The path MUST be a writable file on the local filesystem.

---

## ADDED Architecture Decisions

### Decision: BFF does not own business logic

The BFF delegates to `application.NodeService` and `application.ScenarioService`. It MUST NOT directly call `adapter/storage` or `adapter/httpapi`. This keeps the BFF thin.

### Decision: WebSocket events are best-effort

Slow clients are disconnected after a 30-second read deadline. Events dropped due to a slow client are NOT replayed. This prevents hub backpressure from blocking the entire system.

### Decision: Frontend bundle is embedded at compile time

The frontend build artifact (`web/dist/`) is embedded via `embed.FS` and served directly from memory. No filesystem access is needed at runtime.