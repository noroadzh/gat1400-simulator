# 协议参考 — GA/T 1400.4

> 适用范围：本文档描述本模拟器实现的 GA/T 1400.4《应用平台接口协议要求》REST 接口。
> 字段命名遵循原标准。错误码、字段名、URL 路径均与官方规范一致。

## 一、基础信息

### 1.1 BaseURL

```
http://<host>:14000/VIID/
```

### 1.2 媒体类型

所有请求与响应的 `Content-Type` 均为：

```
Content-Type: application/VIID+JSON
```

> 备注：自定义媒体类型 `application/VIID+JSON` 由本模拟器在 echo 的 Binder 中识别与解码。

## 二、认证

### 2.1 摘要认证（Digest Auth）

仅 `/VIID/System/Register` 与 `/VIID/System/UnRegister` 需要 HTTP Digest 认证（RFC 2617，qop=auth）。

请求头格式：

```
Authorization: Digest username="admin", realm="com.gat1400.simulator",
    nonce="<handler-side-nonce>", uri="/VIID/System/Register",
    qop=auth, nc=00000001, cnonce="<client-nonce>",
    response="<computed-md5>", opaque=""
```

服务端挑战：

```
HTTP/1.1 401 Unauthorized
WWW-Authenticate: Digest realm="com.gat1400.simulator", qop="auth", nonce="<hex>", opaque=""
```

### 2.2 User-Identify 头

除 Register / UnRegister 之外，其它接口都通过请求头标识调用方：

```
User-Identify: <device-id>
```

服务端据此更新节点 `LastSeenAt` 字段，将节点状态置为 `online`。

## 三、响应与错误码

每个响应都包含顶层 `ResponseStatus` 对象：

```json
{
  "ResponseStatus": {
    "StatusCode": 0,
    "StatusString": "OK",
    "Description": ""
  }
}
```

| StatusCode | StatusString | 含义 |
|---|---|---|
| 0 | OK | 成功 |
| 1 | INVALID | 请求格式错误或缺少必填字段 |
| 2 | NOTFOUND | 引用的资源不存在 |
| 3 | UNAUTHORIZED | 认证失败 / 节点不存在 |
| 4 | SERVER_ERROR | Internal error on the server side |

## 四、路由清单

### 4.1 系统类（System）

| 方法 | 路径 | 认证 |
|------|------|------|
| POST | `/VIID/System/Register` | Digest |
| POST | `/VIID/System/UnRegister` | Digest |
| POST | `/VIID/System/Keepalive` | User-Identify |
| GET  | `/VIID/System/Time` | 无 |

#### POST /VIID/System/Register

请求体：

```json
{
  "RegisterObject": {
    "DeviceID": "41000000005030312222",
    "DeviceName": "Camera-01",
    "Manufacturer": "Hikvision",
    "Model": "DS-2CD3T86FWDV2-I3S",
    "Firmware": "V5.6.11"
  }
}
```

响应：仅包含 `ResponseStatus`（成功时 StatusCode=0）。

#### POST /VIID/System/Keepalive

请求体：

```json
{
  "KeepaliveObject": {
    "DeviceID": "41000000005030312222"
  }
}
```

服务端将节点状态置为 `online`，更新 `LastSeenAt`。

#### GET /VIID/System/Time

返回服务端当前时间（RFC3339）。

```json
{
  "ResponseStatus": { "StatusCode": 0, "StatusString": "OK" },
  "Time": "2026-09-28T10:30:00+08:00"
}
```

### 4.2 资源对象（§5.2 数据服务）

> 通用规则：所有资源对象接口对 12 种 Kind 提供一致的能力——集合批量写入（标准信封）、
> 列表查询、单条查询/更新/删除、Info/Data 子资源。每个 Kind 的路由段是 Kind 的复数形式，
> 主键字段名按 Kind 区分（如 `PersonID`、`MotorVehicleID`）。写入后的对象视为不透明
> payload 整体保存与回显，第三方自定义字段不被丢弃。

#### 4.2.1 支持的 12 种 Kind

| Kind（单数） | URI 段（复数） | 主键字段 | 中文 |
|---|---|---|---|
| `Person` | `Persons` | `PersonID` | 人员 |
| `Face` | `Faces` | `FaceID` | 人脸 |
| `MotorVehicle` | `MotorVehicles` | `MotorVehicleID` | 机动车 |
| `NonMotorVehicle` | `NonMotorVehicles` | `NonMotorVehicleID` | 非机动车 |
| `Thing` | `Things` | `ThingID` | 物品 |
| `Scene` | `Scenes` | `SceneID` | 场景 |
| `VideoSlice` | `VideoSlices` | `VideoSliceID` | 视频片段 |
| `Image` | `Images` | `ImageID` | 图像 |
| `File` | `Files` | `FileID` | 文件 |
| `Case` | `Cases` | `CaseID` | 案件 |
| `VideoLabel` | `VideoLabels` | `VideoLabelID` | 视频标签 |
| `AnalysisRule` | `AnalysisRules` | `AnalysisRuleID` | 分析规则 |

> 主键字段映射实现见 `internal/domain/resource.IDOf(kind)`；新增 Kind 只需同步更新
> `Kind` 常量、`CollectionOf`、`IDOf` 三处，路由注册自动生效。

#### 4.2.2 端点总表

| 方法 | 路径 | 含义 |
|------|------|------|
| POST | `/VIID/<Collection>` | 批量写入（标准信封） |
| GET | `/VIID/<Collection>` | 列表查询 |
| GET | `/VIID/<Collection>/<id>` | 单条查询 |
| PUT | `/VIID/<Collection>/<id>` | 单条更新 |
| DELETE | `/VIID/<Collection>/<id>` | 单条删除（软删除：从内存 map 移除） |
| GET | `/VIID/<Collection>/<id>/Info` | Info 子资源（当前占位：返回父对象） |
| GET | `/VIID/<Collection>/<id>/Data` | Data 子资源（同上占位） |
| POST | `/VIID/<Collection>/<id>/Data` | Data 子资源写入（同上占位） |
| DELETE | `/VIID/<Collection>/<id>/Data` | Data 子资源删除（同上占位） |

`<Collection>` 取自上表的 URI 段列；`<id>` 是该 Kind 的主键字段值。

#### 4.2.3 POST /VIID/<Collection>（批量写入）

请求体采用标准 §5.2 信封：

```json
{
  "PersonList": {
    "PersonObject": [
      {
        "PersonID": "P00000001",
        "Name": "Alice",
        "Gender": "female",
        "ShotTime": "2026-09-28T10:00:00+08:00",
        "DeviceID": "41000000005030312222"
      },
      { "PersonID": "P00000002", "Name": "Bob" }
    ]
  }
}
```

响应：

```json
{
  "ResponseStatus": { "StatusCode": 0, "StatusString": "OK" },
  "ItemCount": 2
}
```

- 缺主键的对象会被**静默忽略**，不影响其它对象的写入；`ItemCount` 仅统计成功写入的数量。
- 写入成功后服务端会调用 `ports.NotifyResource` 通知进程内订阅者；通知失败仅记日志，
  不阻断响应。

#### 4.2.4 GET /VIID/<Collection>（列表查询）

返回当前该 Kind 已存储的全部对象：

```json
{
  "ResponseStatus": { "StatusCode": 0, "StatusString": "OK" },
  "PersonList": { "PersonObject": [ {"PersonID":"P00000001",...}, ... ] }
}
```

空集合时 `PersonObject` 为空数组 `[]`，不返回 404。

#### 4.2.5 GET /VIID/<Collection>/<id>（单条查询）

- 命中：返回 `200 OK` + `{ResponseStatus, <Kind>: {<obj>}}`（单数信封）。
- 未命中：返回 `404 Not Found`，body 含 `ResponseStatus` 错误信封，状态码 2（NOTFOUND）。

#### 4.2.6 PUT /VIID/<Collection>/<id>（单条更新）

请求体为整条对象，按 URL 中的 `<id>` 替换存储。响应：

```json
{ "ResponseStatus": { "StatusCode": 0, "StatusString": "OK" } }
```

#### 4.2.7 DELETE /VIID/<Collection>/<id>（单条删除）

软删除：从内存 map 中移除该主键。响应 `200 OK`。删除后再 GET 返回 404。

#### 4.2.8 Info / Data 子资源

Info 与 Data 子资源为占位实现：GET 时返回父对象本身（外层包 `Info` 字段），POST/PUT/DELETE
均接受并返回 `200 OK`。生产实现可在此处扩展为完整元信息或二进制数据载荷。

#### 4.2.9 错误码

| 场景 | HTTP 状态 | StatusCode |
|---|---|---|
| body 缺 `<Kind>List` 包裹 | 400 | 1（INVALID） |
| body 缺 `<Kind>Object` 数组 | 400 | 1（INVALID） |
| 单条 GET 未命中 | 404 | 2（NOTFOUND） |
| 写入成功 | 200 | 0（OK） |

### 4.3 级联类（Cascade）

| 方法 | 路径 | 用途 | 形态 |
|------|------|------|------|
| POST | `/VIID/Subscribes` | 创建订阅（布控任务） | 创建 / Body 删除（双形态） |
| GET  | `/VIID/Subscribes` | 订阅列表 | - |
| DELETE | `/VIID/Subscribes/<id>` | 取消订阅 | 路径删除 |
| POST | `/VIID/SubscribeNotifications` | 平台推送通知 | - |
| POST | `/VIID/Dispositions` | 订阅命中后的告警上报 | 创建 / Body 删除（双形态） |
| DELETE | `/VIID/Dispositions/<id>` | 撤销告警 | 路径删除 |

#### Subscribe 双形态删除说明

`POST /VIID/Subscribes` 同时支持创建与 body 删除两种语义，**优先级为删除优先**：

- **创建**：body 含 `SubscribeList` ⇒ 创建订阅（§5.4 GA/T 1400.4）
- **删除**：body 含 `SubscribeIDList`（顶层）或 `DeleteOperate.SubscribeIDList`（嵌套）⇒ 取消指定订阅
- **路径删除**：`DELETE /VIID/Subscribes/<id>` 与 body 删除结果等价（同一 ID 产生相同状态码 200）

> 字段判别顺序：`SubscribeIDList` / `DeleteOperate.SubscribeIDList` ⇒ 删除；`SubscribeList` ⇒ 创建。同一 body 同时包含两者时以删除优先（防止误删）。

#### Disposition 双形态删除说明

`POST /VIID/Dispositions` 与 `/VIID/Subscribes` 同理：

- **删除**：`DispositionIDList`（顶层）或 `DeleteOperate.DispositionIDList`（嵌套）
- **路径删除**：`DELETE /VIID/Dispositions/<id>` 与 body 删除等价

#### POST /VIID/Subscribes

```json
{
  "SubscribeList": {
    "SubscribeObject": [
      {
        "SubscribeID": "SUB00000001",
        "Title": "男性布控",
        "Resource": "Person",
        "SubscribeMethod": "notify",
        "Filter": "Gender=male",
        "Timeout": 86400
      }
    ]
  }
}
```

#### POST /VIID/Dispositions

```json
{
  "DispositionList": {
    "DispositionObject": [
      {
        "DispositionID": "DISP001",
        "SubscribeID": "SUB00000001",
        "MatchObject": { "PersonID": "P001" },
        "AppearTime": "2026-09-28T10:30:00+08:00"
      }
    ]
  }
}
```

### 4.4 目录类（Catalog）

> 备注：以下路由不需要任何认证，由模拟器静态提供。

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/VIID/APEs` | 卡口实体（Access Point Entity） |
| GET | `/VIID/APSs` | 卡口服务器 |
| GET | `/VIID/Tollgates` | 收费站 |
| GET | `/VIID/Lanes` | 车道 |

响应示例：

```json
{
  "ResponseStatus": { "StatusCode": 0, "StatusString": "OK" },
  "APEList": {
    "APEObject": [
      { "APEID": "41000000000000000001", "Name": "卡口001" }
    ]
  }
}
```

## 五、DeviceID 编码规则

DeviceID 为 20 位十进制字符串，按 `8+2+2+2+6` 切分：

| 位置 | 长度 | 含义 | 示例 |
|------|------|------|------|
| 0–8 | 8 | 区划码（SiteCode） | `41000000` |
| 8–10 | 2 | 行业代码（IndustryCode） | `30` |
| 10–12 | 2 | 设备类型（TypeCode） | `01` |
| 12–14 | 2 | 子类型（SubTypeCode） | `01` |
| 14–20 | 6 | 设备序号（Sequence） | `000001` |

完整示例：

```
41000000 30 01 01 000001
└─Site──┘ └Ind┘ └T─┘ └Sub┘ └Seq─┘
```

## 六、Nonce 与重放保护

- 服务端 nonce 为 32 位小写十六进制字符串，使用 `crypto/rand` 生成
- 默认有效期 30 秒
- 客户端每次请求必须递增 `nc`（nonce counter）
- 服务端按 `nonce` 单次消费语义去重（`NonceStore.Consume`），同一 nonce 仅允许被消费一次

### 6.1 重放请求响应示例

**请求**（重放一次已被消费的 Authorization 头）：

```
POST /VIID/System/Register HTTP/1.1
Host: 192.168.1.10:14000
Authorization: Digest username="admin",realm="com.gat1400.simulator",
   nonce="9c4f2a8b1e3d7c5f6a9b2e4d8c1f3a5b7e9d2c4f6a8b1e3d5c7f9a2b4e6d8c1f",
   uri="/VIID/System/Register",qop=auth,nc=00000001,cnonce="ab12cd34ef56ab78",
   response="<ha1:ha2 digest>",opaque=""
```

**响应**（401，附带新的 WWW-Authenticate challenge，提示上一 nonce 已消费）：

```
HTTP/1.1 401 Unauthorized
WWW-Authenticate: Digest realm="com.gat1400.simulator",
   qop="auth,auth-int",
   nonce="<new 32-hex>",
   opaque="",
   algorithm=MD5,
   stale=false
Content-Type: application/json

{"ResponseStatus":{"StatusCode":1403,"Brief":"nonce already consumed"}}
```

### 6.2 客户端处理流程

1. 客户端发起无 Authorization 头的请求 ⇒ 服务端返回 401 + 新 nonce
2. 客户端用 `(HA1, HA2, nonce, nc, cnonce, qop)` 计算 `response`
3. 客户端重发请求 + Authorization 头（`nc=00000001`）
4. 服务端 `NonceStore.Consume(nonce)` 验证成功 ⇒ 业务逻辑处理
5. 若客户端尝试重用同一 nonce（如 nc 未递增或回放旧请求）⇒ 401 + `nonce already consumed`

> 注：`stale=false` 表明身份凭证本身有效，问题出在 nonce 已被使用。客户端必须获取新 nonce 后重试，不应中断登录状态。

---

> 下一节建议阅读：[docs/USER_GUIDE.md](./USER_GUIDE.md)（场景 / 配置 / 接口使用）