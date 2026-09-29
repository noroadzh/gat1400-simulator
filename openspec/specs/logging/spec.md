# logging Specification

## Purpose
为模拟器建立统一的本地结构化日志能力：可配置日志级别与格式、可选 stdout / 本地文件双 sink、
按大小或按天的文件轮转、`--profile` 启动参数预设默认级别、关键路径埋点与 HTTP 请求级 trace
id 透传，让运维、生产与测试 / 集成环境在同一份代码上各取所需，并让一次请求的所有日志可被
按 trace_id 串联查询。

## Requirements

### Requirement: 日志级别与格式可通过配置文件覆盖

模拟器 MUST 在配置文件（`configs/default.yaml` / `configs/local.yaml`）的 `log` 段
提供下列字段并允许覆盖：

- `level`：`debug | info | warn | error` 之一（默认 `info`）
- `format`：`json | text`（默认 `json`）
- `stdout`：布尔（默认 `true`），是否同时输出到 stdout
- `file`：字符串（默认 `""`），输出到本地的 file 路径；空字符串表示禁用
- `rotation`：`size | daily`（默认 `size`），仅在 `file` 非空时生效
- `maxSizeMB`：整数（默认 `100`），仅当 `rotation=size` 时生效
- `maxBackups`：整数（默认 `7`），保留的历史文件份数
- `maxAgeDays`：整数（默认 `30`），仅当 `rotation=daily` 时生效
- `compress`：布尔（默认 `true`），历史文件是否 gzip 压缩

未提供 `log` 段时 MUST 退回 Default()，保证旧配置零修改即可运行。

#### Scenario: 默认配置零迁移

- **WHEN** 用户的 `configs/local.yaml` 不包含 `log:` 段
- **THEN** 模拟器 MUST 退回到 stdout + JSON + level=info + 不写文件的现状行为，
  且 MUST NOT 输出任何关于"日志配置缺失"的告警

#### Scenario: 通过 yaml 把 level 调到 debug

- **WHEN** `configs/local.yaml` 含 `log: {level: debug}`
- **THEN** 模拟器 MUST 把根 logger 级别设为 `debug`，所有 component logger MUST 继承该级别

#### Scenario: 切换 JSON / Text 格式

- **WHEN** `log.format: text`
- **THEN** 模拟器 MUST 使用 `slog.NewTextHandler` 输出可读文本；
  反之默认 MUST 使用 `slog.NewJSONHandler`

### Requirement: 多 sink（stdout + 文件）可独立开关

模拟器 MUST 支持同时启用 stdout 与文件两种输出，二者通过配置项独立控制：

- `log.stdout: false` 且 `log.file: ""` MUST 退回到默认行为（仅 stdout info JSON）
- `log.stdout: true` 且 `log.file: "./logs/sim.log"` MUST 同时写两份，stdout 与文件内容
  MUST 一致（除 sink 标识差异）
- `log.stdout: false` 且 `log.file: "./logs/sim.log"` MUST 仅写文件，stdout MUST 不再出现
  任何本系统日志

#### Scenario: 仅写文件不写 stdout

- **WHEN** `log.stdout=false` 且 `log.file="./logs/sim.log"`
- **THEN** stdout MUST 不再出现本系统日志，所有业务日志 MUST 仅出现于
  `./logs/sim.log`（及其轮转文件）

#### Scenario: stdout 与文件双写

- **WHEN** `log.stdout=true` 且 `log.file="./logs/sim.log"`
- **THEN** 同一条 Info 日志 MUST 同时出现于 stdout 与文件（时间戳级别与消息内容一致），
  且文件追加而非覆盖

### Requirement: 文件 sink 支持按大小或按天轮转

模拟器 MUST 在 `log.file` 非空时启用本地文件输出，并按 `log.rotation` 选择策略：

- `size`：单文件超过 `log.maxSizeMB` MiB 时滚动；历史保留 `log.maxBackups` 份
- `daily`：每日 00:00（本地时区）滚动一个文件；历史保留 `log.maxAgeDays` 天

历史文件 MUST 在达到上限时被删除；`log.compress=true` MUST 把历史文件 gzip 压缩为
`<name>-YYYY-MM-DD_HH-MM-SS.gz`。

#### Scenario: 按大小轮转

- **WHEN** `log.rotation=size`、`log.maxSizeMB=1`、单条日志写入后累计超过 1 MiB
- **THEN** 写入方 MUST 自动滚动到新文件；旧的超限文件 MUST 按 `maxBackups` 保留，
  超出的最旧文件 MUST 被删除

#### Scenario: 按天轮转

- **WHEN** `log.rotation=daily` 且跨过本地时间 00:00
- **THEN** 模拟器 MUST 创建一个新的日期命名的日志文件继续写入；旧文件 MUST 保留
  直到超过 `maxAgeDays` 才被删除

#### Scenario: 关闭文件 sink 时不创建日志目录

- **WHEN** `log.file=""` 或未设置 `log` 段
- **THEN** 模拟器 MUST NOT 创建 `logs/` 目录或任何日志文件

### Requirement: --profile 启动参数为 level 提供预设

模拟器 MUST 支持 `--profile=prod|dev|test` 启动参数（无值默认 `prod`），
在 yaml `log.level` 缺省或为空时套用预设默认值；yaml 显式给出 level MUST 始终优先：

- `prod`：默认 `log.level=warn`（"BUG 日志"，对应用户描述）
- `dev`：默认 `log.level=info`
- `test`：默认 `log.level=debug`

#### Scenario: 生产 profile 默认 BUG 日志

- **WHEN** 启动命令为 `gat1400-sim --profile=prod` 且 yaml 中无 `log.level`
- **THEN** 模拟器 MUST 把根 logger 级别设为 `warn`，Info 与 Debug 日志 MUST NOT
  被写出

#### Scenario: 测试 / 集成 profile 默认详细日志

- **WHEN** 启动命令为 `gat1400-sim --profile=test` 且 yaml 中无 `log.level`
- **THEN** 模拟器 MUST 把根 logger 级别设为 `debug`

#### Scenario: yaml 显式 level 覆盖 profile

- **WHEN** `--profile=prod` 但 yaml 含 `log: {level: debug}`
- **THEN** 模拟器 MUST 使用 `debug`（yaml 优先于 profile 预设）

#### Scenario: 未知 profile 报错并退出非零

- **WHEN** 启动命令为 `--profile=foo`
- **THEN** 模拟器 MUST 在启动阶段把未知 profile 报错并以非零退出码终止，
  stderr MUST 含可被人阅读的错误信息

### Requirement: 根 logger 通过 slog.SetDefault 全局可见

模拟器 MUST 在构造根 logger 后调用 `slog.SetDefault(logger)`，使得遗留的
`slog.Default()` 调用（如 `internal/ui/ws.go`）也命中同一份 handler，
避免出现"两套 logger、输出格式不一致"的问题。

#### Scenario: 遗留 slog.Default 调用走同一份 logger

- **WHEN** `internal/ui/ws.go` 等模块直接调用 `slog.Default().Info(...)`
- **THEN** 输出 MUST 与所有注入 logger 的模块走相同的 handler、相同的 level 过滤、
  相同的 sink（stdout / file）

### Requirement: 关键状态变更必须可被日志查询

模拟器 MUST 在下列关键路径打印可被运维查询的结构化日志，属性键 MUST 稳定：

- 节点 / 场景 CRUD（`event=node_create|list_start|list_stop` 等，`node_id`, `scenario_id`）
- SIP 注册 / 注销成功或失败（`event=sip_register|sip_unregister`, `node_id`,
  `status_code`, `reason`）
- HTTP 入口 / 出口（`event=http_entry|http_exit`, `trace_id`, `method`,
  `path`, `status`, `duration_ms`）
- 抓包写入（`event=capture_write`, `node_id`, `count`）

`event` 字段 MUST 出现在所有 Info / Debug 级别日志里（HTTP 中间件除外，由 `trace_id`
代表）。

#### Scenario: 节点上线被日志记录

- **WHEN** 一个节点被场景引擎物化并触发 HTTP 监听注册
- **THEN** 日志 MUST 至少包含一条 Info 日志，键 MUST 包含 `event=node_create`、
  `node_id`、可读的 description

#### Scenario: SIP 注册失败原因可被查询

- **WHEN** 一个节点注册失败
- **THEN** 日志 MUST 包含一条 Warn 或 Error 日志，键 MUST 包含 `event=sip_register`、
  `node_id`、`status_code`、`reason`

### Requirement: HTTP 请求级 trace id 透传

模拟器 MUST 在 BFF（`internal/ui`）与协议端（`internal/adapter/httpapi`）的 HTTP 入口
中间件为每个请求生成一个 trace id（16 字节 hex 字符串），并存入 `context.Context`；
下游 handler / 业务调用 MUST 能从 ctx 取出 trace id 并通过 `slog.With(trace_id)` 绑定
到当前 goroutine 的 logger。

请求响应 header `X-Trace-Id` MUST 回传该值，方便客户端日志关联。

#### Scenario: 同一请求的多条日志共享同一 trace_id

- **WHEN** 一次 HTTP 请求从 BFF 进入并转发到协议端，两层中间件都参与
- **THEN** 该请求生命周期内所有 `slog` 日志 MUST 共享同一个 `trace_id` 属性，
  且响应 header `X-Trace-Id` MUST 等于该值

#### Scenario: 客户端未提供 X-Trace-Id 时自动生成

- **WHEN** 客户端请求不携带 `X-Trace-Id`
- **THEN** 中间件 MUST 生成一个新的 trace id 并在响应中回传

### Requirement: 启动阶段不可达错误保留 stderr 兜底

模拟器 MUST 把启动阶段（`run()` 函数体）所有错误统一改走 root logger.Error 输出；
仅当 logger 初始化本身失败时（即 logger 不可用），保留 `fmt.Fprintf(os.Stderr, ...)`
兜底，保证用户至少能看到错误原因。

#### Scenario: 启动失败原因出现在日志与 stderr 之一

- **WHEN** `run()` 任意一步失败
- **THEN** 如果根 logger 可用，错误 MUST 以 `slog.Error` 形式输出（包含 `event=startup_failure`）
  并以非零退出码终止
- **AND** 仅当 logger 初始化失败时，错误 MUST 走 stderr；二者 MUST NOT 同时存在
  同一进程里双写导致重复

### Requirement: 健康检查子命令不写日志

模拟器 MUST 让 `-healthcheck` 子命令不输出任何 slog 日志（避免污染 stdout）；
其错误信息保留 stderr 输出（符合 Docker / Compose healthcheck 约定）。

#### Scenario: healthcheck 子命令静默

- **WHEN** 进程以 `-healthcheck` 启动
- **THEN** 无论成功或失败，MUST NOT 出现任何结构化日志；
  失败时 stderr MUST 保留 `healthcheck failed: <reason>` 一行可读信息
