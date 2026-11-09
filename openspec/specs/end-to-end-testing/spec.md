# end-to-end-testing — 主规格

> 合并来源：`changes/refine-test-coverage/specs/end-to-end-testing/spec.md`

## Purpose

跨包端到端（`test/e2e/`）、契约回放（`test/contract/`）与多包协作集成测试的独立线索；并约束编排核心 `Engine` 的单元测试覆盖。

## Requirements

### Requirement: 端到端 / 契约 / 集成测试 MUST 单独成线

跨包端到端(`test/e2e/`)、契约回放(`test/contract/`)、以及涉及多包协作的集成用例，均 MUST 归属 `end-to-end-testing` capability 下的 spec，而不是分散到各 adapter 的 capability。该 capability 的 spec 描述以下事项：

- 跨包测试入口（如 `test/e2e/`、`test/contract/golden_test.go`）
- 契约回放的 fixture 格式与版本演进策略
- 哪些场景是端到端、哪些是单包集成的分类规则
- 端到端测试在 CI 中的运行时机（每 PR 必跑）

#### Scenario: 端到端测试归属明确

- **WHEN** 一个测试用例涉及 ≥ 2 个 `internal/` 包的协作
- **THEN** 该用例的目录归属 MUST 由 `end-to-end-testing` capability 决定（避免散落在各 adapter capability）
- **AND** 端到端测试 MUST 通过 `-coverpkg=./internal/...,./cmd/...` 纳入覆盖率统计

#### Scenario: 契约 fixture 演进

- **WHEN** `test/contract/golden/` 下某 fixture 需要更新（协议行为变更）
- **THEN** 该更新 MUST 走 PR review，PR 描述 MUST 说明协议变更来源（GA/T 1400.x 修订、Issue 编号、还是回归修复）
- **AND** commit message MUST 包含 `contract:` 前缀

### Requirement: 编排核心 Engine 单独测试覆盖

`internal/adapter/scenario.Engine`（节点生命周期 + AutoStart + 事件回调）MUST 拥有独立单元测试，覆盖率 ≥ 80%。

覆盖 MUST 至少包括：

- `Start` / `Stop` 在节点已存在 / 不存在 / 已停止 / 启动失败等状态下的行为
- `AutoStart` 相关副作用路径
- 事件回调在 Engine 关闭后 MUST NOT 触发副作用
- 并发 Start/Stop 同一节点 ID 的竞态（`-race` 通过）

#### Scenario: Engine Start/Stop 完整路径覆盖

- **WHEN** 测试以 `Engine` 为 SUT 跑生命周期分支
- **THEN** `go test -coverpkg=./internal/adapter/scenario ./internal/adapter/scenario` 输出 MUST ≥ 80%
- **AND** 测试 MUST 包含至少 1 个并发竞态相关用例或 `-race` 下通过的生命周期用例

#### Scenario: Engine 关闭后回调不触发

- **WHEN** `Engine.Stop(id)` 已返回
- **THEN** 任何先前注册的回调在事件到达时 MUST NOT 产生副作用
- **AND** 该行为 MUST 由测试断言覆盖（或等价回归路径）

### Requirement: 双向协议压测 MUST 有真实 TCP e2e

`test/e2e/` MUST 包含至少一条用例：本进程 UAS 监听真实端口，UAC（wire 客户端）经真实 TCP 完成 Register → 集合推送 → Catalog 上报 → DispositionNotification 主路径；禁止 mock `http.Handler` 替代 socket。

#### Scenario: 回环全协议主路径 e2e

- **WHEN** 运行该 e2e（空闲端口）
- **THEN** 上述各步骤 MUST 返回协议成功
- **AND** UAS 侧 MUST 可查询到对应持久化或内存对象

### Requirement: 契约黄金样本 MUST 覆盖新增 UAC 请求形

对新增高层客户端构造的代表性请求/响应（至少 Collection 分页列表与 DispositionNotification POST）MUST 有 `test/contract/golden/` 样本或等价契约断言。

#### Scenario: DispositionNotification 契约

- **WHEN** 契约测试回放 DispositionNotification 推送样本
- **THEN** 路径、Content-Type 与关键包装字段 MUST 与 PROTOCOL 一致
