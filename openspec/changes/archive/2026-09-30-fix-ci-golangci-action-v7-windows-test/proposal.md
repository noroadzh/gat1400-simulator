# Proposal

## Why

上一轮归档 change `fix-ci-lint-version-mismatch` 将 CI lint 工具从 v1.61.0 升级到 v2.7.2(golangci-lint v2),同时将 `golangci/golangci-lint-action` 锁定在 `@v6.5.1`。但 action v6 的版本校验逻辑只认识 v1 系列版本字符串,对 v2.x 版本字符串直接拒绝,导致 Lint job 在 prepare 阶段崩溃。

同时,`TestDailyWriter_RotateIsSerialized` 在 Windows CI(`windows-latest`)上失败:测试通过 `t.Cleanup` 只关闭了 `dailyWriter` 的后台 goroutine,但未关闭底层的 `lumberjack.Logger` 持有的文件句柄,导致 `t.TempDir()` cleanup 时 Windows 报告"另一个进程正在使用该文件"。

## What Changes

- `.github/workflows/ci.yml` Lint job:`golangci/golangci-lint-action@v6.5.1` → `@v7.0.1`(golangci-lint v2 所需的最低 action 版本)
- `internal/app/logging/daily_test.go`:`TestDailyWriter_RotateIsSerialized` 在 `t.Cleanup` 中追加 `_ = rot.Close()`,确保 `lumberjack.Logger` 在 TempDir cleanup 前释放文件句柄
- `openspec/specs/ci-quality-gates/spec.md`:追加 scenario,说明 golangci-lint v2 必须与 action v7 配对

## Capabilities

### New Capabilities

(无)

### Modified Capabilities

- `ci-quality-gates`:在 Requirement「静态检查工具 MUST 与 go.mod 声明的 Go 版本兼容」下追加新 Scenario,明确 golangci-lint v2 与 action v6 不兼容,必须升级 action 到 v7。

## Impact

| 影响范围 | 说明 |
|---|---|
| `.github/workflows/ci.yml` | 仅 Lint job 的 action 版本一行 |
| `internal/app/logging/daily_test.go` | 仅 `TestDailyWriter_RotateIsSerialized` 一处加 1 行 |
| `openspec/specs/ci-quality-gates/spec.md` | delta spec 仅追加 1 个 scenario |
| go.mod / 生产代码 | 不涉及 |
| CI Test job | windows-latest 失败修复后,三平台全绿 |
