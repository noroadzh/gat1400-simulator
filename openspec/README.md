# OpenSpec — GAT 1400 模拟器变更管理

本目录遵循 [OpenSpec](https://www.codebuddy.ai/docs/zh/openspec/Overview) 变更管理规范，对项目所有架构决策与功能变更进行结构化追踪。

## 目录结构

```
openspec/
├── config.yaml          # OpenSpec 配置（schema、context、规则）
├── CHANGELOG.md         # 变更总览，所有已归档 change 的里程碑清单
├── README.md            # 本文件
├── specs/               # 主规格（由各 change delta spec 合并而来）
│   ├── domain.md
│   ├── adapter-httpapi.md
│   ├── adapter-wire.md
│   ├── scenario.md
│   ├── web-bff.md
│   └── testing.md
└── changes/             # 变更包（每个 change 一个子目录）
    ├── archive/         # 已完成并归档的 change
    └── (active)/        # 进行中的 change
```

## Change 生命周期

每个 change 经历以下阶段：

1. **propose** — 撰写 `proposal.md`，描述动机与非目标
2. **design** — 撰写 `design.md`，给出具体方案
3. **specs** — 撰写 `specs.md`（delta spec），头部英文 `## ADDED Requirements`，含 `### Requirement:` / `#### Scenario: WHEN/THEN`，正文 MUST 关键字，中文正文
4. **tasks** — 撰写 `tasks.md`，分解为可独立验证的任务
5. **implement** — 按 tasks 实现
6. **verify** — `opsx verify`（或 `openspec verify-change`）验证实现与 specs 一致
7. **archive** — `opsx archive`（或 `openspec archive-change`）将 change 移入 `archive/`，同步 delta spec 到 `specs/`

## 已完成 Change（按实现顺序）

| # | Change ID | 名称 | 状态 |
|---|-----------|------|------|
| 1 | bootstrap-scaffold | 项目骨架：go.mod、Makefile、CI | 已归档 |
| 2 | domain-models | 领域模型：Node/Resource/Subscription/Disposition/Scenario/ResponseStatus/ID 生成器 | 已归档 |
| 3 | adapter-httpapi | 协议 REST 路由：System/Collection/Cascade/Catalog + Binder + 抓包中间件 | 已归档 |
| 4 | adapter-wire | HTTP 客户端：Digest 二次握手 + nonce 持久化 + User-Identify + VIID+JSON | 已归档 |
| 5 | scenario-engine | 场景引擎：YAML 加载 + 节点编排 + 资源工厂 + 异常注入 | 已归档 |
| 6 | web-control-plane | Web BFF：echo 控制面 API + WS Hub + Vue3/ElementPlus embed.FS | 已归档 |
| 7 | testing-and-docs | 测试矩阵与文档：黄金样本 + e2e + 6 份文档 | 已归档 |

详情见 [CHANGELOG.md](./CHANGELOG.md)。

## 规范

- Delta spec 头部必须为英文：`## ADDED Requirements`、`### Requirement:`、`#### Scenario:`
- 需求正文必须含 `MUST` 关键字
- 需求正文推荐中文（正文可中文，标题英文）
- Tasks 文件中每个任务可独立验证
- 归档后 change 目录移入 `archive/`，不可再修改