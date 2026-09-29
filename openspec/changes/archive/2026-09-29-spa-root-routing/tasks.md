# Tasks

## 1. Refactor installRoutes

- [ ] 1.1 Remove `s.e.GET("/ui/*", ...)` and `s.e.GET("/", ...)` redirect from `internal/ui/server.go::installRoutes`; verify file compiles (`go build ./internal/ui/...`)
- [ ] 1.2 Add `spaHandler(c echo.Context) error` function in `internal/ui/server.go` that tries to serve `uiStaticFS` files and falls back to `index.html`; verify the function handles empty path (`/`) correctly
- [ ] 1.3 Mount `s.e.GET("/*", spaHandler)` after the API and WS groups in `installRoutes`; verify route registration order with `go test -race ./internal/ui/...`

## 2. SPA Fallback Tests

- [ ] 2.1 Add `TestSPAFallback_RootReturnsIndexHTML` to `internal/ui/server_test.go` verifying `GET /` returns 200 with `text/html` body containing `<div id="app">`
- [ ] 2.2 Add `TestSPAFallback_UnknownPathReturnsIndexHTML` verifying `GET /dashboard` returns 200 with `text/html` body containing `<div id="app">`
- [ ] 2.3 Add `TestAPIPathNotAffectedByFallback` verifying `GET /api/control/nodes` returns 200 with `application/json` content-type
- [ ] 2.4 Run `go test -race -count=1 ./internal/ui/...` and confirm all tests pass (existing + 3 new)

## 3. Verification

- [ ] 3.1 `go vet ./...` returns zero warnings
- [ ] 3.2 `go build ./...` succeeds
- [ ] 3.3 Manual smoke: start server with `go run ./cmd/gat1400-sim`, curl `/` returns SPA HTML, curl `/nodes` returns SPA HTML, curl `/api/control/system/health` returns JSON
- [ ] 3.4 `openspec validate spa-root-routing` returns zero errors