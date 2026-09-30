## 任务

### 任务：定义 Server 结构体

- [x] 创建 `internal/adapter/httpapi/server.go`
- [x] 定义 `Server`，包含 `e *echo.Echo` 及依赖（NodeService、ScenarioService、Recorder、NonceStore、IDGenerator）
- [x] 实现 `NewServer(log, nodeSvc, scenSvc, rec, nonce, idGen, cfg)`
- [x] 实现 `Start(addr) error` 和 `Shutdown(ctx) error`

### 任务：实现 System 路由

- [x] `POST /VIID/System/Register` —— Digest + 请求体 `RegisterObject.DeviceID`，调用 `nodeSvc.MarkSeen`
- [x] `POST /VIID/System/UnRegister` —— Digest，标记节点为已停止
- [x] `POST /VIID/System/Keepalive` —— User-Identify，调用 `MarkSeen`
- [x] `GET /VIID/System/Time` —— 返回 RFC3339 时间戳

### 任务：实现 Collection 路由

- [x] 对每种 Kind（Person、Face、Vehicle、Plate、NonMotorVehicle、Image、Object）实现 POST/GET/PUT/DELETE
- [x] 资源存储使用 kind 分桶 map
- [x] `/Info`、`/Data` 子路由用于元数据 blob

### 任务：实现 Cascade 路由

- [x] `/VIID/Subscribes` CRUD
- [x] `/VIID/SubscribeNotifications` POST + GET（带 `subscribeId` 查询参数）
- [x] `/VIID/Dispositions` CRUD

### 任务：实现 Catalog 路由

- [x] `/VIID/APEs` GET → 固定数据列表
- [x] `/VIID/APSs`、`/VIID/Tollgates`、`/VIID/Lanes` GET → 固定数据列表

### 任务：实现 Digest 中间件

- [x] `middleware.go::DigestAuth(realm, username, password, qop, nonceStore)`
- [x] 解析 `Authorization` 请求头
- [x] 计算期望响应值，使用常量时间比较
- [x] 缺失/无效认证时返回 401 + WWW-Authenticate

### 任务：实现 UserIdentify 中间件

- [x] `middleware.go::UserIdentify(nodeSvc)`
- [x] 提取 `User-Identify` 请求头
- [x] 调用 `MarkSeen`，未知节点返回 404

### 任务：实现 Capture 中间件

- [x] `middleware.go::CaptureMiddleware(recorder, defaultNodeID)`
- [x] 缓冲请求体，调用 `Next()`，然后记录 capture
- [x] 无 default 时回退到请求头中的 NodeID

### 任务：自定义 Binder

- [x] `binder.go::NewVIBinder`，处理 `application/VIID+JSON` 和 `application/json`

### 任务：响应辅助函数

- [x] `response.go::OK`、`Invalid`、`NotFound`、`Unauthorized`、`ServerError`

### 任务：集成测试

- [x] `internal/adapter/httpapi/server_test.go`
- [x] 覆盖全部四个路由族
- [x] 验证缺失认证时触发 Digest 挑战
- [x] 验证 collection/cascade 的 CRUD 往返
- [x] 验证 catalog 每个路由族至少返回一条数据

### 验证

- `go test ./internal/adapter/httpapi/...` → exit 0
- 所有路由返回 `application/VIID+JSON` Content-Type
- Digest 中间件拒绝使用已消费 nonce 的请求