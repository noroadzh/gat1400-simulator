# Design: spa-root-routing

## Overview

Refactor the Echo route registration in `internal/ui/server.go` so the SPA is served at `/` (root) with a proper fallback handler. All other routes (`/api/*`, `/ws/*`) are registered before the catch-all so Echo dispatches them correctly.

## Route Registration Order

Echo matches routes in the order they are registered. The fix is to register specific routes first, then mount the SPA catch-all last:

```
installRoutes() execution order:
  1. api := s.e.Group("/api/control")          ← /api/control/* dispatched here first
  2. ws := s.e.Group("/ws")                     ← /ws/* dispatched here second
  3. s.e.GET("/*", spaHandler)                 ← catch-all: tries file, falls back to index.html
```

The `/` path does **not** get a dedicated route — it falls into the catch-all and is served by `index.html`.

## SPA Handler Implementation

The SPA handler (`spaHandler`) wraps `uiStaticFS` (the `embed.FS`) with a try-files fallback:

```go
// spaHandler serves files from uiStaticFS, falling back to index.html for any path
// that is not a real file. This implements the SPA fallback: GET /dashboard → index.html.
func spaHandler(c echo.Context) error {
    path := c.Request().URL.Path

    // Normalize: remove leading slash for embed.FS
    fsPath := strings.TrimPrefix(path, "/")
    if fsPath == "" {
        fsPath = "index.html"
    }

    // Try to open the file; if it exists, serve it
    f, err := uiStaticFS.Open(fsPath)
    if err == nil {
        defer f.Close()
        return c.File(path)
    }

    // Fallback: serve index.html (SPA entry point)
    return c.File("/index.html")
}
```

Key properties:
- Files that exist (e.g. `index.html`, `favicon.ico`) are served correctly
- Routes that do not exist as files (e.g. `/dashboard`, `/nodes/abc`) fall back to `index.html`
- `/api/*` and `/ws/*` never reach this handler because they are registered first

## API Route Protection

With the catch-all at `/*`, Echo must register specific routes before the catch-all so they win. Since Echo uses the first matching route (prefix groups), `s.e.Group("/api/control")` and `s.e.Group("/ws")` registered before `s.e.GET("/*", ...)` guarantee correct dispatch.

## Existing File: internal/ui/server.go

Current `installRoutes()`:

```go
func (s *Server) installRoutes() {
    api := s.e.Group("/api/control")
    // ... all REST endpoints ...

    // Static SPA — Vue app lives under /ui/
    s.e.GET("/ui/*", echo.WrapHandler(http.StripPrefix("/ui/", http.FileServer(http.FS(uiStaticFS)))))
    s.e.GET("/", func(c echo.Context) error {
        return c.Redirect(http.StatusFound, "/ui/")
    })

    ws := s.e.Group("/ws")
    ws.GET("/events", s.handleEvents)
}
```

After change — `installRoutes()`:

```go
func (s *Server) installRoutes() {
    api := s.e.Group("/api/control")
    // ... all REST endpoints unchanged ...

    ws := s.e.Group("/ws")
    ws.GET("/events", s.handleEvents)

    // SPA catch-all: registered last, catches all non-API/WS paths
    // Try-files fallback: real file → serve it; unknown path → index.html
    s.e.GET("/*", spaHandler)
}
```

## Testing

Four test cases for `TestSPAFallback`:

| Case | Method+Path | Expected |
|---|---|---|
| Root | `GET /` | 200, Content-Type: text/html, body contains `<div id="app">` |
| SPA path | `GET /dashboard` | 200, Content-Type: text/html, body contains `<div id="app">` |
| API | `GET /api/control/nodes` | 200, Content-Type: application/json, unaffected |
| WebSocket upgrade | `WS /ws/events` | HTTP 101 Upgrade, unaffected |

The test harness uses `httptest.NewServer(echo.WrapFS(uiStaticFS))` for static file serving and `httptest.NewServer(e)` for full routing tests.

## Open Questions

None — the implementation is straightforward.
