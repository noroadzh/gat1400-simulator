# 文档索引 — GA/T 1400 协议模拟器

本目录为项目**用户可读文档**。变更管理（OpenSpec）在仓库根 `openspec/`（本地维护）；AI 执行计划在 `.cursor/plan/`。

文档按**使用阶段分目录**，目录名前缀为序号。

```
docs/
├── README.md                 ← 本索引
├── 01-architecture/          阶段 1 认识系统
├── 02-protocol/              阶段 2 协议对接
├── 03-user-guide/            阶段 3 上手联调
├── 04-testing/               阶段 4 测试验收
├── 05-operations/            阶段 5 部署运维
├── 06-changelog/             阶段 6 变更追溯
└── 07-standards/             阶段 2 附属：国标 PDF
```

---

## 阶段总览

| 阶段 | 目录 | 目的 | 入口 |
|:----:|------|------|------|
| **1** | [`01-architecture/`](./01-architecture/) | 搞清组件、拓扑、分层 | [ARCHITECTURE.md](./01-architecture/ARCHITECTURE.md) |
| **2** | [`02-protocol/`](./02-protocol/) | 实现 / 联调 GA/T 1400.4 | [PROTOCOL.md](./02-protocol/PROTOCOL.md) |
| **2†** | [`07-standards/`](./07-standards/) | 国标 PDF 只读对照 | [README.md](./07-standards/README.md) |
| **3** | [`03-user-guide/`](./03-user-guide/) | 启动、配置、场景、BFF | [USER_GUIDE.md](./03-user-guide/USER_GUIDE.md) |
| **4** | [`04-testing/`](./04-testing/) | 单测、契约、e2e、CI | [TESTING.md](./04-testing/TESTING.md) |
| **5** | [`05-operations/`](./05-operations/) | Docker / systemd / 监控 | [OPERATIONS.md](./05-operations/OPERATIONS.md) |
| **6** | [`06-changelog/`](./06-changelog/) | 版本与行为变更摘要 | [CHANGELOG.md](./06-changelog/CHANGELOG.md) |

† `07-standards` 归属阶段 2（协议对照），序号排在正文文档之后以免与阅读主路径混淆。

**新人推荐路径**：`01 → 02 → 03`；上线前补 `04 → 05`；发版对照 `06`；字段争议查 `07` + `02`。

---

## 阶段 1 — 认识系统（`01-architecture/`）

| 文档 | 读者 | 内容 |
|------|------|------|
| [ARCHITECTURE.md](./01-architecture/ARCHITECTURE.md) | 开发 | 组件拓扑、数据流、端口、分层纪律、技术决策 |

---

## 阶段 2 — 协议对接（`02-protocol/` + `07-standards/`）

| 文档 | 读者 | 内容 |
|------|------|------|
| [PROTOCOL.md](./02-protocol/PROTOCOL.md) | 开发 / 对接方 | VIID / VIAS 路由、认证、12 Kind、Catalog、Cascade、错误码（**协议真相源**） |
| [07-standards/](./07-standards/) | 协议研发 | 国标 PDF 原文（只读对照，不作 API 契约） |

> 改协议字段/路径：先改代码与 `test/contract/golden/`，再改 `02-protocol/PROTOCOL.md`，并记入 `06-changelog/CHANGELOG.md`。

---

## 阶段 3 — 上手联调（`03-user-guide/`）

| 文档 | 读者 | 内容 |
|------|------|------|
| [USER_GUIDE.md](./03-user-guide/USER_GUIDE.md) | 使用方 / QA / 开发 | 快速开始、配置、场景 YAML、Web BFF API、排障 |

---

## 阶段 4 — 测试验收（`04-testing/`）

| 文档 | 读者 | 内容 |
|------|------|------|
| [TESTING.md](./04-testing/TESTING.md) | 开发 | 测试组织、黄金样本、真实 TCP e2e、覆盖率、CI 门禁 |

---

## 阶段 5 — 部署运维（`05-operations/`）

| 文档 | 读者 | 内容 |
|------|------|------|
| [OPERATIONS.md](./05-operations/OPERATIONS.md) | 运维 / SRE | Docker Compose、TLS、systemd、监控、备份、升级、容量 |

---

## 阶段 6 — 变更追溯（`06-changelog/`）

| 文档 | 读者 | 内容 |
|------|------|------|
| [CHANGELOG.md](./06-changelog/CHANGELOG.md) | 全员 | 用户可读变更日志 |

---

## 维护约定

- **协议行为变更**：代码 + `test/contract/golden/` + `02-protocol/PROTOCOL.md` + `06-changelog/CHANGELOG.md`
- **架构 / 端口变更**：同步 `01-architecture/`、`03-user-guide/`、`05-operations/`、根 `README.md`
- **文档语言**：中文为主；字段名 / 路径保持标准英文原样
- **不要**把 OpenSpec proposal/tasks、临时调试笔记写进本目录（执行 plan → `.cursor/plan/`）
