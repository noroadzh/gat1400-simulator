# Proposal: vue-componentize-and-dockerize

## Change Information

- **Change ID**: `vue-componentize-and-dockerize`
- **Date**: 2026-09-29
- **Status**: Planning
- **Priority**: High

## Why

Change 2 (`spa-root-routing`) 已经把 BFF 的 SPA fallback 实现到根路径，但 `internal/ui/dist/index.html` 仍是手写的 380 行 CDN 单文件。这个"单文件 SPA"在以下场景下不可维护：

1. **状态管理混乱**：所有 Vue 逻辑堆在 `<script>` 里，无组件边界，无类型安全
2. **无法做集成测试**：没有 npm 工具链，无 Vitest/Playwright
3. **无法做类型检查**：纯 JS，Element Plus API 用错只能运行时发现
4. **无法做独立部署**：前端资源死绑在 Go 二进制里，不能 nginx/CDN 独立扩缩
5. **违反设计目标**：openspec spec 说 "Vue 3 + Element Plus 构建产物"，实际是手写 CDN 加载

此外，项目完全没有容器化。用户在生产部署时需要手动构建 Go 二进制 + 拷贝配置，没有标准化的 Docker 镜像和 docker-compose 编排。

## What Changes

- **[NEW] `web/` 完整 Vue 3 项目**
  - `package.json`：vue ^3.4, vue-router ^4, element-plus ^2, vite ^5, typescript ^5, @vitejs/plugin-vue, vue-tsc
  - `vite.config.ts`：`build.outDir = '../internal/ui/dist'`, `base: '/'`, 代理 `/api` 到 backend:14080
  - `tsconfig.json`：strict 模式
  - `src/main.ts`、`src/App.vue`、`src/router.ts`
  - `src/api/control.ts`：fetch 封装（/api/control/*）
  - `src/api/ws.ts`：WebSocket 自动重连
  - `src/views/`：Dashboard / Nodes / Scenarios / Resources / Subscriptions / Captures / Config
  - `src/components/`：Sidebar / Topbar / StatCard / LiveCaptureTable / NodeFormDialog
- **[DELETE] `internal/ui/static.go`**（//go:embed 整段移除，前端不再 embed）
- **[MODIFY] `.gitignore`**：`/internal/ui/dist/` 加入（作为 npm 构建产物，不再进 git）
- **[NEW] `Dockerfile.frontend`**：node:20-alpine build → nginx:alpine
- **[NEW] `Dockerfile.backend`**：golang:1.25-alpine build → distroless
- **[NEW] `docker-compose.yml`**：backend:14080 + frontend:8080 + shared `data/` volume
- **[NEW] `nginx.conf`**：SPA fallback + 反代 `/api/`、`/ws/` 到 backend
- **[NEW] `.dockerignore`**
- **[MODIFY] `.github/workflows/ci.yml`**：新增 frontend job（Node 20, npm ci, build, typecheck, upload artifact）

## Capabilities

### New Capabilities

#### `frontend-build`

独立 Vue 3 前端项目，可独立 build 出静态资源。产物路径 `internal/ui/dist/`（作为 npm 产物，gitignored）。

#### `containerization`

Docker 多阶段构建 + docker-compose 编排，标准化生产部署。

## Impact

- **Affected specs**: `web-bff`（embed 部分移除，新增 frontend-build 和 containerization 要求）
- **Affected code**:
  - `internal/ui/static.go` [DEL]
  - `.gitignore` [MOD]
  - `.github/workflows/ci.yml` [MOD]
- **New files**: `web/**`（约 15 个文件）、`Dockerfile.*`、`docker-compose.yml`、`nginx.conf`、`.dockerignore`
- **Breaking changes**: 无（BFF API 契约不变）

## Non-Goals

- 不引入 Pinia / Vuex（状态少，`<script setup>` + ref 够用）
- 不改 BFF REST API 路径（`/api/control/*` 契约保持）
- 不做单元测试（首版：靠 typecheck + build + 手测；后续 change 可加 Vitest）
- 不做多副本 scaling（docker-compose 单机部署足够）

## Success Criteria

1. `cd web && npm ci && npm run build` 产出 `internal/ui/dist/`（包含 index.html + assets/）
2. `go build ./...` 依然成功（Go 不再依赖 embed，dist 只是宿主目录）
3. `go test ./...` 全绿（embed 移除不破坏测试）
4. `docker-compose up` 能拉起 frontend:8080 + backend:14080，浏览器访问 `http://localhost:8080` 能看到 dashboard 并正常调 API
5. `docker-compose config` 语法合法
6. CI 绿：新增 frontend job 跑通 npm ci + build + typecheck

## Design Highlights

- **UI 设计风格**：Cyberpunk Neon + Glassmorphism + Dark Mode（与现有 CDN SPA 保持一致的视觉语言）
- **路由**：vue-router 4，7 个路由对应 7 个 view
- **组件通信**：props/emit 为主；跨组件共享用简单的 composable
- **Element Plus**：全量 import（保持简单；按需 import 留给优化 change）
- **容器化**：两个独立 Dockerfile，frontend 用 nginx:alpine 做静态服务 + 反向代理，backend 用 distroless 最小化攻击面
- **CI**：新增 frontend job 独立跑（与 go test 并行），降低 CI 时间
