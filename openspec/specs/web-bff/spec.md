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

BFF MUST 在根路径服务内嵌的 `web/dist/` 目录。任意非 API 路径 MUST 回退到 `index.html`（SPA 路由）。

#### Scenario: 未知前端路径

- **WHEN** 客户端 GET `/nodes`
- **THEN** 响应 MUST 是内嵌的 `index.html`
- **AND** Content-Type MUST 为 `text/html`。

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

## ADDED Architecture Decisions

### Decision: BFF 不包含业务逻辑

BFF 委派给 `application.NodeService` 与 `application.ScenarioService`。MUST NOT 直接调用 `adapter/storage` 或 `adapter/httpapi`。保持 BFF 薄。

### Decision: WebSocket 事件尽力而为

慢客户端在 30 秒读超时后被断开。被慢客户端丢弃的事件不会回放。避免 Hub 反压阻塞全系统。

### Decision: 前端 bundle 在编译期嵌入

前端构建产物（`web/dist/`）通过 `embed.FS` 嵌入，直接在内存中服务。运行时无需文件系统访问。