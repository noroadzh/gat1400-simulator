# Tasks — 结构化本地日志体系

## 1. 配置与依赖

- [x] 1.1 在 `go.mod` 增加 `gopkg.in/natefinch/lumberjack.v2`，跑 `go mod tidy` 后
      `go.sum` 同步更新；完成后 `grep lumberjack go.mod` 返回 1 行，`go build ./...` 通过
- [x] 1.2 在 `internal/app/config/config.go` 增加 `LogConfig` 结构体
      （Level/Format/Stdout/File/Rotation/MaxSizeMB/MaxBackups/MaxAgeDays/Compress）
      并挂到 `Config.Log`，`Default()` 给出现状对齐的默认值
      （level=info, format=json, stdout=true, file=""）；完成后 `go build ./internal/app/config/...`
      通过
- [x] 1.3 在 `configs/default.yaml` 增加 `log:` 段（level=info, format=json,
      stdout=true, file=""），与 `Default()` 完全一致；完成后 `yaml -c
      configs/default.yaml log.level` 返回 `info`

## 2. logging 子包实现

- [x] 2.1 新建 `internal/app/logging/logger.go`：导出 `New(cfg *config.LogConfig,
      profile string) (*slog.Logger, func() error, error)`，返回根 logger 与关闭函数；
      内部按 `cfg.Format` 选择 JSON / Text handler；按 `cfg.Stdout` / `cfg.File`
      用 `io.MultiWriter` 合并 sink；`cfg.File` 非空时用 lumberjack 包装 writer
      并按 `Rotation` 设置 MaxSize/MaxBackups/MaxAge/Compress；最后调用
      `slog.SetDefault`；完成后 `go build ./internal/app/logging/...` 通过
- [x] 2.2 在同包实现 `parseLevel(s string) (slog.Level, error)` 与
      `levelFromProfile(p string) (slog.Level, error)`：`--profile=prod|dev|test`
      分别映射 warn/info/debug；空字符串返回 `nil, nil` 表示沿用 yaml；完成后
      `go vet ./internal/app/logging/...` 通过
- [x] 2.3 新建 `internal/app/logging/logger_test.go`：覆盖 `parseLevel` 4 档、
      `levelFromProfile` 三档 + 未知 profile 返回 `(nil,err)`、
      `New` 单 stdout / 单 file / 双 sink 三种情形、关闭函数幂等；用 `tempdir`
      隔离 file sink；完成后 `go test -race -count=1 ./internal/app/logging/...` 全绿

## 3. main.go 启动流程改造

- [x] 3.1 在 `cmd/gat1400-sim/main.go` 顶部用 `flag` 解析 `--profile` 默认
      `prod`，把 `-healthcheck` 提前到 flag 解析之前；`flag.Parse` 报错时走 stderr
      兜底并以非零退出；完成后 `go build ./...` 通过，`./gat1400-sim -h` 输出
      profile 帮助
- [x] 3.2 把 `cmd/gat1400-sim/main.go:55-56` 的硬编码 logger 替换为
      `logger.New(cfg.Log, profile)`；调用顺序改为「配置 Load → logger 构造 →
      组件构造（注入）→ SetDefault」；完成后 `go run ./cmd/gat1400-sim --profile=prod`
      启动后 stdout 仅 Warn 及以上级别输出
- [x] 3.3 把 `cmd/gat1400-sim/main.go:42, 49` 的 `fmt.Fprintf(os.Stderr, ...)`
      改造：`run()` 内的错误走 `slog.Default().Error("startup_failure", ...)`；
      `flag.Parse` 与 logger 不可达阶段保留 stderr 兜底；完成后 `go vet ./...` 通过

## 4. 中间件与 trace_id

- [x] 4.1 新建 `internal/app/logging/trace.go`：导出 `WithTraceID(ctx, id) ctx.Context`
      与 `FromContext(ctx) *slog.Logger`，后者从 ctx 取出 trace_id 并返回
      `slog.Default().With("trace_id", id)`；完成后 `go build ./internal/app/logging/...`
      通过
- [x] 4.2 新建 `internal/ui/middleware_trace.go`：导出
      `TraceMiddleware(logger *slog.Logger) echo.MiddlewareFunc`，从请求 header
      `X-Trace-Id` 读取或生成 16 字节 hex，把 trace_id 写入 ctx 并在响应 header
      回传；完成后 `go build ./internal/ui/...` 通过
- [x] 4.3 在 `internal/ui/server.go` 的 `registerRoutes` 中把 `TraceMiddleware`
      挂在 `e.Use(...)` 链最外侧；完成后 `go build ./...` 通过
- [x] 4.4 在 `internal/adapter/httpapi/middleware.go` 的现有中间件
      （capture middleware 等）增加 trace_id 支持：从请求 header 读取或生成，
      写入 ctx，在响应 header 回传；同一进程内多次调用同 ID 不重复生成；完成后
      `go build ./internal/adapter/httpapi/...` 通过
- [x] 4.5 新建 `internal/ui/middleware_trace_test.go`：覆盖 `X-Trace-Id` 透传、
      自动生成、ctx logger 共享；完成后 `go test -race -count=1 ./internal/ui/...`
      全绿

## 5. 关键路径埋点

- [x] 5.1 在 `internal/app/application/services.go` 的 `NodeService.UpsertNode` /
      `RemoveNode` 与 `ScenarioService.Start` / `Stop` 处打 Info 日志（含
      `event=node_create|node_remove|scenario_start|scenario_stop`、`node_id` /
      `scenario_id`）；完成后 `go test -race -count=1 ./internal/app/application/...`
      全绿
- [x] 5.2 在 `internal/adapter/wire/client.go` 的 SIP Register / Unregister
      成功与失败路径打 Info / Warn 日志（含 `event=sip_register|sip_unregister`、
      `node_id`、`status_code`、`reason`）；完成后 `go build ./internal/adapter/wire/...`
      通过
- [x] 5.3 在 `internal/adapter/scenario/engine.go` 与 `dispatcher.go` 的节点
      物化、pacing tick、订阅推送处打 Info / Debug 日志（含 `event=engine_*`、
      `node_id`）；完成后 `go test -race -count=1 ./internal/adapter/scenario/...`
      全绿
- [x] 5.4 在 `internal/adapter/capture/recorder.go` 的批量写入处打 Info 日志
      （含 `event=capture_write`、`node_id`、`count`）；完成后
      `go build ./internal/adapter/capture/...` 通过

## 6. 替换 slog.Default 兜底

- [x] 6.1 在 `internal/ui/ws.go:40` 把 `slog.Default()` 替换为通过构造函数注入
      的 logger；完成后 `go build ./internal/ui/...` 通过

## 7. 端到端验证与归档

- [x] 7.1 跑 `go vet ./... && go test -race -count=1 ./...` 全部 `ok`；新增的
      `internal/app/logging`、`internal/ui/middleware_trace` 测试均绿
- [x] 7.2 用 `subagent:code-reviewer` 审查整个 change
      （`internal/app/logging/`、`internal/app/config/config.go`、
      `cmd/gat1400-sim/main.go`、`internal/ui/middleware_trace.go`、
      各 component 埋点），重点检查 logger 构造顺序、fanout 关闭幂等、
      trace_id 上下文穿透；完成后 reviewer 报告 `critical=0`、warning ≤ 3
      （**reviewer 初报 critical=2 / warning=5**；已就地修复：抽 emitFatal 落
      `event=startup_failure`、run() 返回 ready 替代包级 loggerReady、
      SetStdoutSink 降级为包内私有、FromContext godoc 示例对齐 base、
      countObjects 透传 X-Trace-Id、`Default().Log.Level` 与 yaml 统一为空串）
- [x] 7.3 跑 `openspec validate structured-logging --strict`，无错误输出，退出码 0；
      `openspec archive structured-logging` 归档成功，`openspec/specs/logging/spec.md`
      合并完成；完成后 `ls openspec/changes/archive/` 出现
      `2026-09-29-structured-logging`
- [x] 7.4 `git add . && git commit -m "feat(logging): 统一结构化日志（profile + 双
      sink + 文件轮转 + trace id 透传）"`；完成后 `git log --oneline -1` 显示新
      commit，`git status` 干净