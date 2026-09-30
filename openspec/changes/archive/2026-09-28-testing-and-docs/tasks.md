## 任务

### 任务：Golden 样本文件

- [ ] `test/contract/golden/register.json` —— Digest 握手 401 → 200
- [ ] `test/contract/golden/persons_post.json` —— POST + GET + DELETE
- [ ] `test/contract/golden/subscribes.json` —— Subscribe CRUD
- [ ] `test/contract/golden/catalog.json` —— APE/APS/Tollgate/Lane GET
- [ ] `test/contract/golden/keepalive.json` —— User-Identify keepalive

### 任务：Golden 样本运行器

- [ ] `test/contract/golden_test.go` —— 加载 `*.json`，回放，比对
- [ ] 标志：使用 `--golden` 启用（默认无 golden 文件则跳过）
- [ ] 不一致时报告 diff

### 任务：E2E 测试：protocol_e2e_test.go

- [ ] `net.Listen` 一个真实 TCP 端口
- [ ] 在该端口上启动 `NewServer`
- [ ] 创建指向该端口的 `wire.NewClient`
- [ ] Register → 200，PostJSON → 200，验证 ResourceStore
- [ ] `defer listener.Close()` 清理

### 任务：E2E 测试：capture_e2e_test.go

- [ ] 启动带 CaptureMiddleware 的协议服务器
- [ ] 发送一个带 body 的 POST
- [ ] 查询 CaptureStore → 验证其中包含该请求 body 的条目

### 任务：Makefile 目标

- [ ] `make test-golden` —— 运行 golden 样本测试
- [ ] `make test-e2e` —— 运行 e2e 测试
- [ ] `make test-all` —— 单元 + golden + e2e

### 任务：文档文件

- [ ] `docs/ARCHITECTURE.md` —— 系统概览、组件图（ASCII）
- [ ] `docs/PROTOCOL.md` —— 路由表、请求/响应形态、错误码
- [ ] `docs/USER_GUIDE.md` —— 快速上手、配置 YAML、场景 YAML 语法
- [ ] `docs/OPERATIONS.md` —— 部署、TLS、监控、日志
- [ ] `docs/TESTING.md` —— 运行测试、golden 样本、e2e、CI
- [ ] `docs/CHANGELOG.md` —— 发布说明（与 openspec/CHANGELOG.md 镜像）

### 任务：OpenSpec delta 规范，覆盖全部 7 个 change

- [x] bootstrap-scaffold
- [x] domain-models
- [x] adapter-httpapi
- [x] adapter-wire
- [x] scenario-engine
- [x] web-control-plane
- [x] testing-and-docs（proposal/design/specs 已写；tasks 与文档待补）

### 任务：主 spec 同步

- [ ] `openspec/specs/domain.md` —— 合并的 domain-model 需求
- [ ] `openspec/specs/adapter-httpapi.md` —— 合并的协议需求
- [ ] `openspec/specs/adapter-wire.md` —— 合并的客户端需求
- [ ] `openspec/specs/scenario.md` —— 合并的引擎需求
- [ ] `openspec/specs/web-bff.md` —— 合并的 BFF 需求
- [ ] `openspec/specs/testing.md` —— 合并的测试需求

### 验证

- `go test ./test/...` → exit 0
- `go test -cover ./internal/...` → 全部包 ≥ 90%
- 所有 `docs/*.md` 文件存在且非空
- 所有 `openspec/specs/*.md` 文件存在且非空