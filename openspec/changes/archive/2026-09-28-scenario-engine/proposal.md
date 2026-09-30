# 提案：场景引擎

## 状态
已归档 —— 实现为 YAML 驱动的编排引擎。

## 背景动机

在代码里手工配置 50+ 个节点与资源推送序列无法维护。我们需要：
- 一种基于 YAML 的场景格式，用于描述节点拓扑、资源推送序列、订阅关系与故障注入
- 一个加载场景文件并启动全部节点的引擎
- 一个资源工厂，用于产出贴近真实的 Person/Face/Vehicle 对象（随机属性、可选的预制图片）
- 一张订阅矩阵，用于在平台之间串联 subscribes（级联）
- 一个故障注入框架，用于测试边界情况（延迟、丢包、报文畸形）

## 目标

- YAML 场景格式（`configs/scenarios/` 下的 `*.yaml` 文件）
- `ScenarioEngine.Start(ctx, s)` 与 `ScenarioEngine.Stop(id)` 方法
- ResourceFactory 接口，含 `Fake`（随机）与 `Static`（来自文件）两种实现
- FaultInjector，支持 `Delay`、`Drop`、`Reorder`、`Malformed` 四种类型
- 支持自动启动（`ScheduleSpec.AutoStart: true`）

## 非目标

- 不做场景编辑 UI（由 web-control-plane 负责）
- 不做运行中场景的实时迁移
- 不做跨进程分布式协调

## 待定问题

无。