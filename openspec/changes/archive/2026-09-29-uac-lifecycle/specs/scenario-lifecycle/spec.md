# Spec Delta

## Purpose

场景引擎在节点 materialise 时主动发起 Register，运行时按配置周期驱动 Keepalive 心跳，Stop 时统一发送 UnRegister，完成完整设备节点生命周期闭环。同时提供真实 POST 路径投递 SubscribeNotification 到订阅端点。

## ADDED Requirements

### Requirement: 节点 materialise 时 MUST 按 RegisterDropRate 决定是否发送 Register

`Engine.Start` 调用 `materialise` 时，MUST 根据 `ExceptionSpec.RegisterDropRate` 计算是否跳过本次 Register（`RegisterDropRate=0` 时 100% 发送，`RegisterDropRate=1.0` 时 100% 跳过）。未跳过时，引擎 MUST 通过 `OutboundDispatcher.DispatchRegister` 向目标平台的 `/VIID/System/Register` 发起 HTTP POST。

#### Scenario: RegisterDropRate=0 时注册成功

- **WHEN** 节点注册，`RegisterDropRate=0`
- **THEN** `DispatchRegister` MUST 被调用
- **AND** HTTP 请求 MUST 携带该节点的 DeviceID 作为 `User-Identify` 头。

#### Scenario: RegisterDropRate=1.0 时跳过注册

- **WHEN** 节点注册，`RegisterDropRate=1.0`
- **THEN** `DispatchRegister` MUST NOT 被调用
- **AND** 节点状态 MUST 保持 unknown（不标记为 online）。

### Requirement: 每个 device 节点 MUST 按 keepaliveInterval 周期发送心跳

`Engine.Start` 启动后，MUST 为每个 device 节点启动一个独立 goroutine，按 `keepaliveInterval`（默认 30s）+ jitter 驱动 `OutboundDispatcher.DispatchKeepalive` 调用。goroutine 在 `Engine.Stop` 时随 context cancel 退出。

#### Scenario: 心跳周期 30s + jitter

- **WHEN** `keepaliveInterval=30s`，jitter 随机偏移 ±5s
- **THEN** 两次心跳间隔 MUST 在 25s–35s 之间
- **AND** 节点心跳不应无限重试（超时 5s 后记录错误日志并继续下次心跳）。

#### Scenario: Stop 时 goroutine 退出

- **WHEN** `Engine.Stop` 被调用
- **THEN** 所有 keepalive goroutine MUST 在一个心跳周期内退出（通过 context cancel）
- **AND** Stop MUST 不阻塞。

### Requirement: Engine.Stop 时 MUST 对每个 device 节点发送 UnRegister

`Engine.Stop` 执行退出路径前，MUST 对每个已完成 materialise 的 device 节点调用 `OutboundDispatcher.DispatchUnregister`。

#### Scenario: Stop 触发注销

- **WHEN** `Engine.Stop` 被调用，且有 2 个 device 节点处于 online 状态
- **THEN** 引擎 MUST 发起 2 次 `DispatchUnregister`（各带节点 DeviceID）
- **AND** UnRegister 失败 MUST NOT 阻断 Stop 流程（仅记录日志）。

### Requirement: postNotification MUST 真实 POST 到 SubscribeNotification 端点

当订阅触发通知事件时，`postNotification` MUST 通过 `OutboundDispatcher.DispatchSubscribeNotification` 将通知 POST 到订阅端点（`<subscribe.NotificationURL>`），而非仅记录日志。

#### Scenario: 订阅触发通知

- **WHEN** 订阅配置了 `NotificationURL`，触发事件时
- **THEN** `DispatchSubscribeNotification` MUST 被调用
- **AND** 目标 URL MUST 等于 `NotificationURL`
- **AND** HTTP 失败 MUST 记录错误日志但不阻塞主流程。

### Requirement: keepaliveInterval 为 0 时 MUST 禁用心跳循环

当 `keepaliveInterval <= 0` 时，引擎 MUST NOT 为任何节点启动 keepalive goroutine。

#### Scenario: 心跳禁用

- **WHEN** `keepaliveInterval=0`
- **THEN** `runKeepalive` MUST NOT 被调用
- **AND** `Engine.IsRunning` 仍返回 true（心跳禁用不等于场景未运行）。

### Requirement: 异常注入 KeepaliveJitter MUST 影响心跳间隔

`ExceptionSpec.KeepaliveJitter` 字段（范围 0.0–1.0）MUST 在每次心跳间隔计算中加入随机偏移：`interval = baseInterval * (1 + jitter * (rand - 0.5))`，使心跳间隔在 `[baseInterval*(1-jitter/2), baseInterval*(1+jitter/2)]` 范围内随机浮动。

#### Scenario: jitter=0.2 时心跳间隔波动 10%

- **WHEN** `KeepaliveJitter=0.2`，`keepaliveInterval=30s`
- **THEN** 实际心跳间隔 MUST 在 27s–33s 之间
- **AND** 对同一 seed，同一节点 MUST 产生相同的 jitter 值（确定性）。

### Requirement: Engine.SetKeepaliveInterval MUST 设置心跳周期

`Engine` MUST 提供 `SetKeepaliveInterval(d time.Duration)` 方法，在 `Start` 之前调用时设置后续心跳 goroutine 的周期。`Start` 后调用行为未定义（调用方负责在 Start 前设置）。

#### Scenario: 设置心跳周期为 60s

- **WHEN** 调用 `engine.SetKeepaliveInterval(60 * time.Second)`
- **THEN** 后续启动的 keepalive goroutine MUST 使用 60s 基准周期
- **AND** 已运行的 goroutine 不受影响（允许在 Start 前配置）。