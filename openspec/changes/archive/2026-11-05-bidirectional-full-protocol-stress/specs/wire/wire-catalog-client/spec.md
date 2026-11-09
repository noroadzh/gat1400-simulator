# Spec Delta

## Purpose

为 UAC 提供 Catalog 与 VIIDServers 高层客户端，使模拟器能对真实平台上报/查询采集设备目录与级联服务器信息，支撑设备侧压测与联调。

## ADDED Requirements

### Requirement: Catalog 子类型 MUST 支持 APEs 列表与按 ID 查询

`Client.Catalog()` 返回的子类型 MUST 提供对 `/VIID/APEs` 的 GET 列表与 `/VIID/APEs/:id` 的 GET，解析标准列表包装并返回对象切片或单对象。

#### Scenario: 列出 APEs

- **WHEN** 调用方请求 APEs 列表且对端返回合法 `APEList`
- **THEN** 客户端 MUST 返回解析后的对象切片且无 error

#### Scenario: 按 ID 获取 APE

- **WHEN** 调用方以合法 ID GET `/VIID/APEs/:id` 且对端返回 200
- **THEN** 客户端 MUST 返回该 APE 对象

### Requirement: Catalog 子类型 MUST 支持 APEs 上报与删除

Catalog 子类型 MUST 提供 POST/PUT `/VIID/APEs`（批量上报）与 DELETE `/VIID/APEs/:id`，复用 Digest 透明重试与 `User-Identify`。

#### Scenario: 上报 APE

- **WHEN** 调用方 POST 合法 `APEList` body
- **THEN** 对端 MUST 收到 `application/VIID+JSON` 请求且成功时调用方无 error

### Requirement: Catalog 子类型 MUST 覆盖 APSs、Tollgates、Lanes

Catalog 子类型 MUST 对 `/VIID/APSs`、`/VIID/Tollgates`、`/VIID/Lanes` 提供与 APEs 对等的列表、按 ID、上报、删除能力（路径与包装字段遵循 `docs/02-protocol/PROTOCOL.md`）。

#### Scenario: Tollgates 列表

- **WHEN** 调用方 GET `/VIID/Tollgates`
- **THEN** 客户端 MUST 解析列表包装并返回切片

### Requirement: Catalog 子类型 MUST 支持 VIIDServers CRUD 子集

Catalog（或并列的 VIIDServers 入口）MUST 支持 GET 列表、GET/PUT/DELETE by id、以及 POST/PUT 集合上报 `/VIID/VIIDServers`。

#### Scenario: 创建级联服务器条目

- **WHEN** 调用方 POST 合法 VIIDServer 对象列表
- **THEN** 成功时 MUST 无 error，失败时 MUST 返回含状态码语义的 error
