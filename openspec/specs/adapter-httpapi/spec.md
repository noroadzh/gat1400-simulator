# 适配器：HTTP API — 主规格

> 合并来源：`archive/adapter-httpapi/specs.md`

## Purpose

定义 HTTP API 服务端的行为规范：协议路由、Content-Type、Digest 认证、User-Identify 头处理、资源 CRUD、Cascade 订阅布控、Catalog 静态数据、Capture 中间件与 Nonce 重放检测。

## Requirements

### Requirement: 所有协议路由 MUST 返回 application/VIID+JSON

协议服务端的每个响应 MUST 携带 `Content-Type: application/VIID+JSON`。Content-Type MUST 在 Body 写入 wire 之前设置。

#### Scenario: Persons POST

- **WHEN** 客户端 POST `/VIID/Persons`
- **THEN** 响应 Content-Type MUST 等于 `application/VIID+JSON; charset=UTF-8`。

### Requirement: System Register/UnRegister MUST 强制 Digest 认证

`/VIID/System/Register` 与 `/VIID/System/UnRegister` MUST 强制 HTTP Digest 认证。缺少合法 `Authorization: Digest ...` 头的请求 MUST 返回 `401 Unauthorized` 并附带 `WWW-Authenticate: Digest realm="..."`。

#### Scenario: 首次 POST 缺少 Authorization

- **WHEN** 客户端 POST `/VIID/System/Register` 且未携带 `Authorization`
- **THEN** 响应 MUST 为 `401`
- **AND** `WWW-Authenticate` 头 MUST 包含 `Digest realm="<configured realm>"`。

#### Scenario: 携带合法 Digest

- **WHEN** 客户端 POST `/VIID/System/Register`，Authorization 按 RFC 2617 正确计算
- **THEN** 响应 MUST 为 `200`，且 `ResponseStatus.StatusCode=0`。

### Requirement: User-Identify 头 MUST 触发心跳更新

任何携带 `User-Identify: <node-id>` 的请求，服务端 MUST 调用 `NodeService.MarkSeen(ctx, node-id)`，更新节点的 `LastSeenAt` 并将 `Status` 置为 `online`。

#### Scenario: 已知节点心跳

- **WHEN** 已知节点 POST `/VIID/System/Keepalive`，头 `User-Identify: DEV-1`
- **THEN** 节点状态 MUST 为 `online`，`LastSeenAt` MUST 在最近 5 秒内。

#### Scenario: 未知节点 User-Identify

- **WHEN** 请求携带 `User-Identify: unknown`
- **THEN** 服务端 MUST 返回 `404 Not Found`（响应体 MUST 含 `StatusCode=2`）。

### Requirement: 资源集合 CRUD MUST 受支持

每个 Kind（`Person` / `Face` / `Vehicle` 等）MUST 支持 POST（批量插入）、GET（列表 / 单条）、DELETE。

#### Scenario: POST /VIID/Persons 携带 2 条

- **WHEN** 客户端 POST `{"PersonList": {"PersonObject": [...]}}`，含 2 条记录
- **THEN** 响应 MUST 为 `200`，`ItemCount=2`。

#### Scenario: 插入后 GET /VIID/Persons

- **WHEN** 客户端 GET `/VIID/Persons`
- **THEN** 响应 MUST 在 `PersonList.PersonObject` 中包含两条记录。

#### Scenario: DELETE 删除单条

- **WHEN** 客户端 DELETE `/VIID/Persons/p1`
- **THEN** 后续 GET `/VIID/Persons/p1` MUST 返回 `404`。

### Requirement: Cascade 路由 MUST 同时支持 URL 路径与 Body 操作符

`/VIID/Subscribes`、`/VIID/SubscribeNotifications`、`/VIID/Dispositions` MUST 同时支持两种删除形态：URL 路径删除（`DELETE /VIID/Subscribes/:id`）与 Body 操作符删除（`POST /VIID/Subscribes` body 含 `SubscribeIDList` 或 `DeleteOperate`）。两种形态在功能上 MUST 等价——同一 ID 通过任一方式删除后，对应记录 MUST 从 `subscribeRepo.subs` 中移除。

#### Scenario: Cascade 端点存在

- **WHEN** 客户端向 `/VIID/Subscribes`、`/VIID/SubscribeNotifications`、`/VIID/Dispositions` 发送合法请求
- **THEN** 每个端点 MUST 响应 200 并返回符合 VIID 协议的 JSON 体。

#### Scenario: URL 路径删除订阅

- **WHEN** 客户端 `DELETE /VIID/Subscribes/SUB00000001`
- **THEN** 响应 MUST 为 200，`ResponseStatus.StatusCode=0`
- **AND** 后续 `GET /VIID/Subscribes` MUST 不含该 ID。

#### Scenario: Body 操作符删除订阅

- **WHEN** 客户端 `POST /VIID/Subscribes`，body 含 `{"SubscribeIDList": ["SUB00000001"]}` 或 `{"DeleteOperate": {"SubscribeIDList": [...]}}`
- **THEN** 响应 MUST 为 200
- **AND** 后续 `GET /VIID/Subscribes` MUST 不含该 ID（与 URL 路径删除等价）。

#### Scenario: URL 路径删除布控

- **WHEN** 客户端 `DELETE /VIID/Dispositions/DISP001`
- **THEN** 响应 MUST 为 200
- **AND** 后续 `GET /VIID/Dispositions` MUST 不含该 ID。

#### Scenario: Body 操作符删除布控

- **WHEN** 客户端 `POST /VIID/Dispositions`，body 含 `{"DispositionIDList": ["DISP001"]}`
- **THEN** 响应 MUST 为 200
- **AND** 后续 `GET /VIID/Dispositions` MUST 不含该 ID。

### Requirement: Catalog 路由 MUST 静态返回

`/VIID/APEs`、`/VIID/APSs`、`/VIID/Tollgates`、`/VIID/Lanes` MUST 返回来自 `internal/adapter/httpapi/catalog.go` 的静态种子数据。每个集合 MUST 至少包含 1 条记录。

#### Scenario: 静态种子可访问

- **WHEN** 客户端 GET `/VIID/APEs`
- **THEN** 响应 MUST 返回至少 1 条 APE 记录，且 `ResponseStatus.StatusCode=0`。

### Requirement: 所有响应 MUST 包含 ResponseStatus

每个成功 / 错误响应 MUST 包含顶层 `ResponseStatus` 字段，含 `StatusCode`（整数 0–4）和 `StatusString`（`OK` / `INVALID` / `NOTFOUND` / `UNAUTHORIZED` / `SERVER_ERROR`）。

#### Scenario: 成功响应携带 ResponseStatus

- **WHEN** 任意端点返回 200
- **THEN** 响应体顶层 MUST 包含 `ResponseStatus.StatusCode=0` 且 `StatusString="OK"`。

### Requirement: Capture 中间件 MUST 记录每次请求

每个入站请求 MUST 被记录 NodeID、Method、Path、URL、Remote、Status、Header、RequestBody、ResponseBody。记录 MUST 异步写入 CaptureStore（非阻塞）。

#### Scenario: POST Body 完整保留

- **WHEN** 一次 POST 携带 1KB JSON Body，响应为 200
- **THEN** CaptureStore MUST 包含一条记录，含原始请求 Body 与响应 Body。

### Requirement: 系统端点 MUST 标注 GA/T 1400.4 节号

`internal/adapter/httpapi/system.go`、`collection.go`、`cascade.go`、`catalog.go` 中每个路由处理函数 MUST 在函数注释中标明对应的 GA/T 1400.4 节号（如 `// GA/T 1400.4 §5.4 Subscribe`），便于审计与维护。

#### Scenario: System 端点注释节号

- **WHEN** 阅读 `system.go` 中 `handleRegister` 函数
- **THEN** 函数 docstring MUST 出现 `§5.1`、`§5.2`、`§5.3` 或 `§5.4` 之一。

#### Scenario: Cascade 端点注释节号

- **WHEN** 阅读 `cascade.go` 中 `handleSubscribeCreate` 函数
- **THEN** 函数 docstring MUST 引用 GA/T 1400.4 §5.4（订阅与通知）节号。

#### Scenario: Catalog 端点注释节号

- **WHEN** 阅读 `catalog.go` 中任意 handle 函数
- **THEN** 函数 docstring MUST 引用 GA/T 1400.4 §5.5（目录）节号。

### Requirement: Nonce 重放 MUST 被拒绝

Digest 请求使用相同 `nc` 的 nonce 已被消费过，服务端 MUST 返回 `401 Unauthorized`。`verifyAuthorization` MUST 实际调用 `NonceStore.Consume(nonce)` 以标记 nonce 已被消费；调用失败（如 nonce 不存在或已过期）MUST 返回错误并触发 401 挑战。

#### Scenario: 重放 nonce 被拒

- **WHEN** Digest 请求复用此前已被服务端消费过的 nonce
- **THEN** 服务端 MUST 返回 `401 Unauthorized`，并不执行下游业务逻辑
- **AND** `NonceStore.Consume` MUST 返回 `ErrNonceUnknown`。

#### Scenario: 合法 nonce 通过校验

- **WHEN** Digest 请求携带未过期且未被消费的 nonce
- **THEN** `verifyAuthorization` MUST 返回 `nil`
- **AND** `NonceStore.Consume` MUST 成功标记该 nonce 已消费一次。

---

## ADDED Architecture Decisions

### Decision: HTTP 路由使用 labstack/echo

Echo 提供路由分组、中间件链、自定义 Binder，直接映射 4 类路由。

### Decision: NonceStore 持久化到 SQLite

Nonce 写入 SQLite 而非进程内 map。可横向扩展（多进程共享一份 nonce 表）并支持重启。

### Decision: Digest 中间件先于路由处理器执行

Digest MUST 在任何业务逻辑之前校验，避免鉴权失败时仍消耗昂贵的处理器资源（防 DoS）。