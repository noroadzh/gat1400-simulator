# Spec Delta — scenario

> 本文件为 `openspec/specs/scenario.md` 的增量规格。
> 原 spec 中 8 条 Requirement 保持不变，此处新增生命周期（Register/Keepalive/UnRegister/通知投递）的需求。

## MODIFIED Requirements

### Requirement: Dispatcher MUST 支持 System 类与 Cascade 类出站方法

`OutboundDispatcher` MUST 提供以下方法：
- `DispatchRegister(ctx, target, body) error`
- `DispatchKeepalive(ctx, target) error`
- `DispatchUnregister(ctx, target) error`
- `DispatchSubscribe(ctx, target, body) error`
- `DispatchSubscribeNotification(ctx, target, body) error`
- `DispatchDisposition(ctx, target, body) error`

所有方法均 MUST 复用既有的抓包记录逻辑（recorder 非空时记录）。所有方法 MUST 内部复用 `wire.Client.PostJSON`，享受 Digest 401 自动重试。

#### Scenario: DispatchRegister POST 到 /VIID/System/Register
WHEN `Engine.materialise` 触发 `DispatchRegister`
THEN HTTP 请求 MUST 指向 `<target.HTTPListen>/VIID/System/Register`
AND MUST 携带 `User-Identify: <target.ID>` 头。

#### Scenario: DispatchKeepalive POST 到 /VIID/System/Keepalive
WHEN keepalive goroutine 触发 `DispatchKeepalive`
THEN HTTP 请求 MUST 指向 `<target.HTTPListen>/VIID/System/Keepalive`
AND 失败 MUST 返回 error 而非 panic。

#### Scenario: DispatchUnregister POST 到 /VIID/System/UnRegister
WHEN `Engine.Stop` 触发 `DispatchUnregister`
THEN HTTP 请求 body MUST 包含 `UnRegisterObject.DeviceID = <target.ID>`
AND 失败 MUST 仅记录日志，不影响 Stop 流程。

## ADDED Requirements

<!-- 本 delta 暂无独立 ADDED 条目；新增能力见 scenario-lifecycle/spec.md -->