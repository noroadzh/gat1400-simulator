# Proposal: spa-root-routing

## Why

The current BFF mounts the Vue SPA at `/ui/*` with a 302 redirect from `/`, which is functionally a 2-hop UX (`/` → `/ui/` → app). It also lacks a true SPA fallback — `/ui/dashboard` (a non-existent file) returns a `404 page not found` from `http.FileServer` rather than falling back to `index.html`. With Change 3 set to migrate to Docker / nginx and the SPA destined for the root path, this change moves the SPA to `/` and implements a proper fallback in the BFF.

## What Changes

- **Remove** the `/ui/*` route handler in `internal/ui/server.go::installRoutes`
- **Remove** the `GET /` redirect to `/ui/`
- **Add** a catch-all SPA handler mounted on `/` that:
  - Tries to serve `internal/ui/dist/<request-path>` as a file
  - Falls back to `internal/ui/dist/index.html` for any path that does not exist as a file
  - Preserves `/api/*` and `/ws/*` routes, which are registered before the catch-all so Echo dispatches them first
- **Add** BFF unit tests covering:
  - `GET /` returns the SPA index HTML
  - `GET /dashboard` (unknown path) returns the SPA index HTML (fallback)
  - `GET /api/control/nodes` returns JSON, unaffected by the catch-all
  - `GET /ws/events` upgrades to WebSocket, unaffected by the catch-all
- **BREAKING**: any external client hitting `http://host/ui/*` will receive the SPA fallback (still 200 + index.html) instead of `404`. This is intentional — the SPA always lived under `/ui/` only as a transient arrangement.

## Capabilities

### Modified Capabilities

- **web-bff** (`openspec/specs/web-bff/spec.md`): The "BFF MUST 在 / 路径服务嵌入的 Vue3 SPA" requirement is refined to explicitly require SPA fallback (any non-API/WS path returns `index.html`). Two new scenarios are added: "根路径返回 SPA 入口" and "未知前端路径回退到 SPA 入口".

### New Capabilities

None.

## Impact

- **Code**: `internal/ui/server.go` (route mount order), `internal/ui/server_test.go` (new tests)
- **API**: `/api/*` and `/ws/*` paths unchanged; `/` and `/<anything>` semantics change
- **Deploy**: When Change 3 introduces nginx, the BFF catch-all is no longer needed for production traffic (nginx serves SPA directly). However it remains active to support `make dev` and single-binary deployments.