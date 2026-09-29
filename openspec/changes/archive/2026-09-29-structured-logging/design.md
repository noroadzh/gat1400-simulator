# Design — 结构化本地日志体系

> 配套 proposal.md（动机）与 specs/logging/（行为契约）。本文聚焦"如何实现"。

## Context

**当前状态**
- `cmd/gat1400-sim/main.go:55` 硬编码 `slog.New(slog.NewJSONHandler(os.Stdout,
  &slog.HandlerOptions{Level: slog.LevelInfo}))`，随后 `slog.SetDefault(logger)`。
- 所有 component 通过构造函数注入 `*slog.Logger`（无 `slog.Default()` 调用，
  仅有 `internal/ui/ws.go:40` 例外）。
- 配置结构体 `internal/app/config/config.go` 没有 `LogConfig`。
- `cmd/gat1400-sim/main.go:42, 49` 通过 `fmt.Fprintf(os.Stderr, ...)` 输出
  healthcheck 失败 / 启动 fatal。
- e2e 测试在 `test/e2e/*` 全部使用 `slog.NewTextHandler(io.Discard, nil)`，
  不会被新增 stdout 输出污染。

**约束**
- 保持分层纪律（app / domain / adapter / ui），logging 子包放在 `internal/app/logging`。
- 不引入除 `lumberjack` 外的第三方日志库（`log/slog` 即可）。
- 组件构造函数签名不变，logger 仍走构造函数注入。
- 现有 `slog.SetDefault` 调用必须保留，使遗留 `slog.Default()` 也走新根 logger。

## Goals / Non-Goals

**Goals**
- 配置化：日志级别、格式、stdout / 文件双 sink、按大小或按天轮转、保留份数。
- 启动参数 `--profile=prod|dev|test` 提供 level 默认值；yaml 显式 level 优先。
- 关键路径埋点：节点 / 场景 CRUD、SIP 注册 / 注销、HTTP 入口出口、抓包写入。
- HTTP 请求级 `trace_id` 在 BFF 与协议端中间件生成并透传到下游日志。
- 替换启动阶段 `fmt.Fprintf(os.Stderr, ...)`，统一走 root logger（仅 logger
  初始化失败时保留 stderr 兜底）。

**Non-Goals**
- 不引入分布式 trace（OpenTelemetry / Jaeger）；trace_id 仅在进程内串联。
- 不改造 `slog.Handler` 本身，仅用标准库 + lumberjack。
- 不改变测试侧 `io.Discard` 模式（已有测试全部沿用）。
- 不动 `-healthcheck` 子命令的 stderr 输出（保留 Docker 约定）。
- 不做日志脱敏 / 字段白名单（后续需要可由独立 spec 处理）。

## Decisions

### Decision 1: 多 sink 用 io.MultiWriter 而不是自定义 Handler

**为什么**：slog 支持 `slog.NewJSONHandler` 与 `slog.NewTextHandler` 的基础能力足够，
没必要自写 `slog.Handler`。在 slog.HandlerOptions 内：
  - 单 sink 时直接 `JSONHandler(w, opts)` / `TextHandler(w, opts)`
  - 多 sink 时 `io.MultiWriter(stdoutW, fileW)` 包装成单一 `io.Writer`
- stdout / file 各自的可配置开关天然映射为"是否包含在 MultiWriter 列表里"。

**替代方案考虑**
- A. 自实现 `multiHandler []slog.Handler` 并实现 Enabled/Handle/WithAttrs/WithGroup：
  灵活但要重新处理 attribute 合并，且与 `slog.SetDefault` 链路有冲突（SetDefault
  要求替换全局 Handler），故被否决。
- B. 多个根 logger 通过 fanout 调用：测试与代码复杂度都会上涨，否决。

### Decision 2: 文件 sink 用 lumberjack 而非自实现轮转

**为什么**：lumberjack 是 Go 社区事实标准的日志轮转库（被 zap / logrus 默认采用），
按大小、按时间、易数保留、压缩均开箱即用，且纯 Go 无 CGO。

**替代方案考虑**
- A. 用 `cron` 自定义按天轮转：实现成本高、时区与跨日一致性需要测试覆盖，
  性价比低，否决。
- B. 写入 `os.Stderr` 由 docker / systemd 收集：与本地用户诉求不符，否决。

### Decision 3: profile 在 main.go 解析，配置加载前只取命令行值

**为什么**：`--profile` 必须在 yaml 加载前解析（用于给 yaml 缺省时的 level 提供
默认值），但 yaml 中显式 `log.level` 必须覆盖 profile。流程：

```
1. parseFlags() → profile, cfgPathOverride, err
2. logger.New(cfg, profile) → 内部使用 cfg.Log.Level 或 profile 预设
3. config.Load(...)(GAT1400_CONFIG) → cfg
4. 重新用 cfg.Log 实际值构建 root logger
```

主路径为：在 `config.Load` **之后** 才用最终的 `cfg.Log` + profile 构造根 logger；
profile 仅用于当 `cfg.Log.Level == ""` 时的 fallback。

### Decision 4: trace_id 通过 context 透传

**为什么**：用 `context.Context` 携带 trace_id 字符串，下游通过 helper
`logging.FromContext(ctx) *slog.Logger` 取带 trace_id attribute 的子 logger，
避免每个 handler 显式 With。HTTP 中间件在响应 header 写回 `X-Trace-Id`，
客户端可按此关联。

**替代方案考虑**
- A. goroutine-local：用 `runtime.Goexit` / `context.local` hack，无标准做法，
  放弃。
- B. 直接传 logger：与项目"通过构造函数注入 logger"风格冲突，否决。

### Decision 5: 启动 stderr 兜底保留，仅在 logger 不可用时出现

**为什么**：`fmt.Fprintf(os.Stderr, "fatal: ...")` 在 logger 初始化前/初始化
失败时仍有意义——保证用户至少看到错误信息。spec 要求两者不能同时存在；
实现上：构造 root logger 失败 → 走 stderr；构造后任何错误 → root logger.Error。

### Decision 6: 不修改现有 component 构造函数签名

**为什么**：所有现有 service / adapter / ui 已通过构造函数注入 logger；
本次改动只**调整传入的 logger 来源**（从 main.go 统一构造后传入），不改字段。
新加中间件（`internal/ui/middleware_trace.go`）以独立文件新增，不动既有文件
签名。

## Risks / Trade-offs

- **[Risk] lumberjack 在 `log.file` 路径所在目录不存在时静默失败** →
  **Mitigation**：`logging.New` 在打开前 `os.MkdirAll(filepath.Dir(file), 0o755)`，
  失败时报错并退。
- **[Risk] stdout 与 file 双写时性能下降** →
  **Mitigation**：默认 `stdout=true, file=""`（不双写）；双写只在用户显式开启
  时生效；写入耗时是 O(消息大小) 的两次序列化，可控。
- **[Risk] `--profile=test` 让日志刷屏干扰单测输出** →
  **Mitigation**：测试侧继续走 `slog.NewTextHandler(io.Discard)`，与生产 logger
  完全隔离；中间件测试通过 mock logger 验证 trace_id 绑定即可。
- **[Risk] 引入 trace_id 后日志格式变化导致现有日志解析规则失效** →
  **Mitigation**：trace_id 作为 attribute 附加，不改变顶层字段；
  JSON 输出顺序由 slog 保证稳定。
- **[Risk] 配置文件 `log` 段与现有 `Log` 关键字冲突** →
  **Mitigation**：命名使用 `log`（小写，与 `node` / `protocol` / `control` /
  `storage` / `auth` 风格一致）；不存在冲突。
- **[Risk] `--profile` 命令行参数与未来 CLI 子命令（如 `-healthcheck`）冲突** →
  **Mitigation**：使用标准 `flag` 包解析，把 `-healthcheck` 优先判断，
  其余走 `--profile`；spec 已规定"未知 profile 报错退出"。

## Migration Plan

**部署步骤**
1. 本 change 提交到 main 分支，CI 跑 `go test -race` + `go vet` + `go build`。
2. 用户无需修改任何 yaml；默认行为与现状一致（stdout + JSON + info）。
3. 想要本地落盘：复制 `log: {file: "./logs/sim.log", maxSizeMB: 100,
   maxBackups: 7}` 到 `configs/local.yaml` 即可。
4. 想要不同环境：复制 `log: {level: debug}` 到对应环境的 yaml，或在启动命令加
   `--profile=dev|test` 不改 yaml。

**回滚**
- 单 commit 回滚：所有改动局限在
  `internal/app/logging/`（新）+
  `internal/app/config/config.go`（+LogConfig）+ `cmd/gat1400-sim/main.go`（logger
  构造迁移）+
  `internal/ui/middleware_trace.go`（新）+
  `internal/adapter/httpapi/middleware.go`（加 trace_id）+ 各 component 埋点 +
  `configs/default.yaml`（+log 段）。
- 无 schema 变更；配置兼容（缺 `log:` 段时与现状一致）。

**Open Questions**: 无。