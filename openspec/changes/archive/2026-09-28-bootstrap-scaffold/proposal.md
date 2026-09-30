# 提案：项目脚手架

## 状态
已归档 —— 已实现为项目基础工程。

## 背景动机

我们需要一个最小且生产级的 Go 项目骨架，满足：
- `go build` 开箱即用即可通过
- CI 确定性、可复现
- 建立六边形 / 端口-适配器架构的目录约定
- 支持多平台构建（Linux amd64、macOS arm64）

## 目标

- `go.mod` 中依赖版本锁定
- 标准布局：`cmd/`、`internal/`、`configs/`、`data/`、`web/`、`test/`
- `Makefile` 提供 `build`、`test`、`lint`、`run`、`clean` 目标
- GitHub Actions CI：lint → test（带 race）→ build（多平台）
- `.golangci.yml` 启用必要的 linter（vet、staticcheck、gofmt、goimports、revive）
- 通过 `modernc.org/sqlite` 使用纯 Go SQLite（不依赖 CGO）

## 非目标

- 不实现任何 GA/T 1400 协议（推迟到 domain-models）
- 不做数据库迁移或 ORM（推迟到 adapter-storage）
- 不提供 Docker/Kubernetes 清单（推迟到运维文档）

## 待定问题

无 —— 全部在 design.md 中确定。