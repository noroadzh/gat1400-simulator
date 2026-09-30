# Proposal

## Why

GA/T 1400.4 资源对象 API（12 种 Kind × POST/GET/PUT/DELETE + Info/Data 子资源）在
`internal/adapter/httpapi/collection.go` 已经实现并有单元测试覆盖，但这一整块能力在项目
其他三个层面都是缺位的：

1. **OpenSpec 治理缺位**：`openspec/specs/` 没有任何主 spec 描述资源对象 API 的行为契约，
   已归档的 12 个 change 也没有产出过对应 delta，导致这块能力处于"代码存在但规范缺失"状态。
2. **协议文档缺位**：`docs/PROTOCOL.md` 端点表只列了 System 端点、Catalog 四类目录
   （APEs/APSs/Tollgates/Lanes）、Subscribes、Dispositions，完全没有 12 种 Resource 集合的
   端点与响应信封格式。
3. **BFF 控制面缺位**：`internal/ui/server.go` 没有 `/api/control/resources*` 端点，Web UI
   或外部工具无法通过控制面浏览、注入、删除资源对象。
4. **前端缺位**：`web/src/views/ResourcesView.vue` 是 64 行硬编码静态表，URI 列写的是
   `/VIAS/api/v1/<Resource>`——这个前缀属于另一份标准（VIAS 视图接入协议），与本项目实现的
   `/VIID/<Collection>` 完全不一致；表中的 12 个类型名也与 `internal/domain/resource.AllKinds`
   枚举对不上（多出 TrafficEquipment/APE/Alarm，少 Cases/VideoSlices/VideoLabels/AnalysisRules），
   而且整张表不连BFF，数字永远是假的。

现在把这条链路端到端补齐，让"资源对象"从只有协议端实现变成可以用、看得见、可被规范约束的
完整能力，同时修掉前端把 `/VIAS/` 写进 URI 列的协议认知错误。

## What Changes

- **新增 OpenSpec 主 spec `adapter-resource-collection`**：把已实现的 12 种 Kind 资源对象 API
  的行为契约显式写进规范——集合批量写入（标准信封）、列表查询、单条查询/更新/删除、
  Info/Data 子资源、12 种 Kind 全覆盖、主键字段校验、认证要求、错误码。
- **新增 BFF 控制面端点组 `/api/control/resources/*`**（薄透传，HTTP 转发到协议端
  `/VIID/<Collection>`），暴露：
  - `GET /api/control/resources` — 列出 12 种 Kind 元数据（kind / collection / idField / 描述 / 计数）
  - `GET /api/control/resources/:kind/list` — 列表查询
  - `GET /api/control/resources/:kind/list/:id` — 单条查询
  - `POST /api/control/resources/:kind/list` — 批量写入（标准信封）
  - `PUT /api/control/resources/:kind/list/:id` — 单条更新
  - `DELETE /api/control/resources/:kind/list/:id` — 单条删除
  - `GET /api/control/resources/:kind/list/:id/info` — Info 子资源
  - 协议端 4xx/5xx 与响应 body 原样透传，不解析、不改写
- **BFF 单元测试**：新增资源路由的 200/400/404 覆盖。
- **前端 API 客户端**：`web/src/api/control.ts` 新增 `listResourceKinds` / `listResources` /
  `getResource` / `createResource` / `updateResource` / `deleteResource` / `getResourceInfo`，
  以及 `ResourceKindMeta` / `ResourceObject` 两个 TypeScript 接口。
- **前端 ResourcesView 重写**：从 12 行死表改为数据驱动——12 个资源类型卡片（带实时计数
  badge，计数从 BFF 返回的真实数据来），选中 Kind 后渲染`el-table` 列表，支持搜索、
  分页、原始 JSON 查看器、行内删除（`el-popconfirm`）、以及"POST 测试"对话框（手动注入
  一条资源对象用于协议联调）。URI 列修正为 `/VIID/<Collection>`。
- **协议文档**：`docs/PROTOCOL.md` 新增「资源对象（§5.2）」端点章节，覆盖 12 种 Kind ×
  POST/GET/PUT/DELETE + Info/Data 子资源、请求/响应信封、字段规则、错误码。
- **不破坏现有行为**：协议端 `collection.go` 已注册的路径与方法完全不动；现有 4 个
  collection 单元测试保留；BFF 旧端点不动；端口依赖不变。

## Capabilities

### New Capabilities
- `adapter-resource-collection`: GA/T 1400.4 资源对象 API（12 种 Kind）的集合批量写入、
  列表/单条查询、单条更新与删除、Info/Data 子资源、主键字段校验与认证要求。

### Modified Capabilities
- `web-bff`: 新增资源对象控制面端点组 `/api/control/resources/*` 的需求——端点矩阵、
  透传语义（状态码与body 原样转发）、Kind 参数校验、错误码映射。

## Impact

**后端（新增）**
- `internal/ui/resources.go`（新文件）— BFF 资源端点组 handler（约 200 行）
- `internal/ui/server.go` — 新增路由注册（约 5 行）+ 协议端 base URL 字段
- `internal/ui/server_test.go` — 新增资源端点测试（约 150 行）
- `internal/app/ports/*.go` — 可能需要 `ResourceReader` port（若不直连协议端）
- `internal/adapter/httpapi/*.go` — **不改**（已实现）

**前端（修改）**
- `web/src/api/control.ts` — +7 方法、+2 interface
- `web/src/views/ResourcesView.vue` — 重写 64 → 约 300 行
- `web/src/styles/global.css` — 可能需要新增卡片网格与 JSON 查看器样式

**文档（修改）**
- `docs/PROTOCOL.md` — 新增「资源对象」章节
- `docs/ARCHITECTURE.md` — 可能需要更新 BFF 端点清单（若已列出）
- `docs/USER_GUIDE.md` — 可能需要更新 BFF API 清单（若已列出）
- `docs/CHANGELOG.md` — 新增条目

**OpenSpec（新增/修改）**
- `openspec/changes/resource-collection-end-to-end/` — proposal / design / tasks / specs delta
- `openspec/specs/adapter-resource-collection/spec.md`（archive 时新建）
- `openspec/specs/web-bff/spec.md`（archive 时合并 ADDED Requirements）

**兼容性**
- 无 BREAKING：`/VIID/<Collection>` 协议端路径与方法不变；BFF 只增端点；前端只改内部实现。
- 依赖不变，无新端口（沿用 14080 控制面 / 14000 协议端），无新 Go/npm 依赖。

**性能**
- BFF 透传是 O(请求体) 的单次 HTTP 转发，无额外序列化层；前端 12 卡片 + 按需加载列表，
  首屏只拉一次 `/api/control/resources`。
