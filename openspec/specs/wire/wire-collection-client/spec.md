# wire/wire-collection-client Specification

## Purpose
为 UAC 提供 12 Kind 业务对象集合的推送、查询与分页列表客户端，使模拟器能对真实平台持续灌入与拉取 Faces/Persons 等资源以做全协议压测。

## Requirements

### Requirement: Collection 子类型 MUST 支持按 Kind 推送

`Client.Collection()` MUST 提供按资源 Kind 向 `/VIID/<Collection>` POST 标准列表包装 body 的方法，Kind 与集合名映射 MUST 与领域 `CollectionOf` 一致。

#### Scenario: 推送 Face 列表

- **WHEN** 调用方对 Face Kind POST 合法 `FaceList` body
- **THEN** 客户端 MUST POST 到 `/VIID/Faces` 且 Content-Type 为 `application/VIID+JSON`

### Requirement: Collection 子类型 MUST 支持按 ID 的 GET/PUT/DELETE

对每个支持的 Kind，Collection 子类型 MUST 提供 `/VIID/<Collection>/:id` 的 GET、PUT、DELETE。

#### Scenario: 更新后删除对象

- **WHEN** 调用方对同一 ID 依次 PUT 再 DELETE
- **THEN** 两次请求 MUST 分别使用 PutJSON 与 DeleteJSON 路径且成功时无 error

### Requirement: Collection 子类型 MUST 支持分页列表查询

列表 GET MUST 支持查询参数 `PageRecordNum` 与 `RecordStartNo`，并解析响应中的 `TotalNum`（若存在）与对象列表。

#### Scenario: 分页拉取

- **WHEN** 调用方以 `PageRecordNum=10`、`RecordStartNo=0` 列表查询
- **THEN** 客户端 MUST 将参数附加到 URL 并返回解析后的对象切片与 TotalNum（缺失时 TotalNum 可为 0）

### Requirement: Collection 高层方法 MUST 复用 Digest 透明重试

所有 Collection 方法 MUST 仅通过底层 PostJSON/GetJSON/PutJSON/DeleteJSON 发请求，不得绕过 Digest 重试。

#### Scenario: 401 后重试成功

- **WHEN** 对端首次返回 Digest 401 挑战
- **THEN** 客户端 MUST 自动重试一次且调用方仅看到最终结果
