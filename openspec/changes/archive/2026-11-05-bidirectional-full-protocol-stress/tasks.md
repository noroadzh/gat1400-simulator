# Tasks

## 1. wire 底层与 Cascade 补全

- [x] 1.1 在 `internal/adapter/wire/client.go` 实现 `PutJSON` / `DeleteJSON`（复用 `do`），并补充单元测试验证 Digest 401 重试与 Content-Type
- [x] 1.2 扩展 `Cascade`：DispositionList / DispositionDelete / DispositionNotificationPush / DispositionNotificationList，单测覆盖路径与包装字段
- [x] 1.3 更新 `adapter-wire` / `wire-cascade-client` 相关注释与 `go test ./internal/adapter/wire/...` 通过

## 2. Catalog / Collection / VIAS 客户端

- [x] 2.1 新增 `Catalog` 子类型（APEs/APSs/Tollgates/Lanes/VIIDServers），单测用 httptest 假对端校验路径
- [x] 2.2 新增 `Collection` 子类型（12 Kind POST/GET/PUT/DELETE + 分页列表），Kind 映射对齐 domain；`go test` 通过
- [x] 2.3 新增 `VIAS` 子类型（System + Tasks + TaskControls + AnalysisRules + Capability GET），单测路径断言通过

## 3. 场景引擎与压测样例

- [x] 3.1 扩展 `OutboundDispatcher`：DispositionNotification / CatalogUpsert / CollectionPush /（可选）VIASTaskCreate，dispatcher 单测路径断言通过
- [x] 3.2 引擎或 YAML 动作接线，使压测场景能按 Rate 推送集合与可选 Catalog/通知；相关 engine 测试或新测通过
- [x] 3.3 新增 `configs/scenarios/` 双向压测样例（AutoStart=false，localhost 占位），`Loader` 可成功解析

## 4. e2e / 契约 / 文档

- [x] 4.1 新增真实 TCP 回环 e2e：Register → Collection 推送 → Catalog 上报 → DispositionNotification；`make e2e` 相关用例通过
- [x] 4.2 补充契约黄金样本或断言（至少分页列表与 DispositionNotification）；`go test ./test/contract/...` 通过
- [x] 4.3 更新 `docs/` 双向压测拓扑与验收清单，并与 PROTOCOL 交叉引用无冲突

## 5. 集成验收

- [x] 5.1 `make test` 与 `make e2e` 全绿；抽查 UAC 新 API 对本地 `:14000` 手工或脚本冒烟成功

## Workflow follow-up

- 实现完成后按项目流程 archive `bidirectional-full-protocol-stress`
- 外场对接真实平台时仅改配置/env，不改默认样例密钥
