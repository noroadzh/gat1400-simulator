# Spec Delta

## MODIFIED Requirements

### Requirement: Dispatcher MUST 支持 System 类与 Cascade 类出站方法

`OutboundDispatcher` MUST 提供以下方法：
- `DispatchRegister(ctx, target, body) error`
- `DispatchKeepalive(ctx, target) error`
- `DispatchUnregister(ctx, target) error`
- `DispatchSubscribe(ctx, target, body) error`
- `DispatchSubscribeNotification(ctx, target, body) error`
- `DispatchDisposition(ctx, target, body) error`
- `DispatchDispositionNotification(ctx, target, body) error`
- `DispatchCatalogUpsert(ctx, target, collection, body) error`（collection 为 APEs/APSs/Tollgates/Lanes/VIIDServers 之一）
- `DispatchCollectionPush(ctx, target, collection, body) error`
- `DispatchVIASTaskCreate(ctx, target, body) error`（可选启用；未接线时测试可跳过）

所有方法均 MUST 复用既有的抓包记录逻辑（recorder 非空时记录）。写操作 MUST 内部复用 `wire.Client` 的 PostJSON/PutJSON/DeleteJSON（按动词），享受 Digest 401 自动重试。

#### Scenario: DispatchRegister POST 到 /VIID/System/Register

- **WHEN** `Engine.materialise` 触发 `DispatchRegister`
- **THEN** HTTP 请求 MUST 指向 `<target.HTTPListen>/VIID/System/Register`
- **AND** MUST 携带 `User-Identify: <target.ID>` 头。

#### Scenario: DispatchKeepalive POST 到 /VIID/System/Keepalive

- **WHEN** keepalive goroutine 触发 `DispatchKeepalive`
- **THEN** HTTP 请求 MUST 指向 `<target.HTTPListen>/VIID/System/Keepalive`
- **AND** 失败 MUST 返回 error 而非 panic。

#### Scenario: DispatchUnregister POST 到 /VIID/System/UnRegister

- **WHEN** `Engine.Stop` 触发 `DispatchUnregister`
- **THEN** HTTP 请求 body MUST 包含 `UnRegisterObject.DeviceID = <target.ID>`
- **AND** 失败 MUST 仅记录日志，不影响 Stop 流程。

#### Scenario: DispatchCollectionPush POST 到集合路径

- **WHEN** 触发 `DispatchCollectionPush` 且 collection=`Faces`
- **THEN** HTTP 请求 MUST 指向目标上的 `/VIID/Faces`

#### Scenario: DispatchDispositionNotification POST

- **WHEN** 触发 `DispatchDispositionNotification`
- **THEN** HTTP 请求 MUST 指向 `/VIID/DispositionNotifications`
