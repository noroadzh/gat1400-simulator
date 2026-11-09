# Proposal: 硬化错误处理（消除静默吞错与契约违反）

## 背景

项目当前以"测试通过 + 端到端 green"作为质量底线，但实测发现 `internal/adapter/httpapi/{cascade,system}.go`、`internal/app/logging/logger.go`、`internal/adapter/storage/capture_reader.go` 等 6 处使用了 `_ = ` 静默吞错模式：

1. **Cascade Bind 三处**（`cascade.go:238, 363, 378`）：`_ = c.Bind(&body)` 后继续用零值 `body` 处理订阅/通知/布控请求。畸形 JSON 不会被拒为 400，而是被路由到 "空对象" 的逻辑。
2. **Cascade 双 store 重试**（`cascade.go:185, 328`）：`id, _ = s.subRepo.store(body)` 在生成 ID 后的二次存储失败时不会向调用方暴露。
3. **日志轮转失败被吞**（`logger.go:276`）：`_ = d.rot.Rotate()`，长期跑（模拟器典型是长跑场景）会撑爆磁盘。
4. **抓包表头反序列化失败被吞**（`capture_reader.go:108`）：BFF 抓包详情页会展示空 Header / Request / Response。
5. **UnRegister / MarkSeen 失败被吞**（`system.go:127, 141`）：节点状态可能与实际不一致，错误排查困难。

这些点违反 spec：
- `adapter-httpapi/spec.md` 隐含假设是 `Cascade` 路由**对畸形输入有定义良好的行为**（成功/错误响应均含 `ResponseStatus`），但当前行为是"不报错但走错路"；
- `logging/spec.md` 的"关键状态变更必须可被日志查询" requirement 隐含"系统级副作用（节点上下线、轮转）失败时**应被记录**，但目前连 stderr 兜底都缺。

> 注：扫描脚本最初把 `internal/ui/resources.go:81` 标为"无 Timeout"，但 cross-line 检查确认实际**已有 `Timeout: 10 * time.Second`**，是误报，从本变更中移除。

## 目标

把上述 6 处 `_ = ` 模式替换为显式错误路径或日志记录，并把这些契约写进 `spec.md`（不是改个 _pseudocode_）：

- 协议端的 Cascade / System 路由**对畸形 Body 返回 400**（与现有 System Register 的 Bind 错误处理保持一致）。
- 抓包读路径的 Header / Request / Response 反序列化失败**写入一条 Warn 日志**，但仍展示能反序列化成功的字段，避免 UI 闪断。
- 日志轮转失败**写入一条 Error 日志**。
- UnRegister / MarkSeen 失败**写入一条 Warn 日志**（`event=node_unregister_failed` / `event=node_markseen_failed`）。

## 范围

### IN

- 修改 `internal/adapter/httpapi/cascade.go` 4 个 handler（Create / BulkDelete / DispositionCreate / DispositionBulkDelete）的 Bind + store 错误处理。
- 修改 `internal/adapter/httpapi/system.go` UnRegister / Keepalive 的 `_ = ` 模式，注入 `*slog.Logger` 后打 Warn 日志。
- 修改 `internal/app/logging/logger.go` 的 `dailyWriter.maybeRotate`，轮转失败打 Error 日志（通过外部注入 logger，或经由 error 传递）。
- 修改 `internal/adapter/storage/capture_reader.go` 的反序列化静默吞错，改为日志记录失败（注入 logger）。
- 同步在 `adapter-httpapi/spec.md` / `logging/spec.md` / `web-bff/spec.md` 中**新增**两个 Requirement：协议端 Body 解析错误契约、抓包反序列化失败可观测性。
- 给上述 4 个文件加单测覆盖新错误路径（target ≥ 80% line for touched files）。

### OUT

- 不动 `client.go:134, 155`、`ws.go:118–152`、`storage.go:50` 等连接关闭 / shutdown 路径的 `_ = `，那些是带外资源、与业务契约无关。
- 不动 `services.go:301`（`engine.Stop` 在 RemoveNode 失败后的兜底），那是编排逻辑。
- 不改协议语义、HTTP 状态码语义（除 400 之外）、Digest 行为、Cascade 的成功/失败响应格式。
- 不重构 `subscribeRepo.store` 返回值语义（仍是 `(id, bool)`，但调用方升级）。

## 影响

- 行为变更：畸形 JSON 现在返回 400 而不是 200 + 空对象。
- 日志：UnRegister / MarkSeen / 轮转 / 抓包反序列化失败将可见（Info/Warn/Error），不会膨胀 INFO 量（仅在出错时出现）。
- 性能：每个 Bind 增加一次 error 检查，< 1µs，可忽略。
- 二进制：无变化（无新依赖）。
- CI：受影响的单测增加 ~6 个 case，应在 < 5s 内跑完。
