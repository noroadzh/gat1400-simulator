# Spec Delta

## ADDED Requirements

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