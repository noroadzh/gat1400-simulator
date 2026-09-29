# GA/T 1400 协议模拟器

> 全栈实现：Go（后端 + BFF）· Vue 3（前端 SPA）· Docker（多阶段 + Compose）· OpenSpec（变更管理）

## 项目概述

GA/T 1400.4《应用平台接口协议要求》全栈模拟器，基于 Go 开发，能够在**单进程内**同时扮演符合标准的"设备（UAC）"和"平台（UAS）"两种角色，支持任意拓扑结构，用于验证平台协议栈、压力测试、设备对接调试。

整套模拟器围绕"**节点（Node）**"这一统一抽象构建：device（IPC/NVR，门禁）、platform-small（小平台）、platform-large（大平台）都是 Node 的具体角色；同一进程可任意实例化多节点、任意拓扑，且节点间可发起订阅 / 通知 / 级联。

### 已实现能力概览（路线图 #1–#17 已完成）

| 能力域 | 范围 |
|--------|------|
| 协议服务端 | `/VIID/System`（Register/Keepalive/UnRegister/Time）、`/VIID/Catalog/*`（APEs/APSs/Tollgates/Lanes）、`/VIID/Subscribes`、`/VIID/Dispositions`、`/VIID/SubscribeNotifications`、`/VIID/Resources/*`（12 种 Kind × POST/GET/PUT/DELETE + Info/Data） |
| 认证 | HTTP Digest（RFC 2617，qop=auth）+ 服务端 Nonce 重放保护 + `User-Identify` 头心跳维护 |
| Cascade 双形态 | `DELETE /VIID/Subscribes/:id` 与 `POST /VIID/Subscribes` body `DeleteOperate` 等价；Dispositions 同理 |
| 设备侧（UAC） | 自动 Digest 重试、`crypto/rand` 客户端 nonce、跨进程重启的 NonceStore |
| 资源对象 API | 12 种 Kind（Persons/Aps/Apes/Tollgates/Lanes/MotorVehicles/NonMotorVehicles/Faces/VideoSlices/VideoLabels/Cases/AnalysisRules）完整 CRUD + Info/Data 子资源 |
| 场景引擎 | YAML 拓扑加载 + AutoStart + 节点生命周期（Register → Keepalive → UnRegister）+ 订阅/通知级联 |
| Web 控制面 | Vue 3 + Element Plus SPA，7 个视图（仪表盘 / 节点 / 场景 / 资源 / 订阅 / 抓包 / 配置），实时 WebSocket 推送 |
| BFF 透传 | `/api/control/resources/*` 7 端点薄代理到协议端 `/VIID/<Collection>`，含 5s 缓存 + singleflight 防击穿 |
| 容器化 | distroless 后端镜像（~20MB）+ nginx 前端镜像 + docker-compose 编排 + healthcheck |
| 抓包 | 协议中间件全量记录，SQLite 持久化 + 列表/筛选/详情视图 |

### 运行模式

| 模式 | 说明 | 典型用途 |
|------|------|---------|
| `device` | 模拟摄像机 / NVR / 门禁等前端设备，主动推送结构化数据 | 验证设备协议栈；压力测试平台 |
| `platform-small` | 模拟小型平台，接收设备推送 | 级联验证 |
| `platform-large` | 模拟大型平台，支持订阅 / 告警上报 | 完整拓扑测试 |
| 混合模式 | 同一进程跑多个 device + 多个 platform | 端到端联调 |

### 技术栈

| 层级 | 技术选型 |
|------|---------|
| 协议服务端 | Go 1.25 / [labstack/echo](https://echo.labstack.com/) |
| 持久化 | [modernc.org/sqlite](https://modernc.org/sqlite/)（纯 Go，无 CGO） |
| 控制面 BFF | Go 1.25 + Echo + WebSocket Hub（`:14080`，JSON API + WS） |
| 前端 SPA | Vue 3.4 + `<script setup>` + Element Plus 2 + Vite 5 + TypeScript 5 |
| 前端容器 | nginx:alpine，自带 SPA fallback + `/api` `/ws` 反向代理 |
| 后端容器 | distroless/static-debian12，二进制 ~20MB |
| 日志 | `log/slog`（结构化 JSON / text 可切换） |
| 协议 | HTTP Digest 认证（RFC 2617，qop=auth）、`application/VIID+JSON`、Nonce 重放保护 |
| 并发同步 | `golang.org/x/sync/singleflight`（BFF 缓存防击穿） |
| 测试 | `go test -race` / 黄金样本契约 / 真实 TCP socket e2e / `vue-tsc --noEmit` |

## 项目结构

```
gat1400-simulator/
├── cmd/
│   └── gat1400-sim/          # 可执行程序入口（main.go）
│
├── configs/                  # 配置文件
│   ├── default.yaml          # 默认配置
│   └── scenarios/            # YAML 场景文件（节点拓扑/资源/故障）
│
├── data/                     # SQLite 数据目录（运行时生成，gitignored）
│   ├── captures.db           # 抓包数据
│   └── nonces.db             # Digest nonce 表
│
├── docs/                     # 项目文档（中文）
│   ├── ARCHITECTURE.md       # 架构说明：组件拓扑、数据流、分层纪律
│   ├── PROTOCOL.md           # 协议参考：12 Kind 资源对象、System、Catalog、Cascade、错误码
│   ├── USER_GUIDE.md         # 用户指南：配置、场景 YAML、Web BFF API
│   ├── OPERATIONS.md         # 运维指南：本地部署、Docker、nginx、TLS
│   ├── TESTING.md            # 测试指南：黄金样本、e2e、覆盖率
│   └── CHANGELOG.md          # 变更日志
│
├── openspec/                 # 变更管理（gitignored，仅本地维护）
│   ├── specs/                # 主规格（11 个 capability）
│   │   ├── adapter-httpapi/          # 协议服务端契约
│   │   ├── adapter-resource-collection/  # 12 Kind 资源对象 API
│   │   ├── adapter-wire/             # 设备侧 UAC 客户端
│   │   ├── config-keepalive/         # 心跳间隔配置
│   │   ├── domain/                   # 领域模型
│   │   ├── scenario/                 # 场景引擎
│   │   ├── scenario-lifecycle/       # 节点生命周期（Register/Keepalive/UnRegister）
│   │   ├── testing/                  # 测试规范
│   │   ├── web-bff/                  # BFF 控制面契约
│   │   └── wire/                     # UAC 行为契约
│   └── changes/              # 在途 / 归档的 change
│       └── archive/          # 已归档 change（按日期目录）
│
├── web/                      # 前端 SPA 工程（Vite + Vue 3 + TS）
│   ├── package.json          # vue / element-plus / vite / typescript
│   ├── tsconfig.json         # TypeScript strict 模式
│   ├── vite.config.ts        # build.outDir=web/dist，开发代理 /api→14080
│   ├── index.html            # Vite 入口模板
│   ├── public/               # 静态资源
│   └── src/
│       ├── main.ts           # createApp + ElementPlus + Router
│       ├── App.vue           # 根组件（layout shell）
│       ├── router.ts         # Vue Router 配置
│       ├── api/
│       │   ├── control.ts    # fetch 封装（/api/control/*）
│       │   ├── resources-meta.ts  # 12 Kind 中文元数据 fallback
│       │   └── ws.ts         # WebSocket 自动重连
│       ├── components/
│       │   ├── Sidebar.vue
│       │   ├── Topbar.vue
│       │   ├── StatCard.vue
│       │   ├── LiveCaptureTable.vue
│       │   └── NodeFormDialog.vue
│       ├── views/
│       │   ├── DashboardView.vue
│       │   ├── NodesView.vue
│       │   ├── ScenariosView.vue
│       │   ├── ResourcesView.vue  # 12 Kind 卡片 + 列表/详情/JSON viewer
│       │   ├── SubscriptionsView.vue
│       │   ├── CapturesView.vue
│       │   └── ConfigView.vue
│       └── styles/global.css # 玻璃拟态 + 霓虹渐变主题
│
├── internal/                 # 内部包（禁止外部导入）
│   ├── domain/               # 领域层：零外部依赖
│   │   ├── node/             # 节点模型：Node、Role、Capability、Status
│   │   ├── resource/         # 资源模型：Resource、Kind、Disposition（12 Kind 枚举）
│   │   ├── scenario/         # 场景模型：Scenario、NodeSpec、ResourceSpec
│   │   ├── response/         # 响应模型：ResponseStatus、Code
│   │   └── ids/              # ID 生成器：DeviceID（20位）、Nonce、UUID
│   │
│   ├── app/                  # 应用层：编排 domain 与 port
│   │   ├── application/      # 应用服务：NodeService、ScenarioService
│   │   ├── config/           # 配置加载：viper、flag、环境变量
│   │   └── ports/            # 端口接口：NodeStore、ResourceStore、CaptureStore
│   │
│   ├── adapter/              # 适配层：实现 port，集成第三方库
│   │   ├── httpapi/          # 协议服务端（GA/T 1400.4 REST，:14000）
│   │   │   ├── server.go     # Echo 实例、路由注册
│   │   │   ├── system.go     # /VIID/System/*（含 verifyAuthorization + Nonce 重放检测）
│   │   │   ├── catalog.go    # /VIID/Catalog/{APEs,APSs,Tollgates,Lanes}
│   │   │   ├── collection.go # /VIID/Resources/<Kind> 12 Kind × POST/GET/PUT/DELETE + Info/Data
│   │   │   ├── cascade.go    # /VIID/Subscribes、/VIID/Dispositions、/VIID/SubscribeNotifications
│   │   │   ├── registry.go   # 多节点监听器注册表
│   │   │   ├── middleware.go # Digest + User-Identify + Capture 中间件
│   │   │   └── server_test.go
│   │   ├── wire/             # 设备侧 HTTP 客户端（UAC，Digest 自动重试）
│   │   │   ├── client.go     # 底层 HTTP + Digest 401 重试 + 单调 nonce 表
│   │   │   ├── uac.go        # 高层 UAC 封装：System.Register/Keepalive/UnRegister、Cascade.*
│   │   │   ├── json.go       # 通用 JSON 编解码辅助
│   │   │   └── *_test.go
│   │   ├── storage/          # SQLite 持久化（captures + nonces）
│   │   │   ├── capture_store.go   # 抓包写入
│   │   │   ├── capture_reader.go  # 抓包查询
│   │   │   ├── nonce_store.go     # Digest nonce 表（Consume 重放检测）
│   │   │   ├── helpers.go         # 通用 SQL 工具
│   │   │   └── storage.go         # 初始化与迁移
│   │   ├── scenario/         # YAML 场景引擎
│   │   │   ├── engine.go     # Engine.AutoStart / Start / Stop / runKeepalive
│   │   │   ├── dispatcher.go # OutboundDispatcher：Register/Keepalive/Subscribe/Disposition
│   │   │   ├── factory.go    # NodeSpec → *domain.node.Node 实例化
│   │   │   └── loader.go     # 场景目录扫描 + YAML 解析
│   │   ├── capture/          # 抓包 recorder
│   │   ├── generator/        # 资源生成器（可选扩展）
│   │   └── media/            # 媒体流（可选扩展）
│   │
│   └── ui/                   # Web 控制面 BFF（:14080，纯 JSON + WS）
│       ├── server.go         # Echo 实例、路由、SPA fallback
│       ├── resources.go      # /api/control/resources/* 资源对象透传（5s 缓存 + singleflight）
│       ├── ws.go             # WebSocket Hub（实时推送）
│       ├── api/              # REST 处理器
│       ├── middleware/       # Auth、CORS 中间件
│       └── server_test.go
│
├── test/                     # 集成 / e2e / 契约 / 单元测试
│   ├── contract/             # 黄金样本契约测试
│   ├── e2e/                  # 端到端测试（真实 TCP socket）
│   ├── integration/          # 跨包集成测试
│   └── unit/                 # 补充单元测试
│
├── Dockerfile.backend        # 多阶段：go build → distroless
├── Dockerfile.frontend       # 多阶段：npm build → nginx:alpine
├── docker-compose.yml        # backend:14080 + frontend:8080 + shared data/
├── nginx.conf                # SPA fallback + /api /ws 反代到 backend
├── .dockerignore             # 构建期排除项
│
├── .github/workflows/ci.yml  # CI 流水线（lint + test + frontend + build）
├── Makefile                  # build / test / e2e / lint / release 快捷命令
└── go.mod                    # Go 模块依赖（含 golang.org/x/sync）
```

## 快速开始

### 1. 本地构建

```bash
# 后端
make build           # 输出 bin/gat1400-sim
make test            # 单元 + 集成测试（-race）
make e2e             # 端到端测试（需要空闲端口）

# 前端
cd web && npm install && npm run build   # 输出 web/dist/
cd web && npm run dev                     # 开发模式（自动代理 /api → :14080）
```

### 2. 本地启动（开发模式）

```bash
# 终端 1：启动后端（含协议端 :14000 + BFF :14080）
./bin/gat1400-sim --config configs/default.yaml

# 终端 2：启动前端开发服务器（端口 5173，自动代理 /api 到 :14080）
cd web && npm run dev

# 浏览器访问
#   http://localhost:5173  → 前端开发服务器
#   http://localhost:14080/api/control/system/health  → BFF 健康检查
#   http://localhost:14000/VIID/System/Time           → 协议端直连
```

### 3. Docker 部署（生产模式）

```bash
# 一键拉起整套服务
docker-compose up -d

# 端口映射
#   http://localhost:8080  → 前端 SPA（nginx，自动反代 /api /ws 到 backend）
#   http://localhost:14080 → 后端 BFF（JSON + WS，仅容器间通信也可用）

# 健康检查（容器内执行）
docker-compose exec backend ./gat1400-sim -healthcheck
```

浏览器访问 `http://localhost:8080` 即可使用 Web 控制台。

## Web 控制面（BFF API 速查）

完整字段与示例见 `docs/USER_GUIDE.md`。控制面所有接口均挂在 `/api/control/*`，并以 WebSocket 在 `/ws/events` 推送实时事件。

| 域 | 端点 | 说明 |
|----|------|------|
| 系统 | `GET /api/control/system/health` | 健康检查 |
| 节点 | `GET / POST /api/control/nodes` | 节点列表 / 新建 |
| 场景 | `GET / POST /api/control/scenarios` | 场景列表 / 加载 |
| 场景 | `POST /api/control/scenarios/{id}/start` | 启动场景 |
| 场景 | `POST /api/control/scenarios/{id}/stop` | 停止场景 |
| 资源 | `GET /api/control/resources` | 12 Kind 元数据（kind / collection / idField / 计数） |
| 资源 | `GET /api/control/resources/{kind}/list` | 列表查询（透传 `/VIID/{Collection}`） |
| 资源 | `GET /api/control/resources/{kind}/list/{id}` | 单条查询 |
| 资源 | `POST /api/control/resources/{kind}/list` | 批量写入（标准信封） |
| 资源 | `PUT /api/control/resources/{kind}/list/{id}` | 单条更新 |
| 资源 | `DELETE /api/control/resources/{kind}/list/{id}` | 单条删除 |
| 资源 | `GET /api/control/resources/{kind}/list/{id}/info` | Info 子资源 |
| 抓包 | `GET /api/control/captures` | 抓包列表（分页 + 筛选） |

> 资源对象端点的 status code / headers / body 全部原样透传协议端，BFF 不做解析或改写；5 秒内存缓存（`singleflight.Group` 防击穿）。

## 文档导航

| 文档 | 内容 |
|------|------|
| [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | 系统架构、组件拓扑、数据流、分层纪律 |
| [docs/PROTOCOL.md](docs/PROTOCOL.md) | GA/T 1400.4 协议路由、12 Kind 资源对象、System、Catalog、Cascade、错误码 |
| [docs/USER_GUIDE.md](docs/USER_GUIDE.md) | 配置、场景 YAML、Web BFF API |
| [docs/OPERATIONS.md](docs/OPERATIONS.md) | 本地部署、Docker、TLS、systemd、监控 |
| [docs/TESTING.md](docs/TESTING.md) | 黄金样本、e2e、覆盖率、CI |
| [docs/CHANGELOG.md](docs/CHANGELOG.md) | 变更历史 |

## 端口分配

| 端口 | 服务 | 协议 | 说明 |
|------|------|------|------|
| `8080` | 前端 SPA（nginx） | HTTP | 仅在 docker-compose 模式下暴露 |
| `5173` | 前端开发服务器 | HTTP | `npm run dev` 使用 |
| `14080` | 控制面 BFF | HTTP + WS | JSON API + WebSocket 实时推送 |
| `14000` | 协议服务端 | HTTP | GA/T 1400.4 REST API |
| `14101+` | 设备节点本地监听 | HTTP | device 节点对外暴露时按需占用 |

## 变更管理

本项目遵循 [OpenSpec](https://www.codebuddy.ai/docs/zh/openspec/Overview) 变更管理规范。每个 change 走生命周期 `propose → design → specs → tasks → implement → verify → archive`：

- 主规格沉淀在 `openspec/specs/<capability>/spec.md`（当前 11 个 capability）
- 在途与归档 change 在 `openspec/changes/`，已归档按日期归档到 `openspec/changes/archive/<YYYY-MM-DD>-<slug>/`
- delta spec 头部使用 `## ADDED Requirements` / `### Requirement:` / `#### Scenario:`（WHEN / THEN），需求正文需含 `MUST`
- 完整提案 / 设计 / 任务 / 规格 详见各 change 目录

## 路线图进度

| # | 主题 | 状态 |
|---|------|------|
| 1–13 | 项目脚手架 / 协议骨架 / Web 控制面 / Vue 组件化 / Docker / 抓包与异常注入 / … | ✅ 已归档 |
| 14 | web-management-ui | ✅ 已归档 |
| 15 | scenario-engine | ✅ 已归档 |
| 16 | uac-lifecycle（节点 Register/Keepalive/UnRegister + Cascade 双形态） | ✅ 已归档 |
| 17 | 资源对象 API 端到端打通（OpenSpec spec + 协议文档 + BFF 透传 + 前端 ResourcesView） | ✅ 已归档（2026-09-29） |

后续增量以"小步快跑 + 单一职责 change"形式持续演进，每个 change 独立 verify 后归档。
