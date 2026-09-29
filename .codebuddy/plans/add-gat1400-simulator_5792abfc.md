---
name: add-gat1400-simulator
overview: 基于 GA/T 1400.4 接口协议，构建企业级 Go 模拟器，生成具备采集设备 / 视图库服务端 / 视图库客户端三类节点、可任意拓扑与混部，提供完整 HTTP/REST + HTTP Digest 鉴权、订阅通知/布控告警、YAML 场景包引擎、Vue3 Web BFF 控制面板、内嵌报文抓包、协议级黄金样本测试矩阵与完整企业级文档。
todos:
  - id: bootstrap-scaffold
    content: 搭建项目骨架：go.mod、Makefile、目录结构、CI（lint/test/build 多平台）、README 初始化
    status: completed
  - id: domain-models
    content: 实现 domain 模型：Node/Role/Resource/Subscription/Disposition/Scenario/统一 ResponseStatus；含校验与20 位编码生成器
    status: completed
    dependencies:
      - bootstrap-scaffold
  - id: adapter-httpapi
    content: 实现协议 REST 路由与统一响应壳：System/Collection/DataService/Cascade 四类，含 Resource 仓库与抓包中间件
    status: completed
    dependencies:
      - domain-models
  - id: adapter-wire
    content: 实现 HTTP 客户端与 Digest 二次握手，含 401 重试、nonce 持久化、User-Identify 头与 application/VIID+JSON 序列化
    status: completed
    dependencies:
      - adapter-httpapi
  - id: scenario-engine
    content: 实现场景引擎：YAML 加载、节点编排、资源工厂（伪数据 + 可选素材）、订阅矩阵、异常注入开关
    status: completed
    dependencies:
      - adapter-wire
  - id: web-control-plane
    content: 实现 Web BFF：echo 控制面 API、WS 实时事件；前端 Vue3+ElementPlus 仪表盘/节点/场景/资源/订阅/抓包 6 个页面与 embed.FS 注入
    status: completed
    dependencies:
      - scenario-engine
  - id: testing-and-docs
    content: 补齐测试矩阵与文档：黄金样本、单测/集成/e2e（真 socket 双进程）、ARCHITECTURE/PROTOCOL/USER_GUIDE/OPERATIONS/TESTING/CHANGELOG
    status: completed
---

## 产品概述

补齐 gat1400-simulator 的 UAC（客户端）高层协议语义：让 `wire.Client` 拥有 Register/Keepalive/Unregister/Subscribe/Disposition 等高层方法，让 `OutboundDispatcher` 覆盖 System 与 Cascade 类 URI，让 `scenario.Engine` 在节点生命周期内自动触发这些动作，从而形成"上线 → 心跳 → 推送 → 订阅/布控 → 注销"的完整闭环。

## 核心特性

- **System 类高层封装**：Register / UnRegister / Keepalive / GetTime 高层方法，复用现有 PostJSON/Digest 机制
- **Cascade 类高层封装**：SubscribeCreate / SubscribeDelete / SubscribeList / DispositionCreate / DispositionList- **OutboundDispatcher 扩展**：Dispatch 覆盖 `/VIID/System/*` 与 `/VIID/Cascade/*` 路由，与 Collection 类保持同一调用形式
- **Engine 生命周期联动**：materialise 时调用 Register，Stop 时调用 Unregister；按 ExceptionSpec.KeepaliveJitter 触发心跳抖动
- **Notification stub 修复**：dispatchNotifications 实际通过 wire client POST 到 SubscribeNotifications

## 技术栈

- 语言：Go 1.25（沿用项目当前栈）
- HTTP 客户端：现有 `internal/adapter/wire.Client`（labstack/echo/v4 仅服务端用，客户端为标准库 `net/http`）
- 鉴权：RFC 2617 Digest（qop=auth，cnonce via `crypto/rand`）
- 抓包：复用 `internal/adapter/capture.Recorder`
- 日志：`log/slog`
- 配置：现有 `internal/app/config.Config`

## 实现方法

### 1. wire 包高层 API（`internal/adapter/wire/client.go`）

新增以下方法，**全部复用 `do()` 已实现的 Digest 401 自动重试与 User-Identify 注入**：

- `Register(ctx, baseURL, deviceID, registerObject) error`
- `UnRegister(ctx, baseURL, deviceID) error`
- `Keepalive(ctx, baseURL, deviceID) error`
- `GetTime(ctx, baseURL) (time.Time, error)`
- `SubscribeCreate(ctx, baseURL, payload) (string, error)`
- `SubscribeDelete(ctx, baseURL, subscribeID) error`
- `SubscribeList(ctx, baseURL) ([]map[string]any, error)`
- `DispositionCreate(ctx, baseURL, payload) (string, error)`
- `DispositionList(ctx, baseURL) ([]map[string]any, error)`

### 2. OutboundDispatcher 扩展（`internal/adapter/scenario/dispatcher.go`）

新增 3 个高层方法，与现有 `Dispatch` 同形式：

- `DispatchRegister(ctx, target, registerObj) error`
- `DispatchKeepalive(ctx, target) error`
- `DispatchUnregister(ctx, target) error`
- `DispatchSubscribe(ctx, target, payload) error`
- `DispatchDisposition(ctx, target, payload) error`

URL 构造复用 `baseURL(target)`，所有响应进入 capture 中间件（与 Dispatch 行为一致）。

### 3. Engine 生命周期联动（`internal/adapter/scenario/engine.go`）

- **`materialise`**末尾追加：依据 `Exceptions.RegisterDropRate` 概率随机跳过 Register 调用；否则调用 `dispatcher.DispatchRegister`
- **新增 goroutine `runKeepalive(ctx, scenario)`**：每个有 HTTPListen 的 device 节点启动一个 `time.Ticker`，周期 30s（可配置），调用 `DispatchKeepalive`；`KeepaliveJitter` 加随机偏移
- **Stop 流程**：除 cancel 外，对每个 device 节点调用 `dispatcher.DispatchUnregister`（带超时，避免阻塞 shutdown）
- **修复 `postNotification`**：通过 `wire.Client` 实际 POST `/VIID/SubscribeNotifications`，payload 与目标都已从 `dispatchNotifications` 推导出来

### 4. 错误处理与可靠性

- Register失败时 log warn 但不阻塞节点物化（设备可以"半上线"，后续推送仍能 work）
- Unregister 失败 best-effort，3s 超时；shutdown 路径不依赖其成功
- Keepalive 失败按 ExceptionSpec触发一次重试（最多1 次）
- 所有调用统一捕获并写入 capture recorder，便于 Web BFF 抓包面板查看 UAC→UAS 流量

### 5. 配置层（`internal/app/config/config.go`）

- 在 Config.Protocol / 新 Config.UAC 节加入 `KeepaliveInterval`（默认 30s）、`UnregisterTimeout`（默认 3s）
- 透传给 Engine构造函数

### 6. 分层纪律保持

- `wire` 包不依赖 `httpapi`（反向）；`dispatcher` 不依赖 `httpapi`；`engine` 不依赖 `httpapi`
- 所有 wire 调用通过现有 `Client.PostJSON/GetJSON`，保持 wire 的纯 HTTP 客户端定位

## 架构设计

```
┌─────────────────────────────────────────────────────────────┐
│ scenario.Engine                                              │
│  ├── materialise()  → Dispatcher.DispatchRegister (随机丢弃) │
│  ├── runKeepalive() → Dispatcher.DispatchKeepalive (30s+jitter)│
│  ├── emit()         → Dispatcher.Dispatch (Collection)       │
│  ├── dispatchNotifications() → Dispatcher.DispatchSubscribeNotification│
│  └── Stop()         → Dispatcher.DispatchUnregister (3s timeout)│
└──────────────────────┬──────────────────────────────────────┘
                       │ 依赖
┌──────────────────────▼──────────────────────────────────────┐
│ scenario.OutboundDispatcher (扩展)                          │
│  Dispatch, DispatchRegister, DispatchKeepalive,             │
│  DispatchUnregister, DispatchSubscribe, DispatchDisposition │
│  DispatchSubscribeNotification                              │
│  ↓ 仅依赖 wire.Client                                       │
└──────────────────────┬──────────────────────────────────────┘
                       │ 依赖
┌──────────────────────▼──────────────────────────────────────┐
│ wire.Client (扩展)                                          │
│  PostJSON/GetJSON (existing)                                │
│  Register / UnRegister / Keepalive / GetTime                │
│  SubscribeCreate / SubscribeDelete / SubscribeList          │
│  DispositionCreate / DispositionList                         │
│  ↓ 仅依赖 net/http + crypto/rand + slog │
└─────────────────────────────────────────────────────────────┘
```

## 目录结构（仅本任务变更）

```
internal/adapter/wire/
├── client.go           # [MODIFY] 新增 8 个高层方法
└── client_test.go      # [NEW] httptest 验证 Digest 流程与高层方法

internal/adapter/scenario/
├── dispatcher.go       # [MODIFY] 扩展 OutboundDispatcher 增加6 个 Dispatch* 方法
├── engine.go           # [MODIFY] materialise/Stop 联动 + 新增 runKeepalive goroutine + 修复 postNotification
├── engine_test.go      # [NEW] 验证 Register/Unregister/Keepalive 调用序列
└── dispatcher_test.go  # [NEW] 验证各 Dispatch* 路由 URL 正确internal/app/config/
└── config.go           # [MODIFY] 新增 KeepaliveInterval / UnregisterTimeout 字段

cmd/gat1400-sim/
└── main.go             # [MODIFY] 传递新配置给 NewEngine

configs/
└── default.yaml        # [MODIFY] 增加 uac 段配置项

test/e2e/
└── uac_lifecycle_e2e_test.go  # [NEW] 真实 socket 验证 Register→Keepalive→Unregister 全链路
```

## 关键代码结构

```
// wire.Client 新增高层方法（仅签名）
type Client interface {
    // existing
    PostJSON(ctx context.Context, url, deviceID string, body any) (map[string]any, int, error)
    GetJSON(ctx context.Context, url, deviceID string) (map[string]any, int, error)

    // System 类（GA/T 1400.4 第5节）
    Register(ctx context.Context, baseURL, deviceID string, registerObj map[string]any) error
    UnRegister(ctx context.Context, baseURL, deviceID string) error
    Keepalive(ctx context.Context, baseURL, deviceID string) error
    GetTime(ctx context.Context, baseURL string) (time.Time, error)

    // Cascade 类（GA/T 1400.4 第5.4节）
    SubscribeCreate(ctx context.Context, baseURL string, payload map[string]any) (string, error)
    SubscribeDelete(ctx context.Context, baseURL, subscribeID string) error
    SubscribeList(ctx context.Context, baseURL string) ([]map[string]any, error)
    DispositionCreate(ctx context.Context, baseURL string, payload map[string]any) (string, error)
    DispositionList(ctx context.Context, baseURL string) ([]map[string]any, error)
}

// OutboundDispatcher 扩展
func (d *OutboundDispatcher) DispatchRegister(ctx context.Context, target node.Node, registerObj map[string]any) error
func (d *OutboundDispatcher) DispatchKeepalive(ctx context.Context, target node.Node) error
func (d *OutboundDispatcher) DispatchUnregister(ctx context.Context, target node.Node) error
func (d *OutboundDispatcher) DispatchSubscribe(ctx context.Context, target node.Node, payload map[string]any) error
func (d *OutboundDispatcher) DispatchDisposition(ctx context.Context, target node.Node, payload map[string]any) error
func (d *OutboundDispatcher) DispatchSubscribeNotification(ctx context.Context, target node.Node, payload map[string]any) error
```