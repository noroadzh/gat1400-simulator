# 任务

## 1. 重构 installRoutes

- [ ] 1.1 从 `internal/ui/server.go::installRoutes` 移除 `s.e.GET("/ui/*", ...)` 与 `s.e.GET("/", ...)` 重定向；验证文件编译通过（`go build ./internal/ui/...`）
- [ ] 1.2 在 `internal/ui/server.go` 中添加 `spaHandler(c echo.Context) error` 函数，尝试服务 `uiStaticFS` 文件并在失败时回退到 `index.html`；验证函数正确处理空路径（`/`）
- [ ] 1.3 在 `installRoutes` 中的 API 与 WS 组之后挂载 `s.e.GET("/*", spaHandler)`；用 `go test -race ./internal/ui/...` 验证路由注册顺序

## 2. SPA Fallback 测试

- [ ] 2.1 在 `internal/ui/server_test.go` 中添加 `TestSPAFallback_RootReturnsIndexHTML`，验证 `GET /` 返回 200，Content-Type 为 `text/html`，body 含 `<div id="app">`
- [ ] 2.2 添加 `TestSPAFallback_UnknownPathReturnsIndexHTML`，验证 `GET /dashboard` 返回 200，Content-Type 为 `text/html`，body 含 `<div id="app">`
- [ ] 2.3 添加 `TestAPIPathNotAffectedByFallback`，验证 `GET /api/control/nodes` 返回 200，Content-Type 为 `application/json`
- [ ] 2.4 运行 `go test -race -count=1 ./internal/ui/...` 并确认全部测试通过（既有测试 + 3 个新测试）

## 3. 验证

- [ ] 3.1 `go vet ./...` 零警告
- [ ] 3.2 `go build ./...` 成功
- [ ] 3.3 手动冒烟：用 `go run ./cmd/gat1400-sim` 启动服务，curl `/` 返回 SPA HTML，curl `/nodes` 返回 SPA HTML，curl `/api/control/system/health` 返回 JSON
- [ ] 3.4 `openspec validate spa-root-routing` 零错误