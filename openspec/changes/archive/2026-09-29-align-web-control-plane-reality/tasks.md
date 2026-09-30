# 任务

## 1. 文件系统修正

- [x] 1.1 彻底删除 `web/` 目录（`rm -rf web/`），并验证 `ls web/` 返回 "No such file or directory"
- [x] 1.2 将 `/internal/ui/dist/` 追加到 `.gitignore` 并附说明注释，验证 `internal/ui/dist/` 被忽略（`git check-ignore -v internal/ui/dist/index.html` 返回路径）
- [x] 1.3 确认 `internal/ui/dist/index.html` 本地仍存在（在 Change 2 路由更新后由 embed 提供）

## 2. Spec 对齐

- [x] 2.1 更新 `openspec/specs/web-bff/spec.md` 架构决策，准确描述 CDN SPA 状态并前向引用 Change 3 的 Vite 迁移
- [x] 2.2 运行 `openspec validate align-web-control-plane-reality` 并确认零错误

## 3. 验证

- [x] 3.1 `go build ./...` 零错误通过
- [x] 3.2 `go test -race ./internal/ui/...` 通过（既有测试不受本 change 影响）
- [x] 3.3 `git status` 只显示预期内的修改/删除（`web/` 消失、`.gitignore` 更新、spec delta）
