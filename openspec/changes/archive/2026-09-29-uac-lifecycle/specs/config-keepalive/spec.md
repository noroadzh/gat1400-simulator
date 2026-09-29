# Spec Delta

## Purpose

在 Config 中新增 `KeepaliveInterval` 字段，使运维可通过 YAML 配置心跳周期，无需重新编译即可调参。默认值 30s 符合 GA/T 1400 常见实践。

## ADDED Requirements

### Requirement: Config MUST 支持 YAML 解析 keepaliveInterval

`config.Config` 结构体 MUST 包含 `KeepaliveInterval time.Duration` 字段，且该字段 MUST 可通过 YAML 文件 `keepaliveInterval` 键解析。解析失败 MUST 返回错误。

#### Scenario: YAML 中写 "30s"

- **WHEN** YAML 配置包含 `keepaliveInterval: "30s"`
- **THEN** `Config.KeepaliveInterval` MUST 等于 `30 * time.Second`
- **AND** `config.Load()` MUST 不返回 error。

#### Scenario: YAML 中省略 keepaliveInterval

- **WHEN** YAML 配置中无 `keepaliveInterval` 字段
- **THEN** `Config.KeepaliveInterval` MUST 默认为 `30 * time.Second`
- **AND** `config.Load()` MUST 不返回 error。

#### Scenario: YAML 中写 "0s"

- **WHEN** YAML 配置包含 `keepaliveInterval: "0s"`
- **THEN** `Config.KeepaliveInterval` MUST 等于 `0`（禁用心跳）
- **AND** `config.Load()` MUST 不返回 error。

### Requirement: Config.Default MUST 返回含 keepaliveInterval=30s 的默认值

`config.Default()` 返回的 `Config` 实例 MUST 将 `KeepaliveInterval` 设为 `30 * time.Second`。

#### Scenario: Default 返回值含 keepaliveInterval

- **WHEN** 调用 `config.Default()`
- **THEN** `result.KeepaliveInterval` MUST 等于 `30 * time.Second`。