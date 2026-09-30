# Tasks

## 1. 协议端契约对齐（OpenSpec + 文档）

- [x] 1.1 在 `openspec/changes/resource-collection-end-to-end/specs/adapter-resource-collection/spec.md` 中落地 9 条 requirements 与对应 scenarios（集合路由覆盖 12 Kind、标准信封、列表信封、单条 CRUD、Info/Data 占位、主键字段映射、不透明 payload、继承通用行为、写入后通知、Kind 路由表稳定性）；完成后用 `openspec instructions specs` 校验 delta 格式（每 Requirement 配 `#### Scenario:`、WHEN/THEN 各 1+）
- [x] 1.2 在 `openspec/changes/resource-collection-end-to-end/specs/web-bff/spec.md` 中增 6 条 ADDED Requirements（BFF 暴露资源端点组、薄透传、Content-Type 透传、不重实现业务、日志、兼容既有行为）；完成后 `openspec status --change resource-collection-end-to-end` 输出 `applyRequires: [tasks]`、`isPlanningComplete: true`
- [x] 1.3 在 `docs/PROTOCOL.md` 新增「资源对象（§5.2）」章节，端点表覆盖 12 种 Kind × POST/GET/PUT/DELETE + Info/Data 子资源、列出标准信封格式与错误码；完成后 `grep -E "POST.*VIID/(Persons|Faces|MotorVehicles)" docs/PROTOCOL.md` 返回 ≥ 4 行

## 2. BFF 控制面实现

- [x] 2.1 在 `internal/ui/server.go` 中为 `Server` 加 `protocolBaseURL string` 字段并在 `NewServer` 中从配置项 `protocol.listen`（默认 `http://127.0.0.1:14000`）初始化；完成后 `go vet ./internal/ui/...` 通过
- [x] 2.2 新增 `internal/ui/resources.go` 文件：实现 `GET /api/control/resources` handler，遍历 `resource.AllKinds` 生成 12 项 `ResourceKindMeta{kind, collection, idField, description, count}` 元数据（count 通过对协议端串行 12 次 `GET /VIID/<Collection>` 获取），加 5s 内存缓存；完成后 `go build ./...` 通过
- [x] 2.3 在 `internal/ui/resources.go` 中实现透传 handler：`GET /api/control/resources/:kind/list`、`GET /api/control/resources/:kind/list/:id`、`POST /api/control/resources/:kind/list`、`PUT /api/control/resources/:kind/list/:id`、`DELETE /api/control/resources/:kind/list/:id`、`GET /api/control/resources/:kind/list/:id/info`：用 `net/http` 转发到 `protocolBaseURL + /VIID/<Collection>...`，保留请求体与响应状态码；非 12 种合法 `:kind` 立即返回 400；完成后 `go vet ./internal/ui/...` 通过
- [x] 2.4 在 `internal/ui/server.go` `registerRoutes` 中调用 `s.registerResourceRoutes()` 注册上述 7 条路由；完成后 `go build ./...` 通过
- [x] 2.5 在 `internal/ui/server_test.go` 新增 5 个单元测试：`TestResourceAPI_ListAllKinds`（12 项返回）、`TestResourceAPI_GetByKind_Success`、`TestResourceAPI_GetByKind_NotFound`（协议端 404 透传）、`TestResourceAPI_CreateAndDelete_RoundTrip`、`TestResourceAPI_InvalidKind_Returns400`；用 `httptest.Server` mock 协议端响应；完成后 `go test -race -count=1 ./internal/ui/...` 全绿

## 3. 前端 API 客户端与视图

- [x] 3.1 在 `web/src/api/control.ts` 中新增 `ResourceKindMeta` 与 `ResourceObject` 两个 TS interface，以及 `listResourceKinds()`、`listResources(kind)`、`getResource(kind, id)`、`createResource(kind, body)`、`updateResource(kind, id, body)`、`deleteResource(kind, id)`、`getResourceInfo(kind, id)` 七个方法（沿用既有 `request<T>` 封装）；完成后 `cd web && npx vue-tsc --noEmit` 通过
- [x] 3.2 在 `web/src/api/resources-meta.ts`（新文件）落 Kind 中文描述表，作为 BFF 返回 description 的前端 fallback；完成后 `cd web && npx vue-tsc --noEmit` 通过
- [x] 3.3 重写 `web/src/views/ResourcesView.vue`：从 64 行硬编码表改为数据驱动——顶部 12 卡片网格（`el-card` + glass-card 样式 + 计数 badge，URL `?kind=Persons` 同步选中状态）、选中 Kind 后展开 `el-table`（ID 列 + 原始 JSON 列）、顶部操作栏（POST 测试 `el-dialog`、搜索框、分页）、行内 `el-popconfirm` 删除按钮；URI 列从 `/VIAS/api/v1/<Resource>` 改为 `/VIID/<Collection>`；完成后 `cd web && npx vue-tsc --noEmit && npm run build` 通过
- [x] 3.4 在 `web/src/api/control.ts` 中确认 `listResourceKinds` 返回值兼容 BFF 信封 `{kinds: [...]}`；前端组件不要假设 `ResourceObject` 必有 `Name` 字段（按不透明 payload 约定）；完成后 `cd web && npm run build` 通过

## 4. 端到端验证与归档

- [x] 4.1 跑 `go test -race -count=1 ./...`，全部 `ok`；`cd web && npx vue-tsc --noEmit && npm run build`，无报错且生成 `web/dist/index.html`；完成后两条命令均无错误输出
- [x] 4.2 用 `subagent:code-reviewer` 审查整个 change（`internal/ui/resources.go`、`internal/ui/server.go`、`web/src/views/ResourcesView.vue`、`web/src/api/control.ts`、`docs/PROTOCOL.md`、proposal/design/specs 四个文件），重点检查 BFF 透传是否丢状态码、ResourcesView 资源泄漏（WebSocket listener、el-dialog 实例）、spec delta 头部合规（`## ADDED Requirements`、`### Requirement:`、`#### Scenario:` 与 WHEN/THEN）；完成后 reviewer 报告 `critical=0`、warning ≤ 3
- [x] 4.3 跑 `openspec validate resource-collection-end-to-end --strict`，无错误输出，退出码 0；`openspec archive resource-collection-end-to-end` 归档成功，`openspec/specs/adapter-resource-collection/spec.md` 与 `openspec/specs/web-bff/spec.md` 合并完成；完成后 `ls openspec/changes/archive/` 出现 `2026-09-29-resource-collection-end-to-end`
- [x] 4.4 `git add . && git commit -m "feat(resource): 资源对象 API 端到端（BFF + 前端 + spec）" && git push origin main`；完成后 `git log --oneline -1` 显示新 commit，`git status` 干净