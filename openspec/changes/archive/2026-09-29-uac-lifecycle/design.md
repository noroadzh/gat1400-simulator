# Design — uac-lifecycle

## Context

GA/T 1400 模拟器的服务端（UAS）接口已完整实现，`wire.Client` 提供底层 `PostJSON/GetJSON`，场景引擎的 `OutboundDispatcher` 通过它们 POST 资源到 Collection URI。但以下能力缺失：

1. `wire.Client` 无高层 System/Cascade API，`Client.Register/UnRegister/Keepalive/Subscribe/Disposition` 无法直接调用
2. `Engine.Start` 的 `materialise` 只创建节点，未触发 HTTP Register
3. `Engine.Stop` 只 cancel goroutine，未触发 HTTP UnRegister
4. `postNotification` 是 stub，不实际 POST
5. 心跳周期硬编码为 30s，无法通过配置调参

详见 `proposal.md` - Why。

## Goals / Non-Goals

**Goals:**
- 为 `wire.Client` 增加 `System`（Register/UnRegister/Keepalive/ServerTime）和 `Cascade`（SubscribeCreate/Delete/List/DispositionCreate/SubscribeNotification）高层子类型，内部通过既有 `PostJSON/GetJSON` 实现
- 将 `OutboundDispatcher` 从只支持 Collection 类扩展到 System 类与 Cascade 类
- 将场景引擎改造为完整节点生命周期：materialise 触发 Register → 周期 Keepalive → Stop 触发 UnRegister
- `postNotification` 变活，真正 POST 到订阅端点
- 心跳周期与 jitter 可通过配置文件调整

**Non-Goals:**
- 不实现 GB/T 28181 的 SIP 信令（那是另一个 simulator 的职责）
- 不改变资源分发逻辑（ResourceSpec pacing 行为不变）
- 不实现定时任务或 cron 式调度（Schedule.Duration 已在现有设计中支持）
- 不重构 `Engine` 的 `running` map 锁粒度（当前互斥已够用）

## Decisions

### Decision: wire 子类型用 Go struct 而非 interface

`System` 和 `Cascade` 作为 `Client` 的持有字段（struct），而非 `Dispatcher` 那样的 interface。

**Rationale：** `System` 和 `Cascade` 依赖 `Client` 的私密字段（`nonce`、`username`、`password`），无法独立构造。struct 字段天然满足这一约束，无需 interface 的全部动态性。

**Alternative：** interface `Systemer { Register(...); UnRegister(...) }`。引入 interface 的额外间接寻址在高频心跳路径（30s/次）下得不偿失。Go 惯用法中，"组合而非继承"的 struct embedding 更适合此场景。

### Decision: keepalive goroutine 按节点 ID 独立，而非全局一个

每个 device 节点对应一个独立 goroutine，周期调用 `DispatchKeepalive`。goroutine 通过 `node.ID` 闭包捕获目标，context cancel 时自然退出。

**Rationale：** 各节点可配置不同的心跳周期（未来可扩展 `NodeSpec.KeepaliveInterval`）。独立 goroutine 使各节点心跳互不干扰。

**Alternative：** 全局一个 ticker，对所有节点批量发送心跳。缺点是节点增减需要重新构建 ticker，且无法独立退出。

### Decision: Engine.dispatcher 字段改为 `*OutboundDispatcher`

当前 `Engine.dispatcher` 是 `Dispatcher` 接口（只有 `Dispatch`）。生命周期改造需要 `DispatchRegister/Keepalive/Unregister` 等新方法。

**Rationale：** 新方法只属于 `*OutboundDispatcher`，不在 `Dispatcher` 接口中。若维持接口，调用方需要 type-assert 或在接口追加方法。前者破坏类型安全，后者污染所有 fake 实现。

**Alternative：** 在 `Dispatcher` 接口追加新方法，修改所有 fake 实现。成本高，且 engine 层本就不需要 fake——测试时 `OutboundDispatcher` 直接用 httptest 服务器即可。

**Trade-off：** engine 无法再被 `fakeDispatcher` 替换。但 engine 的核心逻辑（pacing、subscription、schedule）不依赖 dispatcher 的具体实现，这些路径已有单元测试覆盖。

### Decision: Dispatcher.record() 返回 void

抓包记录（recorder.Record）不返回 error，直接忽略记录失败。

**Rationale：** 抓包是诊断辅助功能，不应因存储失败污染主业务流程。返回 void 也避免了 `Recorder` 实现者的负担。

### Decision: keepaliveInterval=0 禁用心跳

当配置为 0 时，`Engine.run` 不为任何节点启动 keepalive goroutine。

**Rationale：** 兼容"只推送资源不维护心跳"的场景（如测试时快速验证推送路径）。无需引入单独的"心跳开关"配置项。

## Risks / Trade-offs

- **[Risk] Stop 阻塞 5s 硬超时** → 已在当前设计中存在（`select { case <-h.done; case <-time.After(5s) }`），UnRegister 发起的 goroutine 即使在 5s 后也会继续运行。**无新增风险。**

- **[Risk] DispatchRegister 失败导致场景无法运行** → 若 Register 被 DropRate 跳过，节点状态保持 unknown，仪表盘显示正常（灰显），心跳继续运行。**可接受。**

- **[Risk] HTTP POST 超时导致 keepalive goroutine 泄漏 goroutine** → `DispatchKeepalive` 有 5s context timeout，超时后 goroutine 记录日志并进入下一周期。**可接受。**

- **[Risk] RegisterDropRate 判定与 materialise 并发** → `materialise` 在 `Engine.mu` 锁外执行，Register 异步发起。节点已在 `provisioner.UpsertNode` 中创建，无需等待 Register 成功。**设计一致。**

## Migration Plan

1. **单包增量**：所有改动在 `internal/adapter/wire/`、`internal/adapter/scenario/`、`internal/app/config/`、`cmd/gat1400-sim/` 内，对外 API（HTTP API、Web UI）完全不受影响。
2. **配置向后兼容**：`keepaliveInterval` 有默认值 30s，YAML 中省略时行为不变。
3. **部署**：直接替换二进制，`configs/default.yaml` 新增一行 `keepaliveInterval: "30s"`（或保持省略使用默认值）。
4. **回滚**：替换旧二进制即可，无需数据迁移。

## Open Questions

无。所有设计决策在现有技术约束内可确定，无需延后。