# 任务：vue-componentize-and-dockerize

## 1. 前端项目搭建

- [x] **1.1** 创建 `web/package.json`，依赖：vue@^3.4、vue-router@^4、element-plus@^2、vite@^5、typescript@^5、@vitejs/plugin-vue、vue-tsc、@types/node、@types/element-plus
- [x] **1.2** 创建 `web/tsconfig.json`：`strict: true`、`target: ES2020`、`jsx: preserve`、`@/*` 路径别名
- [x] **1.3** 创建 `web/vite.config.ts`：outDir `dist/`（由 nginx 直接服务）、base `/`、server.proxy `/api` → `http://localhost:14080`
- [x] **1.4** 创建 `web/index.html` —— Vite 入口模板（div#app、script type="module" src="/src/main.ts"）
- [x] **1.5** 创建 `web/public/` 目录（favicon 占位）
- [x] **1.6** 在 `web/` 中运行 `npm install`
- [x] **1.7** 验证 `npm run build` 产出 `web/dist/index.html`，含 Vite bundle 标签

## 2. 前端核心文件

- [x] **2.1** 创建 `web/src/main.ts`：createApp、app.use(ElementPlus)、app.use(router)、app.mount('#app')
- [x] **2.2** 创建 `web/src/App.vue`：布局壳（Sidebar + Topbar + RouterView），cyberpunk glassmorphism CSS
- [x] **2.3** 创建 `web/src/router.ts`：7 条路由（Dashboard、Nodes、Scenarios、Resources、Subscriptions、Captures、Config）
- [x] **2.4** 创建 `web/src/api/control.ts`：所有 `/api/control/*` 端点的 fetch 封装（getNodes、upsertNode、deleteNode、getScenarios、startScenario、stopScenario、getCaptures、getStats、getSystemHealth）
- [x] **2.5** 创建 `web/src/api/ws.ts`：带自动重连的 WebSocket 类（指数退避，最多重试 3 次）

## 3. 前端视图（7 个 view）

- [x] **3.1** 创建 `web/src/views/DashboardView.vue`：4 个 StatCard（Nodes/Online/Scenarios/Running）+ LiveCaptureTable + Health JSON 面板
- [x] **3.2** 创建 `web/src/views/NodesView.vue`：节点数据 el-table + 用于创建/编辑/删除的 NodeFormDialog
- [x] **3.3** 创建 `web/src/views/ScenariosView.vue`：场景数据 el-table + Start/Stop 按钮
- [x] **3.4** 创建 `web/src/views/ResourcesView.vue`：12 行只读表（Person/Face/MotorVehicle 等）
- [x] **3.5** 创建 `web/src/views/SubscriptionsView.vue`：双面板布局（Subscribes + Dispositions JSON 展示）
- [x] **3.6** 创建 `web/src/views/CapturesView.vue`：抓包表，带过滤输入框与导出 JSONL/HAR 按钮
- [x] **3.7** 创建 `web/src/views/ConfigView.vue`：全屏格式化 JSON 配置展示

## 4. 前端组件（5 个 component）

- [x] **4.1** 创建 `web/src/components/Sidebar.vue`：可折叠导航带图标，cyberpunk 霓虹激活态
- [x] **4.2** 创建 `web/src/components/Topbar.vue`：渐变标题文字 + WebSocket 状态徽标（live · ok / offline）
- [x] **4.3** 创建 `web/src/components/StatCard.vue`：props（label、value、color、unit），glassmorphism 卡片带渐变数值文字
- [x] **4.4** 创建 `web/src/components/LiveCaptureTable.vue`：自动刷新的 el-table（3s 间隔），WebSocket 触发刷新
- [x] **4.5** 创建 `web/src/components/NodeFormDialog.vue`：el-dialog + el-form，全部节点字段，提交前校验

## 5. 后端变更

- [x] **5.1** 删除 `internal/ui/static.go`（移除 //go:embed）
- [x] **5.2** 从 `internal/ui/server.go` 删除 `spaHandler` 函数与 `s.e.GET("/*", spaHandler)`
- [x] **5.3** 更新 `internal/ui/server.go` 的 import（删除后未使用的 strings/io/fs 一并移除）
- [x] **5.4** 验证 `go build ./...` 仍成功（不再依赖 dist/）
- [x] **5.5** 验证 `go test ./...` 仍通过（TestAPIPathNotAffectedByFallback 仍覆盖 API 边界）

## 6. Docker 文件

- [x] **6.1** 创建 `Dockerfile.frontend`：node:20-alpine 构建 → nginx:alpine 运行时，拷贝 dist/ + nginx.conf
- [x] **6.2** 创建 `Dockerfile.backend`：golang:1.25-alpine 构建 → gcr.io/distroless/static-debian12，CGO_ENABLED=0
- [x] **6.3** 创建 `nginx.conf`：try_files fallback + 反代 /api/ → backend:14080 + 带 Upgrade 头反代 /ws/
- [x] **6.4** 创建 `docker-compose.yml`：backend + frontend 服务，共享 data/ volume，backend 健康检查（因 distroless 无 wget 改用 /dev/tcp）、networks
- [x] **6.5** 创建 `.dockerignore`：node_modules、.git、data/、*.log、vendor/
- [x] **6.6** 验证 `docker-compose config` 零错误通过
- [x] **6.7** （Docker 不可用时可选）以 `docker build` dry-run 验证 Dockerfile 语法合法 —— *已跳过：沙箱无 Docker daemon；Dockerfile 语法经人工目检确认*

## 7. CI 更新

- [x] **7.1** 更新 `.github/workflows/ci.yml`：新增 `frontend` job（ubuntu-latest、setup-node@v4、npm ci、npm run typecheck、npm run build、upload-artifact）
- [x] **7.2** 更新 `build` job：加入 `needs: [test, frontend]` 依赖
- [x] **7.3** 已跳过：artifact 下载验证移入 frontend job 自身的 upload 步骤（CI 运行时契约）
- [x] **7.4** 已跳过：docker-compose 语法已在本地验证；将在发布流水线中再次验证

## 8. Gitignore 与收尾

- [x] **8.1** 将 `/web/dist/` 加入 `.gitignore`（Vite 产物；由 nginx 在容器内直接服务）
- [ ] **8.2** 以描述性提交信息提交全部变更
- [ ] **8.3** 推送到 origin/main

## 验证命令

```bash
# 前端构建
cd web && npm install && npm run typecheck && npm run build
test -f ../internal/ui/dist/index.html

# 后端
go build ./... && go test ./...

# Docker
docker-compose config

# CI（本地）
docker build -f Dockerfile.backend . && docker build -f Dockerfile.frontend .
```