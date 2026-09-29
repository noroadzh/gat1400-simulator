# GAT 1400 协议模拟器

> GA/T 1400.4《应用平台接口协议要求》全栈模拟器（Go + Vue3）

## 项目概述

GAT 1400 协议模拟器是一个基于 Go 开发的命令行 / Web 应用，能够在**单进程内**同时扮演符合 GA/T 1400.4 标准的"设备（UAC）"和"平台（UAS）"两种角色，支持任意拓扑结构，用于验证平台协议栈、压力测试、设备对接调试。

**运行模式：**

| 模式 | 说明 | 典型用途 |
|------|------|---------|
| `device` | 模拟摄像机 / NVR / 门禁等前端设备，主动推送结构化数据 | 验证设备协议栈；压力测试平台 |
| `platform-small` | 模拟小型平台，接收设备推送 | 级联验证 |
| `platform-large` | 模拟大型平台，支持订阅 / 告警上报 | 完整拓扑测试 |
| 混合模式 | 同一进程跑多个 device + 多个 platform | 端到端联调 |

**技术栈：**

| 层级 | 技术选型 |
|------|---------|
| 协议服务端 | Go 1.21+ / [labstack/echo](https://echo.labstack.com/) |
| 持久化 | [modernc.org/sqlite](https://modernc.org/sqlite/)（纯 Go，无 CGO） |
| 前端 | Vue 3 + Element Plus + Vite，`embed.FS` 内嵌 |
| 日志 | `log/slog`（结构化 JSON / text 可切换） |
| 协议 | HTTP Digest Auth（RFC 2617）、`application/VIID+JSON` |
| 测试 | `go test` / `httptest.Server` / 真实 socket e2e / 黄金样本 |

## 项目结构

```
gat1400-simulator/
│
├── cmd/
│   └── gat1400-sim/          # 可执行程序入口（main.go）
│
├── configs/                  # 配置文件
│   ├── simulator.yaml        # 主配置
│   └── scenarios/            # YAML 场景文件（节点拓扑/资源/故障）
│
├── data/                     # SQLite 数据目录（运行时生成）
│   ├── captures.db           # 抓包数据
│   └── nonces.db             # Digest nonce 表
│
├── docs/                     # 项目文档（中文）
│   ├── ARCHITECTURE.md       # 架构说明：组件拓扑、数据流、分层纪律
│   ├── PROTOCOL.md           # 协议参考：路由、字段、错误码
│   ├── USER_GUIDE.md         # 用户指南：配置、场景 YAML、Web BFF API
│   ├── OPERATIONS.md         # 运维指南：部署、TLS、systemd、监控
│   ├── TESTING.md            # 测试指南：黄金样本、e2e、覆盖率
│   └── CHANGELOG.md          # 变更日志
│
├── internal/                 # 内部包（禁止外部导入）
│   ├── domain/               # 领域层：零外部依赖
│   │   ├── node/             # 节点模型：Node、Role、Capability、Status、Error
│   │   ├── resource/         # 资源模型：Resource、Kind、Disposition
│   │   ├── scenario/         # 场景模型：Scenario、NodeSpec、ResourceSpec
│   │   ├── response/         # 响应模型：ResponseStatus、Code
│   │   └── ids/              # ID 生成器：DeviceID（20位）、Nonce、UUID、SubscribeID
│   │
│   ├── app/                  # 应用层：编排 domain 与 port
│   │   ├── application/      # 应用服务：NodeService、ScenarioService
│   │   ├── config/           # 配置加载：viper、flag、环境变量
│   │   └── ports/            # 端口接口：NodeStore、ResourceStore、CaptureStore、Notifier
│   │
│   ├── adapter/              # 适配层：实现 port，集成第三方库
│   │   ├── httpapi/          # 协议服务端（:19001）
│   │   │   ├── server.go     # Server、路由注册、中间件
│   │   │   ├── system.go     # System 路由（Register/UnRegister/Keepalive/Time）
│   │   │   ├── collection.go # Collection 路由（Person/Face/Vehicle 等 CRUD）
│   │   │   ├── cascade.go    # Cascade 路由（Subscribe/Disposition）
│   │   │   ├── catalog.go    # Catalog 路由（APE/APS/Tollgate/Lane，静态）
│   │   │   ├── middleware.go # DigestAuth、UserIdentify、Capture 中间件
│   │   │   ├── registry.go   # 节点 HTTP 监听器注册（多节点）
│   │   │   ├── server_test.go# HTTP API 集成测试
│   │   │
│   │   ├── wire/             # 设备侧 HTTP 客户端（UAC）
│   │   │   ├── client.go     # Client、Digest 自动重试（RFC 2617 §3）
│   │   │   ├── uac.go        # UAC 原子操作（Register/Subscribe/Disposition）
│   │   │   ├── json.go       # VIID+JSON 编解码
│   │   │   └── uac_test.go   # UAC 单元测试
│   │   │
│   │   ├── storage/          # SQLite 持久化
│   │   │   ├── storage.go    # DB 连接、schema 初始化
│   │   │   ├── capture_store.go  # 抓包写入
│   │   │   ├── capture_reader.go # 抓包查询
│   │   │   ├── nonce_store.go    # nonce 持久化（防重放）
│   │   │   └── storage_test.go   # SQLite 集成测试
│   │   │
│   │   ├── scenario/         # YAML 场景引擎
│   │   │   ├── loader.go     # LoadAll(*.yaml)
│   │   │   ├── engine.go     # ScenarioEngine（Start/Stop/AutoStart）
│   │   │   ├── dispatcher.go # 按节点分发到 NodeService
│   │   │   ├── factory.go    # ResourceFactory（FakeFactory / StaticFactory）
│   │   │   └── factory_test.go# 工厂测试
│   │   │
│   │   ├── capture/          # 抓包 recorder
│   │   │   └── recorder.go   # 异步写入 CaptureStore
│   │   │
│   │   ├── generator/        # 资源生成器（人脸/车辆图片，可选扩展）
│   │   └── media/            # 媒体流（视频抽帧/HLS/RTSP，可选扩展）
│   │
│   └── ui/                   # Web 控制面 BFF（:19000）
│       ├── server.go         # Echo 实例、路由、中间件
│       ├── ws.go             # WebSocket Hub（实时推送）
│       ├── api/              # REST 处理器
│       ├── middleware.go     # Auth、CORS 中间件
│       ├── static.go         # embed.FS 前端 bundle
│       ├── dist/             # Vue3 构建产物（内嵌）
│       └── server_test.go    # BFF API 测试
│
├── test/                     # 集成 / e2e 测试（跨包，不受 internal/ 边界约束）
│   ├── contract/             # 黄金样本契约测试
│   │   ├── golden/           # JSON 线级样本
│   │   │   ├── register.json      # Digest 握手
│   │   │   ├── persons_post.json  # Person 批量推送
│   │   │   ├── subscribes.json    # 订阅创建
│   │   │   ├── catalog.json       # APE 目录查询
│   │   │   └── keepalive.json     # Keepalive 心跳
│   │   └── golden_test.go    # 黄金样本运行器
│   │
│   └── e2e/                  # 端到端测试（真实 TCP socket）
│       ├── protocol_e2e_test.go  # Register → 推送 → 状态验证
│       └── capture_e2e_test.go   # Capture 中间件端到端
│
├── openspec/                 # OpenSpec 变更管理
│   ├── config.yaml           # OpenSpec 配置
│   ├── README.md             # OpenSpec 说明
│   ├── CHANGELOG.md          # 变更总览
│   ├── specs/                # 主规格（合并自各 change delta spec）
│   │   ├── domain.md              # 领域模型规格
│   │   ├── adapter-httpapi.md     # HTTP API 适配规格（System/Collection/Cascade/Catalog）
│   │   ├── adapter-wire.md        # HTTP 客户端适配规格（Digest 认证）
│   │   ├── config-keepalive.md   # Keepalive 配置规格
│   │   ├── scenario.md            # 场景引擎规格（AutoStart）
│   │   ├── scenario-lifecycle.md  # 场景生命周期规格
│   │   ├── testing.md             # 测试规格（contract/e2e）
│   │   ├── web-bff.md             # Web BFF 规格
│   │   ├── wire/                  # Wire 协议规格
│   │   │   ├── wire-system-client.md   # 系统客户端规格（Register/Subscribe）
│   │   │   └── wire-cascade-client.md # 级联客户端规格（Disposition）
│   │
│   └── changes/              # 变更包
│       └── archive/          # 已归档的 7 个 change
│           ├── bootstrap-scaffold/
│           ├── domain-models/
│           ├── adapter-httpapi/
│           ├── adapter-wire/
│           ├── scenario-engine/
│           ├── web-control-plane/
│           └── testing-and-docs/
│
├── .github/workflows/        # GitHub Actions CI 流水线
├── Makefile                  # 构建、测试、打包快捷命令
└── go.mod                    # Go 模块依赖
```

## 快速开始

### 编译

```bash
go build -o bin/gat1400-simulator ./cmd/gat1400-sim/
# 或
make build
```

### 启动

```bash
./bin/gat1400-simulator --config configs/simulator.yaml
```

启动后访问：

| 地址 | 用途 |
|------|------|
| `http://localhost:19000` | Web 控制台（SPA + WebSocket） |
| `http://localhost:19001/VIID/` | GA/T 1400.4 协议服务端 |

### 运行测试

```bash
make test          # 单元测试
make test-contract # 黄金样本测试
make test-e2e      # e2e 测试（真实 socket）
make test-all      # 全部测试 + 覆盖率门禁
```

## 文档

| 文档 | 内容 |
|------|------|
| [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | 系统架构、组件拓扑、数据流、分层纪律 |
| [docs/PROTOCOL.md](docs/PROTOCOL.md) | GA/T 1400.4 协议路由、请求/响应、错误码 |
| [docs/USER_GUIDE.md](docs/USER_GUIDE.md) | 配置、场景 YAML、Web BFF API |
| [docs/OPERATIONS.md](docs/OPERATIONS.md) | 部署、Docker、TLS、systemd、监控 |
| [docs/TESTING.md](docs/TESTING.md) | 黄金样本、e2e、覆盖率、CI |
| [docs/CHANGELOG.md](docs/CHANGELOG.md) | 变更历史 |
| [openspec/README.md](openspec/README.md) | OpenSpec 变更管理规范 |
| [openspec/CHANGELOG.md](openspec/CHANGELOG.md) | 已归档 change 里程碑 |

## 变更管理

本项目遵循 [OpenSpec](https://www.codebuddy.ai/docs/zh/openspec/Overview) 变更管理规范，所有功能变更走 `propose → design → specs → tasks → implement → verify → archive` 生命周期。

详情见 [openspec/README.md](openspec/README.md)。