# OpenSpec 变更日志

> 本文件记录所有已归档 change 的里程碑。与 `docs/CHANGELOG.md` 镜像，但按 OpenSpec change 粒度组织。

---

## change-007 — 2026-09-28

**ID：** `testing-and-docs`  
**状态：** 已归档

### 变更内容

- `test/contract/golden_test.go` — 黄金样本运行器：将 `test/contract/golden/*.json` 逐一回放到真实 HTTP 服务，校验响应状态码与 body shape
- `test/contract/golden/` — 5 个 JSON 线级样本：register.json / persons_post.json / subscribes.json / catalog.json / keepalive.json
- `test/e2e/protocol_e2e_test.go` — E2E：真 TCP socket，Register → Digest 握手 → 推送 Person → 验证节点 online
- `test/e2e/capture_e2e_test.go` — E2E：真 TCP socket，验证 Capture 中间件持久化
- `docs/` — 6 份中文文档：ARCHITECTURE / PROTOCOL / USER_GUIDE / OPERATIONS / TESTING / CHANGELOG
- `openspec/specs/` — 6 个主规格文件（合并 7 个 delta spec）
- `README.md` — 重写为中文，含完整项目结构树

### 新增 Requirement

- 黄金样本 MUST 覆盖全部 4 类路由（System / Collection / Cascade / Catalog）
- E2E 测试 MUST 使用真实 TCP socket
- E2E 测试 MUST 验证端到端可观测效应（状态、存储、心跳）
- `internal/` 包 MUST ≥ 90% 测试覆盖率
- 文档 MUST 与代码同步

---

## change-006 — 2026-09-27

**ID：** `web-control-plane`  
**状态：** 已归档

### 变更内容

- `internal/ui/server.go` — Echo BFF（`:19000`），含 `/api/control/*` REST 处理器
- `internal/ui/ws.go` — WebSocket Hub，实时推送 node.status / capture.received / scenario.state 事件
- `internal/ui/static.go` — `embed.FS` 前端 bundle
- `internal/ui/dist/` — Vue 3 + Element Plus 构建产物
- `internal/ui/middleware.go` — Auth、CORS 中间件
- `internal/ui/api/` — 节点/场景/资源/订阅/抓包处理器

---

## change-005 — 2026-09-26

**ID：** `scenario-engine`  
**状态：** 已归档

### 变更内容

- `internal/adapter/scenario/loader.go` — `LoadAll(dir)` 读取 `*.yaml` 场景文件
- `internal/adapter/scenario/engine.go` — `ScenarioEngine`，Start / Stop / AutoStart
- `internal/adapter/scenario/dispatcher.go` — 按节点分发到 `NodeService`
- `internal/adapter/scenario/factory.go` — `FakeFactory`（随机数据）/ `StaticFactory`
- `internal/domain/scenario/` — `Scenario` / `NodeSpec` / `ResourceSpec` / `FaultSpec`

### 新增 Requirement

- 场景 MUST 支持 YAML 加载，格式错误返回错误
- 每个节点 MUST 产生恰好一个 `Runnable`
- `ResourceFactory` 对同一 seed 产出确定性 ID
- `FaultInjector` 对同一 seed 产生可复现的故障判定

---

## change-004 — 2026-09-25

**ID：** `adapter-wire`  
**状态：** 已归档

### 变更内容

- `internal/adapter/wire/client.go` — `Client.PostJSON`，Digest 自动重试（401 → 重发）
- `internal/adapter/wire/digest.go` — RFC 2617 Digest 摘要计算
- `internal/adapter/wire/json.go` — `application/VIID+JSON` 编解码
- `internal/adapter/storage/nonce_store.go` — SQLite nonce 持久化（防重放）

### 新增 Requirement

- 客户端 MUST 透明处理 Digest 401 挑战（自动重试一次）
- `cnonce` MUST 使用 `crypto/rand` 生成
- NonceStore MUST 跨进程重启保留
- 重复 nonce+nc MUST 被服务端拒绝
- Digest 比较 MUST 使用 `crypto/subtle.ConstantTimeCompare`

---

## change-003 — 2026-09-24

**ID：** `adapter-httpapi`  
**状态：** 已归档

### 变更内容

- `internal/adapter/httpapi/server.go` — `Server`、路由注册、中间件链
- `internal/adapter/httpapi/system.go` — System 路由（Register/UnRegister/Keepalive/Time）
- `internal/adapter/httpapi/collection.go` — Collection 路由（Person/Face/Vehicle 等 CRUD）
- `internal/adapter/httpapi/cascade.go` — Cascade 路由（Subscribe/Disposition）
- `internal/adapter/httpapi/catalog.go` — Catalog 路由（APE/APS/Tollgate/Lane，静态数据）
- `internal/adapter/httpapi/middleware.go` — DigestAuth、UserIdentify、Capture 中间件
- `internal/adapter/httpapi/registry.go` — 节点 HTTP 监听器注册（多节点）

### 新增 Requirement

- 所有响应 MUST `Content-Type: application/VIID+JSON`
- System Register/UnRegister MUST 强制 Digest 认证
- `User-Identify` 头 MUST 触发节点心跳
- 每个 Kind MUST 支持 POST/GET/DELETE
- Capture 中间件 MUST 异步记录每次请求
- Nonce 重放 MUST 被拒绝

---

## change-002 — 2026-09-23

**ID：** `domain-models`  
**状态：** 已归档

### 变更内容

- `internal/domain/node/node.go` — `Node`、`Role`、`Capability`、`Status`、`Error`
- `internal/domain/resource/resource.go` — `Resource`、`Kind`、`Disposition`
- `internal/domain/scenario/scenario.go` — `Scenario`、`NodeSpec`、`ResourceSpec`、`FaultSpec`
- `internal/domain/response/response.go` — `ResponseStatus`、`Code`
- `internal/domain/ids/ids.go` — `Generator`（DeviceID / Nonce / UUID / SubscribeID）

### 新增 Requirement

- DeviceID MUST 为 20 位十进制字符串
- DeviceID MUST 遵循 `8+2+2+2+6` 布局（SiteCode+IndCode+TypeCode+SubTypeCode+Seq）
- `Resource.Kind` MUST 为已知 7 个枚举值之一
- `ResponseStatus.StatusCode` MUST 为 0–4 枚举
- Nonce MUST 使用 `crypto/rand` 生成
- UUID MUST 符合 RFC 4122 v4
- SubscribeID MUST 为 12 位大写字母数字
- 领域层 MUST 零外部依赖

---

## change-001 — 2026-09-22

**ID：** `bootstrap-scaffold`  
**状态：** 已归档

### 变更内容

- `go.mod` — Go 1.21+，依赖：echo / viper / sqlite / gopacket 等
- `Makefile` — build / test / test-all / clean 目标
- `.golangci.yml` — golangci-lint 配置
- `.github/workflows/ci.yml` — GitHub Actions CI（lint + test + coverage）
- `cmd/gat1400-sim/main.go` — 程序入口