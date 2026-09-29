# Spec Delta: web-bff

## MODIFIED Requirements

### Requirement: BFF MUST 在 / 路径服务嵌入的 Vue3 SPA

BFF MUST 在根路径（`/`）服务内嵌的 Vue3 SPA 资源。SPA 资源位置 MUST 为 `internal/ui/dist/`。任意非 API 路径（即除 `/api/*`、`/ws/*`、SPI 协议端点之外）MUST 回退到 `index.html`，由前端路由处理。

#### Scenario: 未知前端路径

- **WHEN** 客户端 GET `/nodes`（或任何非 `/api/*`、`/ws/*` 的路径）
- **THEN** 响应 MUST 是 SPA 入口 HTML
- **AND** Content-Type MUST 为 `text/html`。

#### Scenario: 根路径返回 SPA 入口

- **WHEN** 客户端 GET `/`
- **THEN** 响应 MUST 是 SPA 入口 HTML
- **AND** Content-Type MUST 为 `text/html`。

#### Scenario: API 路径不受 SPA fallback 影响

- **WHEN** 客户端 GET `/api/control/nodes`
- **THEN** 响应 MUST 由 `/api/control` 路由组处理，Content-Type 为 `application/json`
- **AND** 不走 SPA fallback。

#### Scenario: WebSocket 路径不受 SPA fallback 影响

- **WHEN** 客户端请求 `GET /ws/events` 携带 `Upgrade: websocket` 头
- **THEN** 响应 MUST 是 WebSocket 101 协议升级响应
- **AND** 不走 SPA fallback。