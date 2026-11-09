# Delta: scenario

## ADDED Requirements

### Requirement: 场景引擎 AutoStart 取消路径错误 MUST 显式处理

`application.NodeService.StopNode` 调用 `engine.Stop(id)` 的错误 MUST NOT 用 `_ =` 静默忽略。当前落点 `internal/app/application/services.go:301` MUST 改为在收到错误时记 `log.Warn` 并在响应中包含失败原因（业务上节点已确认接收，响应仍为 200）。

#### Scenario: AutoStart 取消失败被记录

- **WHEN** `engine.Stop(id)` 返回非 nil 错误
- **THEN** 维护者 MUST 看到 `log.Warn` 输出含该 node id
- **AND** HTTP 响应 MUST 仍为 200
- **AND** 抓包记录 MUST 包含错误信息字段