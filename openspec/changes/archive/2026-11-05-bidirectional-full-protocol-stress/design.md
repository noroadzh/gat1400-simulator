# Design

## Context

See proposal.md — Why。现状：`wire.Client` 仅有 PostJSON/GetJSON；`uac.go` 覆盖 System + 部分 Cascade；`OutboundDispatcher` 同步偏窄；UAS `httpapi` 对 VIID/Catalog/Cascade/VIIDServers/VIAS 已较全，是压测入站侧主要依赖。字段与路径以 `docs/02-protocol/PROTOCOL.md` 与 `test/contract/golden/` 为准。

## Goals / Non-Goals

**Goals:**
- UAC 高层 API 覆盖双向压测所需主路径（Catalog / 12 Kind 集合 / Cascade 补全 / VIAS 子集）。
- 场景引擎可编排压测流量；回环 e2e 验收；样例不对外部 autoStart。
- 分层单向依赖不变：高层 wire → Client.do；scenario → wire。

**Non-Goals:**
- 不实现完整 GA/T 1400 全部可选附属接口或多媒体流媒体协议。
- 不内置分布式压测集群 / Prometheus 导出（可后续）；本变更以日志/计数/事件为最小可观测。
- 不改 Digest 算法与默认开发凭据策略。
- 不强制 Web UI 完整压测控制台（BFF 触发可选，非阻塞）。

## Decisions

1. **按子类型扩展 wire，而非巨型 Client 方法集**  
   新增 `Catalog` / `Collection` / `VIAS`，Cascade 补方法；与现有 `System`/`Cascade` 一致。  
   *备选*：仅暴露通用 `Request(method, path, body)` — 拒绝，因丢失协议包装与 Kind 映射约束。

2. **底层先补 PutJSON/DeleteJSON**  
   所有写/删统一走 `do()`，避免各子类型复制 Digest。Delete 默认无 body；若某路径需要 body，提供可选 overload 或 `DeleteJSONWithBody`。

3. **Collection Kind 映射复用 domain `CollectionOf`/`Kind`**  
   wire 可依赖 domain 常量（或薄映射表）；禁止硬编码半套集合名。12 Kind 增删仍须同步既有约束。

4. **Dispatcher 增量方法 + 场景 YAML 显式动作**  
   引擎对压测场景按 YAML `resources`/`actions`（或现有 Rate 推送扩展）调用新 Dispatch*；Catalog/DispositionNotification 用独立 goroutine 或与资源推送同周期，避免隐式魔法。

5. **真实对端与回环分流**  
   - CI/e2e：localhost 回环。  
   - 外场：`configs/` 或 env `GAT1400_STRESS_UPSTREAM` 注入；样例 YAML 占位 `http://127.0.0.1:14000`。

6. **可观测性最小集**  
   slog 计数字段（ok/fail）+ 可选 WS 事件；不做独立 metrics 服务。

## Risks / Trade-offs

- [真实平台方言差异] → Mitigation：客户端保持黄金样本严格；差异用场景 `Exceptions` / 文档记录，不 silently 放宽协议。  
- [高压打满本机 UAS SQLite] → Mitigation：压测样例默认低 Rate；文档注明可切内存或调 page size。  
- [VIAS 对端缺失] → Mitigation：VIAS 客户端与 e2e 可 build-tag 或跳过；本机 VIAS 路由已存在则回环覆盖。  
- [API 表面积膨胀] → Mitigation：子类型边界清晰；未用方法不进 BFF。

## Migration Plan

1. 合并后旧场景 YAML 行为不变（仅新增方法与样例）。  
2. 文档增加双向压测拓扑与验收清单。  
3. 回滚：移除新样例与客户端文件即可；无 DB schema 强制迁移（沿用 `viid_object`）。

## Open Questions

- 外场压测是否需要 BFF「一键启动」按钮：本变更不阻塞，可二期。  
- Disposition 删除优先 DeleteOperate POST 还是 DELETE by id：实现时与 UAS 已注册路由对齐，两者若皆存在则客户端提供明确命名方法。
