# Proposal — 结构化本地日志体系

## Why

项目目前只有一份"硬编码"的 slog：

- `cmd/gat1400-sim/main.go:55` 直接 `slog.New(slog.NewJSONHandler(os.Stdout, ...))`，level 写死
  `LevelInfo`，格式写死 JSON，无法通过配置切换。
- 所有组件（application / adapter / ui）通过构造函数注入 `*slog.Logger`，但**没有任何
  file sink**：进程一退出所有日志即丢；容器环境下日志完全依赖 stdout / 外部收集器，
  本地排障极不方便。
- 关键路径埋点过疏：节点增删、SIP 注册/注销、HTTP 入口/出口、抓包写入等关键事件**只有
  Info 一行裸消息**，跨组件溯源几乎无法做。
- `internal/ui/ws.go:40` 还在直接用 `slog.Default()`，绕开 main.go 的注入链路，
  在 main 还没初始化 logger 时（如热加载场景）会用到 go 默认 logger。
- `cmd/gat1400-sim/main.go:42, 49` 的 `fmt.Fprintf(os.Stderr, ...)` 直接走 stderr，
  既不走 slog 也不带时间/级别/trace，运维采集时无法与正常日志合并。
- 启动阶段配置尚未加载，无法表达"生产 BUG 日志 / 测试更详细"的诉求；当前只能改源码。

## What Changes

- **新增 `log` 配置段**：在 `internal/app/config` 增加 `LogConfig`
  （level / format / file / rotation / stdout）；`configs/default.yaml` 与
  `configs/local.yaml` 增加 `log:` 段。
- **新增 `--profile=prod|dev|test` 启动参数**：根据 profile 自动套一组默认 level，
  再让 yaml / 配置项覆盖；满足"生产用 BUG 日志，测试/集成用更详细"的需求。
- **统一 Logger 初始化入口**：在 `cmd/gat1400-sim/main.go` 中先解析 `--profile`，
  再加载配置，最后用 `internal/app/logging.New(cfg, profile)` 构建唯一根 logger，
  通过构造函数注入到所有组件；`slog.SetDefault` 同步，保证 `internal/ui/ws.go`
  这类遗留 `slog.Default()` 调用也走到同一份 handler。
- **多 sink（stdout / file）**：file sink 使用 `gopkg.in/natefinch/lumberjack.v2`
  实现按大小或按天轮转（`log.rotation: size | daily`），可在配置里独立开关 stdout / file，
  支持 fanout 输出。
- **关键路径埋点**：在 application 服务、scenario engine、httpapi middleware、
  ui BFF middleware、capture recorder、wire SIP client 的关键状态变化处统一打
  Info / Debug 日志，并给所有 HTTP 入口生成 `trace_id`，通过 `slog.With`
  透传到下游，使一次请求的所有日志都带同一 trace。
- **替换 stderr 直接写**：把 `cmd/gat1400-sim/main.go:42, 49` 的
  `fmt.Fprintf(os.Stderr, ...)` 改为通过 root logger.Error/warn 输出（在 logger 化之前
  不可达的错误保留 stderr 作为兜底）。
- **测试**：新增 `internal/app/logging` 单元测试覆盖 level 解析、rotator fanout；
  新增 BFF / httpapi middleware trace id 测试（已有测试保持 discard logger 模式）。

## Capabilities

### New Capabilities
- `logging`: 模拟器统一结构化日志体系——配置化（level/format/file/rotation/profile）、
  多 sink（stdout + 本地文件 + 轮转）、关键路径埋点、HTTP 请求级 trace id 透传。

### Modified Capabilities
（不修改任何现有主 spec；本变更仅新增 `logging` 能力，所有 spec-level 行为变更集中在
新增 spec 内声明。）

## Impact

**新增 / 修改代码**
- `internal/app/logging/logger.go`（新）— Logger 构建、fanout、level 解析、profile 默认值
- `internal/app/logging/logger_test.go`（新）— 单测
- `internal/app/config/config.go`（改）— 增加 `LogConfig`
- `internal/app/config/config_test.go`（改，若存在）— 配置叠加测试
- `configs/default.yaml`（改）— 增加 `log:` 段
- `configs/local.yaml`（改，若存在）
- `cmd/gat1400-sim/main.go`（改）— `--profile` 解析、调用 `logging.New`、
  替换 stderr 直接写
- `internal/ui/middleware_trace.go`（新）— BFF trace id 中间件
- `internal/adapter/httpapi/middleware.go`（改）— 已有中间件加 trace id
- `internal/app/application/services.go`（改）— `NodeService` / `ScenarioService`
  关键状态变更埋点
- `internal/adapter/scenario/engine.go`、`dispatcher.go`（改）— 节点物化、pacing、
  订阅推送埋点
- `internal/adapter/wire/client.go`（改）— SIP 注册/注销埋点
- `internal/adapter/capture/recorder.go`（改）— 抓包写入埋点
- `internal/ui/server.go`、`ws.go`（改）— 注入替换 `slog.Default()`
- `go.mod` / `go.sum`（改）— 新增 `gopkg.in/natefinch/lumberjack.v2`

**兼容性**
- 无 BREAKING：组件构造函数签名不变，仅增加 logger 注入来源（沿用已注入方式）。
- 配置兼容：未配置 `log:` 段时使用 Default() 给出 `level=info, format=json, stdout=true,
  file=""` 的现状行为，零迁移成本。

**新增测试**
- `internal/app/logging/logger_test.go` — level 解析、fanout、rotator 关闭后路径
- `internal/ui/middleware_trace_test.go` — trace id 生成、上下文透传
- `internal/app/config/config_test.go`（新增用例）— `LogConfig` YAML 解析

**性能**
- Logger fanout 写到两个 io.Writer 是 O(消息大小)，生产配置默认只开 stdout，无回退。
- 文件轮转异步执行，不阻塞业务调用。
- trace id 仅在请求进入时生成一次，每次 `slog.With(trace_id)` 走 attribute 复用，
  无序列化开销。

**OpenSpec**
- `openspec/changes/structured-logging/` — proposal / design / tasks / specs/logging/
- archive 时新建 `openspec/specs/logging/spec.md`（主 spec）