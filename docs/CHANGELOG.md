# 变更日志

> 变更管理遵循 OpenSpec 规范（`openspec/` 目录，gitignored）。本文件按用户可读粒度记录。

---

## [未发布]

### feat(bff,web) — 资源对象 API 端到端打通（路线图 #17）

- BFF `/api/control/resources/*` 7 个端点：枚举 12 Kind、`list` / `list/:id` / `POST list` / `PUT list/:id` / `DELETE list/:id` / `info`，全部透传协议端 `/VIID/<Collection>`
- 协议文档 `docs/PROTOCOL.md` §5.2 补齐 12 Kind × 6 端点表（POST/GET/PUT/DELETE + Info/Data），与协议端实现一一对应
- 前端 `ResourcesView.vue` 从硬编码 12 行 `/VIAS/api/v1/...` 重写为：12 卡片网格 + 选中 Kind 拉取真实列表 + POST 测试对话框 + JSON 查看器 + 删除 + 分页 + URI 列修正为 `/VIID/<Collection>`
- 新增 `web/src/api/resources-meta.ts` Kind 中文元数据、`web/src/api/control.ts` 5 个 BFF 调用方法
- 新增 `internal/ui/resources_handler.go` + `server_test.go` 6 个用例覆盖 200/404/400 边界与 query string 透传
- 新建 OpenSpec 主 spec `adapter-resource-collection/spec.md`（8 Requirement 覆盖 12 Kind CRUD/Info/Data 契约），同步 web-bff spec delta

### chore(repo) — 治理 .gitignore 与本地化文档

- `.gitignore` 新增忽略 `.codebuddy/`、`openspec/`、`web/node_modules/`、`web/dist/`、`*.tsbuildinfo`
- 停止跟踪已误提交到 git 的 `.codebuddy/`（22 文件）、`openspec/`（76 文件）、`web/node_modules/`（~11990 文件）
- README、docs/ 全面本地化为中文，端口更新为当前实际值（`:14080` / `:14000`），架构图同步新增 docker-compose 编排视图
- `docs/ARCHITECTURE.md` / `OPERATIONS.md` / `USER_GUIDE.md` / `PROTOCOL.md` 端口全部由 `:1900x` 改为 `:1408x` / `:1400x`，digest realm 由 `viid` 改为 `com.gat1400.simulator`

---

## [v0.2.0] — 2026-09-29

### feat(web) — Vue 3 组件化前端

- 新增 `web/` Vite 工程：Vue 3.4 + `<script setup>` + Element Plus 2 + TypeScript 5
- 拆 7 view + 5 component：`DashboardView` / `NodesView` / `ScenariosView` / `ResourcesView` / `SubscriptionsView` / `CapturesView` / `ConfigView`；`Sidebar` / `Topbar` / `StatCard` / `LiveCaptureTable` / `NodeFormDialog`
- `web/src/api/control.ts` fetch 封装；`web/src/api/ws.ts` WebSocket 自动重连
- 移除 `internal/ui/static.go` 的 `//go:embed all:dist`（BFF 不再嵌入前端）

### feat(docker) — 多阶段容器化

- `Dockerfile.backend`：`golang:1.25-alpine` → `gcr.io/distroless/static-debian12:nonroot`，二进制 ~20MB
- `Dockerfile.frontend`：`node:20-alpine` → `nginx:alpine`，自服务 SPA
- `docker-compose.yml`：`backend:14080` + `frontend:8080` + 共享 `data/` volume + healthcheck
- `nginx.conf`：SPA fallback + `/api/` `/ws/` 反向代理到 backend

### feat(routing) — SPA 根路径

- BFF 移除 `/ui/*` 与 `/` 重定向；非 API/WS 路径回退 `index.html`（SPA fallback）
- `internal/ui/server_test.go` 18 测试全绿（含 `TestSPAUnknownPathReturnsIndexHTML` / `TestAPIPathNotAffectedByFallback` / `TestWebSocketPathNotAffectedByFallback`）

### chore(spec) — 状态对齐

- 承认 archive `web-control-plane` 中未交付任务（package.json / vite.config.ts），不再修改 archive
- 删除已废弃的 `web/` 副本（dist 已在 .gitignore 但实际为空壳）

### CI

- `.github/workflows/ci.yml` 新增 `frontend` job（ubuntu/Node 20/npm ci/typecheck/build/upload-artifact）
- `build` job 改为 `needs: [test, frontend]`

---

## v0.1.0 — 2026-09-28

### 新增

- **bootstrap-scaffold**：项目骨架，含 Go 1.21+ / Makefile / GitHub Actions / golangci-lint / `modernc.org/sqlite`
- **domain-models**：`Node`/`Role`/`Capability`/`Status`、`Resource`/`Kind`、`Subscription`/`Disposition`、`Scenario`、`ResponseStatus`/`Code`、`IDGenerator`（20 位 DeviceID）
- **adapter-httpapi**：完整 GA/T 1400.4 REST API —— System / Collection / Cascade / Catalog 四类路由、Digest 中间件、User-Identify 中间件、Capture 中间件、自定义 VIID+JSON Binder
- **adapter-wire**：HTTP 客户端，支持 RFC 2617 Digest 自动重试、SQLite nonce 持久化、User-Identify 头注入
- **scenario-engine**：YAML 场景加载、`ScenarioEngine.Start/Stop`、`FakeFactory`（随机数据）、`ProbabilityFaultInjector`（delay/drop/reorder/malformed）
- **web-control-plane**：Echo BFF（`:14080` JSON API + WebSocket Hub，Vue3 SPA 通过独立 nginx 容器服务）
- **testing-and-docs**：internal 包 100% 单元测试覆盖、黄金样本测试、真实 TCP socket 的 e2e 测试、6 份文档

### 功能列表

| 功能 | 细节 |
|------|------|
| 协议服务端 | `:14000` —— 4 类路由（System / Collection / Cascade / Catalog） |
| BFF | `:14080` —— JSON API + WebSocket（`/api/control/*` 与 `/ws/events`） |
| 前端 SPA | Vue 3 + Vite，docker-compose 模式由 nginx 自服务（`:8080`） |
| Digest 认证 | RFC 2617 qop=auth，SQLite nonce 重放保护 |
| 抓包 | 每请求记录：NodeID / Method / Path / Status / Header / Body |
| 场景 | YAML 描述拓扑：nodes / resources / subscriptions / faults |
| 异常注入 | `delay` / `drop` / `reorder` / `malformed`，按概率触发 |
| WebSocket | 实时事件：node.status / captures / scenario.started / scenario.stopped |

### 修复

- `randomHex` nonce 生成：改用 `crypto/rand`（旧版本曾误用 `time.Now`）
- capture 测试中的 `defer cr.Close()`：改为 `defer cs.Close()`
- `DigestAuth` 中间件：在所有 System 路由上正确识别 `Authorization` 头存在与否

### 文档

- `docs/ARCHITECTURE.md` —— 组件拓扑图（docker-compose + 进程内部双视图）、数据流、分层纪律
- `docs/PROTOCOL.md` —— 完整路由参考、请求 / 响应、错误码
- `docs/USER_GUIDE.md` —— 快速开始、配置、场景 YAML、Web BFF API
- `docs/OPERATIONS.md` —— Docker Compose 部署、TLS、systemd、监控、备份、升级
- `docs/TESTING.md` —— 运行测试、黄金样本、e2e、覆盖率、CI
- `docs/CHANGELOG.md` —— 本文件

---

> 提示：完整 OpenSpec 变更日志见各归档 change 内的 `tasks.md`（每个 change 含实现细节与验收条目）。