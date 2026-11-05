# 双向全协议压测 — 执行计划

**OpenSpec**: `openspec/changes/bidirectional-full-protocol-stress`  
**目标**: 对真实平台/设备做双向全协议压测与联调（Sim 作 UAC 打真实 UAS；Sim 作 UAS 承接真实 UAC；回环验收）。

## 验收标准

1. UAC：`PutJSON`/`DeleteJSON` + Catalog / Collection(12 Kind+分页) / Cascade 补全 / VIAS 子集可用。
2. 场景：压测 YAML（AutoStart=false）可编排出站主路径；Dispatcher 覆盖新动作。
3. e2e：真实 TCP 回环 Register → 集合推送 → Catalog → DispositionNotification。
4. 契约/文档与 PROTOCOL 一致；无生产密钥入库。

## 分期（对应 tasks.md）

| 阶段 | 内容 | tasks |
|------|------|-------|
| A | wire 底层 + Cascade 补全 | 1.x |
| B | Catalog / Collection / VIAS 客户端 | 2.x |
| C | Dispatcher + 压测样例 | 3.x |
| D | e2e / 契约 / 文档 | 4.x–5.x |

## 非目标

完整可选协议面、分布式压测集群、强制 Web 压测控制台。

## 进度

- 状态：已归档
- 归档路径：`openspec/changes/archive/2026-11-05-bidirectional-full-protocol-stress`
- 主 specs 已同步（added 24 / modified 2）
- `make test` / `make e2e` 通过
