# 提案：Web 控制面

## 状态
已归档 —— 已以 BFF + Vue3 前端的形式实现。

## 背景动机

运维人员需要一个基于浏览器的控制台，用于：
- 实时查看节点状态（online/stopped/error）
- 增删节点而无需编辑 YAML
- 启停场景并观察其进度
- 浏览资源集合（Person、Face、Vehicle 等）
- 配置订阅与处置动作
- 审阅捕获的 HTTP 流量（请求/响应检查器）

## 目标

- 基于 Echo 的 BFF，REST API 位于 `/api/control/...`
- WebSocket Hub 推送实时事件（节点状态、捕获到达）
- Vue3 + ElementPlus 前端，通过 `embed.FS` 嵌入
- 6 个页面：Dashboard、Nodes、Scenarios、Resources、Subscriptions、Captures
- 运行时无外部 CDN 依赖（全部打包内联）

## 非目标

- 不做多用户 / RBAC
- 不做历史分析（只有实时状态）
- 不做场景编辑（运维使用 YAML + BFF 触发）

## 待定问题

无。