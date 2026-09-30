# Spec Delta

## Purpose

提供 wire.Client 的高层 Cascade 类 API（订阅与布控），让上层调用方可以通过子类型 `Cascade` 发起 Subscribe/Disposition 请求与推送 SubscribeNotification，封装 HTTP 路径与 body 构造，复用既有的 Digest 401 自动重试。

## ADDED Requirements

### Requirement: Cascade.SubscribeCreate MUST POST 到 /VIID/Subscribes

`Client.Cascade()` 返回的 `Cascade` 子类型 MUST 提供 `SubscribeCreate(ctx, baseURL, body)` 方法，内部 POST 到 `<baseURL>/VIID/Subscribes`，从响应体解析并返回首个 `SubscribeID`。

#### Scenario: 成功创建订阅

- **WHEN** 调用 `SubscribeCreate` 携带含 `SubscribeList` 的合法 body
- **THEN** 客户端 MUST POST 到 `/VIID/Subscribes`
- **AND** 调用方 MUST 收到非空 `SubscribeID`，或响应失败时的 error。

### Requirement: Cascade.SubscribeDelete MUST POST 到 /VIID/Subscribes 含 DeleteOperate

`Cascade` 子类型 MUST 提供 `SubscribeDelete(ctx, baseURL, subscribeID)` 方法，内部 POST 到 `<baseURL>/VIID/Subscribes`，body 携带 `DeleteOperate.SubscribeID = subscribeID`。

#### Scenario: 删除订阅

- **WHEN** 调用 `SubscribeDelete` 携带合法 12 位 SubscribeID
- **THEN** 服务端 MUST 收到含该 ID 的 body 并返回 200。

### Requirement: Cascade.SubscribeList MUST GET /VIID/Subscribes 并返回列表

`Cascade` 子类型 MUST 提供 `SubscribeList(ctx, baseURL)` 方法，内部 GET `<baseURL>/VIID/Subscribes`，从响应解析 `SubscribeList.SubscribeObject` 数组并返回。

#### Scenario: 列出订阅

- **WHEN** 服务端存在至少 1 个订阅
- **THEN** 返回切片长度 MUST >= 1
- **AND** 服务端无订阅时 MUST 返回空切片且无 error。

### Requirement: Cascade.DispositionCreate MUST POST 到 /VIID/Dispositions

`Cascade` 子类型 MUST 提供 `DispositionCreate(ctx, baseURL, body)` 方法，内部 POST 到 `<baseURL>/VIID/Dispositions`，从响应解析并返回首个 `DispositionID`。

#### Scenario: 成功创建布控

- **WHEN** 调用 `DispositionCreate` 携带合法 body
- **THEN** 调用方 MUST 收到非空 `DispositionID`，或响应失败时的 error。

### Requirement: Cascade.SubscribeNotification MUST POST 到 /VIID/SubscribeNotifications

`Cascade` 子类型 MUST 提供 `SubscribeNotification(ctx, baseURL, body)` 方法，内部 POST 到 `<baseURL>/VIID/SubscribeNotifications`，供订阅端推送通知到上游节点。

#### Scenario: 推送订阅通知

- **WHEN** 调用 `SubscribeNotification` 携带合法 body
- **THEN** 服务端 MUST 收到 POST 并返回 200。

### Requirement: Cascade 高层方法 MUST 透传 Digest 401 重试

所有 Cascade 子类型方法 MUST 复用底层 `PostJSON/GetJSON` 的 Digest 401 自动重试逻辑。

#### Scenario: Digest challenge 透明重试

- **WHEN** 服务端首次返回 401 + Digest challenge
- **THEN** 客户端 MUST 自动重试一次
- **AND** 调用方 MUST 仅收到最终成功响应。