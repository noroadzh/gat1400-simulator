## 任务

> 注：以下勾选为归档后核对产物回填，任务均已在归档前实际完成。

### 任务：Golden 样本文件

- [x] `test/contract/golden/register.json` —— Digest 握手 401 → 200
- [x] `test/contract/golden/persons_post.json` —— POST + GET + DELETE
- [x] `test/contract/golden/subscribes.json` —— Subscribe CRUD
- [x] `test/contract/golden/catalog.json` —— APE/APS/Tollgate/Lane GET
- [x] `test/contract/golden/keepalive.json` —— User-Identify keepalive

### 任务：Golden 样本运行器

- [x] `test/contract/golden_test.go` —— 加载 `*.json`，回放，比对
- [x] 标志：使用 `--golden` 启用（默认无 golden 文件则跳过）
- [x] 不一致时报告 diff

### 任务：E2E 测试：protocol_e2e_test.go

- [x] `net.Listen` 一个真实 TCP 端口
- [x] 在该端口上启动 `NewServer`
- [x] 创建指向该端口的 `wire.NewClient`
- [x] Register → 200，PostJSON → 200，验证 ResourceStore
- [x] `defer listener.Close()` 清理

### 任务：E2E 测试：capture_e2e_test.go

- [x] 启动带 CaptureMiddleware 的协议服务器
- [x] 发送一个带 body 的 POST
- [x] 查询 CaptureStore → 验证其中包含该请求 body 的条目

### 任务：Makefile 目标

- [x] `make test-golden` —— 运行 golden 样本测试
- [x] `make test-e2e` —— 运行 e2e 测试
- [x] `make test-all` —— 单元 + golden + e2e

### 任务：文档文件

- [x] `docs/ARCHITECTURE.md` —— 系统概览、组件图（ASCII）
- [x] `docs/PROTOCOL.md` —— 路由表、请求/响应形态、错误码
- [x] `docs/USER_GUIDE.md` —— 快速上手、配置 YAML、场景 YAML 语法
- [x] `docs/OPERATIONS.md` —— 部署、TLS、监控、日志
- [x] `docs/TESTING.md` —— 运行测试、golden 样本、e2e、CI
- [x] `docs/CHANGELOG.md` —— 发布说明（与 openspec/CHANGELOG.md 镜像）

### 任务：OpenSpec delta 规范，覆盖全部 7 个 change

- [x] bootstrap-scaffold
- [x] domain-models
- [x] adapter-httpapi
- [x] adapter-wire
- [x] scenario-engine
- [x] web-control-plane
- [x] testing-and-docs（proposal/design/specs 已写；tasks 与文档随后已补齐，归档时一并核对回填）

### 任务：主 spec 同步

> 注：当前 `openspec/specs/` 已按模块分目录（domain/、adapter-httpapi/、adapter-wire/、scenario/、web-bff/、testing/ 等），需求已合并；此处勾选按当时约定追溯回填。

- [x] `openspec/specs/domain.md` —— 合并的 domain-model 需求
- [x] `openspec/specs/adapter-httpapi.md` —— 合并的协议需求
- [x] `openspec/specs/adapter-wire.md` —— 合并的客户端需求
- [x] `openspec/specs/scenario.md` —— 合并的引擎需求
- [x] `openspec/specs/web-bff.md` —— 合并的 BFF 需求
- [x] `openspec/specs/testing.md` —— 合并的测试需求

### 验证

- `go test ./test/...` → exit 0
- `go test -cover ./internal/...` → 全部包 ≥ 90%
- 所有 `docs/*.md` 文件存在且非空
- 所有 `openspec/specs/*.md` 文件存在且非空