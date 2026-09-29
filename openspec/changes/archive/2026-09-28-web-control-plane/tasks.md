## Tasks

### Task: BFF Server skeleton

- [x] `internal/ui/server.go` — `Server` struct, `NewServer`, `Start`, `Shutdown`
- [x] Configure echo with default middleware (Logger, Recover, CORS)

### Task: System endpoints

- [x] `GET /api/control/system/health` — returns service info
- [x] `GET /api/control/system/info` — returns revision, Go version, uptime
- [x] `GET /api/control/stats` — aggregate counts

### Task: Nodes endpoints

- [x] `GET/POST/DELETE /api/control/nodes`
- [x] `GET /api/control/nodes/:id`
- [x] Bindings: `bindNode` for JSON → domain.Node conversion

### Task: Scenarios endpoints

- [x] `GET /api/control/scenarios`
- [x] `GET /api/control/scenarios/:id`
- [x] `POST /api/control/scenarios/:id/start|stop`

### Task: Resources endpoints

- [x] `GET /api/control/resources/:kind` — read from `httpapi.ResourceStore`

### Task: Subscriptions endpoints

- [x] `GET/POST/PUT/DELETE /api/control/subscriptions`

### Task: Captures endpoints

- [x] `GET /api/control/captures?limit=N`
- [x] `GET /api/control/captures/export/jsonl`
- [x] `GET /api/control/captures/export/har`

### Task: WebSocket Hub

- [x] `internal/ui/hub.go` — Hub with broadcaster, Register/Unregister clients
- [x] `internal/ui/api_control.go::handleWS` — upgrade connection, register with hub

### Task: Frontend embedding

- [x] `frontend.go` — `embed.FS` for `web/dist/`
- [x] SPA fallback handler

### Task: Frontend build

- [x] `web/package.json` — Vue 3 + ElementPlus + Vite
- [x] `web/vite.config.ts` — output to `web/dist/`
- [x] 6 page components: Dashboard, Nodes, Scenarios, Resources, Subscriptions, Captures

### Task: Unit tests

- [x] `internal/ui/server_test.go` — all BFF endpoints
- [x] Cover health, info, stats, CRUD round-trip, export endpoints
- [x] Cover hub broadcast mechanism

### Verification

- `pnpm build` in `web/` produces `dist/`
- `go test ./internal/ui/...` → exit 0
- Browser can load `http://localhost:19000/` and see the dashboard