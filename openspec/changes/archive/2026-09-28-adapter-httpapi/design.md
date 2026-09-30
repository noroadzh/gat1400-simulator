# 设计：GA/T 1400 HTTP API 适配器

## 服务端结构

```
internal/adapter/httpapi/
├── server.go         # Server 结构体、路由注册、生命周期
├── system.go         # /VIID/System/* 处理器
├── collection.go     # /VIID/Persons、Faces、Vehicles 等
├── cascade.go        # /VIID/Subscribes、Notifications、Dispositions
├── catalog.go        # /VIID/APEs、APSs、Tollgates、Lanes
├── middleware.go     # DigestAuth、UserIdentify、Capture
├── binder.go         # VIID+JSON 与 JSON 的自定义 Binder
├── config.go         # AuthConfig、路由路径
└── response.go       # 辅助函数：OKResponse()、NotFound() 等
```

## 路由表

| 方法 | 路径 | 处理器 | 认证 |
|---|---|---|---|
| POST | `/VIID/System/Register` | System/Register | Digest |
| POST | `/VIID/System/UnRegister` | System/UnRegister | Digest |
| POST | `/VIID/System/Keepalive` | System/Keepalive | User-Identify |
| GET  | `/VIID/System/Time` | System/Time | 无 |
| POST/GET/PUT/DELETE | `/VIID/Persons` `/VIID/Persons/:id` | Collection | User-Identify |
| POST/GET/PUT/DELETE | `/VIID/...` 对应 Face、Vehicle、Plate、Image 等 | Collection | User-Identify |
| POST/GET/PUT/DELETE | `/VIID/Subscribes` `/VIID/Subscribes/:id` | Cascade | User-Identify |
| POST/GET | `/VIID/SubscribeNotifications` | Cascade | User-Identify |
| POST/GET/PUT/DELETE | `/VIID/Dispositions` `/VIID/Dispositions/:id` | Cascade | User-Identify |
| GET | `/VIID/APEs` `/VIID/APSs` `/VIID/Tollgates` `/VIID/Lanes` | Catalog | 无 |

## 中间件链

顺序很重要。应用于所有路由的中间件链为：

1. CaptureMiddleware —— 在任何其他逻辑之前记录请求进入与退出
2. UserIdentifyMiddleware —— 提取 `User-Identify` 请求头，调用 `MarkSeen`
3. DigestAuthMiddleware —— 仅用于受保护的 System 路由（Register/UnRegister）
4. （路由处理器）
5. 响应序列化

抓包中间件使用内存中的 `bytes.Buffer` 记录请求体（响应体在请求头被消费之后读取；请求头在消费前拷贝）。

## Digest 认证细节

```
Server:  Digest realm="<r>", nonce="<n>", qop="auth", opaque=""
Client:  Authorization: Digest username="u", realm="r", nonce="n", uri="...",
         qop=auth, nc=00000001, cnonce="<c>", response="<h>", algorithm=MD5
```

期望的响应值计算方式为：
```
MD5(MD5(u:n:r) + ":" + n + ":" + nc + ":" + c + ":" + qop + ":" + MD5(method + ":" + uri))
```

服务端在 SQLite（`storage.NewNonceStore`）中存储 nonce → 时间戳。超过 30s 的 nonce 会被拒绝。带相同 `nc` 的重复 nonce 会被检测并拒绝（重放保护）。

## Binder

Echo 的默认 Binder 能处理 `application/json`，但忽略 `application/VIID+JSON`。我们注册一个自定义 Binder，将两种内容类型都委托给默认 Binder 处理。无需做转换——VIID+JSON 本质就是 JSON，只是 MIME 标记不同。

## 抓包中间件细节

每个请求记录一条 Capture：

```go
type Capture struct {
    NodeID    string
    Direction string // "inbound" or "outbound"
    Method    string
    Path      string
    URL       string
    Remote    string
    Status    int
    Header    http.Header
    Request   io.Reader
    Response  io.Reader
}
```

`Recorder` 持久化到 `ports.CaptureStore`（SQLite）。查询接口通过 BFF（`internal/ui`）暴露，但协议服务端本身不直接提供查询。

## 响应信封

所有协议响应都包含一个顶层的 `ResponseStatus` 对象：

```json
{
  "ResponseStatus": {
    "StatusCode": 0,
    "StatusString": "OK"
  },
  ...domain-specific keys...
}
```

Server 使用辅助函数：

```go
func OK(c echo.Context, body any) error
func Invalid(c echo.Context, msg string) error  // 400
func NotFound(c echo.Context, id string) error  // 404
func Unauthorized(c echo.Context, realm string) error // 401 + WWW-Authenticate
func ServerError(c echo.Context, err error) error   // 500
```