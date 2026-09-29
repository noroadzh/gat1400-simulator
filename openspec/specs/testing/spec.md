# 测试 — 主规格

> 合并来源：`archive/testing-and-docs/specs.md`

## Purpose

定义测试策略与质量门槛：黄金样本覆盖 4 类路由、真实 TCP socket E2E、端到端正确性断言、黄金样本失配报告失败、internal 包 ≥ 90% 覆盖率、文档与代码同步。

## Requirements

### Requirement: 黄金样本 MUST 覆盖全部 4 类路由

黄金样本集 MUST 至少为以下每类包含一个用例：
- System 路由（Register、Keepalive、Time）
- Collection 路由（至少 Person POST + GET）
- Cascade 路由（至少 Subscribe POST + GET）
- Catalog 路由（至少 APE GET）

#### Scenario: 缺少 APE Catalog 样本

- **WHEN** 测试套件以 `--golden` 标志运行
- **THEN** 每个 `test/contract/golden/*.json` 中含 catalog 步骤的样本 MUST 通过
- **AND** 缺失的 catalog 样本 MUST NOT 让测试静默跳过。

### Requirement: 黄金样本 MUST 覆盖 Nonce 重放检测用例

`test/contract/golden/` 目录 MUST 至少包含一个 replay.json 黄金样本，覆盖 Digest 重放场景：两次 Register 使用同一 nonce+nc，第二次 MUST 被服务端 401 拒绝。

#### Scenario: replay.json 样本存在

- **WHEN** 测试套件以 `--golden` 标志运行
- **THEN** `test/contract/golden/replay.json` MUST 存在
- **AND** 测试运行器 MUST 加载并验证该样本，断言第二次响应状态码为 401。

### Requirement: E2E 测试 MUST 使用真实 TCP socket

E2E 测试 MUST NOT mock HTTP 层。服务端 MUST 通过 `net.Listen` 或 `httptest.Server` 绑定真实端口。客户端 MUST 通过真实 TCP 连接。

#### Scenario: 真实 socket 往返

- **WHEN** `TestProtocolE2E_RegisterAndPush` 运行
- **THEN** 客户端 MUST 发送 TCP SYN 到服务端
- **AND** 响应 MUST 沿同一 TCP 连接返回
- **AND** MUST NOT 使用任何 mock 或假 HTTP transport。

### Requirement: E2E 测试 MUST 验证端到端正确性

每个 E2E 测试 MUST 在 HTTP 状态之外至少断言一项可观测效应：
- 服务端内存状态（ResourceStore、NodeService）反映客户端请求
- CaptureStore 含每次请求的记录
- Register 后 NodeService 报告节点为 online

#### Scenario: Register 后 NodeService 状态验证

- **WHEN** E2E 测试完成 Register 步骤
- **THEN** `NodeService` MUST 报告对应节点状态为 `online`。

### Requirement: 黄金样本运行器 MUST 报告失配为失败

加载并回放黄金样本时，任何响应状态码或关键 Body 字段的差异 MUST 导致 `t.Fatal`。

#### Scenario: 状态码失配

- **WHEN** 黄金样本期望 200，但服务端返回 401
- **THEN** 测试 MUST 失败并打印失配描述。

### Requirement: 全部 internal 包 MUST ≥ 90% 测试覆盖率

`internal/` 下每个包 MUST 达到至少 90% 行覆盖率（`go test -cover` 输出）。覆盖率报告 MUST 在 CI 中发布。

#### Scenario: 覆盖率报告在 CI 发布

- **WHEN** CI 执行 `go test -cover` 后
- **THEN** 覆盖率报告 MUST 出现在 CI 输出中
- **AND** 任何包低于 90% MUST 导致 CI 失败。

### Requirement: 文档 MUST 与代码同步

任何引入新路由、新配置项、新 YAML 字段的变更，对应文档 MUST 在同一变更中更新。

#### Scenario: 新路由后文档同步

- **WHEN** 代码引入新 VIID 路由 `/VIID/NewEndpoint`
- **THEN** 对应文档 MUST 在同一 change 中更新
- **AND** 不得存在路由有代码无文档。

---

## ADDED Architecture Decisions

### Decision: 黄金样本以 JSON 入库

黄金样本是纯 JSON 文件，纳入版本控制。便于评审、diff、协议语义变更时更新。

### Decision: E2E 测试位于 test/e2e/

E2E 测试在顶层 `test/` 目录，而非 `internal/`，因为它们跨越多层，不应受 `internal/` 包边界约束。

### Decision: 文档存放于 docs/

文档由 OpenSpec 配置生成，存放在 `docs/`，作为项目 README 的一部分展示。实现细节保留在源码注释中。