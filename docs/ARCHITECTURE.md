# 架构说明 — GAT 1400 协议模拟器

> 文档配套代码版本：v0.1.0
> 最后更新：2026-09-28

本文档从全局到细节描述了 GAT 1400 协议模拟器的系统组成、模块划分、数据流转路径，并给出代码层面的分层纪律。

## 一、系统概述

GAT 1400 协议模拟器是基于 Go 语言开发的纯命令行 / Web 应用，能够在**单个进程**内同时扮演符合 GA/T 1400.4《应用平台接口协议要求》的视频图像信息管理系统的"设备（UAC）"和"平台（UAS）"两种角色。

模拟器运行模式如下：

| 模式 | 含义 | 典型用途 |
|------|------|---------|
| **设备模式（device）** | 模拟摄像机 / NVR / 门禁等前端设备，主动向上级平台推送人脸、车辆、人体等结构化数据 | 验证下级设备协议栈；压力测试平台入库吞吐 |
| **平台模式（platform-small / platform-large）** | 模拟下级 / 上级平台，接收设备推送、维持心跳、保存订阅 / 布控任务 | 验证平台级联、数据流转 |
| **混合模式** | 同一进程内同时跑若干 device 节点 + 若干 platform 节点 | 端到端联调，验证拓扑完整性 |

> 备注：UAC（User Agent Client）与 UAS（User Agent Server）是 SIP / GAT 1400 协议中的术语，前者是主动发起请求的一方，后者是响应请求的一方。

## 二、组件拓扑图

```
┌────────────────────────────────────────────────────────────────┐
│                  gat1400-simulator 进程                       │
│                                                                │
│   ┌────────────────┐             ┌────────────────────────┐  │
│   │ cmd/gat1400-sim│             │  Web BFF（:19000）     │  │
│   │  启动入口      │             │  echo REST + WebSocket │  │
│   └───────┬────────┘             └──────────┬─────────────┘  │
│           │                                  ▲                │
│           ▼                                  │                │
│   ┌──────────────────────────────────────────────────────┐  │
│   │                internal/app/                        │  │
│   │    ┌────────────────┐  ┌──────────────────────────┐│  │
│   │    │  NodeService   │  │  ScenarioService         ││  │
│   │    │  CRUD + 心跳   │  │  启动 / 停止 / 自启动    ││  │
│   │    └──────┬─────────┘  └───────────┬──────────────┘│  │
│   │           │                         │                │  │
│   │    ┌──────┴─────────────────────────┴──────────────┐│  │
│   │    │              ports/ 端口层（接口）            ││  │
│   │    │  NodeStore | ResourceStore | CaptureStore    ││  │
│   │    └──────────────────────────┬───────────────────┘│  │
│   └───────────────────────────────┼─────────────────────┘  │
│                                   │                          │
│   ┌───────────────────────────────┼──────────────────────┐  │
│   │              internal/domain/ 领域层                 │  │
│   │  node | resource | scenario | response | subscription│  │
│   └───────────────────────────────┬──────────────────────┘  │
│                                   │                          │
│   ┌───────────────────────────────┼──────────────────────┐  │
│   │              internal/adapter/ 适配层                │  │
│   │                                                        │  │
│   │  httpapi  ─────────── 协议服务端（:19001）           │  │
│   │    ├─ Digest 中间件（RFC 2617 qop=auth）            │  │
│   │    ├─ User-Identify 中间件                          │  │
│   │    └─ Capture 中间件（抓包）                        │  │
│   │                                                        │  │
│   │  wire  ────────────── 设备侧 HTTP 客户端（UAC）    │  │
│   │    ├─ Digest 自动重试（401 → 携带挑战再发）         │  │
│   │    └─ NonceStore（SQLite 持久化 nonce）            │  │
│   │                                                        │  │
│   │  scenario  ────────── YAML 场景引擎                 │  │
│   │    ├─ LoadAll → ScenarioEngine → Runnable          │  │
│   │    ├─ ResourceFactory（fake / static）             │  │
│   │    └─ FaultInjector（delay / drop / reorder）      │  │
│   │                                                        │  │
│   │  storage  ─────────── SQLite 持久化                 │  │
│   │    ├─ CaptureStore（抓包数据）                      │  │
│   │    └─ NonceStore（nonce 重放保护）                 │  │
│   │                                                        │  │
│   │  capture / generator / media ── 可选扩展模块        │  │
│   └────────────────────────────────────────────────────┘  │
│                                                              │
│   ┌──────────────────────────────────────────────────────┐ │
│   │           internal/ui/  Web 控制面                   │ │
│   │   ├─ REST 处理器（/api/control/*）                  │ │
│   │   ├─ WebSocket Hub（实时事件）                      │ │
│   │   └─ embed.FS → Vue3 SPA（前端构建产物）            │ │
│   └──────────────────────────────────────────────────────┘ │
└────────────────────────────────────────────────────────────────┘
```

## 三、核心数据流

### 3.1 设备推送流程（device 模式 → platform 模式）

```
   device 节点（设备模式）
        │
        │ wire.Client.PostJSON("/VIID/Persons", body)
        │   TCP SYN → :19001
        ├─ HTTP POST /VIID/Persons
        │   Content-Type: application/VIID+JSON
        │   User-Identify: <device-id>
        ▼
   httpapi.Server（:19001）
        ├─ CaptureMiddleware（记录请求、响应）
        ├─ UserIdentifyMiddleware（更新心跳）
        └─ Route → collection.PersonsPOST()
                │
                ▼
          resourceRepo.Put("Person", entry)
                │
                ▼
          BFF WebSocket → 浏览器（推送、捕获、监听）
```

### 3.2 Digest 认证握手

```
wire.Client ──POST /VIID/System/Register──► httpapi.Server
       ◄── 401 Unauthorized
       ◄── WWW-Authenticate: Digest nonce="...", realm="viid"

    // wire.Client 解析挑战，按 RFC 2617 计算 response
    │
    ▼
   ──POST /VIID/System/Register
     Authorization: Digest username="admin", realm="viid",
                    nonce="...", uri="/VIID/System/Register",
                    qop=auth, nc=00000001, cnonce="...",
                    response="<md5>" ──►
       ◄── 200 OK
       ◄── { ResponseStatus: { StatusCode: 0, StatusString: "OK" } }
```

### 3.3 订阅 / 告警流程

```
platform-large
       │
       │ POST /VIID/Subscribes（订阅布控任务）
       ▼
httpapi.Server → subscribeRepo.Add()
       │
       │ device 推送触发了匹配条件（如"男性"）
       ▼
订阅命中 → 平台生成 Disposition
       │
       │ POST /VIID/Dispositions（告警上报）
       ▼
subscribeRepo.RecordDisposition()
```

## 四、端口分配

| 端口 | 用途 | 说明 |
|------|------|------|
| `:19000` | Web 控制面 BFF | 内嵌 SPA + REST + WebSocket |
| `:19001` | 协议服务端 | GA/T 1400.4 REST API |
| `:19101+` | 设备节点本地监听 | 仅当 device 节点对外暴露时占用 |

## 五、持久化（SQLite）

模拟器的全部持久化数据存放在 `data/` 目录：

| 文件 | 内容 |
|------|------|
| `data/captures.db` | 抓包数据：每一次请求的 NodeID / Method / Path / Header / Body |
| `data/nonces.db` | 服务端 nonce 表，重放攻击检测 |

> 实现说明：SQLite 驱动使用 `modernc.org/sqlite` —— 纯 Go 实现，无 CGO，方便跨平台编译。

## 六、分层纪律（强制）

```
internal/app/      ──→ 仅依赖 internal/domain/ 与 internal/app/ports/
internal/adapter/  ──→ 依赖 internal/domain/、internal/app/，以及第三方库
internal/domain/   ──→ 仅依赖标准库
```

执行机制：

- `golangci-lint` 的 `exported` 检查
- CI 流水线阻断违规合并
- `internal/` 前缀天然阻断外部包跨模块导入

## 七、关键技术决策

| 决策 | 取舍 |
|------|------|
| HTTP 框架 | labstack/echo —— 支持路由分组、自定义 Binder、Middleware 链 |
| 数据库 | modernc.org/sqlite —— 纯 Go，无 CGO 依赖 |
| ID 生成 | 自研 `ids.Generator`，符合 GA/T 1400 规定的 20 位 DeviceID |
| 随机源 | `crypto/rand`（nonce、UUID 等），严禁 `time.Now()` 充当熵源 |
| 前端嵌入 | `embed.FS`，启动时 SPA 直接在内存中服务 |
| 日志 | `log/slog`（结构化 JSON，可切换为 text） |
| OutboundDispatcher | 客户端 UAC 各业务方法（Register/Subscribe/Disposition/Notification）由统一的 `dispatcher.OutboundDispatcher` 调度，便于在 e2e 中注入 mock |
| Engine.AutoStart | 与 `Engine.Start` 并存的入口；前者根据 `Scenario.Schedule.AutoStart` 决定是否启动，后者无条件启动；通过 `AutoStart(false)` 可实现幂等 no-op |

## 九、可扩展模块

下列模块已存在但暂未启用，可在后续 change 中扩展：

| 模块 | 用途 |
|------|------|
| `internal/adapter/generator/` | 资源生成器（人脸、车辆图片） |
| `internal/adapter/media/` | 媒体流（视频抽帧、HLS、RTSP） |

---

> 下一节建议阅读：[docs/PROTOCOL.md](./PROTOCOL.md)（协议字段细节）