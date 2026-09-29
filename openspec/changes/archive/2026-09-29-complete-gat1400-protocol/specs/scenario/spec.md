# Spec Delta: scenario

## MODIFIED Requirements

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

## ADDED Requirements

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