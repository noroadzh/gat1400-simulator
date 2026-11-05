# Design: 硬化错误处理

## 关键设计决策

### 决策 1: 协议端 Body 解析错误统一返回 400 + StatusCode=1

与现有 `handleUnRegister` (`system.go:119-124`) 保持一致：

```go
if err := c.Bind(&body); err != nil {
    return c.JSON(http.StatusBadRequest, response.Error("", response.CodeInvalid, "invalid request body"))
}
```

`response.Error` 在 `internal/domain/response/response.go` 已存在，StatusCode=1 (INVALID)。这与 spec 中"所有响应 MUST 包含 ResponseStatus" Requirement 一致。

### 决策 2: System 路由中副作用失败走日志而非错误响应

UnRegister / MarkSeen 是收到客户端"已注册"请求的"清理动作"——客户端已完成它要做的协议动作，**不应因本地清理失败而收到错误**。但失败必须可见，固走 `logger.Warn` 带 `event=..._failed` 属性键。

`event` 命名规范沿用 `logging/spec.md:142-148`：

- `node_unregister_failed`
- `node_markseen_failed`
- `log_rotate_failed`（logging spec）
- `capture_parse_failed`（web-bff spec）

### 决策 3: `dailyWriter` 持 logger 引用

`dailyWriter` 是写入器，本身不持 `*slog.Logger`。在 `newDailyWriter` 增加 `log *slog.Logger` 参数。`maybeRotate` 在持锁内 `log.Error` 调用，slog 写 stdout/文件是单次锁内调用、可接受（实测 < 1ms）。如未来需进一步解耦可改为 channel + 单独 worker，本变更不引入此复杂度。

### 决策 4: 抓包解析失败不清行

`_ = json.Unmarshal(...)` 当前是"整行继续返回，只是三个字段是空"。决策保留该行为，**仅加日志**——因为整行丢掉会让用户在 UI 上看到抓包"消失"，比显示部分字段更糟。

### 决策 5: Server 注入 logger 字段已存在 vs 新增

需先读 `server.go` 确认 `*Server` 是否已有 `log *slog.Logger` 字段。若有则复用；若无则新增并在 `cmd/gat1400-sim/main.go` 构造 `httpapi.New(...)` 处传入根 logger。

## 风险与缓解

| 风险 | 缓解 |
|---|---|
| Bind 失败现返回 400，旧 e2e 用例可能期待 200 | 在 `test/e2e` 中显式断言畸形 Body → 400 + ResponseStatus.StatusCode=1 |
| 日志量短期增加（无 IO 失败时不会出现，仅失败时） | `event=..._failed` 失败率应 < 1/万，INFO 量不增长 |
| `Server` 注入 logger 改 constructor signature | 若是 public API 改动，需查 `cmd/gat1400-sim/main.go` 同步 |
| 持锁内 `log.Error` 偶发阻塞 | 当前 `logging` 包是同步写，不引入 worker；如未来出现 perf 瓶颈再优化 |
