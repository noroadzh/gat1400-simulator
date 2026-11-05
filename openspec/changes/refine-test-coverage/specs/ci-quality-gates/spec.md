# Delta: ci-quality-gates

## ADDED Requirements

### Requirement: `.golangci.yml` MUST 提交并定义最小规则集

仓库根目录 MUST 存在 `.golangci.yml`（golangci-lint v2 格式），至少启用以下 linter：`revive`、`gocritic`、`unparam`、`gocyclo`。配置文件变更 MUST 与 CI 工作流中 `golangci-lint-action` 的 `version` 字段在同一变更中提交。

#### Scenario: `.golangci.yml` 缺失

- **WHEN** 仓库根目录不存在 `.golangci.yml`
- **THEN** 维护者 MUST 在补齐 CI 静态门禁的同一变更中提交该文件
- **AND** 该文件 MUST 至少包含 `revive` `gocritic` `unparam` `gocyclo` 四个 linter 的启用项

#### Scenario: 规则集需随 Go 版本升级时同步升级

- **WHEN** `go.mod` 中 `go X.Y` 升级
- **THEN** 若 `.golangci.yml` 中的 v1 格式键（如 `run.deadline`、`output.formats`）与 golangci-lint v2 不兼容
- **AND** 维护者 MUST 在同一变更中迁移到 v2 格式

### Requirement: 覆盖率统计 MUST 使用 `-coverpkg=./internal/...`

CI 的 `go test` 步骤 MUST 传 `-coverpkg=./internal/...` 标志，覆盖率阈值（若有） MUST 针对整体 `internal/...` 包计算，而非各包自测的子集。`test/contract` 与 `test/e2e` 目录的测试代码 MUST 计入覆盖率统计。

#### Scenario: 默认覆盖率模式

- **WHEN** `go test ./...` 不带 `-coverpkg`
- **THEN** 实际覆盖率仅统计 `cmd/gat1400-sim` 与 `internal/...` 中被 `internal/...` 测试调用到的语句
- **AND** `test/contract/` 与 `test/e2e/` 目录因 import 路径不匹配可能显示 `[no statements]`，从而被误判为 0% 覆盖

#### Scenario: 修复后

- **WHEN** `go test ./... -coverpkg=./internal/...`
- **THEN** 端到端与契约测试中触达的生产代码 MUST 计入覆盖率
- **AND** 报告 MUST 不再出现 `[no statements]` 条目