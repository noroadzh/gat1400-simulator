# Design — refine-test-coverage

## 决策摘要

- 覆盖率口径统一为 `-coverpkg=./internal/...,./cmd/...` + `coverage.out`（CI 与 Makefile 一致）。
- 废除「internal 一刀切 ≥90%」，改为分层目标（见主 `testing` / `end-to-end-testing` specs）。
- Engine 补单测；契约 / e2e 目录归属独立 capability。
- `ScenarioService.Stop` 对 `engine.Stop` 错误改为 `log.Warn`，不再 `_ =` 静默。

## 非目标

- 不强制 PR 因覆盖率阈值红灯；产物可审计即可。
- 不改造既有 e2e / golden 实现，仅归类与统计。
