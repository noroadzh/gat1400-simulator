# 设计：带 Digest 认证的 HTTP 客户端

## 客户端结构

```
internal/adapter/wire/
├── client.go    # Client 结构体、http.Client 封装、PostJSON
├── digest.go    # parseChallenge、buildAuthorization、digestPassword
└── nonce_store.go # 在 storage.NonceStore 之上的 NonceStore 包装
```

```go
type Client struct {
    BaseURL   string        // 例如 "http://localhost:19001"
    NodeID    string        // User-Identify 的值
    User      string        // Digest 用户名
    Password  string        // Digest 密码
    HTTP      *http.Client  // 标准库，可配置超时
    nonces    *NonceStore   // SQLite 持久化的 nonce 存储
    idGen     ids.Generator
    log       *slog.Logger
}
```

## 认证流程

```
Client.PostJSON(ctx, "/VIID/System/Register", body)

  Step 1: POST /VIID/System/Register (不带 Authorization)
  ← 401 WWW-Authenticate: Digest realm="viid", nonce="abc", qop="auth", opaque=""

  Step 2: 解析 challenge → NonceStore.Issue(nonce, serverRealm)
  Step 3: 构建 Authorization 请求头（RFC 2617 §3.2.2）
  Step 4: POST /VIID/System/Register (带 Authorization)
  ← 200 OK + ResponseStatus.StatusCode=0
```

## Digest 计算

```
HA1  = MD5(username ":" realm ":" password)
HA2  = MD5(method ":" requestURI)
resp = MD5(HA1 ":" nonce ":" nc ":" cnonce ":" qop ":" HA2)
```

参数说明：
- `username`、`password`、`realm` —— 来自 challenge / config
- `nonce` —— 来自服务端 `WWW-Authenticate`
- `nc` —— nonce 计数器，8 位十六进制，每个请求自增（例如 `00000001`）
- `cnonce` —— 客户端 nonce，16 位十六进制，通过 `ids.Nonce()` 按请求生成
- `qop` —— `"auth"`（不支持 auth-int）
- `algorithm` —— `"MD5"`（不支持 MD5-sess）

## Nonce 持久化

NonceStore 包装了 `storage.NonceStore`（SQLite）。服务端重启后，先前签发的 nonce 仍可能有效。客户端会保存自己使用过的 nonce 以避免重放。SQLite 文件位于 `data/nonces.db`。

## PostJSON 签名

```go
func (c *Client) PostJSON(ctx context.Context, path string, body any) (*http.Response, error)
```

请求体进行 JSON 编码。Content-Type 始终为 `application/VIID+JSON`。调用方负责关闭响应体。