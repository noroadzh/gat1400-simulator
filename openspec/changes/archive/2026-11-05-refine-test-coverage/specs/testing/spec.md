# Delta: testing

## REMOVED Requirements

### Requirement: 全部 internal 包 MUST ≥ 90% 测试覆盖率

**移除原因**：该单一 90% 阈值与仓库现实长期脱节——`internal/adapter/scenario`（编排核心，576 行）与 `cmd/gat1400-sim`（引导入口）实测分别为 35.0% / 3.7%，导致该 Requirement 事实上从未被满足，却仍作为契约留在主 spec 中，丧失约束力。取而代之的是 `包覆盖率 MUST 满足分层目标且 MUST 可观测` 中的分层目标。

**迁移方式**：以 ADDED 的分层覆盖率要求为准。分层要求不要求任何代码回退；当前低于目标值的包在本次变更中通过补测试提升至目标线，其余包按 Scenario「低覆盖率包的豁免必须显式」走显式豁免流程。

## ADDED Requirements

### Requirement: 包覆盖率 MUST 满足分层目标且 MUST 可观测

仓库 MUST 在 CI 与本地 `make test` / `make cover` 中以**同一口径**统计覆盖率（`-coverpkg=./internal/...,./cmd/...`），覆盖率 MUST 作为产物输出（`coverage.out`）。该口径 MUST 覆盖跨包测试（`test/e2e`、`test/contract`）对其所触达包的贡献，MUST NOT 出现 `[no statements]` 的包被排除在统计之外。

各包基线目标（语句覆盖率）：

| 层级 | 目标 | 例外说明 |
|---|---|---|
| `internal/domain/**`、`internal/app/ports/**` | ≥ 95% | 域逻辑与端口接口 MUST 完全覆盖 |
| `internal/adapter/scenario/**` | ≥ 80% | 编排核心，MUST 重点覆盖 |
| `internal/adapter/**`（除 scenario） | ≥ 70% | 受外部资源/网络/文件系统约束的边角允许未覆盖 |
| `internal/ui/**`（BFF） | ≥ 70% | WebSocket 长连接与视图渲染允许未覆盖 |
| `cmd/gat1400-sim/**` | ≥ 50% | 入口与启动/关闭路径覆盖即可 |

#### Scenario: CI 以归一口径统计覆盖率

- **WHEN** PR 触发 `.github/workflows/ci.yml` 的 Test job
- **THEN** `go test` 命令 MUST 包含 `-coverpkg=./internal/...,./cmd/...` 与 `-coverprofile=coverage.out`
- **AND** 产物 `coverage.out` MUST 作为 artifact 上传，保留至少 7 天
- **AND** job summary MUST 输出逐包覆盖率行

#### Scenario: 本地与 CI 口径一致

- **WHEN** 开发者本地执行 `make cover`
- **THEN** 该命令 MUST 与 CI Test job 使用相同的 `-coverpkg` 集合
- **AND** 本地不得出现 CI 未统计的包

#### Scenario: 低覆盖率包的豁免必须显式

- **WHEN** 某包的覆盖率低于对应层级目标
- **THEN** PR 描述 MUST 包含该包的豁免理由，理由 MUST 显式写出「为什么未达目标」与「后续计划」
- **AND** reviewer MUST 在 PR 评审中确认豁免理由