# Delta: web-bff spec — vue-componentize-and-dockerize

## MODIFIED Requirements

### Requirement: BFF MUST 在 / 路径服务嵌入的 Vue3 SPA

BFF MUST NOT embed or serve the frontend SPA. The SPA MUST be a standalone Vue 3
build artifact served by a dedicated nginx container on port 80. The backend (port
14080) MUST be a pure API + WebSocket server. A reverse proxy MUST route `/api/` and `/ws/`
to the backend, and all other paths to the SPA.

#### Scenario: 未知前端路径

- **WHEN** 客户端 GET `/nodes`（或任何非 `/api/*`、`/ws/*` 的路径）
- **THEN** 响应 MUST 是 SPA 入口 HTML
- **AND** Content-Type MUST 为 `text/html`。

#### Scenario: 根路径返回 SPA 入口

- **WHEN** 客户端 GET `/`
- **THEN** 响应 MUST 是 SPA 入口 HTML
- **AND** Content-Type MUST 为 `text/html`。

#### Scenario: API 路径不受 SPA fallback 影响

- **WHEN** 客户端 GET `/api/control/nodes`（或任意 `/api/*`、`/ws/*` 路径）
- **THEN** BFF MUST 直接处理该请求（MUST NOT 回退到 `index.html`）
- **AND** Content-Type MUST 为 `application/json`。

#### Scenario: WebSocket 路径不受 SPA fallback 影响

- **WHEN** 客户端建立 `ws://host/ws/events`（或任意 `/ws/*` 路径）
- **THEN** BFF MUST 升级协议为 WebSocket（MUST NOT 回退到 `index.html`）

#### Scenario: Frontend served by nginx, not by BFF

- **WHEN** browser requests any non-API path on the frontend container (e.g. `/`, `/dashboard`)
- **THEN** nginx returns `index.html` or the matching static asset
- **AND** the Go binary has no `//go:embed` directive for `internal/ui/dist/`

#### Scenario: API path unaffected by nginx SPA fallback

- **WHEN** frontend requests `/api/control/nodes` on port 8080
- **THEN** nginx proxies to `http://backend:14080/api/control/nodes`
- **AND** returns JSON (not the SPA fallback)

#### Scenario: WebSocket path proxied with upgrade headers

- **WHEN** frontend opens `ws://localhost:8080/ws/events`
- **THEN** nginx proxies to `http://backend:14080/ws/events` with `Upgrade` and `Connection` headers

---

## ADDED Requirements

### Requirement: Frontend Build Pipeline

The frontend MUST be built with Vite and MUST output to `web/dist/`. The nginx container
MUST be the runtime serving the built SPA, NOT the Go binary.

#### Scenario: npm build produces dist

- **WHEN** developer runs `npm run build` in `web/`
- **THEN** `web/dist/` contains `index.html` and `assets/` chunks
- **AND** `index.html` contains a `<script type="module">` tag loading the Vite bundle

#### Scenario: dist is gitignored

- **WHEN** `git status` is run after a build
- **THEN** `web/dist/` does not appear (it is in `.gitignore`)

### Requirement: Containerization

The project MUST be containerized with Docker multi-stage builds and orchestrated with docker-compose.

#### Scenario: Docker multi-stage frontend build

- **GIVEN** `Dockerfile.frontend` with build stage `node:20-alpine` and runtime stage `nginx:alpine`
- **WHEN** `docker build -f Dockerfile.frontend .` succeeds
- **THEN** the image exposes port 80 and serves the SPA at `/`

#### Scenario: Docker multi-stage backend build

- **GIVEN** `Dockerfile.backend` with build stage `golang:1.25-alpine` and runtime stage `gcr.io/distroless/static-debian12`
- **WHEN** `docker build -f Dockerfile.backend .` succeeds
- **THEN** the image exposes port 14080 and runs the gat1400-sim binary

#### Scenario: docker-compose orchestration

- **GIVEN** `docker-compose.yml` defines services `frontend` and `backend`
- **WHEN** `docker-compose up` succeeds
- **THEN** frontend is reachable at `http://localhost:8080`, backend at `http://localhost:14080`
- **AND** backend health check passes before frontend starts

### Requirement: CI must verify frontend build

CI MUST run a frontend job on every branch, and the test/build jobs MUST depend on it.

#### Scenario: Frontend CI job

- **WHEN** CI runs
- **THEN** a `frontend` job runs `npm ci && npm run build && npm run typecheck` in `web/`
- **AND** uploads `frontend-dist` artifact

#### Scenario: Test job depends on frontend build

- **WHEN** CI runs
- **THEN** the `test` job has `needs: [frontend]`

### Requirement: nginx SPA fallback and reverse proxy

nginx MUST implement SPA fallback and MUST reverse-proxy API and WebSocket paths.

#### Scenario: SPA fallback on unknown path

- **WHEN** browser requests `/some/unknown/route` on the frontend container
- **THEN** nginx returns `index.html` with HTTP 200

#### Scenario: API reverse proxy

- **WHEN** browser requests `/api/control/nodes` on port 8080
- **THEN** nginx proxies to `http://backend:14080/api/control/nodes`

#### Scenario: WebSocket reverse proxy

- **WHEN** browser opens WebSocket `/ws/events` on port 8080
- **THEN** nginx proxies to `http://backend:14080/ws/events` with HTTP 1.1 upgrade headers

## Architecture Decisions

### AD-FRONTEND-001: Vite as build tool (over webpack)

- **DECISION**: Use Vite 5 as the frontend build tool
- **RATIONALE**: Faster HMR, simpler config, native ESM dev server, first-class Vue 3 support
- **CONSEQUENCE**: Developers must run `npm install` before `npm run dev`

### AD-FRONTEND-002: npm as package manager (over pnpm)

- **DECISION**: Use npm with `package-lock.json`
- **RATIONALE**: Simplest CI integration (`actions/setup-node` has native npm cache support)
- **CONSEQUENCE**: `npm ci` in CI (not `pnpm install`)

### AD-FRONTEND-003: Separate frontend container (over embedded SPA)

- **DECISION**: Serve SPA from a dedicated nginx container, not from the Go binary
- **RATIONALE**: Independent scaling, smaller Go binary (no embed), standard production pattern
- **CONSEQUENCE**: Backend is pure API + WS server; `static.go` is removed; `go build` no longer depends on `web/dist/`

### AD-FRONTEND-004: Element Plus full import (over tree-shaking)

- **DECISION**: Import all of Element Plus (`import ElementPlus from 'element-plus'`)
- **RATIONALE**: Simpler setup, smaller maintenance surface for v1
- **CONSEQUENCE**: Larger JS bundle; optimize later if bundle size becomes a concern