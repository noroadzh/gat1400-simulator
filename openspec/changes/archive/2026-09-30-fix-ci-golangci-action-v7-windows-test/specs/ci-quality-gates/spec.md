# Spec Delta

## MODIFIED Requirements

### Requirement: 静态检查工具 MUST 与 `go.mod` 声明的 Go 版本兼容

CI 工作流中拉取的 golangci-lint 预编译二进制，其「构建所用 Go 版本」MUST ≥ `go.mod` 中声明的 Go 版本。否则 lint job 在加载 module 阶段直接崩溃并退出码非 0，Lint job MUST 由成功掩盖。

#### Scenario: linter 构建版本早于 module 版本时 Lint job 失败

- **WHEN** CI 拉取的 `golangci-lint` 二进制由 go1.X 预编译，而 `go.mod` 声明 `go 1.Y` 且 `Y > X`
- **THEN** Lint job MUST 在日志中输出 `can't load config: ... Go language version (go1.X) ... lower than the targeted Go version (1.Y)`
- **AND** Lint job MUST 退出码非 0
- **AND** 维护者 MUST 将 linter 升级到由 go ≥ 1.Y 编译的版本，直到退出码 0

#### Scenario: linter 与 module 版本一致时 Lint job 真正执行检查

- **WHEN** linter 构建版本 ≥ module 声明版本
- **THEN** Lint job MUST 真实执行所有默认启用的 linter（errcheck、govet、ineffassign、staticcheck、unused）
- **AND** Lint job 的退出码 MUST 由 issue 数量决定，与平台缓存/Node 弃用提示无关

#### Scenario: golangci-lint v2 与 action v6 不兼容时 Lint job 失败

- **WHEN** 工作流的 `golangci-lint-action` 固定在 `@v6` 系列，而 `version` 字段填写 golangci-lint v2 版本字符串
- **THEN** Lint job MUST 在 prepare environment 阶段失败，MUST NOT 进入 linter 实际执行
- **AND** 日志 MUST 输出 `golangci-lint v2 is not supported by golangci-lint-action v6` 之类的版本兼容性拒绝信息
- **AND** 维护者 MUST 把 `golangci-lint-action` 升级到支持 v2 的版本（`@v7` 及以上）

#### Scenario: action 与 linter 主版本配对一致时 Lint job 正常执行

- **WHEN** `golangci-lint-action` 固定在支持 golangci-lint v2 的版本（`@v7` 及以上），且 `version` 字段为 v2.x 版本
- **THEN** Lint job MUST 越过 prepare environment 阶段并真正下载、执行 linter
- **AND** Lint job 的退出码 MUST 仅由 linter 的 issue 数量决定，MUST NOT 受 action 自身版本校验影响