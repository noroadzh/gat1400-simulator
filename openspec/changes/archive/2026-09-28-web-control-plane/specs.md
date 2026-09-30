## ADDED Requirements

### Requirement: BFF MUST 暴露位于 /api/control/ 下的控制面 REST API

所有控制面端点 MUST 位于 `/api/control/` 前缀之下。协议服务器（`/VIID/...`）MUST 在不同端口上提供服务。

#### Scenario: 不同端口
WHEN 模拟器以默认配置启动
THEN `:19000` MUST 提供 BFF（控制面）
AND `:19001` MUST 提供协议服务器（VIID）。

### Requirement: BFF MUST 在 / 下提供嵌入的 Vue3 SPA

BFF MUST 在根路径下提供嵌入的 `web/dist/` 目录。任何非 API 路径 MUST 回退到 `index.html`（SPA 路由）。

#### Scenario: 未知的前端路径
WHEN 客户端 GET `/nodes`
THEN 响应 MUST 为嵌入的 `index.html`
AND Content-Type MUST 为 `text/html`。

### Requirement: WebSocket MUST 向已连接客户端广播事件

当一个事件被发布到 hub 时，所有已连接的 WebSocket 客户端 MUST 在 100ms 内收到该事件。

#### Scenario: 节点状态变化
WHEN 一个节点的状态从 `online` 变为 `stopped`
THEN 所有已连接的 WebSocket 客户端 MUST 收到类型为 `node.status` 的事件，其中携带该节点的 `id` 与 `status`。

### Requirement: BFF MUST 返回 JSON 响应

所有 REST 端点 MUST 返回 `Content-Type: application/json`。列表响应 MUST 包成 `{items: [...]}`，单个对象响应 MUST 为 `{...object...}`。

### Requirement: BFF MUST 校验创建节点的 payload

当 POST `/api/control/nodes` 缺少必填字段（`id`、`name`、`role`）时，响应 MUST 为 `400 Bad Request` 并给出描述性错误信息。

### Requirement: BFF MUST 持久化捕获并按需提供

BFF MUST 将所有捕获持久化到 `CaptureStore`，并通过 `/api/control/captures` 提供。查询 MUST 支持 `limit`、`nodeId`、`method`、`path` 过滤。

### Requirement: BFF MUST 支持 JSONL 与 HAR 导出

端点 `/api/control/captures/export/jsonl` 与 `/api/control/captures/export/har` MUST 返回 `200 OK`，body 为 `{path, count}`。其中 path MUST 是本地文件系统上的可写文件。

---

## ADDED Architecture Decisions

### Decision: BFF 不持有业务逻辑

BFF 委托给 `application.NodeService` 与 `application.ScenarioService`。它 MUST NOT 直接调用 `adapter/storage` 或 `adapter/httpapi`。这使 BFF 保持薄层。

### Decision: WebSocket 事件尽力而为

慢客户端在 30 秒读超期后被断开。由于慢客户端而被丢弃的事件 NOT 会被回放。这避免了 hub 的反压阻塞整个系统。

### Decision: 前端 bundle 在编译期嵌入

前端构建产物（`web/dist/`）通过 `embed.FS` 嵌入并直接从内存提供。运行时不需要文件系统访问。