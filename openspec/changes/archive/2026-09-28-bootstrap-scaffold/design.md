# 设计：项目脚手架

## 技术选型

| 类别 | 选择 | 理由 |
|---|---|---|
| 语言 | Go 1.21+ | slog 标准库、range-over-func、泛型 |
| SQLite | modernc.org/sqlite | 纯 Go，无 CGO，跨平台构建 |
| HTTP 框架 | labstack/echo/v4 | 成熟、文档齐全、支持中间件分组 |
| 配置 | viper | 支持环境变量、YAML、flag 绑定 |
| 日志 | log/slog | 标准库、结构化、JSON/text handler |
| Lint | golangci-lint | 统一 CLI 整合 20+ linter |

## 目录布局

```
gat1400-simulator/
├── cmd/                  # 应用入口
│   └── simulator/        # 主二进制
├── internal/             # 私有包（六边形分层）
│   ├── app/              # 应用服务（编排）
│   ├── adapter/          # 基础设施适配器（HTTP、存储、wire）
│   ├── domain/           # 领域模型（纯业务逻辑）
│   └── ui/               # Web BFF（控制面）
├── configs/              # YAML 配置（节点定义、场景）
├── data/                 # 运行时数据（SQLite DB、抓包 pcap）
├── web/                  # 内嵌的前端资源
├── test/                 # 契约测试、e2e、golden 样本
├── openspec/             # OpenSpec 变更管理
├── docs/                 # 协议参考、架构文档
├── Makefile
├── go.mod / go.sum
├── .golangci.yml
└── .github/workflows/
```

## 构建目标

- `make build` → `bin/gat1400-simulator`（当前 OS/arch）
- `make test` → `go test -race ./...`
- `make lint` → `golangci-lint run`
- `make run` → `go run ./cmd/simulator/`
- `make clean` → rm bin/

## CI 流水线

1. golangci-lint
2. go test -race ./...
3. goreleaser 跨平台构建（linux/amd64、darwin/arm64）

## 安全约束

- 源代码中不得硬编码凭据
- 凭据从配置文件或环境变量加载
- SQLite 文件权限仅限本人（mode 0600）