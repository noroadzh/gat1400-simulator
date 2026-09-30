# Design

> Motivation: see `proposal.md` — Why。本节只描述**怎么做**与**为什么这样选**。

## Context

- `go.mod` 声明 `go 1.25.5`。当前 CI 在 Lint job 固定拉取 `golangci-lint v1.61.0`（GitHub release 上由 go1.23.1 编译的预编译二进制），加载 go 1.25.5 module 直接报错退出码 3，Lint job 标红。
- 仓库**从未**提交过 `.golangci.yml`；只有仓库根目录下一个空目录 `.golangci.yml/`，路径形态会误导部分工具（虽对 golangci-lint v1.x 无影响——它只在文件存在时加载）。
- golangci-lint 自 v2.x 起每个 minor 版本都明确「built with go X.Y」并要求 module 的 Go 版本 ≤ 构建版本，否则在加载阶段拒绝。已实测 v1.61.0 (go1.23.1)、v1.64.8 (go1.24.1)、v2.0.2 (go1.24.1)、v2.3.1 (go1.24.5) 均与 `go 1.25.5` 不兼容；v2.4.0+ 全部由 go1.25.x 构建，兼容。
- 当用兼容 linter（v2.7.2，go1.25.4）实跑时，仓库存在 27 个真实静态检查 issue 长期被掩盖：errcheck 19、ineffassign 1、staticcheck 3、unused 4。

## Goals / Non-Goals

**Goals**
- 让 CI Lint job 真正执行静态检查并在无 issue 时退出码 0。
- 解决所有 27 个真实静态检查 issue，且不引入新的错误传播路径或运行时行为变化。
- 在 CI 配置中明示静态检查工具与 module Go 版本的兼容性约束，避免未来升级 `go.mod` 时再次踩坑。

**Non-Goals**
- 不重写或重构任何业务逻辑。
- 不修改 `.golangci.yml` 配置（仓库历史上无此文件，保留默认 lint 集）。
- 不修改 Frontend / Test / Build 三个 job。
- 不降级 `go.mod` 的 `go 1.25.5`。
- 不删除 `golangci-lint` 与 `make lint` 的本地入口（仅修复其不能跑的默认版本）。

### 1. 选择 `golangci-lint v2.7.2` 而非最新版

- **Decision**：在 `.github/workflows/ci.yml::lint` 中将 `version: v1.61.0` 改为 `version: v2.7.2`。
- **Why**：v2.7.2 是 v2 系列里**用 go1.25.4 编译的最新稳定版**（发布于 2025-12-07），与 `go 1.25.5` module 兼容。v2.10.1 起切换到 go1.26 编译，又会与 module 不兼容。
- **Alternatives considered**
  - `v2.14.0`（最新版，2026-09 发布，go1.26.0 构建）—— 不兼容 module go1.25.5；升级会触发同样的加载失败。
  - `v2.5.0`（更早但同样兼容）—— 已 1 年无安全更新；v2.7.2 是当前 LTS 候选。
  - `go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.7.2` 由 CI runner 现地编译 —— 引入 2-3 分钟编译开销，与原 action 设计不一致；放弃。

### 2. errcheck 修复策略：defer 闭包包装 vs `_ =` 显式忽略

- **Decision**：`defer x.Close()` 改为 `defer func() { _ = x.Close() }()`；直接调用改为 `_ = f(...)`。
- **Why**：保持原有 `defer` 语义（函数返回时执行），不引入新的错误传播路径，不改变测试中断言。Go 社区事实标准做法。
- **Alternatives considered**
  - 把错误记到全局 logger / `t.Log` —— 引入新依赖与测试噪音。
  - 直接 `x.Close()` 不 defer —— 改变 panic 时清理语义，回退风险高。
  - `//nolint:errcheck` 注释 —— 抑制 lint 但保留隐患；本变更要"真正修复"，不用抑制。

### 3. unused 修复策略：删除 vs `//nolint:unused`

- **Decision**：直接删除未引用的私有方法、辅助函数、字段。
- **Why**：被 lint 标记的均为内部细节，跨包边界已用 grep 验证无外部引用。删除比保留并抑制更彻底。
- **Alternatives considered**
  - 全部 `//nolint:unused` 抑制 —— 留下死代码、未来误用概率高。

### 4. 不补 `.golangci.yml` 配置文件

- **Decision**：删除空目录 `.golangci.yml/`，但不创建新文件。
- **Why**：v2.7.2 默认 lint 集（errcheck、gosimple、govet、ineffassign、staticcheck、unused）已能覆盖本变更目标。零配置最简，避免迁移噪音。
- **Alternatives considered**
  - 补一份显式配置文件锁定 lint 集 —— 增加维护成本；当前规模不值得。

## Risks / Trade-offs

- **Risk**：v2 → v1 配置格式破坏性变更在未来若需要 `.golangci.yml` 时将带来一次性迁移 → **Mitigation**：在 `specs/ci-quality-gates/spec.md` 中明示「lint 工具版本必须与 module Go 版本兼容」，未来升级时与 `go.mod` 同步进行。
- **Risk**：v2.7.2 是 2025-12 版本，无重大安全修复历史，但仍是**当前 LTS 候选**；后续 minor 仍由 go1.25.x 编译 → **Mitigation**：在 spec 中要求"lint 工具版本需在 30 天内有 release"。
- **Risk**：errcheck 抑制可能掩盖未来回归 → **Mitigation**：保留原始 defer 位置语义，未删除任何错误处理路径；若资源泄露应在 e2e 测试中体现，仓库已有 capture_e2e / protocol_e2e 覆盖。
- **Risk**：CI 偶发的 Cache 400 / Node 20 deprecation 提示仍会出现，与本变更无关 → **Mitigation**：显式记录在 spec 的 `### Out of Scope` 中，避免将来误以为是 lint 失败。

## Migration Plan

本变更**无运行时数据迁移**，纯 CI 配置 + 代码静态清理：

1. **本机预检**：用 `/tmp/golangci-lint-2.7.2-darwin-amd64/golangci-lint run ./...` 跑出 27 issue，逐条修复后再跑得 0 issue。
2. **回滚**：所有修改均为局部代码与单行 CI 版本号变更，`git revert <commit>` 即可恢复。
3. **灰度**：CI Lint job 改动立即生效；不涉及运行时功能，不需灰度发布。

## Open Questions

无。