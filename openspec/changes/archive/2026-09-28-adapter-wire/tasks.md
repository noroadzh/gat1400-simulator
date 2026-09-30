## 任务

### 任务：定义 Client 结构体与 PostJSON

- [x] `client.go` —— `Client` 结构体、`NewClient(...)`、`PostJSON(ctx, path, body) (*http.Response, error)`
- [x] 所有 POST/PUT 设置 `Content-Type: application/VIID+JSON`
- [x] 附带 `User-Identify` 请求头

### 任务：实现 parseChallenge

- [x] `digest.go::parseChallenge(header string) (challenge, error)`
- [x] 处理带引号的值与逗号分隔的指令
- [x] 返回类型化结构体：Realm、Nonce、Qop、Opaque、Algorithm

### 任务：实现 buildAuthorization

- [x] `digest.go::buildAuthorization(method, uri, username, password, realm, nonce, qop, nc, cnonce) string`
- [x] 按 RFC 2617 §3.2.2 计算 HA1、HA2、response
- [x] 返回 `Authorization: Digest ...` 请求头值

### 任务：实现 digestPassword

- [x] `digest.go::digestPassword(username, realm, password) string`
- [x] `MD5(username:realm:password)`，小写十六进制

### 任务：实现 NonceStore

- [x] `nonce_store.go` —— 包装 `storage.NewNonceStore`
- [x] `NonceStore.Issue(nonce, realm)` —— 记录 nonce
- [x] `NonceStore.Issue(nonce, realm)` 在 nonce 已消费时返回错误
- [x] `NonceStore.Close()`

### 任务：PostJSON 接入 401 重试

- [x] `Client.PostJSON` 中首次请求不带认证
- [x] 若 401：解析 WWW-Authenticate，构建 Authorization，重试
- [x] 其他状态码：原样返回

### 任务：单元测试

- [x] `digest_test.go` —— parseChallenge 覆盖度、buildAuthorization 字段存在性
- [x] 按 RFC 2617 测试向量校验 HA1/HA2 值

### 验证

- `go test ./internal/adapter/wire/...` → exit 0
- `go test -race ./internal/adapter/wire/...` → exit 0
- 集成：启动 httpapi 服务端，用 wire 客户端 Register → 200 OK