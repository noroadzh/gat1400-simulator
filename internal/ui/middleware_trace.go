package ui

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"time"

	"github.com/labstack/echo/v4"

	applogging "github.com/noroadzh/gat1400-simulator/internal/app/logging"
)

// HeaderTraceID is the canonical HTTP header used to propagate a request-scoped
// trace id between client and server. The middleware echoes whatever value it
// generated (or accepted) on the response so callers can correlate logs.
const HeaderTraceID = "X-Trace-Id"

// traceIDBytes controls the entropy of a freshly generated trace id.
// 16 bytes => 32 hex chars, matching common practice (e.g. OpenTelemetry).
const traceIDBytes = 16

// newTraceID returns a fresh hex-encoded trace id. crypto/rand failure
// degrades to a timestamp-based fallback so the request still gets a usable id.
func newTraceID() string {
	var b [traceIDBytes]byte
	if _, err := rand.Read(b[:]); err == nil {
		return hex.EncodeToString(b[:])
	}
	// last-resort fallback: nanosecond timestamp, still distinct per call.
	return hex.EncodeToString([]byte(time.Now().Format(time.RFC3339Nano)))
}

// TraceMiddleware returns an echo middleware that:
//  1. reads X-Trace-Id from the incoming request, or generates a fresh one
//  2. attaches the id to the request context and the echo context
//  3. echoes the id on the response header X-Trace-Id
//
// Downstream handlers MUST use applogging.FromContext(c.Request().Context(), base)
// to obtain a logger that automatically carries the trace id.
func TraceMiddleware(base *slog.Logger) echo.MiddlewareFunc {
	if base == nil {
		base = slog.Default()
	}
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()
			id := req.Header.Get(HeaderTraceID)
			if id == "" {
				id = newTraceID()
			}

			// Persist onto the underlying request context so handlers can pull it
			// via applogging.FromContext. We also drop it on the echo context so
			// helpers that only have access to echo.Context don't need to dig.
			ctx := applogging.WithTraceID(req.Context(), id)
			c.SetRequest(req.WithContext(ctx))
			c.Set("trace_id", id)

			// Echo back on the response so the client can correlate its own logs.
			c.Response().Header().Set(HeaderTraceID, id)

			// Per-request logger carrying the trace id; reused for entry/exit so
			// the lines stay correlatable by trace_id.
			reqLogger := base.With(slog.String("trace_id", id))
			reqLogger.Debug("http request entry",
				slog.String("event", "http_entry"),
				slog.String("method", req.Method),
				slog.String("path", req.URL.Path),
				slog.String("remote", req.RemoteAddr),
			)
			start := time.Now()
			err := next(c)
			status := c.Response().Status
			level := slog.LevelInfo
			if status >= 500 {
				level = slog.LevelWarn
			}
			reqLogger.LogAttrs(req.Context(), level, "http request exit",
				slog.String("event", "http_exit"),
				slog.String("method", req.Method),
				slog.String("path", req.URL.Path),
				slog.Int("status", status),
				slog.Int64("duration_ms", time.Since(start).Milliseconds()),
			)
			return err
		}
	}
}
