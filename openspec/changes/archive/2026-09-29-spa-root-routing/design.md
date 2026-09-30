# 设计：spa-root-routing

## 概述

重构 `internal/ui/server.go` 中的 Echo 路由注册，使 SPA 服务于 `/`（根）并带有合适的 fallback 处理器。所有其他路由（`/api/*`、`/ws/*`）在 catch-all 之前注册，Echo 才能正确派发。

## 路由注册顺序

Echo 按注册顺序匹配路由。修复方法是在前面注册特定路由，最后挂载 SPA catch-all：

```
installRoutes() 执行顺序：
  1. api := s.e.Group("/api/control")          ← /api/control/* 首先派发到这里
  2. ws := s.e.Group("/ws")                     ← /ws/* 第二派发到这里
  3. s.e.GET("/*", spaHandler)                 ← catch-all：尝试文件，回退到 index.html
```

`/` 路径**不会**有专用路由——它落入 catch-all，由 `index.html` 提供。

## SPA 处理器实现

SPA 处理器（`spaHandler`）包装 `uiStaticFS`（`embed.FS`），提供 try-files fallback：

```go
// spaHandler 从 uiStaticFS 提供文件，对于任何不是真实文件的路径回退到 index.html。
// 这实现了 SPA fallback：GET /dashboard → index.html。
func spaHandler(c echo.Context) error {
    path := c.Request().URL.Path

    // 规范化：去除前导斜杠以适配 embed.FS
    fsPath := strings.TrimPrefix(path, "/")
    if fsPath == "" {
        fsPath = "index.html"
    }

    // 尝试打开文件；如果存在则提供
    f, err := uiStaticFS.Open(fsPath)
    if err == nil {
        defer f.Close()
        return c.File(path)
    }

    // Fallback：提供 index.html（SPA 入口）
    return c.File("/index.html")
}
```

关键属性：
- 存在的文件（如 `index.html`、`favicon.ico`）被正确服务
- 不以文件形式存在的路由（如 `/dashboard`、`/nodes/abc`）回退到 `index.html`
- `/api/*` 与 `/ws/*` 永远不会到达此处理器，因为它们先注册

## API 路由保护

由于 catch-all 在 `/*`，Echo 必须在 catch-all 之前注册特定路由以使其胜出。由于 Echo 使用首个匹配路由（前缀分组），在 `s.e.GET("/*", ...)` 之前注册的 `s.e.Group("/api/control")` 与 `s.e.Group("/ws")` 保证正确派发。

## 现有文件：internal/ui/server.go

当前的 `installRoutes()`：

```go
func (s *Server) installRoutes() {
    api := s.e.Group("/api/control")
    // ... 所有 REST 端点 ...

    // 静态 SPA —— Vue 应用位于 /ui/ 下
    s.e.GET("/ui/*", echo.WrapHandler(http.StripPrefix("/ui/", http.FileServer(http.FS(uiStaticFS)))))
    s.e.GET("/", func(c echo.Context) error {
        return c.Redirect(http.StatusFound, "/ui/")
    })

    ws := s.e.Group("/ws")
    ws.GET("/events", s.handleEvents)
}
```

修改后——`installRoutes()`：

```go
func (s *Server) installRoutes() {
    api := s.e.Group("/api/control")
    // ... 所有 REST 端点不变 ...

    ws := s.e.Group("/ws")
    ws.GET("/events", s.handleEvents)

    // SPA catch-all：最后注册，捕获所有非 API/WS 路径
    // try-files fallback：真实文件 → 提供它；未知路径 → index.html
    s.e.GET("/*", spaHandler)
}
```

## 测试

`TestSPAFallback` 的四个用例：

| 用例 | 方法+路径 | 预期 |
|---|---|---|
| 根路径 | `GET /` | 200，Content-Type: text/html，body 含 `<div id="app">` |
| SPA 路径 | `GET /dashboard` | 200，Content-Type: text/html，body 含 `<div id="app">` |
| API | `GET /api/control/nodes` | 200，Content-Type: application/json，不受影响 |
| WebSocket 升级 | `WS /ws/events` | HTTP 101 Upgrade，不受影响 |

测试 harness 使用 `httptest.NewServer(echo.WrapFS(uiStaticFS))` 进行静态文件服务，使用 `httptest.NewServer(e)` 进行完整路由测试。

## 待定问题

无——实现很直接。