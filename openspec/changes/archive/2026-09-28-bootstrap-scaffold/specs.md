## ADDED Requirements

### Requirement: 项目 MUST 无错误编译

在干净检出的代码上用 Go 1.21+ 执行 `go build ./cmd/...` 时，构建 MUST 以退出码 0 完成，并在 `bin/` 下产出至少一个二进制。

### Requirement: 测试套件 MUST 通过

执行 `go test ./...` 时，全部测试 MUST 以退出码 0 通过。CI 中 MUST 使用 `-race` 标志以检测数据竞争。

### Requirement: golangci-lint MUST 报告零错误

执行 `golangci-lint run ./...` 时，MUST 报告零个 linter 错误。

### Requirement: 所有私有包 MUST 位于 internal/ 之下

`cmd/` 与 `test/` 之外的任何包 MUST NOT 导入 `internal/` 下的包。该约束 MUST 由 CI 强制（golangci-lint 的 `exported` 规则检查）。

### Requirement: SQLite 依赖 MUST 是纯 Go 实现

项目 MUST 使用 `modernc.org/sqlite` 作为 SQLite 驱动。任何 `database/sql` 搭配 `github.com/mattn/go-sqlite3` 的导入 MUST 导致构建失败。

### Requirement: Makefile MUST 提供标准目标

Makefile MUST 定义 `build`、`test`、`lint`、`run`、`clean` 目标。每个目标 MUST 执行对应的标准 Go 或工具链命令。

### Requirement: GitHub Actions CI MUST 在每次 push 时运行

`.github/workflows/ci.yml` 中的工作流 MUST 在对 `main` 与 `release/**` 分支的 `push` 和 `pull_request` 时触发。工作流 MUST 按 lint、test、build 的顺序执行。

### Requirement: Go 模块版本 MUST 声明为 1.21+

`go.mod` 文件 MUST 包含 `go 1.21` 或更高版本，以确保能够使用 slog、range-over-func 与泛型。

---

## ADDED Architecture Decisions

### Decision: 六边形架构

代码库 MUST 划分为三层：
- **domain/**：纯领域模型，无外部依赖。
- **app/**：应用服务，仅依赖 domain 与 ports（接口）。
- **adapter/**：基础设施适配器（HTTP 服务端、HTTP 客户端、SQLite 存储），依赖 domain、app 与外部库。

该分层 MUST 通过确保 `internal/app/` 不包含来自 `internal/adapter/` 的导入来验证。

### Decision: 纯 Go SQLite

SQLite 操作通过 `modernc.org/sqlite` 实现，它编译为纯 Go 的 WebAssembly 且不需要 CGO。这使得 macOS arm64（Apple Silicon）与 Linux amd64 的二进制无需交叉编译工具链即可构建。

### Decision: 标准库日志

所有包 MUST 使用 `log/slog`（而非第三方日志库）进行结构化日志记录。Handler（生产用 JSON、开发用 text）在应用启动时配置。