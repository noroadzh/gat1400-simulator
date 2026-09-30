# 提案：align-web-control-plane-reality

## 为什么做

Web 控制面的实现与 OpenSpec 规范出现了偏差。已归档的 change `2026-09-28-web-control-plane` 声称交付了 `web/package.json`、`web/vite.config.ts` 与 `pnpm build` 工具链——三者均不存在。前端目前是一个手写的、基于 CDN 的 380 行单文件 SPA，而不是 Vite 构建的 Vue 3 应用。本 change 正式承认这一偏差，并确立修正后的状态。

## 变更内容

- **删除** `web/` 目录（空的 `src/` 与 `public/`、被 gitignore 的 `dist/`、无 npm 项目文件）
- **将** `/internal/ui/dist/` **加入** `.gitignore` —— 它成为唯一的 SPA 产物位置，取代 spec 中的 `web/dist/` 引用
- **在** `specs/web-bff/spec.md` 中**声明**：前端目前是基于 CDN 的单文件 SPA，Change 3（`vue-componentize-and-dockerize`）会引入 Vite 构建流水线
- 已归档的 change `2026-09-28-web-control-plane` **不被修改**（它是历史记录）；本提案为后来的读者记录该偏差

## 能力（Capabilities）

### 修改的能力

- **web-bff**（`openspec/specs/web-bff/spec.md`）：修订"前端 bundle 在编译期嵌入"这条架构决策，使其准确反映当前状态："前端是手写的、基于 CDN 的 SPA，从 `internal/ui/dist/` 提供"。并加入指向 Change 3 中 Vite 迁移的前向引用。

### 新增能力

无。

## 影响

- **代码**：`web/` 目录删除；`.gitignore` 更新；`openspec/specs/web-bff/spec.md` delta
- **Git 历史**：无破坏性改写；`web/` 在最近一次 commit 中未被跟踪
- **下游**：Change 2（`spa-root-routing`）与 Change 3（`vue-componentize-and-dockerize`）建立在本 change 确立的修正状态之上
