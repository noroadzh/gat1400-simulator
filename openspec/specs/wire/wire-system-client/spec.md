# Spec: wire-system-client

## Purpose

提供 `wire.Client` 的高层 System 类 API（Register / UnRegister / Keepalive / ServerTime），让上层调用方不必关心 HTTP 路径与 body 构造细节，复用既有的 Digest 401 自动重试、User-Identify 头与 `application/VIID+JSON` 序列化。

## Requirements

### Requirement: System.Register MUST 提交 RegisterObject 到 /VIID/System/Register

`Client.System()` 返回的 `System` 子类型 MUST 提供 `Register(ctx, baseURL, RegisterObject)` 方法，内部构造标准 `RegisterObject` body 并 POST 到 `<baseURL>/VIID/System/Register`，自动附带 `User-Identify` 头与 Digest 认证；返回 200 时 MUST 不返回 error。

#### Scenario: 首次注册遭遇 Digest 401

- **WHEN** 调用 `System.Register` 访问服务端 `/VIID/System/Register`，服务端首次返回 401 + Digest challenge
- **THEN** 客户端 MUST 自动重试一次并提交合法 `Authorization: Digest ...` 头
- **AND** 调用方 MUST 仅收到最终成功响应（无 error）。

#### Scenario: DeviceID 必填且与 User-Identify 一致

- **WHEN** `RegisterObject.DeviceID` 为 20 位合法 DeviceID
- **THEN** 出站 HTTP 请求的 `User-Identify` 头 MUST 等于该 DeviceID
- **AND** 注册成功后服务端 MUST 将节点标记为 online。

### Requirement: System.UnRegister MUST 提交 UnRegisterObject 到 /VIID/System/UnRegister

`System` 子类型 MUST 提供 `UnRegister(ctx, baseURL, deviceID)` 方法，内部 POST 到 `<baseURL>/VIID/System/UnRegister`，body 形如 `{ "UnRegisterObject": { "DeviceID": "<id>" } }`。

#### Scenario: 注销设备节点

- **WHEN** 调用 `System.UnRegister` 携带合法 DeviceID
- **THEN** 客户端 MUST 携带与 DeviceID 一致的 `User-Identify` 头
- **AND** 服务端 MUST 在收到合法注销请求后返回 200。

### Requirement: System.Keepalive MUST POST 到 /VIID/System/Keepalive

`System` 子类型 MUST 提供 `Keepalive(ctx, baseURL, deviceID)` 方法，内部 POST 到 `<baseURL>/VIID/System/Keepalive`，body 为空。

#### Scenario: 心跳请求不强制 Digest 挑战

- **WHEN** 调用 `System.Keepalive`，服务端未返回 Digest challenge
- **THEN** 客户端 MUST 不进行 401 重试，直接透传响应
- **AND** 调用方 MUST 仅收到最终响应。

#### Scenario: 节点被标记为 online

- **WHEN** 调用 `System.Keepalive` 收到 200
- **THEN** 服务端 MUST 在收到心跳后将对应节点状态置为 online。

### Requirement: System.ServerTime MUST GET /VIID/System/Time

`System` 子类型 MUST 提供 `ServerTime(ctx, baseURL)` 方法，内部 GET `<baseURL>/VIID/System/Time`，返回 RFC3339 形式的时间字符串。

#### Scenario: 服务端时间同步

- **WHEN** 调用 `System.ServerTime`
- **THEN** 返回值 MUST 包含 RFC3339 时间戳（含 `T` 分隔符）
- **AND** 网络或解析错误 MUST 通过 error 返回给调用方。
