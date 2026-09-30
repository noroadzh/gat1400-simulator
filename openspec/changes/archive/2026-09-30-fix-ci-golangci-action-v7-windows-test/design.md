# Design

## Context

参见 proposal.md(动机)。当前现状:

- `.github/workflows/ci.yml` Lint job 固定 `golangci/golangci-lint-action@v6.5.1` + `version: v2.7.2`
- action v6 的版本解析器只识别 v1.x 版本字符串,对 v2.x 直接报 `golangci-lint v2 is not supported by golangci-lint-action v6` 并退出
- `internal/app/logging/daily_test.go::TestDailyWriter_RotateIsSerialized` 用 `t.TempDir()` + `lumberjack.Logger` 写日志,但 `t.Cleanup` 只关闭 `dailyWriter` 的后台 goroutine,**未关闭 lumberjack 持有的文件句柄**
- 在 Windows 上,`t.TempDir()` 的 cleanup 走 `RemoveAll`,遇到被 lumberjack 独占的文件即失败
- 三平台中仅 windows-latest 失败是因为 Windows 默认文件锁为 `FILE_SHARE_NONE` 行为,Mac/Linux 允许 delete-while-open

## Goals / Non-Goals

**Goals:**

- 让 CI Lint job 在保持 linter v2.7.2 的前提下真正执行静态检查
- 让 `TestDailyWriter_RotateIsSerialized` 在 ubuntu/mac/windows 三个平台一致通过

**Non-Goals:**

- 不降级 linter 回 v1.x
- 不引入 `.golangci.yml`(沿用默认 lints,与上一轮归档决策一致)
- 不重写 dailyWriter 的 stop/cleanup 协议(`sync.Once` 已能保证幂等)
- 不引入跨平台文件锁差异处理(以「先关后清」作为跨平台最简方案)

## Decisions

### Decision 1: `golangci/golangci-lint-action@v6.5.1` → `@v7.0.1`

**选择:** 升到 v7.0.1(v7 系列最新)。

**理由:**

- action v6 错误消息明确指向 v7 作为最低升级路径
- v7 是支持 golangci-lint v2 的首个 major(官方 fix message 隐含推荐)
- v7 仍在维护(`v7.0.0`、`v7.0.1` 两次 patch),且与当前仓库其他 actions(checkout `@v4.2.2`、setup-go `@v5.2.0`、setup-node `@v4.1.0`、upload-artifact `@v4.4.3`)的版本家族一致——其他 action 都钉在 v4/v5
- 不选 v8 / v9:这两个 major 同样支持 lint v2,但发布较新、对老 linter v1 路径已删除;v7 是最贴近「升级 action 不改 linter」这一窄目标的版本

**备选:**

- `@v9.3.0`(最新 stable)— 拒绝。版本跨度过大,引入不可见风险(上游 issue tracker 上 v8→v9 之间有 node runtime 行为变更记录),与本仓库主流 v4/v5 action 风格不一致
- `@v8.x` — 拒绝。跳过 v7 缺乏理由,且 v7 已足够解决问题
- 保留 `@v6` + 降级 linter 至 v1.x — 拒绝。上一轮已确认 v1 不能加载 `go 1.25.5` module

### Decision 2: `TestDailyWriter_RotateIsSerialized` 在 `t.Cleanup` 中追加 `_ = rot.Close()`

**选择:** 在 `t.Cleanup(func() { d.stop() })` 之后追加 `_ = rot.Close()`,确保 TempDir 回收前 lumberjack 文件句柄已释放。

**理由:**

- 当前 cleanup 序列: `d.stop()` 关闭 dailyWriter 后台 goroutine,但 `rot`(*lumberjack.Logger)是 dailyWriter 持有的底层资源,`stop` 不代理其关闭
- Windows 文件锁语义(`FILE_SHARE_NONE`)要求「关闭 → 删除」严格串行,Linux/macOS 在测试短时间窗口内 delete-while-open 通常能成功,这就是「本地不可见、Windows 必现」的根因
- `rot.Close()` 返回 error,但 cleanup 阶段无法传播,显式 `_ =` 丢弃以满足 errcheck

**备选:**

- 在测试用例体内手动 `defer func(){ _ = rot.Close() }()` — 拒绝。`t.Cleanup` 才是测试框架约定的资源释放点,显式 defer 会让 cleanup 顺序与框架管理顺序不一致
- 修改 `dailyWriter.stop()` 顺带关掉 `rot` — 拒绝。生产代码 stop() 不应该关 rot,后者由 `rotatorCloser.Close()` 统一管理;测试场景临时改 cleanup 是局部最小改动
- 用 `runtime.AddCleanup`(Go 1.24+ 替代方案)— 拒绝。语义等价但引入新 API,无收益

### Decision 3: 不扩大 spec 改动范围

**选择:** delta spec 仅修改 `ci-quality-gates` 一个 requirement,不新增 testing/logging capability 改动。

**理由:**

- 测试句柄释放是单点 bug 修复,属实现细节,不构成新需求
- 现有 `testing` capability 定义的是「真实 socket、黄金样本、覆盖率」等测试质量契约,与「单个测试的资源管理」是两个层级
- `openspec validate --strict` 严禁「为满足验证而发明 requirement」

## Risks / Trade-offs

| 风险 | 缓解 |
|---|---|
| v7.0.1 与当前 lint v2.7.2 在少数默认 patch 上有 behavior diff(如 nilable 检查) | 在 PR 前本地用 `/tmp/golangci-lint-2.7.2-darwin-amd64/golangci-lint run ./...` 验证 0 issue;若出现新增 issue,按上一轮经验逐项修复 |
| Windows 测试仍可能因其他 lumberjack 句柄泄漏失败 | 本次仅修一个测试用例;若 CI 还有其他 Windows-only 失败,作为下一轮 change 处理 |
| `@v7.0.1` 后续被上游弃用 | 升级到 v7 已是 action 主版本主动迁移的官方推荐路径;后续若弃用,作为独立 change 处理 |
| 测试 cleanup 中手动 `_ = rot.Close()` 与生产路径不同步 | 文档化注释说明「测试环境额外 cleanup,生产路径仍由 rotatorCloser.Close() 统一管理」 |

## Migration Plan

| 阶段 | 操作 |
|---|---|
| 实施 | 1. 修改 `.github/workflows/ci.yml` action 版本;2. 修改 `daily_test.go` cleanup;3. 本地验证 lint / build / test -race ./... |
| 提交 | 单个 commit `fix(ci): 升级 golangci-lint-action 到 v7.0.1 兼容 lint v2 + 修 Windows lumberjack 句柄泄漏` |
| 推送 | `git push` 触发 GitHub CI,关注 Lint job 与三平台 Test job |
| 回滚 | 若 CI 出现意外回归,`git revert <sha>` 即可;无破坏性 schema/接口变化,无须数据迁移 |

## Open Questions

无。