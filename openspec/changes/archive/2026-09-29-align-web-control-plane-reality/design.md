# 设计：align-web-control-plane-reality

## 概述

本 change 是一次文档与整洁性修正，不涉及功能代码的修改。目标是使 OpenSpec 文档与实际状态一致，避免后续实现者被误导。

## 当前状态 vs 规范

```
                        OpenSpec 声称                 实际状态
                        ─────────────────────────     ──────────────────────────────────────
web/package.json        存在（task [x]）              不存在
web/vite.config.ts       存在（task [x]）              不存在
web/src/                有 Vue 组件                   空目录
web/public/             有静态资源                    空目录
web/dist/               pnpm build 输出              手写的 CDN SPA（被 gitignore）
internal/ui/dist/        spec 中被引用                git 跟踪的 CDN SPA（实际来源）
```

## 决策：不修改已归档 Change

已归档 change `2026-09-28-web-control-plane` 是历史记录。修改其 `tasks.md` 会改写历史。因此本 change 的 `proposal.md` 显式记录了该偏差。

## Change 工件

| 工件 | 状态 |
|---|---|
| `proposal.md` | ✅ 已写 |
| `design.md` | ✅ 已写（本文） |
| `specs/web-bff/spec.md` | Delta：更新架构决策文本 |
| `tasks.md` | 两项文件系统操作的检查清单 |

## 文件操作

### 1. 删除 `web/` 目录

```bash
rm -rf web/
```

理由：`web/` 完全冗余。`web/dist/` 被 gitignore（`.gitignore` 第 15 行），`web/src/` 与 `web/public/` 为空。没有价值。

### 2. 更新 `.gitignore`

追加到 `.gitignore`：

```
# Frontend SPA build output (Vite output goes here when npm build is introduced)
# Currently served from internal/ui/dist/ via embed.FS
/internal/ui/dist/
```

### 3. Spec delta：`specs/web-bff/spec.md`

修订架构决策"前端 bundle 在编译期嵌入"：

**修改前：**
> The frontend build artifact (`web/dist/`) is embedded via `embed.FS` and served directly from memory.

**修改后：**
> The frontend SPA is currently a hand-written CDN-based single-file application served from `internal/ui/dist/` via `embed.FS`. This directory will be replaced by the Vite build output in Change 3 (`vue-componentize-and-dockerize`).

## 构建验证

没有构建产物被修改。以下命令验证变更安全：

```bash
# 确认 web/ 已消失
ls web/ 2>&1  # → No such file or directory

# 确认 internal/ui/dist/ 仍存在（BFF 所服务）
ls internal/ui/dist/index.html  # → exists

# Go 构建仍然正常
go build ./...

# Go 测试仍然通过
go test -race ./...
```

## 待定问题

无。范围有意保持最小。