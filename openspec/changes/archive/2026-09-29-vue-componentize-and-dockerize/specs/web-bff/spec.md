# Delta: web-bff spec — vue-componentize-and-dockerize

## MODIFIED Requirements

### Requirement: BFF MUST 在 / 路径服务嵌入的 Vue3 SPA

BFF MUST NOT embed 或服务前端 SPA。SPA MUST 是独立的 Vue 3 构建产物，由专用 nginx 容器在 80 端口服务。后端（14080 端口）MUST 是纯 API + WebSocket 服务器。反向代理 MUST 将 `/api/` 与 `/ws/` 路由到后端，其余所有路径路由到 SPA。

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

#### Scenario: 前端由 nginx 提供而非 BFF

- **WHEN** 浏览器请求前端容器上的任意非 API 路径（如 `/`、`/dashboard`）
- **THEN** nginx 返回 `index.html` 或匹配的静态资源
- **AND** Go 二进制中不存在针对 `internal/ui/dist/` 的 `//go:embed` 指令

#### Scenario: API 路径不受 nginx SPA fallback 影响

- **WHEN** 前端在 8080 端口请求 `/api/control/nodes`
- **THEN** nginx 代理到 `http://backend:14080/api/control/nodes`
- **AND** 返回 JSON（而非 SPA fallback）

#### Scenario: WebSocket 路径带 upgrade 头代理

- **WHEN** 前端打开 `ws://localhost:8080/ws/events`
- **THEN** nginx 携带 `Upgrade` 与 `Connection` 头代理到 `http://backend:14080/ws/events`

---

## ADDED Requirements

### Requirement: Frontend Build Pipeline

前端 MUST 使用 Vite 构建，MUST 输出到 `web/dist/`。nginx 容器 MUST 是服务构建后 SPA 的运行时，而非 Go 二进制。

#### Scenario: npm build 产出 dist

- **WHEN** 开发者在 `web/` 中运行 `npm run build`
- **THEN** `web/dist/` 包含 `index.html` 与 `assets/` chunks
- **AND** `index.html` 含有加载 Vite bundle 的 `<script type="module">` 标签

#### Scenario: dist 被 gitignore

- **WHEN** 构建后运行 `git status`
- **THEN** `web/dist/` 不出现（它已在 `.gitignore` 中）

### Requirement: Containerization

项目 MUST 使用 Docker 多阶段构建容器化，并用 docker-compose 编排。

#### Scenario: Docker 多阶段前端构建

- **GIVEN** `Dockerfile.frontend` 构建阶段为 `node:20-alpine`、运行阶段为 `nginx:alpine`
- **WHEN** `docker build -f Dockerfile.frontend .` 成功
- **THEN** 镜像暴露 80 端口并在 `/` 服务 SPA

#### Scenario: Docker 多阶段后端构建

- **GIVEN** `Dockerfile.backend` 构建阶段为 `golang:1.25-alpine`、运行阶段为 `gcr.io/distroless/static-debian12`
- **WHEN** `docker build -f Dockerfile.backend .` 成功
- **THEN** 镜像暴露 14080 端口并运行 gat1400-sim 二进制

#### Scenario: docker-compose 编排

- **GIVEN** `docker-compose.yml` 定义 `frontend` 与 `backend` 服务
- **WHEN** `docker-compose up` 成功
- **THEN** frontend 可在 `http://localhost:8080` 访问，backend 可在 `http://localhost:14080` 访问
- **AND** backend 健康检查在 frontend 启动前通过

### Requirement: CI 必须验证前端构建

CI MUST 在每个分支运行 frontend job，test/build job MUST 依赖它。

#### Scenario: Frontend CI job

- **WHEN** CI 运行
- **THEN** `frontend` job 在 `web/` 中执行 `npm ci && npm run build && npm run typecheck`
- **AND** 上传 `frontend-dist` artifact

#### Scenario: Test job 依赖前端构建

- **WHEN** CI 运行
- **THEN** `test` job 声明 `needs: [frontend]`

### Requirement: nginx SPA fallback 与反向代理

nginx MUST 实现 SPA fallback，MUST 反向代理 API 与 WebSocket 路径。

#### Scenario: 未知路径上的 SPA fallback

- **WHEN** 浏览器请求前端容器上的 `/some/unknown/route`
- **THEN** nginx 以 HTTP 200 返回 `index.html`

#### Scenario: API 反向代理

- **WHEN** 浏览器在 8080 端口请求 `/api/control/nodes`
- **THEN** nginx 代理到 `http://backend:14080/api/control/nodes`

#### Scenario: WebSocket 反向代理

- **WHEN** 浏览器在 8080 端口打开 WebSocket `/ws/events`
- **THEN** nginx 携带 HTTP 1.1 upgrade 头代理到 `http://backend:14080/ws/events`

## Architecture Decisions

### AD-FRONTEND-001: Vite 作为构建工具（而非 webpack）

- **DECISION**：前端构建工具使用 Vite 5
- **RATIONALE**：HMR 更快、配置更简单、原生 ESM dev server、对 Vue 3 一流支持
- **CONSEQUENCE**：开发者必须在 `npm run dev` 之前运行 `npm install`

### AD-FRONTEND-002: npm 作为包管理器（而非 pnpm）

- **DECISION**：使用 npm 与 `package-lock.json`
- **RATIONALE**：CI 集成最简单（`actions/setup-node` 原生支持 npm 缓存）
- **CONSEQUENCE**：CI 中使用 `npm ci`（而非 `pnpm install`）

### AD-FRONTEND-003: 独立前端容器（而非内嵌 SPA）

- **DECISION**：SPA 由专用 nginx 容器服务，而非 Go 二进制
- **RATIONALE**：独立扩缩、Go 二进制更小（无 embed）、标准生产模式
- **CONSEQUENCE**：后端是纯 API + WS 服务器；`static.go` 被删除；`go build` 不再依赖 `web/dist/`

### AD-FRONTEND-004: Element Plus 全量引入（而非 tree-shaking）

- **DECISION**：全量引入 Element Plus（`import ElementPlus from 'element-plus'`）
- **RATIONALE**：搭建更简单，v1 维护面更小
- **CONSEQUENCE**：JS bundle 更大；若 bundle 体积成为问题后续再优化