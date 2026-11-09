# 双向全协议压测

对应 OpenSpec：`bidirectional-full-protocol-stress`。协议字段与路径以 [PROTOCOL.md](../02-protocol/PROTOCOL.md) 为准。

## 拓扑

```
真实设备 (UAC) ──► 本模拟器 UAS :14000
本模拟器 UAC  ──► 真实平台 (UAS)
本模拟器 UAC  ──► 本机 :14000（回环验收）
```

## 验收清单

1. UAC：`wire.Client` 的 System / Cascade（含 DispositionNotifications）/ Catalog / Collection（分页）/ VIAS 可对目标发请求。
2. 场景：`configs/scenarios/bidirectional-stress.yaml` 可加载，`schedule.autoStart: false`。
3. e2e：`TestUAC_Bidirectional_FullProtocolLoopback` 真实 TCP 通过。
4. 外场：仅改节点 `upstream` 或环境配置指向真实平台；**勿**把生产密钥写入仓库。

## 快速回环

```bash
make build
./bin/gat1400-sim --config configs/default.yaml
# 控制面启动场景 bidirectional-stress，或用 wire 客户端脚本打 :14000
make e2e
```

## YAML actions

| type | 行为 |
|------|------|
| `catalog-apes` | POST `/VIID/APEs` |
| `catalog-tollgates` | POST `/VIID/Tollgates` |
| `disposition-notification` | POST `/VIID/DispositionNotifications` |
| `collection-push` | POST `/VIID/<collection>`（默认 Faces） |
| `vias-task` | POST `/VIAS/Tasks` |

资源 `resources[].rate` 仍按 Kind 推送集合；`actions` 用于 Catalog/通知等补充路径。
