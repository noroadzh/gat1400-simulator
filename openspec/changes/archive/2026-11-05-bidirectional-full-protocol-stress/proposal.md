# Proposal

## Why

产品目标是对**真实平台 / 真实设备**做双向全协议压测与联调：本模拟器须同时能以 UAC 主动打真实 UAS，并以 UAS（及 VIAS 子集）承接真实 UAC。当前 UAS 侧 VIID 主路径已较完整，但 UAC 高层封装仍偏窄（System + 部分 Cascade），缺少 Catalog / 集合分页 CRUD / DispositionNotifications / VIAS / Put·Delete 原语，场景引擎也无法编排「全协议压测」流量，导致无法在同一工具链上闭环验收。

## What Changes

- 扩展 `wire.Client`：补齐 `PutJSON` / `DeleteJSON`，并新增 Catalog、Collection、VIAS、Cascade 补全高层 API（均走 Digest 透明重试）。
- 扩展 `OutboundDispatcher` 与场景 YAML：支持 Catalog 上报、集合推送/列表拉取、布控通知、VIAS Tasks 等出站动作；提供可配置 QPS / 并发的压测场景样例。
- 补齐双向 e2e / 契约：一侧模拟器打真实 HTTP 对端（或同源双进程回环）覆盖「设备→平台」与「平台→设备」主路径。
- 文档：`docs/02-protocol` / 运维压测说明标明双向拓扑与验收清单。
- 不改变已公开的协议字段语义与路径；不引入非标准 Content-Type / StatusCode。

## Capabilities

### New Capabilities

- `wire/wire-catalog-client`：UAC 对 `/VIID/APEs|APSs|Tollgates|Lanes|VIIDServers` 的列表 / 按 ID / 上报（POST·PUT）/ 删除封装。
- `wire/wire-collection-client`：UAC 对 12 Kind 集合的 POST 推送、按 ID GET/PUT/DELETE、带 `PageRecordNum`/`RecordStartNo` 的列表查询。
- `wire/wire-vias-client`：UAC 对 `/VIAS/System/*`、`/VIAS/Tasks`、`TaskControls`、`AnalysisRules`、能力查询的高层封装。
- `protocol-stress`：双向全协议压测的场景编排、指标与验收行为（含对真实对端与回环拓扑）。

### Modified Capabilities

- `wire/wire-cascade-client`：补 Disposition 列表/更新/删除、DispositionNotifications 推送与查询/删除。
- `adapter-wire`：底层 MUST 提供 `PutJSON`/`DeleteJSON`；高层子类型 MUST 通过它们（及既有 Post/Get）实现。
- `scenario`：`OutboundDispatcher` MUST 覆盖 Catalog / Collection / DispositionNotification / VIAS 出站；YAML 可声明压测速率与目标。
- `end-to-end-testing`：增加双向协议压测 e2e / 契约最小集（真实 TCP，禁 mock HTTP 层）。

## Impact

- 代码：`internal/adapter/wire`、`internal/adapter/scenario`、可选 BFF 触发压测、`configs/scenarios/`、`test/e2e`、`test/contract`、`docs/`。
- 依赖：无新第三方库；继续 `crypto/rand`、Digest、`application/VIID+JSON`。
- 运行：压测场景默认关闭 autoStart；真实对端地址由配置 / 环境变量注入，密钥不入库。
