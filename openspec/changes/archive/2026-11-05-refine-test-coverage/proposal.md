# Proposal

> 注：主 `openspec/specs/testing/spec.md` 已手动合入分层覆盖率 Requirement；归档时若 delta
> 与主 spec 冲突，以主 spec 为准，可跳过 testing ADDED 或先 REMOVED 再 ADDED。

## Why

`openspec/specs/testing/spec.md` 声明「`internal/` 包 MUST ≥ 90% 覆盖率」,但实测仓库从未通过 `-coverpkg=./internal/...` 跑整体覆盖率:  `go test -cover` 的输出里 `test/contract`、`test/e2e` 都是 `[no statements]`,各包自测覆盖率是孤立数据。  实际数字里 4 个包低于 80%(`scenario 35.0%`、`httpapi 60.6%`、`ui 75.4%`、`storage 78.5%`、`wire 78.7%`),`cmd 3.7%`。  这条 Requirement 与现实长期脱节,既没有 CI 门槛兜底也没有能力补足,实际等于名存实亡。

同时当前 `openspec/specs/` 缺失"端到端契约" capability —— 测试相关全部塞在 `testing` 单 capability 下,没有「跨包端到端 + 契约回放 + 覆盖率归一」这条独立线索,后续 BFF 端到端、SSE 端到端、协议级联等新场景无从分类。

## What Changes

- **新增 capability `end-to-end-testing`**: 把现有散落在 `testing`、`scenario-lifecycle`、`adapter-wire`、`web-bff` 各 capability 里的端到端 / 契约 / 集成测试归到一条独立线索下,并在 spec 中明确覆盖率归一口径与基线包覆盖率分层目标。
- **修订 `openspec/specs/testing/spec.md`**: 把"≥90%"改为**按层分档**: domain/ports ≥95%、adapter ≥70%、cmd ≥50%;明确豁免包(纯入口、无业务逻辑)的退出条件;声明所有 `*_test.go` 文件 MUST 出现在覆盖率统计里(`-coverpkg=./internal/...` 或等价手段)。
- **修订 `openspec/specs/ci-quality-gates/spec.md`**: 在「CI MUST 跑测试」Scenario 下追加一句「CI Test job MUST 在 `go test` 命令里传 `-coverpkg=./internal/...,./cmd/...`,把覆盖率作为 PR 检查产物输出到 codecov / 内部仪表盘」。
- **新增 CI 工作流产物** `coverage.out` 与差分报告 —— 不强制 PR 阻塞,但需有产物可供审计。
- **补 `internal/adapter/scenario/Engine` 单元测试**: 当前 35% 覆盖率(全仓 adapter 最低),它是节点生命周期 + AutoStart 的编排核心,也是最容易出并发 bug 的点。  补 Start/Stop/AutoStart/事件回调 4 类用例,目标 ≥80%。

## Capabilities

### New Capabilities

- `end-to-end-testing`: 新建,定义跨包端到端测试、契约测试、覆盖率归一化目标(基线包 + 排除包 + 报告产物)。

### Modified Capabilities

- `testing`: 把「internal/ 包 MUST ≥ 90% 覆盖率」这一条 Requirement 改写为分层目标,并把"覆盖率必须可观测"作为强制项。
- `ci-quality-gates`: 在「CI MUST 跑测试」Scenario 下追加覆盖率归一口径。

## Impact

| 影响范围 | 说明 |
|---|---|
| `openspec/specs/` | 1 个新增 capability(`end-to-end-testing`),2 个修订(`testing`、`ci-quality-gates`)。 |
| `internal/adapter/scenario/engine_test.go` | 新增/补全用例,文件预计 200~300 行新增。 |
| `.github/workflows/ci.yml` | Test job `go test` 命令追加 `-coverpkg=./internal/...,./cmd/... -coverprofile=coverage.out`;新 step `go tool cover -func=coverage.out | tail` 上传产物(可选 codecov)。 |
| 生产代码 | 不动。 |
| `Makefile` | `make test` / `make cover` 同步更新(若定义)。 |
| 现有契约文件 | `test/contract/golden/`、`test/e2e/` 不动;但其统计纳入覆盖率归一。 |

## Non-Goals

- **不**把所有内部包一刀切拉到 90%: 很多适配层(httpapi、ui)依赖外部服务、文件系统、协议握手的边角,做完整 mock 反而扭曲测试语义,改用分层目标更现实。
- **不**引入 mutation testing / fuzzing: 规模小,投入产出比不匹配,等真有零散问题再单独评估。
- **不**改造现有 e2e / 契约测试代码: 仅做归类与覆盖率统计口径统一,不动实现。
