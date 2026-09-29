# 变更日志

> 本文件镜像 [openspec/CHANGELOG.md](../openspec/CHANGELOG.md)。
> 每次发布后请执行 `make sync-changelog` 重新生成此文件。

---

## [未发布]

### fix(nonce) — 启用 Nonce 重放检测

- `internal/adapter/httpapi/system.go` 中 `verifyAuthorization` 实现真实 Digest 头解析，调用 `NonceStore.Consume(nonce)` 强制单次消费
- `internal/adapter/storage/nonce_store.go` 中 `Consume` 的 WHERE 条件加入 `used_count = 0` 约束，实现严格一次性使用语义
- 新增 `TestNonceReplayRejected` 单测；新增 `test/contract/golden/replay.json` 黄金样本

### feat(engine) — 拆分 Engine.AutoStart 与 Engine.Start

- `internal/adapter/scenario/engine.go` 新增 `Engine.AutoStart(ctx, s)`，仅在 `Scenario.Schedule.AutoStart=true` 时启动
- `Engine.Start` 语义不变，保持向后兼容
- `cmd/gat1400-sim/main.go` 切换为 `AutoStart`，简化启动循环

### feat(cascade) — Subscribe / Disposition 双形态删除

- `internal/adapter/httpapi/cascade.go` 中 `handleSubscribeCreate` / `handleDispositionCreate` 支持 body 删除分支
- 通过 `extractDeleteIDs(body, listKey)` 统一处理 `SubscribeIDList` / `DispositionIDList` 与嵌套 `DeleteOperate` 两种 body 形态
- 删除优先级高于创建路径，防止合法创建 body 被误判为删除

### docs — README/PROTOCOL/ARCHITECTURE 同步

- `README.md` — `internal/adapter/wire/` 文件清单修正为 `client.go`+`uac.go`+`json.go`；`openspec/specs/` 子目录按 10 个 capability 重组
- `docs/PROTOCOL.md §6` — 增加 Nonce 重放响应示例（401 + 新 nonce + stale=false）
- `docs/PROTOCOL.md §4.3` — Cascade 路由表增加"双形态支持"列
- `docs/ARCHITECTURE.md §七` — 关键技术决策表补齐 OutboundDispatcher 与 Engine.AutoStart

---

## v0.1.0 — 2026-09-28

### 新增

- **bootstrap-scaffold**：项目骨架，含 Go 1.21+ / Makefile / GitHub Actions / golangci-lint / `modernc.org/sqlite`
- **domain-models**：`Node`/`Role`/`Capability`/`Status`、`Resource`/`Kind`、`Subscription`/`Disposition`、`Scenario`、`ResponseStatus`/`Code`、`IDGenerator`（20 位 DeviceID）
- **adapter-httpapi**：完整 GA/T 1400.4 REST API —— System / Collection / Cascade / Catalog 四类路由、Digest 中间件、User-Identify 中间件、Capture 中间件、自定义 VIID+JSON Binder
- **adapter-wire**：HTTP 客户端，支持 RFC 2617 Digest 自动重试、SQLite nonce 持久化、User-Identify 头注入
- **scenario-engine**：YAML 场景加载、`ScenarioEngine.Start/Stop`、`FakeFactory`（随机数据）、`ProbabilityFaultInjector`（delay/drop/reorder/malformed）
- **web-control-plane**：Echo BFF（`:19000`），含 REST API + WebSocket Hub，Vue3 + ElementPlus SPA 通过 `embed.FS` 内嵌
- **testing-and-docs**：internal 包 100% 单元测试覆盖、黄金样本测试、真实 TCP socket 的 e2e 测试、6 份文档

### 功能列表

| 功能 | 细节 |
|------|------|
| 协议服务端 | `:19001` —— 4 类路由（System / Collection / Cascade / Catalog） |
| BFF | `:19000` —— 6 个页面：Dashboard / Nodes / Scenarios / Resources / Subscriptions / Captures |
| Digest 认证 | RFC 2617 qop=auth，SQLite nonce 重放保护 |
| 抓包 | 每请求记录：NodeID / Method / Path / Status / Header / Body |
| 场景 | YAML 描述拓扑：nodes / resources / subscriptions / faults |
| 异常注入 | `delay` / `drop` / `reorder` / `malformed`，按概率触发 |
| WebSocket | 实时事件：node.status / captures / scenario.state |

### 修复

- `randomHex` nonce 生成：改用 `crypto/rand`（旧版本曾误用 `time.Now`）
- capture 测试中的 `defer cr.Close()`：改为 `defer cs.Close()`
- `DigestAuth` 中间件：在所有 System 路由上正确识别 `Authorization` 头存在与否

### 文档

- `docs/ARCHITECTURE.md` —— 组件拓扑图、数据流、分层纪律
- `docs/PROTOCOL.md` —— 完整路由参考、请求 / 响应、错误码
- `docs/USER_GUIDE.md` —— 快速开始、配置、场景 YAML、Web BFF API
- `docs/OPERATIONS.md` —— 部署、TLS、systemd、监控、备份、升级
- `docs/TESTING.md` —— 运行测试、黄金样本、e2e、覆盖率、CI
- `docs/CHANGELOG.md` —— 本文件（镜像 openspec/CHANGELOG.md）

### OpenSpec

全部 7 个 change 已在 `openspec/changes/archive/` 归档，每个目录包含 `proposal.md` / `design.md` / `specs.md` / `tasks.md`。详见 `openspec/README.md`。