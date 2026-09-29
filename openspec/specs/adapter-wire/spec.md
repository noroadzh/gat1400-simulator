# 适配器：HTTP 客户端 — 主规格

> 合并来源：`archive/adapter-wire/specs.md`

## Purpose

定义 HTTP 客户端的行为规范：Digest 认证透明重试、crypto/rand cnonce、NonceStore SQLite 持久化、User-Identify 头注入、Content-Type 标准化，以及高层 System/Cascade API 内部必须调用 PostJSON/GetJSON 的约束。

## Requirements

### Requirement: 客户端 MUST 透明处理 Digest 401 挑战

当 `Client.PostJSON` 收到 `401 Unauthorized` 且 `WWW-Authenticate: Digest ...` 时，客户端 MUST 自动重试一次，请求中携带根据挑战计算出的合法 `Authorization: Digest ...` 头。

#### Scenario: Register 首次返回 401

- **WHEN** `Client.PostJSON` 访问 `/VIID/System/Register`，收到 401 与合法 Digest 挑战
- **THEN** 客户端 MUST 仅重试一次，附带计算出的 Authorization 头
- **AND** 最终响应 MUST 返回给调用方。

### Requirement: 客户端 MUST 使用 crypto/rand 生成 nonce

Authorization 头中的 `cnonce` MUST 通过 `crypto/rand` 生成。严禁使用 `time.Now()` 等可预测源。

#### Scenario: cnonce 可预测

- **WHEN** `buildAuthorization` 使用 `time.Now()` 生成的 cnonce
- **THEN** 代码审查 MUST 标注为违反本需求。

### Requirement: NonceStore MUST 跨进程重启保留

Nonce 值 MUST 写入 SQLite 文件。客户端进程重启 MUST NOT 失效仍在有效期内的 nonce。

#### Scenario: 重启后 nonce 仍有效

- **WHEN** 客户端进程重启后发起同一 nonce 的请求
- **THEN** 服务端 MUST 通过 nonce 查询命中 SQLite 中已有的记录，拒绝重复 nc。

### Requirement: 重复的 nonce+nc MUST 被服务端拒绝

对同一 nonce，客户端 MUST 递增 `nc`。对同一 nonce 重复使用相同 `nc`，服务端 MUST 拒绝第二次请求（返回 401）。

#### Scenario: nc 计数器冲突

- **WHEN** 两次 Register 使用同一 nonce 且 `nc=00000001`
- **THEN** 第二次请求 MUST 在服务端失败（401），因为触发了重放检测。

### Requirement: 每个请求 MUST 携带 User-Identify 头

Client 发出的每个 HTTP 请求 MUST 包含 `User-Identify: <node-id>` 头。

#### Scenario: 请求头注入

- **WHEN** Client 发起任意 HTTP 请求
- **THEN** 请求头 MUST 包含 `User-Identify`，值等于配置的节点 DeviceID。

### Requirement: Content-Type MUST 为 application/VIID+JSON

每个 POST / PUT 请求的 Body MUST 以 `Content-Type: application/VIID+JSON` 发送。

#### Scenario: POST Content-Type

- **WHEN** Client 发送任意 POST 请求
- **THEN** 请求头 Content-Type MUST 等于 `application/VIID+JSON`。

### Requirement: Digest 响应 MUST 使用常量时间比较

服务端对摘要值的比较 MUST 使用 `crypto/subtle.ConstantTimeCompare`，防止时序攻击。

#### Scenario: 常量时间比较

- **WHEN** 服务端比对客户端摘要与期望摘要
- **THEN** MUST 使用 `crypto/subtle.ConstantTimeCompare`，不允许 `==` 直接比较。

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

### Requirement: 高层 UAC API MUST 内部调用 PostJSON/GetJSON

`wire.System` 和 `wire.Cascade` 子类型提供的所有高层方法（如 `Register/UnRegister/Keepalive/SubscribeCreate` 等）MUST 通过 `(*Client).PostJSON` 或 `(*Client).GetJSON` 实现，不直接操作 `http.Client`。

> 此约束确保 Digest 401 重试逻辑、User-Identify 头注入与 Content-Type 标准化在任何高层路径下均生效，无需调用方重复处理。

#### Scenario: Register 通过 PostJSON 实现

- **WHEN** `System.Register` 被调用
- **THEN** 内部 MUST 调用 `c.PostJSON(ctx, url, deviceID, body)`
- **AND** 401 重试逻辑 MUST 由 `PostJSON` 透明处理。

#### Scenario: SubscribeList 通过 GetJSON 实现

- **WHEN** `Cascade.SubscribeList` 被调用
- **THEN** 内部 MUST 调用 `c.GetJSON(ctx, url, deviceID)`
- **AND** 401 重试逻辑 MUST 由 `GetJSON` 透明处理。

---

## ADDED Architecture Decisions

### Decision: NonceStore 包装 storage.NonceStore

`adapter/wire.NonceStore` 调用 `storage.NewNonceStore` 内部实现，避免重复 SQLite 逻辑。

### Decision: 客户端最多重试一次

客户端不实现无限重试循环。若第二次仍失败，原样返回错误。避免在服务端故障时死循环。