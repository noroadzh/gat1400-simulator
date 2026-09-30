# Web 控制面 BFF — 主规格

> 合并来源：`archive/web-control-plane/specs.md`

## Purpose

定义 Web BFF 的控制面接口规范：REST API 路由（/api/control/）、Vue3 SPA 嵌入服务、WebSocket 事件广播、节点 CRUD、Capture 持久化与查询、JSONL/HAR 导出。BFF 本身不包含业务逻辑，委派给 application 层。

## Requirements

### Requirement: BFF MUST 在 /api/control/ 下暴露 REST API

所有控制面接口 MUST 位于 `/api/control/` 前缀下。协议服务端（`/VIID/...`）MUST 部署在不同端口。

#### Scenario: 不同端口

- **WHEN** 模拟器以默认配置启动
- **THEN** `:19000` MUST 服务 BFF（控制面）
- **AND** `:19001` MUST 服务协议服务端（VIID）。

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

### Requirement: WebSocket MUST 广播事件到全部已连接客户端

事件发布到 Hub 后，所有已连接 WebSocket 客户端 MUST 在 100ms 内收到。

#### Scenario: 节点状态变更

- **WHEN** 节点状态从 `online` 变为 `stopped`
- **THEN** 所有已连接 WebSocket 客户端 MUST 收到类型为 `node.status` 的事件，含 `id` 与 `status`。

### Requirement: BFF MUST 返回 JSON 响应

所有 REST 端点 MUST 返回 `Content-Type: application/json`。响应 MUST 包为 `{items: [...]}`（列表）或 `{...对象...}`（单条）。

#### Scenario: 节点列表返回 JSON

- **WHEN** 客户端 GET `/api/control/nodes`
- **THEN** Content-Type MUST 为 `application/json`
- **AND** 响应体 MUST 符合 `{items: [...]}` 格式。

### Requirement: BFF MUST 校验节点创建 payload

POST `/api/control/nodes` 缺必填字段（`id`、`name`、`role`）时，响应 MUST 为 `400 Bad Request`，且包含可读错误说明。

#### Scenario: 缺必填字段返回 400

- **WHEN** POST `/api/control/nodes` 缺少 `id` 字段
- **THEN** 响应 MUST 为 `400 Bad Request`
- **AND** 响应体 MUST 含可读错误描述。

### Requirement: BFF MUST 持久化抓包并按需查询

BFF MUST 将全部抓包持久化到 `CaptureStore`，并通过 `/api/control/captures` 暴露。查询 MUST 支持 `limit`、`nodeId`、`method`、`path` 过滤。

#### Scenario: 抓包查询支持过滤

- **WHEN** GET `/api/control/captures?nodeId=DEV-1&limit=10`
- **THEN** 响应 MUST 返回最多 10 条 nodeId=DEV-1 的记录
- **AND** 每条记录 MUST 包含 RequestBody 与 ResponseBody。

### Requirement: BFF MUST 支持 JSONL / HAR 导出

`/api/control/captures/export/jsonl` 与 `/api/control/captures/export/har` MUST 返回 `200 OK`，Body 为 `{path, count}`。path MUST 是本地可写文件。

#### Scenario: JSONL 导出返回文件路径

- **WHEN** GET `/api/control/captures/export/jsonl`
- **THEN** 响应 MUST 为 `200`
- **AND** 响应体 MUST 形如 `{"path":"/tmp/...","count":42}`。

---

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

### Requirement: BFF 暴露资源对象控制面端点组

BFF MUST 在 `/api/control/resources` 路径下注册以下端点，用于把协议端
`/VIID/<Collection>` 已实现的资源对象 API 以控制面 JSON API 形式暴露给 Web UI 与
外部探测工具：

- `GET /api/control/resources` — 返回 12 种 Kind 的元数据列表（含每种 Kind 的当前计数）
- `GET /api/control/resources/:kind/list` — 列表查询，透传协议端 GET `/VIID/<Collection>`
- `GET /api/control/resources/:kind/list/:id` — 单条查询，透传协议端 GET `/VIID/<Collection>/:id`
- `POST /api/control/resources/:kind/list` — 批量写入，透传协议端 POST `/VIID/<Collection>`
- `PUT /api/control/resources/:kind/list/:id` — 单条更新，透传协议端 PUT `/VIID/<Collection>/:id`
- `DELETE /api/control/resources/:kind/list/:id` — 单条删除，透传协议端 DELETE `/VIID/<Collection>/:id`
- `GET /api/control/resources/:kind/list/:id/info` — Info 子资源，透传协议端 GET `/VIID/<Collection>/:id/Info`

`:kind` 路径段 MUST 取自 `internal/domain/resource.AllKinds` 枚举之一；非法 `:kind`
MUST 立即返回 `400 Bad Request`。

#### Scenario: Kind 元数据列表返回 12 种
- **WHEN** 客户端 GET `/api/control/resources`
- **THEN** BFF MUST 返回 `200 OK`，body 形如
  `{"kinds":[{"kind":"Person","collection":"Persons","idField":"PersonID","count":0,"description":"人员"}, ...]}`，
  且 `kinds` 数组长度 MUST 等于 12

#### Scenario: 非法 kind 返回 400
- **WHEN** 客户端 GET `/api/control/resources/Foo/list`
- **THEN** BFF MUST 返回 `400 Bad Request`，body MUST 含 `ResponseStatus` 错误说明

### Requirement: BFF 资源端点为薄透传

BFF MUST 不解析、不修改、不重命名协议端响应 body 字段；协议端返回的 status code、headers
（除 Content-Type 标准化为 `application/json`）与 JSON body MUST 原样转发回客户端。
BFF MUST 不缓存协议端响应。

#### Scenario: 协议端 404 原样透传
- **WHEN** 协议端 GET `/VIID/Persons/unknown` 返回 `404 Not Found` + 错误信封
- **THEN** BFF GET `/api/control/resources/Persons/list/unknown` MUST 返回 `404 Not
  Found`，body MUST 与协议端响应完全一致

#### Scenario: 协议端 200 列表透传
- **WHEN** 协议端 GET `/VIID/Persons` 返回 `200 OK` + 列表信封
- **THEN** BFF GET `/api/control/resources/Persons/list` MUST 返回 `200 OK`，body 必须
  保持 `{ResponseStatus, PersonList:{PersonObject:[...]}}` 信封结构

### Requirement: BFF 资源端点 POST 透传时 Content-Type 透传

BFF 在转发 POST 请求时 MUST 把客户端请求的 `Content-Type` 头原样转发给协议端（如
`application/VIID+JSON` 或 `application/json`），不得擅自改写。响应 `Content-Type` MUST
标准化为 `application/json`。

#### Scenario: 客户端发 application/VIID+JSON
- **WHEN** 客户端 POST `/api/control/resources/Persons/list` 且请求头
  `Content-Type: application/VIID+JSON`
- **THEN** BFF 转发给协议端的请求 MUST 保留 `Content-Type: application/VIID+JSON`，
  且响应 MUST 为 `application/json`

### Requirement: BFF 资源端点不重新实现业务逻辑

BFF MUST NOT 实现资源对象的内存存储、解析、字段校验等业务逻辑；BFF MUST 把这些职责委派
给协议端 `internal/adapter/httpapi.Server`。BFF 与协议端的通信 MUST 仅通过 HTTP
（127.0.0.1:<protocol_port>），不得通过进程内函数直接调用或共享可变状态。

#### Scenario: BFF 进程崩溃不影响协议端状态
- **WHEN** BFF 进程被 SIGKILL，且协议端仍在运行
- **THEN** 协议端 `/VIID/Persons` 已存储的所有资源对象 MUST 仍然可被直接查询到（因协议
  端自有内存存储），不受 BFF 崩溃影响

### Requirement: BFF 资源端点日志

BFF MUST 在每个资源端点请求完成后记 `slog.Debug` 级别日志，包含字段：
- `kind` — Kind 字符串
- `path` — 完整请求路径
- `status` — 实际响应状态码
- `duration_ms` — 处理耗时

失败请求 MUST 升级为 `slog.Warn`，包含 `error` 字段。

#### Scenario: 成功请求记录 Debug 日志
- **WHEN** 客户端 GET `/api/control/resources/Persons/list` 成功
- **THEN** BFF MUST 输出 Debug 级别日志，包含 `kind=Person path=/api/control/resources/Persons/list status=200`

#### Scenario: 协议端 5xx 记录 Warn 日志
- **WHEN** 协议端返回 `500 Internal Server Error`
- **THEN** BFF MUST 输出 Warn 级别日志，含 `kind=<Kind> status=500 error=<错误描述>`

### Requirement: BFF 资源端点与既有 BFF 行为兼容

BFF 资源端点 MUST 复用 `web-bff` 已规范的所有通用行为：响应 Content-Type 为
`application/json`、JSON body 由 `s.apiJSON(c, status, payload)` 生成、错误响应统一使用
`response.Error(id, code, msg)` 信封、不引入新端口（沿用 `:14080`）。

#### Scenario: 错误响应使用 envelope
- **WHEN** 客户端 DELETE `/api/control/resources/Persons/list/unknown` 且协议端返回 404

- **THEN** BFF MUST 透传该 404 响应且 body MUST 含 `ResponseStatus` 错误信封（不裸传）

## ADDED Architecture Decisions

### Decision: BFF 不包含业务逻辑

BFF 委派给 `application.NodeService` 与 `application.ScenarioService`。MUST NOT 直接调用 `adapter/storage` 或 `adapter/httpapi`。保持 BFF 薄。

### Decision: WebSocket 事件尽力而为

慢客户端在 30 秒读超时后被断开。被慢客户端丢弃的事件不会回放。避免 Hub 反压阻塞全系统。

### Decision: 前端 bundle 在编译期嵌入（当前为 CDN SPA，Change 3 升级为 Vite 构建）

**当前状态**：前端是一个手写的、380 行的 CDN 单文件 SPA（`internal/ui/dist/index.html`），通过 Vue 3、Element Plus 的 CDN 加载，所有 Vue SFC 集成在一个 `<script>` 块中。该 SPA 通过 `//go:embed all:dist` 嵌入二进制。

**未来状态**：Change 3 `vue-componentize-and-dockerize` 将引入 `web/` 项目，使用 Vite + Vue 3 SFC + Element Plus，构建产物继续输出到 `internal/ui/dist/`。docker-compose 部署后 nginx 自服务 SPA，不再通过 BFF embed。
