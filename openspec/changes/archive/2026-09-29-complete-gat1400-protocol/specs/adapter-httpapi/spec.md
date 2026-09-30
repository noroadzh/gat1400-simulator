# Spec Delta: adapter-httpapi

## MODIFIED Requirements

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

## ADDED Requirements

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