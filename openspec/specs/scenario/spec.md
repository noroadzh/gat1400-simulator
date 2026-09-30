# 场景引擎 — 主规格

> 合并来源：`archive/scenario-engine/specs.md`

## Purpose

定义场景引擎的核心行为：YAML 加载、节点 materialise、资源 Kind 支持、autoStart、确定性 ID 生成、确定性 FaultInjector、无引擎模式，以及 OutboundDispatcher 对 System 类与 Cascade 类出站方法的支持。

## Requirements

### Requirement: 场景 MUST 支持 YAML 加载

`LoadAll(dir)` 方法 MUST 读取目录下所有 `*.yaml` 文件并解析为 `Scenario`。YAML 格式错误 MUST 返回错误。

#### Scenario: 合法 YAML 文件

- **WHEN** `LoadAll(dir)` 在含 1 个合法 YAML 文件的目录上调用
- **THEN** MUST 返回长度为 1 的切片。

#### Scenario: YAML 格式错误

- **WHEN** `LoadAll(dir)` 在含格式错误 YAML 文件的目录上调用
- **THEN** MUST 返回错误。

### Requirement: 场景 MUST 支持全部 7 个资源 Kind

每个场景 MUST 能够声明以下 Kind：`Person`、`Face`、`Vehicle`、`Plate`、`NonMotorVehicle`、`Image`、`Object`。

#### Scenario: 7 个 Kind 均可声明

- **WHEN** YAML 场景声明 `kind: Person` 至 `kind: Object` 共 7 种
- **THEN** 引擎 MUST 成功解析，无 error。

### Requirement: 每个节点 MUST 产生恰好一个 Runnable

对 YAML 中每个 `NodeSpec`，引擎 MUST 产生恰好一个 `Runnable`。该 Runnable 的 `Start` MUST 为每条资源声明启动至多一个 goroutine。

#### Scenario: 单节点单 Runnable

- **WHEN** YAML 含 1 个 NodeSpec（device 类型，含 3 条资源）
- **THEN** `Engine.Start` MUST 启动恰好 1 个 Runnable
- **AND** 该 Runnable MUST 为 3 条资源启动至多 3 个 goroutine。

### Requirement: ScenarioEngine MUST 暴露 Start 与 AutoStart 两个入口

`Engine` MUST 同时暴露 `Start(ctx, s)` 与 `AutoStart(ctx, s)` 两个方法：
- `Start(ctx, s)` MUST 立即启动场景（无条件启动）
- `AutoStart(ctx, s)` MUST 仅在 `Scenario.Schedule.AutoStart=true` 时调用 `Start`，否则 MUST NOT 启动

两者 MUST 共享相同的状态集（`running map[string]*runHandle`），调用语义对调用方一致。

#### Scenario: autoStart=true 启动场景

- **WHEN** YAML 场景 `Schedule.AutoStart=true`，调用 `engine.AutoStart(ctx)`
- **THEN** `engine.IsRunning(id)` MUST 返回 `true`。

#### Scenario: autoStart=false 不启动

- **WHEN** YAML 场景 `Schedule.AutoStart=false`，调用 `engine.AutoStart(ctx)`
- **THEN** `engine.IsRunning(id)` MUST 返回 `false`
- **AND** MUST NOT 启动任何 goroutine。

#### Scenario: Start 无条件启动

- **WHEN** 调用 `engine.Start(ctx, s)`，无视 `Schedule.AutoStart` 字段值
- **THEN** `engine.IsRunning(id)` MUST 返回 `true`。

### Requirement: ResourceFactory MUST 产出确定性 ID

对同一 `seed` 与 `seq`，工厂 MUST 在每次调用时产出相同的 `Resource.ID`。该特性对黄金样本测试至关重要。

#### Scenario: 确定性 ID 复现

- **WHEN** 用相同 `seed=42, seq=1` 两次调用 `Factory.ResourceID()`
- **THEN** 两次返回 MUST 完全相同。

### Requirement: FaultInjector MUST 可概率且可复现

对同一 seed，同一 target MUST 在多次运行中产生相同的故障判定（基于 `math/rand`，每 target 显式 seed）。

#### Scenario: 相同 seed 产生相同故障

- **WHEN** 对同一 target 用 seed=99 运行 FaultInjector 两次
- **THEN** 两次故障判定 MUST 一致。

### Requirement: 引擎 MUST 支持无引擎模式

当构造时传入 `nil` Runnable 工厂（如测试场景），`Start` MUST 将场景标记为 running 但 MUST NOT 启动任何 goroutine。`IsRunning(id)` 仍 MUST 返回 true。

#### Scenario: nil 工厂启动成功

- **WHEN** 引擎以 `nil` 工厂构造，`Start(id)` 被调用
- **THEN** `IsRunning(id)` MUST 返回 `true`
- **AND** 不得 panic 或启动 goroutine。

### Requirement: 引擎 MUST 对未知场景返回错误

`Start(unknownID)` MUST 返回 `ErrScenarioNotFound`。

#### Scenario: 未知场景返回错误

- **WHEN** `Start("not-exist")`
- **THEN** 返回值 MUST 可通过 `errors.Is(err, ErrScenarioNotFound)` 判定。

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

### Requirement: scenario 适配层 MUST 在关键函数处添加协议节号注释

`internal/adapter/scenario/engine.go` 中的 `fireRegisters`、`runKeepalive`、`dispatchNotifications`、`emit` 函数 MUST 在函数 docstring 中标注 GA/T 1400.4 节号（如 §5.1 Register、§5.4 Subscribe），便于审计与维护。

#### Scenario: Engine 关键函数注释节号

- **WHEN** 阅读 `engine.go` 中 `fireRegisters` 函数
- **THEN** 函数 docstring MUST 引用 GA/T 1400.4 §5.1。

#### Scenario: Keepalive 函数注释节号

- **WHEN** 阅读 `engine.go` 中 `runKeepalive` 函数
- **THEN** 函数 docstring MUST 引用 GA/T 1400.4 §5.3（心跳）。

#### Scenario: Notification 函数注释节号

- **WHEN** 阅读 `engine.go` 中 `dispatchNotifications` 函数
- **THEN** 函数 docstring MUST 引用 GA/T 1400.4 §5.4（订阅通知）。

---

## ADDED Architecture Decisions

### Decision: YAML 是场景唯一真源

场景以 YAML 定义，非 Go 代码。运维可直接替换场景而无需重新编译。

### Decision: 引擎不拥有 HTTP 服务

每个节点（若为 platform）的 HTTP 服务由主程序 bootstrap 启动，引擎不负责。引擎仅驱动客户端行为（Register、推送资源）。

### Decision: 缺省工厂为 FakeFactory

若场景省略 `factory:` 字段，使用 `FakeFactory`。使场景文件保持最小化。