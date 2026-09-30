# Spec Delta

## Purpose

定义 GA/T 1400.4 资源对象 API（§5.2 数据服务）的服务端行为契约——12 种资源类型
（Person / Face / MotorVehicle / NonMotorVehicle / Thing / Scene / VideoSlice / Image /
File / Case / VideoLabel / AnalysisRule）的集合批量写入、列表与单条查询、单条更新与
删除、Info/Data 子资源，让外部设备（UAC）与集成平台可按标准信封交换任意类型的结构化
数据对象。

## ADDED Requirements

### Requirement: 资源对象集合路由覆盖 12 种资源类型

服务端 MUST 在 `/VIID/<Collection>` 路径下为 12 种资源类型注册一致的端点集，其中
`<Collection>` 是各资源类型的复数 URI 段（Persons / Faces / MotorVehicles /
NonMotorVehicles / Things / Scenes / VideoSlices / Images / Files / Cases / VideoLabels /
AnalysisRules）。

#### Scenario: 12 种资源类型集合端点全部可用
- **WHEN** 客户端按 12 种 `<Collection>` 复数名向 `/VIID/<Collection>` 发起 POST/GET 请求
- **THEN** 服务端 MUST 返回 `200 OK`，且 body 包裹在标准响应信封
  `{ResponseStatus, <Kind>List: {<Kind>Object: [...]}}` 之内

### Requirement: 资源对象集合批量写入采用标准信封

服务端 MUST 接受标准 GA/T 1400.4 §5.2 列表信封作为 POST 请求体：
```
{
  "<Kind>List": {
    "<Kind>Object": [ {<obj>}, {<obj>}, ... ]
  }
}
```
服务端 MUST 按 `<Kind>ID` 字段（如 `PersonID`、`FaceID`、`MotorVehicleID` 等）从每个对象中
抽取主键并写入存储；缺主键的对象 MUST 被忽略但不影响其它对象的写入。

#### Scenario: 标准信封成功入存储
- **WHEN** 客户端 POST `/VIID/Persons`，body 为 `{"PersonList":{"PersonObject":[{"PersonID":"p1","Name":"Alice"}]}}`
- **THEN** 服务端 MUST 返回 `200 OK`，body `{ResponseStatus:{...}, ItemCount:1}`，且随后
  `GET /VIID/Persons` MUST 包含 `p1`

#### Scenario: 缺主键对象被静默忽略
- **WHEN** 客户端 POST `/VIID/Persons`，body 包含 `{PersonID:"p1"}` 与 `{}`（无主键）两条
- **THEN** 服务端 MUST 仅写入 `p1`，且响应 `ItemCount` MUST 等于 1

### Requirement: 列表查询返回标准信封

服务端 MUST 在 GET `/VIID/<Collection>` 时返回 `{ResponseStatus, <Kind>List: {<Kind>Object:
[...]}}` 信封，且 `<Kind>Object` 数组 MUST 包含该类型当前所有已存储的对象。

#### Scenario: 空集合返回空数组
- **WHEN** 客户端 GET `/VIID/MotorVehicles` 且服务端未存储任何 `MotorVehicle`
- **THEN** 服务端 MUST 返回 `200 OK`，且 `MotorVehicleList.MotorVehicleObject` MUST 为空数组

### Requirement: 单条查询、更新与删除按主键路由

服务端 MUST 在 `/VIID/<Collection>/:id` 注册：
- `GET` — 按 `<Kind>ID` 检索单条；存在返回 `200 OK` + `{ResponseStatus, <Kind>: {<obj>}}`，
  不存在返回 `404 Not Found` + 标准错误信封
- `PUT` — 按 URL 中的 `:id` 替换整条对象，返回 `200 OK`
- `DELETE` — 从存储中删除该主键，返回 `200 OK`（已实现为软删除：从内存 map 移除即可）

#### Scenario: 单条 GET 命中返回对象
- **WHEN** 客户端 GET `/VIID/Persons/p1` 且 `p1` 已存在
- **THEN** 服务端 MUST 返回 `200 OK`，body 形如 `{"ResponseStatus":{...},"Person":{"PersonID":"p1",...}}`

#### Scenario: 单条 GET 未命中返回 404
- **WHEN** 客户端 GET `/VIID/Persons/unknown`
- **THEN** 服务端 MUST 返回 `404 Not Found`，且 body MUST 包含 `ResponseStatus` 错误信息

#### Scenario: DELETE 后再 GET 返回 404
- **WHEN** 客户端 DELETE `/VIID/Persons/p1` 后再 GET `/VIID/Persons/p1`
- **THEN** 服务端 MUST 在第二次 GET 时返回 `404 Not Found`

### Requirement: Info/Data 子资源占位实现

服务端 MUST 在 `/VIID/<Collection>/:id/Info` 与 `/VIID/<Collection>/:id/Data` 注册
GET/PUT/POST/DELETE 端点。当前实现为占位：返回父对象本身（Info）或后续可扩展为元信息
/ 二进制数据载荷。生产实现可在此处扩展为完整 Info 资源对象。

#### Scenario: Info 子资源 GET 返回父对象
- **WHEN** 客户端 GET `/VIID/Persons/p1/Info` 且 `p1` 已存在
- **THEN** 服务端 MUST 返回 `200 OK`，body `{"ResponseStatus":{...},"Info":{<Person>}}`

### Requirement: 主键字段映射按 Kind 区分

服务端 MUST 按 `internal/domain/resource.IDOf(kind)` 映射主键字段名：Person→PersonID、
Face→FaceID、MotorVehicle→MotorVehicleID、NonMotorVehicle→NonMotorVehicleID、Thing→ThingID、
Scene→SceneID、VideoSlice→VideoSliceID、Image→ImageID、File→FileID、Case→CaseID、
VideoLabel→VideoLabelID、AnalysisRule→AnalysisRuleID。

#### Scenario: 12 种 Kind 主键字段一一对应
- **WHEN** 服务端收到 POST `/VIID/<Collection>` 时
- **THEN** 服务端 MUST 用与 `<Collection>` 对应的 `<Kind>ID` 字段抽取主键，并 MUST NOT 退
  而采用其它字段

### Requirement: 入库对象视为不透明 payload

服务端 MUST 把每条资源对象的剩余字段当作不透明 payload 整体保存与回显，不得擅自序列化
/ 重命名 / 丢弃任意第三方自定义字段。第三方平台（海康、大华、宇视等）可在不通知本模拟器
的情况下扩展字段。

#### Scenario: 第三方扩展字段被完整保留
- **WHEN** 客户端 POST `/VIID/Persons` body 含自定义字段 `VendorExt:{foo:"bar"}`
- **THEN** 后续 GET `/VIID/Persons/<id>` MUST 返回的 Person 对象 MUST 包含 `VendorExt` 字段
  且值与写入时一致

### Requirement: 资源对象 API 继承协议端通用行为

资源对象 API MUST 复用协议端通用行为：Digest 认证（POST/PUT/DELETE 须认证，GET 可选认证，
由项目配置决定）、`application/VIID+JSON` 内容类型、错误响应统一信封
`{ResponseStatus:{...}}`，状态码符合 HTTP 语义（200/400/404/500）。

#### Scenario: 缺认证写入被拒
- **WHEN** 客户端无 Digest 凭据 POST `/VIID/Persons`
- **THEN** 服务端 MUST 返回 `401 Unauthorized` 或 `400 Bad Request`（取决于项目认证策略）

#### Scenario: 错误响应统一信封
- **WHEN** 客户端 POST `/VIID/Persons` body 缺 `PersonList` 包裹
- **THEN** 服务端 MUST 返回 `400 Bad Request`，body MUST 包含 `ResponseStatus` 错误字段
  且错误码 MUST 为标准 `CodeInvalid` 系列

### Requirement: 资源对象写入后通知进程内订阅者

服务端 MUST 在每条资源对象成功写入后调用 `ports.NotifyResource(ctx, log, kind, id)` 通知
进程内订阅者（用于 UAC 链路反向通知）。调用失败 MUST 仅记日志、不阻断响应。

#### Scenario: 写入成功触发订阅通知
- **WHEN** 客户端 POST `/VIID/Persons` 写入 `p1` 成功
- **THEN** `ports.NotifyResource("Person", "p1")` MUST 被调用一次；调用返回错误 MUST
  仅记录 `s.log.Warn`，且 MUST NOT 影响 POST 响应的 `200 OK`

### Requirement: 12 种 Kind 路由表稳定性

服务端 MUST 维护一份有序 Kind 列表（`internal/domain/resource.AllKinds`），POST 集合
路由注册与主键字段映射 MUST 与该列表同步；任何新增 Kind MUST 同时更新三处：Kind 常量、
`CollectionOf` 路由段映射、`IDOf` 主键字段映射。

#### Scenario: 新增 Kind 不需要修改注册代码
- **WHEN** 开发者向 `AllKinds` 加入新条目（如 `KindSnapshot`），并相应实现
  `CollectionOf` 与 `IDOf` 两个 case
- **THEN** `/VIID/Snapshots`、`/VIID/Snapshots/:id` 等端点 MUST 自动可用，无需修改
  `registerCollectionRoutes` / `registerDataServiceRoutes` 注册逻辑