# ci-quality-gates Specification

## Purpose

定义持续集成中静态质量门禁的最低契约：CI Lint job 必须真正执行静态检查、其使用的 lint 工具必须与 `go.mod` 声明的 Go 版本兼容，并且 Lint job 不得被无关平台噪声（缓存 400、Node 弃用提示）误标为失败。

## Requirements

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

### Requirement: Lint job MUST 在零静态检查 issue 时通过

CI Lint job MUST 在仓库代码无 lint issue 时退出码 0，且 MUST NOT 因缓存服务临时不可用、Node 版本弃用提示等平台级噪声误判失败。

#### Scenario: 仓库无 issue 时 Lint job 通过

- **WHEN** 仓库全部 `*.go` 文件经过兼容 linter 检查后报告 0 issue
- **THEN** Lint job MUST 退出码 0
- **AND** Frontend / Test / Build 三个 job 的状态 MUST NOT 反向影响 Lint job 的成功判定

#### Scenario: 仓库存在 issue 时 Lint job 失败

- **WHEN** linter 报告至少 1 条 issue
- **THEN** Lint job MUST 退出码非 0
- **AND** 失败日志 MUST 包含 issue 的 `文件:行号` 与 rule id，便于定位

### Requirement: 静态检查豁免 MUST 显式标注

任何被静态检查豁免（`//nolint:<rule>`、配置文件 skip 等）的代码 MUST 在就近注释中说明豁免理由，禁止无理由抑制。

#### Scenario: 豁免带理由

- **WHEN** 代码使用 `//nolint:errcheck`、`//nolint:unused` 等注释
- **THEN** 该注释 MUST 紧跟一行说明豁免理由（例如「测试清理，错误已在测试结束体现」）

#### Scenario: 仓库根目录 MUST 不存在会被工具误解析为配置文件的空目录

- **WHEN** 静态检查工具按惯例寻找 `.golangci.yml` 等配置文件
- **THEN** 该路径 MUST 是文件或 MUST NOT 存在；空目录形式 MUST 被清理

### Requirement: lint 工具版本升级 MUST 与 `go.mod` 同步

当 `go.mod` 升级 Go 主版本或次版本时，维护者 MUST 在同一变更中把 `golangci-lint-action` 的 `version` 字段切换到与新 Go 版本兼容的预编译版本。

#### Scenario: go.mod 升级后 lint 未跟进

- **WHEN** `go.mod` 中 `go X.Y` 提升到 `go X'.Y'`（X' > X）
- **THEN** 若 `.github/workflows/ci.yml` 的 lint job `version` 字段未相应升级，CI MUST 在合并前失败
- **AND** 维护者 MUST 在合并前完成 lint 版本对齐

## Out of Scope

本 capability 不约束下列平台级 CI 噪声——它们由 GitHub Actions 平台侧控制，与本仓库代码无关：

- `Failed to restore: Cache service responded with 400` —— GitHub cache 服务偶发不可用。
- `Failed to save: ... 0Uqm8agAAAAAChtP5lDPtT7+x0QsIGtdFUEhMMzBFREdFMDEyMQBFZGdl` —— 同上，平台返回 HTML 错误页。
- `Node.js 20 is deprecated. The following actions target Node.js 20 but are being forced to run on Node.js 24: ...` —— GitHub runner 自动把 Node 20 action 升级到 Node 24 运行。

这些提示不计入 Lint job 的成功判定标准。