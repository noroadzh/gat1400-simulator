# Design

> 配套 proposal.md（动机）与 specs/（行为契约）。本文聚焦"如何实现"。

## Context

**当前状态**
- 协议端 `internal/adapter/httpapi/collection.go` 已为 12 种 Kind 注册了
  `/VIID/<Collection>` × POST/GET/PUT/DELETE + `/Info` + `/Data` 路由，并通过
  `internal/domain/resource.AllKinds` 有序枚举 + `CollectionOf/IDOf` 映射实现"新增 Kind
  只需加三个常量"的扩展机制。`server_test.go` 4 个用例 + `go test -race` 全绿。
- `internal/adapter/httpapi/catalog.go` 注册了 4 类目录端点（APEs/APSs/Tollgates/Lanes），
  与 collection 端点共用同一个 Echo 实例，但走的是"内存预填充 + 目录只读"模型。
- BFF `internal/ui/server.go` 当前暴露 5 类端点：`/api/control/nodes/*` /
  `/api/control/scenarios/*` / `/api/control/captures*` / `/api/control/config` /
  `/api/control/subscriptions`。**没有** resources 端点。
- 前端 `web/src/views/ResourcesView.vue` 是 64 行硬编码静态表，URI 列硬写
  `/VIAS/api/v1/<Resource>`——这是把另一份标准（视图接入协议）的 URI 错当成了本项目
  GA/T 1400.4 协议 URI。表中 12 个类型名也对不上 `AllKinds`。
- OpenSpec 主 spec 9 个均无资源对象契约；`openspec/changes/archive/` 12 个归档 change 也
  没产出过对应 delta。

**约束**
- 不改协议端 `collection.go`（已实现并测试稳定）。
- 不引入新端口 / 新依赖。
- 沿用 `internal/domain/resource.AllKinds` 作为唯一权威枚举来源。
- 沿用项目既定分层纪律：app 不依赖 internal/adapter；BFF 只与协议端通过 HTTP 通信，
  不直连协议端的存储或 handler 内部结构。

## Goals / Non-Goals

**Goals**
- 把协议端资源对象 API 通过 BFF `/api/control/resources/*` 端点组以薄透传方式暴露
  给前端。
- 在 `web-bff` 主 spec 与新建的 `adapter-resource-collection` 主 spec 中分别定义
  BFF 端点契约与协议端契约，让 OpenSpec 治理闭环。
- 重写前端 `ResourcesView.vue` 为数据驱动，修复 URI 前缀错引与 Kind 名称错位。
- 在 `docs/PROTOCOL.md` 增补资源对象端点章节。

**Non-Goals**
- 不实现真实持久化（沿用协议端内存存储，进程重启即丢失，与 catalog 一致）。
- 不实现 Info/Data 子资源的完整语义（保留协议端"占位回显父对象"行为）。
- 不实现分页/排序/过滤参数（沿用协议端"返回全量列表"语义；如未来需要，由独立
  change 处理）。
- 不实现 `/VIAS/` 别名路由（保持 `/VIID/` 一份真源）。
- 不实现 WebSocket 推送资源变化事件（沿用既有 `s.hub` 事件机制，不新增事件类型）。

## Decisions

### Decision 1: BFF ↔ 协议端通过 HTTP 透传，而非内存直连

**为什么**：项目分层纪律要求"app 不依赖 internal/adapter"。BFF 当前的实现方式就是通过
HTTP（甚至进程内启动一个 Echo 实例监听内部端口）调用协议端。新增资源端点沿用相同模式：
BFF 启动时拿到协议端 base URL（默认 `http://127.0.0.1:14000`，从配置项 `protocol.listen`
推导），handler 用 `net/http` 转发请求并回写响应。

**替代方案考虑**
- A. 在 BFF 进程内直接调用 `s.adapter.HTTPAPI().Resource()` 方法：被否决，破坏分层纪律，
  且让 BFF 与协议端实现耦合，未来协议端重构会牵连 BFF。
- B. 用 `io.Pipe` 把请求体/响应体在两个 Echo 实例之间流式转发（已用于 captures export）：
  对资源端点同样适用——本设计采用此方案的具体实现，因为请求/响应都是 JSON，无需特殊编码。

### Decision 2: Kind 元数据列表 `count` 字段按需获取

**为什么**：`GET /api/control/resources` 需要返回每种 Kind 的当前对象数量。BFF 通过对每种
Kind 各发一次 `GET /VIID/<Collection>` 然后取 `len(<Kind>Object)` 计算——简单可靠，无须
额外存储。但 12 次串行 HTTP 调用在协议端冷启时可能有几十毫秒延迟。

**优化路径**：未来可让协议端在 catalog 端点 `/VIID/System/Status` 中暴露一次性的 kind 计数
map。本次不引入——保持 BFF 端点实现简单（12 次 GET，可接受）。

### Decision 3: 前端 ResourcesView 不引入 pinia/store

**为什么**：路由切换间保留 Kind 选择状态用 `useRoute().query.kind`（URL 即状态）即可。
引入 pinia store 超出本 change 范围，且 Vue Router 已经原生支持 URL 同步。

**替代方案考虑**
- A. 引入 pinia store：被否决，超范围。
- B. 用 `localStorage` 保留：被否决，URL 同步体验更好（可分享/收藏）。

### Decision 4: BFF 资源 handler 文件独立 `resources.go`

**为什么**：资源端点组 7 条路由 × 各 kind 12 种透传逻辑相对集中，独立文件便于维护与
审查。沿用 `catalog.go` 之于"目录端点"的拆分风格。

### Decision 5: 前端 ResourcesView 卡片网格用 CSS Grid

**为什么**：12 个 Kind 用 CSS Grid `repeat(auto-fill, minmax(160px, 1fr))` 自适应布局，
桌面 6 列、平板 3 列、手机 1 列。沿用 `Dashboard` 的 glass-card 风格。

### Decision 6: URI 前缀错引修正

**为什么**：前端旧 URI 列 `/VIAS/api/v1/...` 是历史遗留 bug——把它直接改成
`/VIID/<Collection>` 与协议端实现对齐。不引入 VIAS 别名路由（保持一份真源，参见 Non-Goals）。
真要兼容 VIAS 客户端可在 nginx 层重写，与本 change 无关。

## Risks / Trade-offs

- **[Risk] BFF 与协议端同时启动时序**：BFF 启动时协议端可能尚未就绪 → **Mitigation**：
  BFF handler 在转发请求失败（连接拒绝）时返回 `502 Bad Gateway` + 标准错误信封，前端
  `ResourcesView` 顶部 banner 显示 "Protocol server unreachable"。
- **[Risk] 12 次串行 GET 计算 count 的延迟**：在大型数据集上可能 >100ms → **Mitigation**：
  BFF 给 `/api/control/resources` 加 5s 内存缓存；缓存键为 protocol base URL。后续可改
  为协议端一次性提供 count map。
- **[Risk] 前端硬编码 Kind 中文描述与 `domain/resource` 不一致** → **Mitigation**：把
  中文描述表迁到 `web/src/api/resources-meta.ts`（独立文件），BFF 返回 `description` 时
  优先使用后端值（如果未来协议端补 description 字段）；前端保留 fallback 静态表。
- **[Risk] DELETE 透传把已删除对象错误回显**：协议端当前实现为同步内存删除，BFF 透传
  立即返回；不存在不一致窗口 → **无 mitigation 必要**。
- **[Risk] PUT/POST 请求体可能很大（视频切片/文件元数据）**：BFF 透传是 O(body) 单次转发，
  无内存复制 → **Mitigation**：必要时给 BFF 加 `MaxBytesReader` 上限（沿用协议端既有的
  限制）。

## Migration Plan

**部署步骤**（本 change 不涉及线上，无灰度需求）
1. 提交本 change 到 `main` 分支。
2. CI 跑后端 `go test -race` + 前端 `vue-tsc --noEmit` + `npm run build`，均绿后合入。
3. 本地 `make dev` 验证：浏览器打开 `http://localhost:8080/#/resources`，确认 12 卡片、
   计数、列表、POST 测试、删除全部正常。

**回滚**：单 commit 回滚即可，所有改动局限在 `internal/ui/resources.go`（新） +
`internal/ui/server.go`（5 行） + `web/src/views/ResourcesView.vue`（重写） +
`web/src/api/control.ts`（增方法） + `docs/PROTOCOL.md`（增章节），无 schema 变更，无
依赖变更。

**Open Questions**: 无。所有技术决策在本设计内闭合。