# Spec Delta

## ADDED Requirements

### Requirement: Cascade.DispositionList MUST GET /VIID/Dispositions

`Cascade` 子类型 MUST 提供 `DispositionList`，GET `<baseURL>/VIID/Dispositions` 并解析 `DispositionList` 包装为对象切片。

#### Scenario: 列出布控

- **WHEN** 对端存在至少一条布控
- **THEN** 返回切片长度 MUST >= 1

### Requirement: Cascade.DispositionDelete MUST 支持按 ID 删除

`Cascade` 子类型 MUST 提供按 DispositionID 删除的方法（DELETE `/VIID/Dispositions/:id` 或协议等价的 DeleteOperate POST），成功时无 error。

#### Scenario: 删除布控

- **WHEN** 调用方删除已存在的 DispositionID
- **THEN** 对端 MUST 返回成功且后续列表不再含该 ID（或 GET by id 为未找到语义）

### Requirement: Cascade.DispositionNotificationPush MUST POST /VIID/DispositionNotifications

`Cascade` 子类型 MUST 提供推送布控命中通知的方法，POST 到 `/VIID/DispositionNotifications`。

#### Scenario: 推送布控通知

- **WHEN** 调用方携带合法 DispositionNotification body
- **THEN** 对端 MUST 收到 POST 且 200 时调用方无 error

### Requirement: Cascade.DispositionNotificationList MUST 支持查询

`Cascade` 子类型 MUST 支持 GET `/VIID/DispositionNotifications`（可带 dispositionId 过滤）并返回通知对象切片。

#### Scenario: 按 dispositionId 列通知

- **WHEN** 调用方以 dispositionId 查询且对端有匹配通知
- **THEN** 返回切片 MUST 非空
