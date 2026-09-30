# Spec Delta: web-bff

## MODIFIED Requirements

### Requirement: BFF MUST 在 / 路径服务嵌入的 Vue3 SPA

BFF MUST 在根路径（`/`）服务内嵌的 Vue3 SPA 资源。SPA 资源位置 MUST 为 `internal/ui/dist/`。任意非 API 路径（即除 `/api/*`、`/ws/*`、SPI 协议端点之外）MUST 回退到 `index.html`，由前端路由处理。

#### Scenario: 根路径返回 SPA 入口

- **WHEN** 浏览器 `GET /`
- **THEN** 响应 MUST 是 SPA 入口 HTML（来自 `internal/ui/dist/index.html`）
- **AND** Content-Type MUST 为 `text/html`。

#### Scenario: 未知前端路径

- **WHEN** 客户端 GET `/nodes`（或任何非 `/api/*`、`/ws/*` 的路径）
- **THEN** 响应 MUST 是 SPA 入口 HTML
- **AND** Content-Type MUST 为 `text/html`。

#### Scenario: 根路径返回 SPA 入口

- **WHEN** 客户端 GET `/`
- **THEN** 响应 MUST 是 SPA 入口 HTML
- **AND** Content-Type MUST 为 `text/html`。

## MODIFIED Architecture Decisions

### Decision: 前端 bundle 在编译期嵌入（当前为 CDN SPA，Change 3 升级为 Vite 构建）

**当前状态**：前端是一个手写的、380 行的 CDN 单文件 SPA（`internal/ui/dist/index.html`），通过 Vue 3、Element Plus 的 CDN 加载，所有 Vue SFC 集成在一个 `<script>` 块中。该 SPA 通过 `//go:embed all:dist` 嵌入二进制。

**未来状态**：Change 3 `vue-componentize-and-dockerize` 将引入 `web/` 项目，使用 Vite + Vue 3 SFC + Element Plus，构建产物继续输出到 `internal/ui/dist/`。docker-compose 部署后 nginx 自服务 SPA，不再通过 BFF embed。

> **历史偏差**：归档 change `2026-09-28-web-control-plane/tasks.md` 声称已交付 `web/package.json` 与 `web/vite.config.ts`，实际从未交付。该 change 的真相由 Change 1 `align-web-control-plane-reality` 对齐。