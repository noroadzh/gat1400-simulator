# 提案：spa-root-routing

## 为什么做

当前 BFF 将 Vue SPA 挂载在 `/ui/*` 下，且从 `/` 做 302 重定向到 `/ui/`，这在功能上等同于两跳 UX（`/` → `/ui/` → 应用）。同时它也缺少真正的 SPA fallback：`/ui/dashboard`（一个不存在的文件）会从 `http.FileServer` 返回 `404 page not found`，而不是回退到 `index.html`。Change 3 即将迁移到 Docker / nginx，且 SPA 注定要落到根路径，本 change 将 SPA 移到 `/` 并在 BFF 中实现正确的 fallback。

## 变更内容

- **移除** `internal/ui/server.go::installRoutes` 中的 `/ui/*` 路由处理器
- **移除** `GET /` 重定向到 `/ui/`
- **新增** 一个挂在 `/` 上的 catch-all SPA 处理器：
  - 尝试以文件形式服务 `internal/ui/dist/<request-path>`
  - 对于任何不以文件形式存在的路径，回退到 `internal/ui/dist/index.html`
  - 保留 `/api/*` 与 `/ws/*` 路由，它们在 catch-all 之前注册，Echo 会优先派发
- **新增** BFF 单元测试，覆盖：
  - `GET /` 返回 SPA index HTML
  - `GET /dashboard`（未知路径）返回 SPA index HTML（fallback）
  - `GET /api/control/nodes` 返回 JSON，不受 catch-all 影响
  - `GET /ws/events` 升级为 WebSocket，不受 catch-all 影响
- **破坏性变更**：任何访问 `http://host/ui/*` 的外部客户端将收到 SPA fallback（仍是 200 + index.html）而不是 `404`。这是有意的——SPA 仅作为临时安排居于 `/ui/` 之下。

## 能力

### 修改的能力

- **web-bff**（`openspec/specs/web-bff/spec.md`）：将 "BFF MUST 在 / 路径服务嵌入的 Vue3 SPA" 需求细化为显式要求 SPA fallback（任何非 API/WS 路径返回 `index.html`）。新增两个场景："根路径返回 SPA 入口" 与 "未知前端路径回退到 SPA 入口"。

### 新增能力

无。

## 影响

- **代码**：`internal/ui/server.go`（路由挂载顺序），`internal/ui/server_test.go`（新增测试）
- **API**：`/api/*` 与 `/ws/*` 路径不变；`/` 与 `/<anything>` 的语义变化
- **部署**：Change 3 引入 nginx 后，BFF catch-all 不再为生产流量所需（nginx 直接服务 SPA）。它仍保留以支持 `make dev` 与单二进制部署。