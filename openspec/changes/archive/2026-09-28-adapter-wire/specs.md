## ADDED Requirements

### Requirement: Client MUST 透明地处理 Digest 401 challenge

当 `Client.PostJSON` 收到 `401 Unauthorized` 及 `WWW-Authenticate: Digest ...` 时，客户端 MUST 自动用根据 challenge 计算出的有效 `Authorization: Digest ...` 请求头重试请求。

#### Scenario: 首次 Register 请求收到 401
WHEN `Client.PostJSON` 向 `/VIID/System/Register` 发出的请求收到带有效 Digest challenge 的 `401`
THEN 客户端 MUST 仅重试一次，附带计算出的 Authorization 请求头
AND 最终响应 MUST 返回给调用方。

### Requirement: Client MUST 使用 crypto/rand 生成 nonce

Authorization 请求头中的 `cnonce` MUST 由 `crypto/rand` 生成。禁止使用 `time.Now()` 或其他可预测来源。

#### Scenario: 可预测的 cnonce
WHEN `buildAuthorization` 使用 `time.Now()` 生成的 `cnonce` 被调用
THEN 这 MUST 在代码审查中被标记为违反本需求。

### Requirement: NonceStore MUST 跨重启持久化

Nonce 值 MUST 存储在 SQLite 文件中。客户端进程重启 MUST 不会让仍在有效期内的 nonce 失效。

### Requirement: 重复的 nonce+nc MUST 被服务端拒绝

针对同一 nonce，客户端 MUST 对每次唯一请求自增 `nc`。同一 nonce 用相同 `nc` 发送两次 MUST 导致服务端拒绝第二次请求，返回 `401`。

#### Scenario: nc 计数器冲突
WHEN 两次 Register 调用对同一 nonce 使用 `nc=00000001`
THEN 第二次请求 MUST 在服务端失败（401），因为检测到重放。

### Requirement: User-Identify 请求头 MUST 出现在每个请求中

客户端发出的每个 HTTP 请求 MUST 包含 `User-Identify: <node-id>` 请求头。

### Requirement: Content-Type MUST 为 application/VIID+JSON

每个 POST/PUT 请求体 MUST 以 `Content-Type: application/VIID+JSON` 发送。

### Requirement: Digest 响应 MUST 使用常量时间比较

服务端 Digest 比较 MUST 使用 `crypto/subtle.ConstantTimeCompare`，以防止时序攻击。

---

## ADDED Architecture Decisions

### Decision: NonceStore 是 storage.NonceStore 的薄包装

这样避免重复实现 SQLite 逻辑。`adapter/wire` 中的 `NonceStore` 内部调用 `storage.NewNonceStore`。

### Decision: 客户端仅重试一次

客户端不实现重试循环。若第二次尝试仍失败，则直接返回错误。这避免了在错误配置的服务端上出现无限循环。