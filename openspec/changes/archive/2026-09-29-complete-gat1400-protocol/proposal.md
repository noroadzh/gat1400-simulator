# Proposal

## Why

在最近一次接口实现完整度盘查中发现，模拟器在以下 4 个方面偏离已发布的 GA/T 1400.4 规范与本仓库的 capability specs：

1. **`verifyAuthorization` 是空桩** —— 服务端 `digestAuth` 中间件接受任意 Authorization 头，`NonceStore.Consume` 从未被调用，nonce 重放保护形同虚设。docs/PROTOCOL.md §6 明确声明"服务端按 `(nonce, nc)` 联合主键去重"，但实现未兑现。
2. **`Engine.AutoStart` 方法缺失** —— `scenario/spec.md` 描述了 `AutoStart(ctx)` 方法用于"按 `ScheduleSpec.AutoStart` 字段决定是否启动"，当前实现只暴露 `Start()`，行为正确但 API 形状不一致。
4. **Cascade 路由存在双形态差异** —— spec 描述 SubscribeDelete 通过 `POST /VIID/Subscribes` body 含 `DeleteOperate` 实现，而实现仅支持 `DELETE /VIID/Subscribes/:id`；同样地，Disposition 批量删除 spec 描述为 `POST /VIID/Dispositions` 含 `DeleteOperate`，实现仅 `DELETE /VIID/Dispositions`。
4. **文档与代码注释稀疏** —— 多个核心文件（system.go、cascade.go、nonce_store.go 等）缺少必要的协议字段、HTTP 头、URI 路径的语义注释；README.md 中关于 `internal/` 目录树的描述与最新代码不一致（adapter-wire 文件名 `digest.go` 不存在，实际是 `uac.go` 和 `client.go`）。

这个变更要把上述 4 个差距一次性收敛：补齐 nonce 重放检测、对齐 autoStart API、补齐 Cascade 双形态路由、补齐代码注释并同步 README/docs/CHANGELOG，使模拟器与 GA/T 1400.4 标准完全对齐，且 README/docs 与代码现状同步。

## What Changes

- **修复 Nonce 重放检测**：`httpapi.verifyAuthorization` 实现真正的 Digest 校验——解析 Authorization 头，调用 `NonceStore.Consume(nonce)` 标记 nonce 被消费；当 nonce 已过期或重复使用返回 401。
- **新增 `Engine.AutoStart` 方法**：根据 `Scenario.Schedule.AutoStart` 字段决定是否调用 `Start`，并保留 `Start` 方法作为无条件启动入口。
- **Cascade 双形态支持**：`POST /VIID/Subscribes` 与 `POST /VIID/Dispositions` 在 body 含 `DeleteOperate` 时按 ID 列表删除（与 `DELETE /:id` 等价）；保留 `DELETE /:id` 端点。
- **代码注释强化**：在 `system.go`、`cascade.go`、`collection.go`、`nonce_store.go`、`client.go` 等核心文件补齐协议字段注释（GA/T 1400.4 §X 节号）、URI 模板、HTTP 头语义、关键不变量的来源。
- **README 同步**：修正 `internal/adapter/wire/` 文件清单（移除不存在的 `digest.go`，改为 `client.go`+`uac.go`+`json.go`）；`openspec/specs/` 子目录结构按 10 个 capability 重新描述；新增 CHANGELOG 条目。
- **docs 同步**：更新 `docs/PROTOCOL.md` §6 增加 Nonce 重放响应字段示例；更新 §4.3 Cascade 路由表格补充双形态说明；更新 `docs/ARCHITECTURE.md` 中 §三适配层描述补齐 OutboundDispatcher 与 engine.AutoStart。

## Capabilities

### Modified Capabilities
- `adapter-httpapi`: 补齐 verifyAuthorization 真实实现、Cascade 双形态路由、System/Collection 端点注释
- `adapter-wire`: 补齐 client.go 与 uac.go 关键路径注释
- `scenario`: 拆分 Engine.AutoStart 与 Engine.Start 语义
- `config-keepalive`: 文档与注释同步（无行为变更）
- `domain`: NonceStore.Consume 文档注释补齐
- `testing`: 黄金样本增加 nonce 重放检测用例

## Impact

- **代码影响**：
  - `internal/adapter/httpapi/system.go` — verifyAuthorization 实现
  - `internal/adapter/httpapi/cascade.go` — SubscribeDelete/DispositionDelete 双形态路由
  - `internal/adapter/scenario/engine.go` — 新增 AutoStart 方法
  - 注释补齐：system.go、collection.go、cascade.go、catalog.go、client.go、uac.go、nonce_store.go
- **测试影响**：
  - `internal/adapter/httpapi/*_test.go` — 增加 nonce 重放检测测试
  - `test/contract/golden/` — 增加 replay.json 黄金样本
- **文档影响**：
  - `README.md` — 修正 internal/ 目录树、openspec/specs/ 描述
  - `docs/PROTOCOL.md` — §6 Nonce 重放响应字段、§4.3 Cascade 双形态
  - `docs/ARCHITECTURE.md` — 适配层描述补齐
  - `CHANGELOG.md` — 新增 entry
- **兼容性**：所有修改向后兼容（仅补齐实现 + 增加新方法）。`DELETE /VIID/Subscribes/:id` 等已部署端点保持不变。
- **依赖**：无新增依赖，使用既有的 `crypto/subtle`、`modernc.org/sqlite`。