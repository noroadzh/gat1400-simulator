## 任务

### 任务：初始化 Go 模块

- [x] 执行 `go mod init github.com/noroadzh/gat1400-simulator`
- [x] 在 `go.mod` 中设置 `go 1.21`
- [x] 验证 `go build ./cmd/...` 在零源文件情况下通过

### 任务：创建目录结构

- [x] 创建 `cmd/simulator/`、`internal/`、`configs/`、`data/`、`web/`、`test/`、`openspec/`、`docs/`
- [x] 为 Git 需要跟踪的空目录添加 `.gitkeep` 文件

### 任务：创建主入口点

- [x] 编写 `cmd/simulator/main.go`，包含一个输出 "hello world" 的最小 `main()`
- [x] 验证 `go run ./cmd/simulator/` 打印问候语并返回 0

### 任务：添加 Makefile

- [x] 定义 `build`、`test`、`lint`、`run`、`clean` 目标
- [x] 定义 `BINARY_NAME`、`GO_LDFLAGS` 变量
- [x] 验证 `make build` 产出 `bin/gat1400-simulator`
- [x] 验证 `make test` 以 `-race` 运行 `go test ./...`

### 任务：添加 golangci-lint 配置

- [x] 创建 `.golangci.yml`，启用 linter：`vet`、`staticcheck`、`gofmt`、`goimports`、`revive`
- [x] 配置 `issues.exclude-rules` 排除测试文件
- [x] 验证 `golangci-lint run ./...` 在最小源码上报告零错误

### 任务：添加 GitHub Actions CI 工作流

- [x] 创建 `.github/workflows/ci.yml`
- [x] 定义任务链：lint → test（带 race）→ build（linux/amd64、darwin/arm64）
- [x] 验证工作流语法（用 `act`，或推送至测试分支）

### 任务：添加 .gitignore

- [x] 忽略 `bin/`、`*.db`、`*.pcap`、` .vscode/`、`.idea/`、`vendor/`
- [x] 包含 `go.sum`（不要忽略）

### 验证

全部任务完成的标准是：
- `go build ./cmd/...` → exit 0，二进制位于 `bin/`
- `go test ./...` → exit 0
- `golangci-lint run ./...` → 零错误
- `make build test lint` → 全绿