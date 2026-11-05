# Delta: adapter-httpapi

## ADDED Requirements

### Requirement: Cascade 与 System 的 Body 解析错误 MUST 返回 400

含 `c.Bind(&body)` 的 GA/T 1400.4 路由，在 `c.Bind` 返回非 nil error 时 MUST 立即返回 `400 Bad Request`，响应体顶层 MUST 包含 `ResponseStatus.StatusCode=1 (INVALID)` 与可读的 `StatusString`。该行为 MUST 适用于全部 Bind 路由，无例外：

`/VIID/Subscribes` (POST/PUT/DELETE)、`/VIID/Dispositions` (POST/PUT/DELETE)、`/VIID/SubscribeNotifications`、`/VIID/DispositionNotifications`、`/VIID/System/Register`、`/VIID/System/UnRegister`。

`_ = c.Bind(&body)` 形式（吞掉解析错误后继续用零值 `body` 变量处理请求）MUST NOT 出现在协议端 handler 中。

#### Scenario: Subscribes 批量删除携带畸形 JSON

- **WHEN** 客户端 `DELETE /VIID/Subscribes` 且 Body 为 `{invalid` 或声明 JSON 但内容非法
- **THEN** 响应 MUST 为 `400`
- **AND** 响应体 MUST 含 `ResponseStatus.StatusCode=1` 且 `StatusString` 非空
- **AND** `subscribeRepo.subs` MUST NOT 被修改（无订阅被误删）

#### Scenario: Dispositions 批量更新携带畸形 JSON

- **WHEN** 客户端 `PUT /VIID/Dispositions` 且 Body 解析失败
- **THEN** 响应 MUST 为 `400` 且含 `ResponseStatus.StatusCode=1`
- **AND** `subscribeRepo.disp` MUST NOT 被修改

#### Scenario: Dispositions 批量删除携带畸形 JSON

- **WHEN** 客户端 `DELETE /VIID/Dispositions` 且 Body 解析失败
- **THEN** 响应 MUST 为 `400` 且含 `ResponseStatus.StatusCode=1`
- **AND** `subscribeRepo.disp` MUST NOT 被修改

#### Scenario: 合法 Body 缺目标字段仍返回 200

- **WHEN** 客户端 `DELETE /VIID/Subscribes` 且 Body 为合法 JSON `{}`，但不含 `SubscribeIDList`
- **THEN** 响应 MUST 为 `200`（缺字段属"空操作"语义，与 Bind 失败不同）

#### Scenario: 无 Body 的 DELETE 不视为错误

- **WHEN** 客户端 `DELETE /VIID/Subscribes` 且完全不带请求体
- **THEN** 响应 MUST 为 `200`（`c.Bind` 对空 body 不返回 error，MUST NOT 被判为畸形）

### Requirement: System 副作用失败 MUST 走日志而非错误响应

`handleUnRegister` 在 `NodeService.RemoveNode` 返回 error 时 MUST 通过 `Server.log` 写一条 `Warn` 级日志，键 MUST 包含 `event=node_unregister_failed`、`node_id`、`error`；响应 MUST 仍为 `200`（客户端协议动作已完成，不应受内部清理失败影响）。

`handleKeepalive` 在 `NodeService.MarkSeen` 返回 error 时 MUST 同上写一条 `Warn`，键 MUST 包含 `event=node_markseen_failed`、`node_id`、`error`。响应 MUST 仍为 `200`。

#### Scenario: UnRegister 失败仍 200 且可观测

- **WHEN** `NodeService.RemoveNode` 返回非 nil error
- **THEN** 日志 MUST 含 `event=node_unregister_failed` 与失败原因
- **AND** 响应 MUST 为 `200` 且 `ResponseStatus.StatusCode=0`

#### Scenario: Keepalive 失败仍 200 且可观测

- **WHEN** `NodeService.MarkSeen` 返回非 nil error
- **THEN** 日志 MUST 含 `event=node_markseen_failed` 与失败原因
- **AND** 响应 MUST 为 `200`

### Requirement: 订阅与布控 id 生成 MUST NOT 静默吞错

`handleSubscribeCreate` / `handleDispositionCreate` 中，当 `subRepo.store` 因 body 缺 `SubscribeID` 而返回 `ok=false` 时，服务端补齐 id 后再次 `store`。此处的 `id, _ =` MUST 携带注释说明其不可能失败（补齐 id 后必然成功），而非无说明的静默忽略。

#### Scenario: 缺 SubscribeID 时补齐并成功

- **WHEN** `POST /VIID/Subscribes` body 为合法 JSON 但不含 `SubscribeID`
- **THEN** 服务端 MUST 生成 `SubscribeID` 并成功存储，返回 `200` 且响应含非空 `SubscribeID`
