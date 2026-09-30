## ADDED Requirements

### Requirement: 所有协议路由 MUST 以 application/VIID+JSON 响应

协议服务端返回的每一个响应 MUST 携带 `Content-Type: application/VIID+JSON`。该 Content-Type MUST 在响应体写入网络之前设置。

#### Scenario: Persons POST
WHEN 客户端向 `/VIID/Persons` 发起 POST
THEN 响应 Content-Type MUST 等于 `application/VIID+JSON; charset=UTF-8`。

### Requirement: System Register/UnRegister MUST 强制 Digest 认证

路由 `/VIID/System/Register` 与 `/VIID/System/UnRegister` MUST 要求 HTTP Digest 认证。缺少有效 `Authorization: Digest ...` 请求头的请求 MUST 收到 `401 Unauthorized`，并携带 `WWW-Authenticate: Digest realm="..."`。

#### Scenario: 首次 POST 缺少 Authorization
WHEN 客户端向 `/VIID/System/Register` 发起 POST 且不带 `Authorization`
THEN 响应 MUST 为 `401`
AND `WWW-Authenticate` 请求头 MUST 包含 `Digest realm="<configured realm>"`。

#### Scenario: 有效的 Digest 请求
WHEN 客户端向 `/VIID/System/Register` 发起 POST，且携带按 RFC 2617 计算出的正确 Digest 响应值
THEN 响应 MUST 为 `200`，且 `ResponseStatus.StatusCode=0`。

### Requirement: User-Identify 请求头 MUST 记录一次心跳

当任意请求携带 `User-Identify: <node-id>` 时，服务端 MUST 调用 `NodeService.MarkSeen(ctx, node-id)`，并更新该节点的 `LastSeenAt` 与 `Status=online`。

#### Scenario: 已知节点保活
WHEN 已知节点发送 POST `/VIID/System/Keepalive`，携带 `User-Identify: DEV-1`
THEN 该节点状态 MUST 为 `online`，且 `LastSeenAt` MUST 落在最近 5 秒内。

#### Scenario: 未知节点的 User-Identify
WHEN 请求携带 `User-Identify: unknown`
THEN 服务端 MUST 返回 `404 Not Found`（响应体 MUST 包含 `StatusCode=2`）。

### Requirement: 资源采集 CRUD MUST 被支持

每种 Kind（`Person`、`Face`、`Vehicle` 等）MUST 支持 POST（批量插入）、GET（列表或单条）、DELETE。

#### Scenario: POST /VIID/Persons 携带 2 条数据
WHEN 客户端 POST `{"PersonList": {"PersonObject": [...]}}`，包含 2 条记录
THEN 响应 MUST 为 `200`，且 `ItemCount=2`。

#### Scenario: 插入后 GET /VIID/Persons
WHEN 客户端 GET `/VIID/Persons`
THEN 响应 MUST 在 `PersonList.PersonObject` 中包含已插入的两条记录。

#### Scenario: DELETE 移除条目
WHEN 客户端 DELETE `/VIID/Persons/p1`
THEN 随后 GET `/VIID/Persons/p1` MUST 返回 `404`。

### Requirement: 级联路由 MUST 被支持

`/VIID/Subscribes`、`/VIID/SubscribeNotifications`、`/VIID/Dispositions` MUST 以完整 CRUD 形式暴露。

### Requirement: 目录路由 MUST 返回静态数据

`/VIID/APEs`、`/VIID/APSs`、`/VIID/Tollgates`、`/VIID/Lanes` MUST 返回由 `internal/adapter/httpapi/catalog.go` 预置的固定目录数据。数据 MUST 保证每个集合至少一条记录。

### Requirement: 所有响应 MUST 包含 ResponseStatus

每个成功或错误响应 MUST 包含顶层 `ResponseStatus` 字段，含 `StatusCode`（整数 0–4）与 `StatusString`（"OK" / "INVALID" / "NOTFOUND" / "UNAUTHORIZED" / "SERVER_ERROR"）。

### Requirement: 抓包中间件 MUST 记录每一个请求

每个入站请求 MUST 被抓包记录 NodeID、Method、Path、URL、Remote、Status、Header、RequestBody、ResponseBody。抓包记录 MUST 异步持久化到 CaptureStore（非阻塞）。

#### Scenario: PostBody 保留
WHEN 一个 POST 携带 1KB 的 JSON 请求体且响应为 200
THEN CaptureStore MUST 包含一条记录，保存原始请求体与响应体。

### Requirement: Nonce 重放 MUST 被拒绝

当 Digest 请求使用的 nonce 已被相同 `nc` 消费过时，服务端 MUST 返回 `401 Unauthorized`。

---

## ADDED Architecture Decisions

### Decision: 使用 labstack/echo 做 HTTP 路由

Echo 提供的路由分组、中间件链和自定义 Binder 能力，与四大路由族一一对应。

### Decision: NonceStore 持久化到 SQLite

Nonce 存储在 SQLite 而非进程内 map，这样可以水平扩展（多个服务端进程可校验同一 nonce），且重启后仍然有效。

### Decision: Digest 中间件在路由处理器之前执行

Digest 校验必须在任何其他逻辑之前进行，避免认证失败时仍执行高开销处理器而造成拒绝服务。