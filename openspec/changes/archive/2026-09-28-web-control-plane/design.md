# Design: Web Control Plane

## BFF Layout

```
internal/ui/
├── server.go          # Server struct, BFF lifecycle, route registration
├── api_control.go     # All REST handlers under /api/control/
├── hub.go             # WebSocket hub, broadcaster, client registry
├── api_bindings.go     # JSON ↔ domain model bindings for BFF requests
└── frontend.go        # embed.FS handler for Vue3 SPA
```

## REST Routes

| Endpoint | Method | Description |
|---|---|---|
| `/api/control/system/health` | GET | `{status, service, version}` |
| `/api/control/system/info` | GET | `{revision, goVersion, uptime}` |
| `/api/control/stats` | GET | Aggregate counts: nodes, scenarios, resources |
| `/api/control/nodes` | GET | List all nodes |
| `/api/control/nodes` | POST | Create node |
| `/api/control/nodes/:id` | GET | Get node |
| `/api/control/nodes/:id` | DELETE | Remove node |
| `/api/control/scenarios` | GET | List scenarios |
| `/api/control/scenarios/:id` | GET | Get scenario |
| `/api/control/scenarios/:id/start` | POST | Start scenario |
| `/api/control/scenarios/:id/stop` | POST | Stop scenario |
| `/api/control/resources/:kind` | GET | Browse resource collection |
| `/api/control/subscriptions` | GET/POST/PUT/DELETE | Manage subscriptions |
| `/api/control/captures?limit=N` | GET | Recent captures |
| `/api/control/captures/export/jsonl` | GET | Download JSONL |
| `/api/control/captures/export/har` | GET | Download HAR |

## WebSocket Hub

A single endpoint `/api/control/ws` accepts WebSocket upgrades. The server maintains a list of connected clients and broadcasts events:

```go
type Event struct {
    Type    string         `json:"type"`     // node.status, capture.received, scenario.state, alert
    Payload map[string]any `json:"payload"`
}
```

Events are emitted by:
- `nodeSvc.SetSyncHook` — node status changes
- `httpapi.CaptureMiddleware` — capture stored
- `scenSvc` — scenario start/stop
- Cascade notification handler — alert/disposition match

The hub uses a mutex-protected slice; broadcasts are non-blocking (channel per client).

## Frontend Embedding

```go
//go:embed all:dist
var distFS embed.FS
```

The frontend is built with `pnpm build` to `web/dist/`, which is embedded at compile time. The BFF serves `dist/` at `/` with SPA fallback (any unknown path returns `index.html`).

## Frontend Pages

1. **Dashboard** — Health card, node counts, scenario counts, recent activity feed (live)
3. **Scenarios** — Cards listing scenarios, status badges, start/stop buttons
4. **Nodes** — Table of nodes with status, capabilities, actions (delete, view)
5. **Resources** — Tabbed list of resource collections, with type/kind filter
6. **Subscriptions** — List and create subscriptions, edit criteria
7. **Captures** — Paginated table of recent captures with method/path/status filter; "view" opens a side panel showing request/response bodies

The frontend uses ElementPlus components (Table, Tag, Dialog, Form, Tabs).

## Lifecycle

```
Server.Start(addr string) error
  ├── Starts echo in goroutine
  ├── Starts hub goroutine
  └── Returns immediately

Server.Shutdown(ctx context.Context) error
  ├── Shutdown echo (drains HTTP requests)
  ├── Closes hub (drains WS clients)
  └── Returns
```

The BFF listens on `:19000` by default. The protocol server listens on `:19001`.