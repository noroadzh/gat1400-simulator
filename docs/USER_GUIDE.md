# 用户指南 — GAT 1400 协议模拟器

> 文档配套代码版本：v0.1.0
> 适用读者：使用本模拟器验证平台 / 设备协议栈的工程师、QA、协议研发

## 一、快速开始

### 1.1 编译

```bash
# 要求 Go 1.21+
go build -o bin/gat1400-simulator ./cmd/gat1400-sim/
```

或使用 Makefile：

```bash
make build
```

### 1.2 启动

```bash
./bin/gat1400-simulator --config configs/simulator.yaml
```

模拟器启动后开启两个 HTTP 服务：

| 端口 | 用途 |
|------|------|
| `:19000` | Web 控制面（BFF），可视化界面 |
| `:19001` | 协议服务端（GA/T 1400.4 REST API） |

### 1.3 打开控制台

浏览器访问：

```
http://localhost:19000
```

可视化管理：

- 节点列表与状态（实时 WebSocket 推送）
- 新增 / 删除节点
- 启动 / 停止场景
- 浏览抓包数据

## 二、配置文件

`configs/simulator.yaml`：

```yaml
server:
  protocol: ":19001"   # 协议端口
  control: ":19000"     # Web BFF 端口
  logLevel: "info"      # debug | info | warn | error

auth:
  realm: "viid"
  username: "admin"
  password: "admin"
  qop: "auth"

storage:
  dataDir: "./data"     # SQLite 数据目录

ids:
  siteCode: 41000000
  industryCode: 30

scenarios:
  dir: "./configs/scenarios"
  autoStart: []         # 启动时自动加载的场景 ID 列表
```

### 2.1 环境变量覆盖

任何配置项都可通过环境变量覆盖（命名规则：`GAT1400_<KEY>`）：

```bash
export GAT1400_SERVER_PROTOCOL=":19001"
export GAT1400_AUTH_PASSWORD="secret"
./bin/gat1400-simulator
```

## 三、场景文件

场景使用 YAML 描述节点的拓扑、资源推送规则、订阅布控、异常注入。

### 3.1 最简示例：单设备 + 单平台

`configs/scenarios/minimal.yaml`：

```yaml
id: minimal
name: 最简拓扑 — 1 设备 + 1 平台
schedule:
  autoStart: true
  interval: 10s

nodes:
  - id: "41000000005030312222"
    role: device             # device / platform-small / platform-large
    httpListen: ":19101"
    upstream: "http://localhost:19001"  # 推送目标
    capabilities: [system, collection]  # system | collection | cascade
    tags: [entrance]

  - id: platform-main
    role: platform-large
    httpListen: ":19001"
    capabilities: [system, collection, cascade]

resources:
  - kind: Person             # 资源类型
    source: "41000000005030312222"   # 推送方节点
    interval: 5s              # 推送频率
    factory: fake             # fake | static

faults:                       # 故障注入
  - target: "41000000005030312222"
    type: delay               # delay | drop | reorder | malformed
    probability: 0.05         # 触发概率
    duration: 500ms           # 延迟时长
```

### 3.2 关键字段说明

| 字段 | 取值 | 说明 |
|------|------|------|
| `nodes[].role` | `device` / `platform-small` / `platform-large` | 节点身份 |
| `nodes[].upstream` | URL | device 节点推送到哪里 |
| `nodes[].capabilities` | `system` / `collection` / `cascade` | 支持的能力集 |
| `resources[].kind` | 枚举 | Person / Face / Vehicle / Plate / ... |
| `resources[].factory` | `fake` / `static` | 数据来源 |
| `resources[].interval` | duration | 推送频率 |
| `faults[].type` | `delay` / `drop` / `reorder` / `malformed` | 故障类型 |
| `faults[].probability` | float 0–1 | 触发概率 |

## 四、节点 ID

节点 ID 即 DeviceID，20 位十进制：

```
41000000 30 01 01 000001
└─Site──┘ └Ind┘ └T─┘ └Sub┘ └Seq─┘
```

生成新 ID：

```bash
# 交互式
go run ./cmd/gat1400-sim/ generate-id --site 41000000 --industry 30
```

或在代码中：

```go
idGen := ids.NewGenerator(41000000, 30)
deviceID := idGen.DeviceID() // 例如 "41000000300101000001"
```

## 五、Web BFF API

所有接口位于 `/api/control/` 命名空间下。

| 方法 | 路径 | 说明 |
|------|------|------|
| GET    | `/api/control/system/health` | 服务健康检查 |
| GET    | `/api/control/nodes` | 节点列表 |
| POST   | `/api/control/nodes` | 新增节点 |
| GET    | `/api/control/nodes/:id` | 单个节点 |
| DELETE | `/api/control/nodes/:id` | 删除节点 |
| GET    | `/api/control/scenarios` | 场景列表 |
| POST   | `/api/control/scenarios/:id/start` | 启动场景 |
| POST   | `/api/control/scenarios/:id/stop` | 停止场景 |
| GET    | `/api/control/resources/:kind` | 浏览资源集合 |
| GET    | `/api/control/captures` | 查询抓包 |
| GET    | `/api/control/captures/export/jsonl` | 导出 JSONL |
| GET    | `/api/control/ws` | WebSocket 实时事件 |

## 六、WebSocket 事件

连接 `ws://localhost:19000/api/control/ws`，推送的事件示例：

```json
{ "type": "node.status", "payload": { "id": "DEV-1", "status": "online" } }
{ "type": "capture.received", "payload": { "nodeId": "DEV-1", "method": "POST", "path": "/VIID/Persons" } }
{ "type": "scenario.state", "payload": { "id": "minimal", "state": "running" } }
```

## 七、常用操作

### 7.1 手动注册设备

```bash
curl -X POST http://localhost:19001/VIID/System/Register \
  -H "Content-Type: application/VIID+JSON" \
  -H "Authorization: Digest ..." \
  -d '{"RegisterObject":{"DeviceID":"41000000005030312222"}}'
```

### 7.2 查询抓包

```bash
curl "http://localhost:19000/api/control/captures?limit=10"
```

### 7.3 启动单个场景

```bash
curl -X POST "http://localhost:19000/api/control/scenarios/minimal/start"
```

## 八、常见问题（Troubleshooting）

### 8.1 Register 返回 401

可能原因：Digest 凭据不匹配。检查配置文件中的 `auth.username` 与 `auth.password`。

### 8.2 节点一直显示离线

检查请求是否携带了 `User-Identify` 头。

### 8.3 抓包数据为空

确认 Capture 中间件已启用（默认启用）；检查 `data/captures.db` 是否存在。

### 8.4 黄金样本测试失败

使用 `-v` 运行查看具体哪一步不匹配：

```bash
go test -v ./test/contract/...
```

---

> 下一节建议阅读：[docs/OPERATIONS.md](./OPERATIONS.md)（部署、监控、升级）