## 任务

### 任务：BFF Server 骨架

- [x] `internal/ui/server.go` —— `Server` 结构体、`NewServer`、`Start`、`Shutdown`
- [x] 配置 echo 的默认中间件（Logger、Recover、CORS）

### 任务：System 端点

- [x] `GET /api/control/system/health` —— 返回服务信息
- [x] `GET /api/control/system/info` —— 返回 revision、Go version、uptime
- [x] `GET /api/control/stats` —— 聚合计数

### 任务：Nodes 端点

- [x] `GET/POST/DELETE /api/control/nodes`
- [x] `GET /api/control/nodes/:id`
- [x] 绑定：JSON → domain.Node 转换的 `bindNode`

### 任务：Scenarios 端点

- [x] `GET /api/control/scenarios`
- [x] `GET /api/control/scenarios/:id`
- [x] `POST /api/control/scenarios/:id/start|stop`

### 任务：Resources 端点

- [x] `GET /api/control/resources/:kind` —— 读取自 `httpapi.ResourceStore`

### 任务：Subscriptions 端点

- [x] `GET/POST/PUT/DELETE /api/control/subscriptions`

### 任务：Captures 端点

- [x] `GET /api/control/captures?limit=N`
- [x] `GET /api/control/captures/export/jsonl`
- [x] `GET /api/control/captures/export/har`

### 任务：WebSocket Hub

- [x] `internal/ui/hub.go` —— 带广播器的 Hub，Register/Unregister 客户端
- [x] `internal/ui/api_control.go::handleWS` —— 升级连接，注册进 hub

### 任务：前端嵌入

- [x] `frontend.go` —— `web/dist/` 的 `embed.FS`
- [x] SPA fallback handler

### 任务：前端构建

- [x] `web/package.json` —— Vue 3 + ElementPlus + Vite
- [x] `web/vite.config.ts` —— 输出到 `web/dist/`
- [x] 6 个页面组件：Dashboard、Nodes、Scenarios、Resources、Subscriptions、Captures

### 任务：单元测试

- [x] `internal/ui/server_test.go` —— 覆盖全部 BFF 端点
- [x] 覆盖 health、info、stats、CRUD 往返、导出端点
- [x] 覆盖 hub 广播机制

### 验证

- 在 `web/` 下 `pnpm build` 产出 `dist/`
- `go test ./internal/ui/...` → exit 0
- 浏览器可加载 `http://localhost:19000/` 并看到 dashboard