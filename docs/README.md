# 文档索引 — GA/T 1400 协议模拟器

本目录为项目**用户可读文档**。变更管理（OpenSpec）在仓库根 `openspec/`（本地维护）；AI 执行计划在 `.cursor/plan/`。

## 阅读顺序（新人）

1. [ARCHITECTURE.md](./ARCHITECTURE.md) — 系统组成、拓扑、分层纪律
2. [PROTOCOL.md](./PROTOCOL.md) — VIID 路由、认证、12 Kind、错误码
3. [USER_GUIDE.md](./USER_GUIDE.md) — 启动、配置、场景 YAML、BFF API
4. [OPERATIONS.md](./OPERATIONS.md) — Docker / systemd / 监控 / 备份
5. [TESTING.md](./TESTING.md) — 单测、黄金样本、e2e、覆盖率
6. [CHANGELOG.md](./CHANGELOG.md) — 版本变更摘要

## 文档一览

| 文档 | 读者 | 内容 |
|------|------|------|
| [ARCHITECTURE.md](./ARCHITECTURE.md) | 开发 | 组件拓扑、数据流、端口、技术决策 |
| [PROTOCOL.md](./PROTOCOL.md) | 开发 / 对接方 | GA/T 1400.4 REST 实现参考（**协议真相源**） |
| [USER_GUIDE.md](./USER_GUIDE.md) | 使用方 / QA | 快速开始、场景、BFF、排障 |
| [OPERATIONS.md](./OPERATIONS.md) | 运维 / SRE | 部署、TLS、监控、升级、容量 |
| [TESTING.md](./TESTING.md) | 开发 | 测试组织、黄金样本、CI 门禁 |
| [CHANGELOG.md](./CHANGELOG.md) | 全员 | 用户可读变更日志 |
| [standards/](./standards/) | 协议研发 | 国标 PDF 原文（只读参考） |

## 国标原文（`standards/`）

| 文件 | 对应 |
|------|------|
| 第1部分：通用技术要求.pdf | GA/T 1400.1 |
| 第2部分：应用平台技术要求.pdf | GA/T 1400.2 |
| 第3部分：数据库技术要求.pdf | GA/T 1400.3 |
| 第4部分：接口协议要求.pdf | GA/T 1400.4（本模拟器主要依据） |

> 实现行为以代码与 [PROTOCOL.md](./PROTOCOL.md) 为准；PDF 仅作标准对照，勿直接当 API 契约复制粘贴。

## 维护约定

- **协议行为变更**：更新代码 + `test/contract/golden/` + `PROTOCOL.md` + `CHANGELOG.md`
- **架构/端口变更**：同步 `ARCHITECTURE.md`、`USER_GUIDE.md`、`OPERATIONS.md`、根 `README.md`
- **文档语言**：中文为主；字段名 / 路径保持标准英文原样
- **不要**把 OpenSpec proposal/tasks、临时调试笔记写进本目录
