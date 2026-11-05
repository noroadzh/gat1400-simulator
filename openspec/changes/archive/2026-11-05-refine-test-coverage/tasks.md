# Tasks

## 1. 重新统计真实覆盖率

- [x] 1.1 跑覆盖率归一口径（见 CI / Makefile）
- [x] 1.2 分层目标写入主 `openspec/specs/testing/spec.md`

## 2. Engine 单元测试补齐

- [x] 2.1 识别 Start/Stop/AutoStart/ListRunning 路径
- [x] 2.2 新增 `internal/adapter/scenario/engine_test.go`（fake provisioner）
- [x] 2.3 本地验证 Engine 相关用例
- [x] 2.4 `go vet` / `go test` 目标纳入本 change

## 3. CI 覆盖率归一

- [x] 3.1 `.github/workflows/ci.yml` 使用 `-coverpkg` + `-coverprofile`
- [x] 3.2 upload-artifact `coverage.out`
- [x] 3.3 `$GITHUB_STEP_SUMMARY` 输出 cover 摘要
- [x] 3.4 `Makefile` `test` 目标同步

## 4. spec 修订

- [x] 4.1 主 `testing/spec.md` 分层目标（并修正导语「≥90%」冲突）
- [x] 4.2 `ci-quality-gates` 主 spec 追加覆盖率归一口径
- [x] 4.3 新建主 capability `end-to-end-testing`
- [x] 4.4 `scenario` 主 spec 合入 Stop 错误显式处理；代码 `log.Warn`

## 5. 本地验证

- [x] 5.1 代码改动落地
- [x] 5.2 `go test ./internal/adapter/scenario/ ./internal/app/application/ -count=1`
- [x] 5.3 CI/`Makefile` 已含 coverprofile（与主 spec 对齐）
- [x] 5.4 design.md 补齐

## 6. 归档

- [x] 6.1–6.4 同步主 specs 后执行 archive
