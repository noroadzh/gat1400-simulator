# 提案：带 Digest 认证的 HTTP 客户端

## 状态
已归档 —— 已实现为协议客户端适配器。

## 背景动机

模拟器需要扮演 GA/T 1400 中的 **设备**（UAC），通过 HTTP 向平台（UAS）推送资源。平台对 `/VIID/System/Register` 与 `/VIID/System/UnRegister` 启用了 HTTP Digest 认证。客户端需要：

1. 发送请求 → 收到 `401 Unauthorized` 及 `WWW-Authenticate: Digest realm="...", nonce="...", qop="auth", opaque="..."`
2. 计算 Digest 响应值
3. 携带 `Authorization: Digest ...` 重试请求
4. 在 nonce 有效期内持久化以复用

## 目标

- 认证对调用方完全透明：调用方只看到成功的响应
- nonce 持久化（SQLite），重启后仍然有效
- 每个请求携带 `User-Identify` 请求头
- 每个请求携带 `Content-Type: application/VIID+JSON`

## 非目标

- 不处理 TLS（由部署层负责——nginx sidecar）
- 不在 stdlib HTTP transport 之外做连接池
- 不支持 multipart 上传

## 待定问题

无。