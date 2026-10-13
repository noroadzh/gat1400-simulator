# OpenSpec Skills 同步（Claude Code）

本目录的脚本把 OpenSpec CLI 官方 skill 从 `~/.agentbro/skills/` 同步到 Claude Code 的 `~/.claude/skills/`，使 Claude Code 与 Codebuddy IDE 共用同一套 OpenSpec 工作流。

## 背景

- 项目采用 [OpenSpec](https://github.com/fission-ai/openspec)（`@fission-ai/openspec` 1.13.1）做规范驱动变更管理
- OpenSpec CLI 自带 12 个官方 skill，安装时落在 `~/.agentbro/skills/`
- Codebuddy IDE 直接使用该目录；Claude Code 需要 skill 在 `~/.claude/skills/`
- 两个 IDE 操作的是同一个 `openspec/` 目录（本项目已 gitignore，仅本地生效）

## 使用

```bash
# 安装/同步全部 12 个 skill（幂等，重复跑安全）
./install-opsx-skills.sh

# 预览将要发生的操作
./install-opsx-skills.sh --dry-run

# 只安装指定 skill
./install-opsx-skills.sh openspec-propose openspec-apply-change
```

## 什么时候需要重跑

- OpenSpec CLI 升级后（skill 内容可能更新）
- 新增 skill 后
- 换新机器初始化环境时

## Skill 清单（12 个）

| Skill | 用途 |
|-------|------|
| `openspec-propose` | 创建新变更提案（proposal → specs → design → tasks） |
| `openspec-apply-change` | 按 tasks 执行实现 |
| `openspec-verify-change` | 验证实现与 specs 一致 |
| `openspec-archive-change` | 归档已完成变更，同步主 spec |
| `openspec-bulk-archive-change` | 批量归档多个变更 |
| `openspec-continue-change` | 继续上次未完成的变更 |
| `openspec-ff-change` | 快速创建全套 artifacts（一键式） |
| `openspec-new-change` | 实验性新 workflow |
| `openspec-onboard` | OpenSpec 入门引导 |
| `openspec-explore` | 探索式规划（无变更产物） |
| `openspec-sync-specs` | 手动同步 specs |
| `openspec-update-change` | 变更中途更新 proposal/tasks |

## 在 Claude Code 中调用

安装后，新会话中可以直接：

```
/openspec-propose <change-name>     # 斜杠命令
```

或自然语言：「走 OpenSpec 流程建一个 fix-xxx change」——Claude 会自动调用对应 skill。

## 前提条件

- OpenSpec CLI 已安装：`npm i -g @fission-ai/openspec`（当前装在 nvmd 管理的 node 下）
- CLI 在 PATH 中（终端 `openspec --version` 可执行）。若不可用，把
  `~/.nvmd/versions/<node-version>/bin` 加进 PATH

## 同步策略

按约定采用**手动同步**（方案 A）：发现 skill 有更新时重跑本脚本即可，不挂 git hook / make target。
