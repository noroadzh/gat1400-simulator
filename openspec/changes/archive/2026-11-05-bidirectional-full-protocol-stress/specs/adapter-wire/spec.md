# Spec Delta

## ADDED Requirements

### Requirement: 客户端 MUST 提供 PutJSON

`Client` MUST 提供 `PutJSON(ctx, url, deviceID, body)`，语义对齐 `PostJSON`（`application/VIID+JSON`、Digest 401 至多重试一次、`User-Identify`）。

#### Scenario: PutJSON Digest 重试

- **WHEN** PutJSON 首次收到 Digest 401 挑战
- **THEN** 客户端 MUST 自动重试一次 PUT 并返回最终状态

### Requirement: 客户端 MUST 提供 DeleteJSON

`Client` MUST 提供 `DeleteJSON(ctx, url, deviceID)`（可选 body 若协议需要），复用同一 Digest 与头注入路径。

#### Scenario: DeleteJSON 成功

- **WHEN** 对端对 DELETE 返回 200
- **THEN** DeleteJSON MUST 返回该状态且 error 为 nil

## MODIFIED Requirements

### Requirement: 高层 UAC API MUST 内部调用 PostJSON/GetJSON

`wire.System`、`wire.Cascade`、`wire.Catalog`、`wire.Collection`、`wire.VIAS` 子类型提供的所有高层方法 MUST 通过 `(*Client).PostJSON`、`GetJSON`、`PutJSON` 或 `DeleteJSON` 实现，不直接操作 `http.Client`。

> 此约束确保 Digest 401 重试逻辑、User-Identify 头注入与 Content-Type 标准化在任何高层路径下均生效，无需调用方重复处理。

#### Scenario: Register 通过 PostJSON 实现

- **WHEN** `System.Register` 被调用
- **THEN** 内部 MUST 调用 `c.PostJSON(ctx, url, deviceID, body)`
- **AND** 401 重试逻辑 MUST 由 `PostJSON` 透明处理。

#### Scenario: SubscribeList 通过 GetJSON 实现

- **WHEN** `Cascade.SubscribeList` 被调用
- **THEN** 内部 MUST 调用 `c.GetJSON(ctx, url, deviceID)`
- **AND** 401 重试逻辑 MUST 由 `GetJSON` 透明处理。

#### Scenario: 集合按 ID 更新通过 PutJSON 实现

- **WHEN** Collection 按 ID 更新被调用
- **THEN** 内部 MUST 调用 `PutJSON`
- **AND** MUST NOT 直接使用 `http.Client`
