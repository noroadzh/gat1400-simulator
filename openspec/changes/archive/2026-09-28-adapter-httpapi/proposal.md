# 提案：GA/T 1400 HTTP API 适配器

## 状态
已归档 —— 已实现为协议 REST 路由器。

## 背景动机

GA/T 1400.4 定义了一套包含四大路由族的 REST API：
- `/VIID/System/...` —— 注册（Register）、注销（UnRegister）、保活（Keepalive）、时间（Time）
- `/VIID/<Resource>...` —— 资源采集（Person、Face、Vehicle 等）
- `/VIID/Subscribes`、`/VIID/SubscribeNotifications`、`/VIID/Dispositions` —— 级联
- `/VIID/APEs`、`/VIID/APSs`、`/VIID/Tollgates`、`/VIID/Lanes` —— 目录

我们需要一个协议服务端，具备以下能力：
- 支持 `application/VIID+JSON` 内容类型
- 通过 HTTP Digest（RFC 2617，qop=auth）完成认证
- 接受客户端发送的 `User-Identify` 请求头
- 返回统一的 `ResponseStatus` 响应信封

## 目标

- 基于 Echo 的服务端，每个协议动词对应一条路由
- 自定义 Binder，同时处理 `application/VIID+JSON` 与 `application/json` 两种内容类型
- 带 SQLite 持久化 nonce 存储的 Digest 认证中间件
- 调用 `NodeService.MarkSeen` 的 User-Identify 中间件
- 记录所有入站/出站请求的抓包中间件

## 非目标

- 客户端重试由 `adapter-wire` 负责
- 认证密钥生成由 `app/config` 负责
- TLS 终结（推迟到运维文档处理——通常部署在 nginx 之后）

## 待定问题

无。
