# Delta: web-bff

## ADDED Requirements

### Requirement: 抓包明细反序列化失败 MUST 被记录为 Warn

`capture_reader.go` 在 `rows.Scan` 后的 `json.Unmarshal(header)` / `json.Unmarshal(request)` / `json.Unmarshal(response)` 任一返回非 nil error 时 MUST 调用注入的 `*slog.Logger` 写一条 `Warn` 级别日志，键 MUST 包含：

- `event=capture_parse_failed`
- `id`（抓包记录主键）
- `field`（失败字段名：`header` | `request` | `response`）
- `error`

行记录 MUST 仍被返回给调用方（不清行），其余已成功反序列化的字段 MUST 仍可被 BFF 展示。

#### Scenario: 损坏 header 不丢行

- **WHEN** SQLite 中某行 header JSON 损坏（e.g. `header TEXT` 存了 `not-json`）
- **THEN** `capture_reader` MUST 仍返回该行
- **AND** `e.Header` 字段为零值（不 panic、不抛错给调用方）
- **AND** 日志 MUST 含 `event=capture_parse_failed`、`id=<row id>`、`field=header`、`error`

#### Scenario: 三字段独立失败

- **WHEN** 同一行 header / request / response 都损坏
- **THEN** 日志 MUST 出现 3 条独立的 `event=capture_parse_failed`，分别带 `field=header|request|response`
- **AND** 三个字段全为零值，但 `e.ID` / `e.Method` / `e.Path` 等非 JSON 字段 MUST 仍正确
