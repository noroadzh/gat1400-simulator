# Spec Delta — adapter-wire

> 本文件为 `openspec/specs/adapter-wire.md` 的增量规格。
> 原 spec 中 7 条 Requirement 的"实质性行为契约"保持不变，此处仅 ADD 一条关于"高层 UAC API 必须复用 PostJSON/GetJSON"的约束。

## MODIFIED Requirements

### Requirement: 高层 UAC API MUST 内部调用 PostJSON/GetJSON

`wire.System` 和 `wire.Cascade` 子类型提供的所有高层方法（如 `Register/UnRegister/Keepalive/SubscribeCreate` 等）MUST 通过 `(*Client).PostJSON` 或 `(*Client).GetJSON` 实现，不直接操作 `http.Client`。

> 此约束确保 Digest 401 重试逻辑、User-Identify 头注入与 Content-Type 标准化在任何高层路径下均生效，无需调用方重复处理。

#### Scenario: Register 通过 PostJSON 实现
WHEN `System.Register` 被调用
THEN 内部 MUST 调用 `c.PostJSON(ctx, url, deviceID, body)`
AND 401 重试逻辑 MUST 由 `PostJSON` 透明处理。

#### Scenario: SubscribeList 通过 GetJSON 实现
WHEN `Cascade.SubscribeList` 被调用
THEN 内部 MUST 调用 `c.GetJSON(ctx, url, deviceID)`
AND 401 重试逻辑 MUST 由 `GetJSON` 透明处理。

## ADDED Requirements

<!-- 本 delta 暂无独立 ADDED 条目；新增能力见新增的 wire-system-client/spec.md 与 wire-cascade-client/spec.md -->