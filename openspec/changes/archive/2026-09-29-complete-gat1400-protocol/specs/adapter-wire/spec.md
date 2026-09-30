# Spec Delta: adapter-wire

## ADDED Requirements

### Requirement: adapter-wire 客户端 MUST 在关键函数处添加协议节号注释

`internal/adapter/wire/client.go` 中的 `do`、`buildRequest`、`buildAuthorization`、`parseChallenge` 函数 MUST 在函数 docstring 中标注 GA/T 1400.4 节号（如 §5.1 Register）以及 RFC 2617 章节号，便于审计与维护。

#### Scenario: client.do 注释节号

- **WHEN** 阅读 `client.go` 中 `do` 函数
- **THEN** 函数 docstring MUST 引用 RFC 2617 §3（Digest 认证握手）。

#### Scenario: client.buildAuthorization 注释节号

- **WHEN** 阅读 `client.go` 中 `buildAuthorization` 函数
- **THEN** 函数 docstring MUST 引用 RFC 2617 §3.2.2.1（response 计算）。

#### Scenario: uac 注释节号

- **WHEN** 阅读 `uac.go` 中 `Register`、`SubscribeCreate`、`SubscribeNotificationPush` 函数
- **THEN** 每个函数 docstring MUST 引用对应 GA/T 1400.4 节号（§5.1 / §5.4）。