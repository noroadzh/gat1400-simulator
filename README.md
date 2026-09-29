# GA/T 1400 协议模拟器

> 全栈实现：Go（后端 + BFF）· Vue 3（前端 SPA）· Docker（多阶段 + Compose）· OpenSpec（变更管理）

## 项目概述

GA/T 1400.4《应用平台接口协议要求》全栈模拟器，基于 Go 开发，能够在**单进程内**同时扮演符合标准的"设备（UAC）"和"平台（UAS）"两种角色，支持任意拓扑结构，用于验证平台协议栈、压力测试、设备对接调试。

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
| 协议 | HTTP Digest 认证（RFC 2617，qop=auth）、`application/VIID+JSON` |
| 测试 | `go test` / 黄金样本契约 / 真实 TCP socket e2e / `vue-tsc --noEmit` |

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
│   ├── PROTOCOL.md           # 协议参考：路由、字段、错误码
│   ├── USER_GUIDE.md         # 用户指南：配置、场景 YAML、Web BFF API
│   ├── OPERATIONS.md         # 运维指南：本地部署、Docker、nginx、TLS
│   ├── TESTING.md            # 测试指南：黄金样本、e2e、覆盖率
│   └── CHANGELOG.md          # 变更日志
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
│       │   ├── ResourcesView.vue
│       │   ├── SubscriptionsView.vue
│       │   ├── CapturesView.vue
│       │   └── ConfigView.vue
│       └── styles/global.css # 玻璃拟态 + 霓虹渐变主题
│
├── internal/                 # 内部包（禁止外部导入）
│   ├── domain/               # 领域层：零外部依赖
│   │   ├── node/             # 节点模型：Node、Role、Capability、Status
│   │   ├── resource/         # 资源模型：Resource、Kind、Disposition
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
│   │   ├── wire/             # 设备侧 HTTP 客户端（UAC，Digest 自动重试）
│   │   ├── storage/          # SQLite 持久化（captures + nonces）
│   │   ├── scenario/         # YAML 场景引擎（Start/Stop/AutoStart）
│   │   ├── capture/          # 抓包 recorder
│   │   ├── generator/        # 资源生成器（可选扩展）
│   │   └── media/            # 媒体流（可选扩展）
│   │
│   └── ui/                   # Web 控制面 BFF（:14080，纯 JSON + WS）
│       ├── server.go         # Echo 实例、路由、中间件
│       ├── ws.go             # WebSocket Hub（实时推送）
│       ├── api/              # REST 处理器
│       └── middleware.go     # Auth、CORS 中间件
│
├── test/                     # 集成 / e2e 测试
│   ├── contract/             # 黄金样本契约测试
│   └── e2e/                  # 端到端测试（真实 TCP socket）
│
├── Dockerfile.backend        # 多阶段：go build → distroless
├── Dockerfile.frontend       # 多阶段：npm build → nginx:alpine
├── docker-compose.yml        # backend:14080 + frontend:8080 + shared data/
├── nginx.conf                # SPA fallback + /api /ws 反代到 backend
├── .dockerignore             # 构建期排除项
│
├── .github/workflows/ci.yml  # CI 流水线（lint + test + frontend + build）
├── Makefile                  # build / test / e2e / lint / release 快捷命令
└── go.mod                    # Go 模块依赖
```

## 快速开始

### 1. 本地构建

```bash
# 后端
make build           # 输出 bin/gat1400-sim
make test            # 单元 + 集成测试
make e2e             # 端到端测试（需要空闲端口）

# 前端
cd web && npm install && npm run build   # 输出 web/dist/
cd web && npm run dev                     # 开发模式（自动代理 /api → :14080）
```

### 2. 本地启动（开发模式）

```bash
# 终端 1：启动后端（含 BFF，端口 14080）
./bin/gat1400-sim --config configs/default.yaml

# 终端 2：启动前端开发服务器（端口 5173，自动代理 /api 到 :14080）
cd web && npm run dev

# 浏览器访问
#   http://localhost:5173  → 前端开发服务器
#   http://localhost:14080/api/control/system/health  → BFF 健康检查
```

### 3. Docker 部署（生产模式）

```bash
# 一键拉起整套服务
docker-compose up -d

# 端口映射
#   http://localhost:8080  → 前端 SPA（nginx，自动反代 /api /ws 到 backend）
#   http://localhost:14080 → 后端 BFF（JSON + WS，仅容器间通信也可用）
```

浏览器访问 `http://localhost:8080` 即可使用 Web 控制台。

## 文档导航

| 文档 | 内容 |
|------|------|
| [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | 系统架构、组件拓扑、数据流、分层纪律 |
| [docs/PROTOCOL.md](docs/PROTOCOL.md) | GA/T 1400.4 协议路由、请求/响应、错误码 |
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

本项目遵循 [OpenSpec](https://www.codebuddy.ai/docs/zh/openspec/Overview) 变更管理规范，完整规范详见各 change 的 `proposal.md` / `design.md` / `tasks.md` / `specs/*.md`。已归档 change 见 [openspec/changes/archive/](openspec/changes/archive/)。

每个 change 经历生命周期：`propose → design → specs → tasks → implement → verify → archive`。