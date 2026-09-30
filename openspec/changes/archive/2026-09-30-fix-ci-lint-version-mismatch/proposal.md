# Proposal

## Why

CI 的 Lint job 持续失败，GitHub 报告「2 errors, 13 warnings, 4 notices」。本地复现确认：仓库 `go.mod` 声明 `go 1.25.5`，而 CI 固定使用 `golangci-lint v1.61.0`——一个由 go1.23.1 预编译的二进制。它在加载 module阶段直接崩溃：

```
can't load config: the Go language version (go1.23) used to build golangci-lint
is lower than the targeted Go version (1.25.5)
```

退出码 3 让 GitHub 把整个 Lint job 标为失败，并连带输出 13 warnings / 4 notices 的降级报告。也就是说，**当前 Lint job 从未真正检查过任何一行代码**，静态质量门禁形同虚设；而一旦 linter 能正常启动，会立刻暴露 27 个长期被掩盖的真实问题（errcheck 19、ineffassign 1、staticcheck 3、unused 4）。

## What Changes

- **BREAKING（仅 CI 工具链）**：`.github/workflows/ci.yml` 的 Lint job 将 `golangci-lint` 从 `v1.61.0` 升级到 `v2.7.2`。v1.61.0 无法加载 go 1.25 module；v2.7.2 由 go1.25.4 构建，可正常加载。不改变任何运行时行为或公开 API。
- 修复 linter 恢复工作后暴露的 27 个真实静态检查问题，分四类：
  - **errcheck（19）**：所有 `defer x.Close()`、测试清理用的 `defer os.Remove(...)`、以及直接调用但未接收返回值的 `Append` / `UpsertNode` / `io.ReadAll` / `SetReadDeadline`，统一改为显式忽略返回值（`defer func() { _ = x.Close() }()` 或 `_ = f(...)`）。保持原语义，不引入新的错误传播路径。
  - **ineffassign（1）**：`scenario/factory.go` 中一个赋值后从未被读取的 `now` 变量。
  - **staticcheck（3）**：`wire/digest_test.go` 的空 `if` 分支、`logging/logger.go` 中冗余的显式函数类型标注、`domain/ids/ids_test.go` 中循环内重复编译的 `regexp`。
  - **unused（4）**：两处未被引用的私有方法（`httpapi/cascade.go`）、一个未被引用的测试辅助函数（`scenario/factory_test.go`）、一个未被使用的结构体字段（`scenario/loader.go` 的 `mu sync.Mutex`）。
- 删除仓库根目录下的空目录 `.golangci.yml/`（不在 git 追踪、未被 `.gitignore` 覆盖），避免工具链把它误认为配置文件路径。
- 在 spec 中固化一条非显然的约束：**静态检查工具的构建 Go 版本不得低于 `go.mod` 声明的 Go 版本**，防止将来升级 `go.mod` 时重蹈覆辙。

## Capabilities

### New Capabilities
- `ci-quality-gates`: 定义持续集成中的静态质量门禁要求——lint 工具链必须与 module 的 Go 版本兼容、Lint job 必须真正执行检查并在零问题时通过、CI 中的测试与 lint 门禁不得被平台噪声掩盖真实结论。

### Modified Capabilities
<!-- 本变更不改变任何既有 capability 的需求语义：所有修改均为静态检查工具链升级与局部代码清理，不改变对外行为、协议、配置或 API。 -->
无。

## Impact

**CI 配置**
- `.github/workflows/ci.yml`（仅 Lint job 的 `version` 字段）

**受影响代码（均为局部清理，不改变行为）**
- `cmd/gat1400-sim/main.go`
- `internal/adapter/httpapi/cascade.go`
- `internal/adapter/scenario/{factory.go,factory_test.go,loader.go}`
- `internal/adapter/storage/{capture_reader.go,storage_test.go}`
- `internal/adapter/wire/{client.go,digest_test.go}`
- `internal/app/logging/logger.go`
- `internal/domain/ids/ids_test.go`
- `internal/ui/{resources.go,ws.go}`
- `test/e2e/{capture_e2e_test.go,protocol_e2e_test.go}`

**依赖与运行环境**
- 新增对 `golangci-lint v2.7.2` 的 CI 工具依赖（仅 CI 环境，不进`go.mod`）
- 不修改 `go.mod` 的 Go 版本，不新增运行时依赖
- 不影响 Frontend / Test / Build 三个 job

**风险**
- golangci-lint v1 → v2 存在配置格式破坏性变更，但本仓库从未提交过 `.golangci.yml`，v2 默认 lint 集即可运行，无配置迁移成本
