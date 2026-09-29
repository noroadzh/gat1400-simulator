# 测试指南 — GA/T 1400 协议模拟器

> 文档配套代码版本：v0.2.0
> 适用读者：参与本项目开发的工程师

## 一、运行测试

### 1.1 全部测试

```bash
make test
# 等价于：go test ./...
```

### 1.2 仅单元测试（internal 包）

```bash
go test ./internal/...
```

### 1.3 黄金样本测试

```bash
go test -v ./test/contract/...
```

### 1.4 端到端测试

```bash
# e2e 测试使用真实 socket；建议最后跑
go test -v ./test/e2e/...
```

### 1.5 包含黄金样本 + e2e 的完整测试

```bash
make test-all
```

### 1.6 启用 race detector

```bash
go test -race ./...
```

## 二、测试组织

```
test/
├── contract/                # 契约 / 黄金样本测试
│   ├── golden/                # JSON 线级样本（*.json）
│   │   ├── register.json
│   │   ├── persons_post.json
│   │   ├── subscribes.json
│   │   ├── catalog.json
│   │   └── keepalive.json
│   └── golden_test.go        # 运行器：把 JSON 样本回放到真实服务
│
└── e2e/                      # 端到端测试（真实 TCP socket）
    ├── protocol_e2e_test.go   # Register → 推送 → 校验状态
    └── capture_e2e_test.go    # Capture 中间件端到端
```

```
internal/
├── domain/         # 单元测试：ids_test, node_test, resource_test 等
├── adapter/         # 单元 + 集成测试
│   ├── httpapi/    # 集成：httptest.Server + 真实处理器
│   ├── wire/       # 单元：digest_test
│   ├── storage/    # 集成：临时 SQLite
│   ├── scenario/   # 单元：loader_test, engine_test
│   └── capture/    # 单元：recorder_test
├── app/            # 单元测试：config_test, services_test
└── ui/             # BFF API 集成
```

## 三、黄金样本（Golden Samples）

黄金样本是**真实协议交互的 JSON 线级录制**，用于校验协议服务端在每条路由上的正确性。

### 3.1 新增黄金样本

1. 启动模拟器并加载一个已知场景
2. 用控制台的"抓包视图"记录请求 / 响应
3. 将 HTTP 头与 Body 复制到 `test/contract/golden/<name>.json`
4. 运行 `go test -v ./test/contract/...` 校验

### 3.2 样本格式

```json
{
  "description": "人类可读的描述",
  "steps": [
    {
      "direction": "outbound",
      "method": "POST",
      "path": "/VIID/Persons",
      "headers": { "User-Identify": "DEV-1" },
      "body": { "PersonList": { "PersonObject": [...] } }
    },
    {
      "direction": "inbound",
      "status": 200,
      "headers": { "Content-Type": "application/VIID+JSON; charset=UTF-8" },
      "body_shape": {
        "ResponseStatus": { "StatusCode": 0, "StatusString": "OK" }
      }
    }
  ]
}
```

> 说明：`body_shape` 字段用于部分匹配——只校验指定键值，其它键不参与比较。

## 四、E2E 测试

E2E 测试**必须使用真实 TCP socket**。不允许 mock HTTP 层。

### 4.1 TestProtocolE2E_RegisterAndPush

测试流程：

```
net.Listen("tcp", "127.0.0.1:0")   ← 真实 TCP
    │
    ├─ httptest.Server（真实 HTTP）
    │     └─ httpapi.Server（真实处理器）
    │
    └─ wire.Client（真实 http.Client）
          ├─ 发 Register → 401 → 携带 Digest 重试 → 200
          ├─ 发 Person → 200
          └─ 校验 nodeSvc 报告节点为 online
```

### 4.2 TestCaptureE2E_RecordsEveryRequest

校验 CaptureMiddleware 将每个请求持久化到 SQLite `captures` 表，并能通过 BFF 回查。

## 五、覆盖率

### 5.1 生成覆盖率报告

```bash
go test -coverprofile=cover.out ./internal/...
go tool cover -html=cover.out -o cover.html
```

### 5.2 覆盖率门槛

CI 强制要求 `internal/` 下每个包达到 ≥ 90% 行覆盖率。

```bash
go test -cover ./internal/... 2>&1 | grep -E "^(ok|---|\s+github)"
```

## 六、Mock 策略

本项目采用**接口式 Mock**。`internal/app/ports/` 中的接口是应用层与适配器层之间的接缝，测试通过 mock port 实现来隔离。

```go
// 生产代码
type NodeService struct { store ports.NodeStore }

// 测试代码：mock port
type mockNodeStore struct{ nodes []node.Node }
func (m *mockNodeStore) List(ctx context.Context) ([]node.Node, error) {
    return m.nodes, nil
}
// ...
svc := NewNodeService(logger, &mockNodeStore{...})
```

## 七、CI 流水线

CI 按以下顺序运行：

1. **Lint**：`golangci-lint run ./...`
2. **单元测试**：`go test -race ./internal/...`
3. **契约测试**：`go test -v ./test/contract/...`
4. **端到端测试**：`go test -v ./test/e2e/...`
5. **覆盖率门禁**：`go test -cover ./...` —— 任一包 < 90% 即失败

任意一步失败都将阻断合并到 `main`。

## 八、调试技巧

### 8.1 详细输出

```bash
go test -v ./internal/domain/ids/...
```

### 8.2 单个用例

```bash
go test -v -run TestDeviceIDFormat ./internal/domain/ids/...
```

### 8.3 启用 SQLite 调试日志

```bash
DEBUG_SQL=1 go test -v ./internal/adapter/storage/...
```

---

> 下一节建议阅读：[docs/CHANGELOG.md](./CHANGELOG.md)（变更历史）