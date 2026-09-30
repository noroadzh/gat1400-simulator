# 设计：测试矩阵与文档

## Golden 样本（test/contract/golden/）

```
test/contract/golden/
├── register.json       # Digest 握手：401 → Digest 应答 → 200
├── persons_post.json   # POST /VIID/Persons → 200 with ItemCount
├── subscribes.json     # Subscribe CRUD → 200
└── catalog.json        # GET /VIID/APEs → 200 with non-empty APEObject
```

每个 JSON 文件是一个自洽的 HTTP 交互：

```json
{
  "description": "System/Register with Digest auth",
  "steps": [
    {
      "direction": "outbound",
      "method": "POST",
      "path": "/VIID/System/Register",
      "headers": {"Content-Type": "application/VIID+JSON", "Authorization": "Digest ..."},
      "body": {"RegisterObject": {"DeviceID": "41000000005030312222"}}
    },
    {
      "direction": "inbound",
      "status": 401,
      "headers": {"WWW-Authenticate": "Digest realm=\"viid\"..."}
    }
  ]
}
```

Golden 测试运行器：
1. 加载 `*.json` 文件
2. 在进程内启动一个真实的协议服务器
3. 回放其中的步骤
4. 比对响应状态码与 body 结构

不一致将被报告为失败。

## E2E 测试（test/e2e/）

```
test/e2e/
├── protocol_e2e_test.go   # 双进程：启动二进制 + 客户端
└── capture_e2e_test.go    # Capture 中间件端到端
```

```go
// protocol_e2e_test.go
func TestProtocolE2E_RegisterAndPush(t *testing.T) {
    // 1. 启动一个全新的协议服务器（httptest.Server 或真实 TCP listener）
    // 2. 创建一个 wire.Client 指向该服务器
    // 3. Client.Register() → 期望 200
    // 4. Client.PostJSON("/VIID/Persons", person) → 期望 200
    // 5. 查询服务器的 ResourceStore → 期望 1 条 person
}
```

E2E 测试使用 `net.Listen`（真实 TCP）来测试完整的 HTTP 栈，包括中间件、header 解析与 body 读取。

## 文档

所有文档都放在 `docs/` 下：

| 文件 | 内容 |
|---|---|
| `ARCHITECTURE.md` | 系统概览、组件图、数据流 |
| `PROTOCOL.md` | GAT 1400.4 协议参考：路由、请求/响应形态、错误码 |
| `USER_GUIDE.md` | 快速上手、配置文件格式、YAML 场景语法 |
| `OPERATIONS.md` | 部署、TLS、监控、日志、升级 |
| `TESTING.md` | 如何运行测试、golden 样本、e2e、CI |
| `CHANGELOG.md` | 逐版本发布说明（与 openspec/CHANGELOG.md 同步） |

`docs/CHANGELOG.md` 通过 `make sync-changelog` 目标由 `openspec/CHANGELOG.md` 自动生成。