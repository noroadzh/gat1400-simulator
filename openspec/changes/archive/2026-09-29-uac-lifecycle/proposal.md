# Proposal

## Why

GA/T 1400 模拟器的服务端（UAS）接口已完整实现，但**客户端侧（UAC）的业务语义封装缺失**。`wire.Client` 目前只有 `PostJSON/GetJSON` 两个通用方法，场景引擎也只推送 Kind 资源，缺少完整的设备节点生命周期管理（Register → Keepalive → UnRegister）和 Cascade 语义（订阅/布控/通知）。这导致模拟器无法端到端验证设备上/下线闭环和订阅通知推送路径。

## What Changes

### wire.Client 高层 UAC 封装
- 新增 `System` 子类型：`Register/UnRegister/Keepalive/ServerTime`
- 新增 `Cascade` 子类型：`SubscribeCreate/SubscribeDelete/SubscribeList/DispositionCreate/SubscribeNotification`
- 均通过已有 `PostJSON/GetJSON` 实现，自动携带 User-Identify 头与 Digest 认证

### scenario/dispatcher.go — OutboundDispatcher 扩展
- 新增 `DispatchRegister(ctx, target, body)` → POST `/VIID/System/Register`
- 新增 `DispatchKeepalive(ctx, target)` → POST `/VIID/System/Keepalive`
- 新增 `DispatchUnregister(ctx, target)` → POST `/VIID/System/UnRegister`
- 新增 `DispatchSubscribe(ctx, target, body)` → POST `/VIID/Subscribes`
- 新增 `DispatchSubscribeNotification(ctx, target, body)` → POST `/VIID/SubscribeNotifications`
- 新增 `DispatchDisposition(ctx, target, body)` → POST `/VIID/Dispositions`
- 提取 `record()` helper 统一出站抓包逻辑

### scenario/engine.go — 完整节点生命周期
- `materialise()` 末尾按 `RegisterDropRate` 异步调用 `DispatchRegister`
- 新增 `runKeepalive()` goroutine：每个 device 节点按 `keepaliveInterval`（默认 30s）+ jitter 发心跳
- `Stop()` 退出前对每个 device 节点调用 `DispatchUnregister`
- `postNotification()` 改为真实 POST 到订阅端点

### 配置透传
- `config.Config` 新增 `KeepaliveInterval` 字段（默认 30s）
- `configs/default.yaml` 新增 `keepaliveInterval: "30s"`
- `main.go` 调用 `engine.SetKeepaliveInterval()`

### 测试
- `internal/adapter/wire/client_uac_test.go`：httptest 验证 Digest 401 重试链路、Keepalive、ServerTime、Cascade 全链路
- `test/e2e/uac_lifecycle_e2e_test.go`：真 TCP socket 验证完整生命周期（Register→Keepalive→Person→UnRegister）和 Cascade 闭环

## Capabilities

### New Capabilities

- `wire-system-client`: wire.Client.System() 高层 System 类 API（Register/UnRegister/Keepalive/ServerTime），封装 HTTP 路径与 body 构造，复用 Digest 401 重试
- `wire-cascade-client`: wire.Client.Cascade() 高层 Cascade 类 API（SubscribeCreate/Delete/List/DispositionCreate），封装订阅与布控的 HTTP 交互
- `scenario-lifecycle`: 场景引擎完整节点生命周期：materialise 触发 Register → runKeepalive 周期心跳 → Stop 触发 Unregister；dispatchNotifications 真实 POST 到订阅端点
- `config-keepalive`: KeepaliveInterval 配置项，透传到场景引擎

### Modified Capabilities

- `adapter-wire`（`openspec/specs/adapter-wire.md`）：现有规格描述了 `PostJSON` 的 Digest 重试行为。新增高层次 API 不修改底层行为，但需在规格中明确"高层 API 内部调用 PostJSON"的实现约束
- `scenario`（`openspec/specs/scenario.md`）：现有规格描述了 YAML 加载与资源分发。新增 lifecycle 行为（Register/Keepalive/Unregister/Notification）作为独立的新需求，不修改现有资源分发行为

## Impact

- **代码**：`internal/adapter/wire/`（新文件 `uac.go`）、`internal/adapter/scenario/dispatcher.go`（扩展）、`internal/adapter/scenario/engine.go`（扩展）、`internal/app/config/config.go`（扩展）、`cmd/gat1400-sim/main.go`（扩展）
- **测试**：`internal/adapter/wire/`（新测试）、`test/e2e/`（新 e2e）
- **配置**：`configs/default.yaml`（新字段）
- **无破坏性变更**：现有 `wire.Client.PostJSON/GetJSON`、场景引擎资源分发逻辑保持不变
