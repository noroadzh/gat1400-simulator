# 设计：Web 控制面

## BFF 布局

```
internal/ui/
├── server.go          # Server 结构体、BFF 生命周期、路由注册
├── api_control.go     # /api/control/ 下的全部 REST handler
├── hub.go             # WebSocket Hub、广播器、客户端注册
├── api_bindings.go    # BFF 请求的 JSON ↔ 领域模型绑定
└── frontend.go        # Vue3 SPA 的 embed.FS handler
```

## REST 路由

| Endpoint | Method | Description |
|---|---|---|
| `/api/control/system/health` | GET | `{status, service, version}` |
| `/api/control/system/info` | GET | `{revision, goVersion, uptime}` |
| `/api/control/stats` | GET | 聚合计数：nodes、scenarios、resources |
| `/api/control/nodes` | GET | 列出全部节点 |
| `/api/control/nodes` | POST | 创建节点 |
| `/api/control/nodes/:id` | GET | 获取节点 |
| `/api/control/nodes/:id` | DELETE | 删除节点 |
| `/api/control/scenarios` | GET | 列出场景 |
| `/api/control/scenarios/:id` | GET | 获取场景 |
| `/api/control/scenarios/:id/start` | POST | 启动场景 |
| `/api/control/scenarios/:id/stop` | POST | 停止场景 |
| `/api/control/resources/:kind` | GET | 浏览资源集合 |
| `/api/control/subscriptions` | GET/POST/PUT/DELETE | 管理订阅 |
| `/api/control/captures?limit=N` | GET | 最近捕获 |
| `/api/control/captures/export/jsonl` | GET | 下载 JSONL |
| `/api/control/captures/export/har` | GET | 下载 HAR |

## WebSocket Hub

一个端点 `/api/control/ws` 接受 WebSocket 升级。服务器维护一份已连接客户端列表并广播事件：

```go
type Event struct {
    Type    string         `json:"type"`     // node.status, capture.received, scenario.state, alert
    Payload map[string]any `json:"payload"`
}
```

事件来源：
- `nodeSvc.SetSyncHook` —— 节点状态变化
- `httpapi.CaptureMiddleware` —— 捕获已存储
- `scenSvc` —— 场景启停
- 级联通知 handler —— 告警/处置匹配

Hub 使用 mutex 保护的切片；广播是非阻塞的（每个客户端一个 channel）。

## 前端嵌入

```go
//go:embed all:dist
var distFS embed.FS
```

前端由 `pnpm build` 构建到 `web/dist/`，在编译期被嵌入。BFF 在 `/` 下提供 `dist/`，并带有 SPA fallback（任何未知路径返回 `index.html`）。

## 前端页面

1. **Dashboard** —— 健康卡片、节点计数、场景计数、最近活动流（实时）
3. **Scenarios** —— 场景卡片列表、状态徽章、启停按钮
4. **Nodes** —— 节点表格，含状态、能力、操作（删除、查看）
5. **Resources** —— 资源集合的 Tab 化列表，支持 type/kind 过滤
6. **Subscriptions** —— 列出与创建订阅，编辑判定条件
7. **Captures** —— 最近捕获的分页表格，含 method/path/status 过滤；"view" 打开侧边面板显示请求/响应 body

前端使用 ElementPlus 组件（Table、Tag、Dialog、Form、Tabs）。

## 生命周期

```
Server.Start(addr string) error
  ├── 在 goroutine 中启动 echo
  ├── 启动 hub goroutine
  └── 立即返回

Server.Shutdown(ctx context.Context) error
  ├── 关闭 echo（排空 HTTP 请求）
  ├── 关闭 hub（排空 WS 客户端）
  └── 返回
```

BFF 默认监听 `:19000`。协议服务器监听 `:19001`。