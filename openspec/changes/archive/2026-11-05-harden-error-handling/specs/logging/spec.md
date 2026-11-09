# Delta: logging

## ADDED Requirements

### Requirement: 日志轮转失败 MUST 被记录为 Error

模拟器在 `dailyWriter.maybeRotate` 调用 `lumberjack.Logger.Rotate()` 失败时 MUST 通过注入的 `*slog.Logger` 写一条 `Error` 级别日志，键 MUST 包含：

- `event=log_rotate_failed`
- `error`（可读字符串）
- 可选 `path`（被轮转的文件路径，便于排错）

该日志 MUST 同步写出（持锁内单次调用），但 MUST NOT 阻塞业务日志（已通过 `slog.Error` 的非阻塞语义保证）。

#### Scenario: 轮转失败被记录

- **WHEN** 模拟器跨过本地时间 00:00 且 `lumberjack.Rotate()` 返回非 nil error
- **THEN** 日志输出 MUST 至少出现一条 `Error` 级条目，含 `event=log_rotate_failed` 与 `error` 字段
- **AND** 业务日志 MUST 继续写入（write 路径不中断）

#### Scenario: 轮转成功无日志噪音

- **WHEN** 跨过 00:00 且 Rotate 成功
- **THEN** MUST NOT 出现 `event=log_rotate_failed` 条目（成功路径不写日志）
