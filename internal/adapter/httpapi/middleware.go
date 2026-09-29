package httpapi

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/noroadzh/gat1400-simulator/internal/adapter/capture"
	"github.com/noroadzh/gat1400-simulator/internal/app/logging"
)

// CaptureMiddleware 记录每个经过 echo 路由器的请求。
//
// nodeID 来自 URL pattern；为空时降级取 User-Identify 头，再降级为 "anonymous"。
// request/response body 均被读取并缓存，以便下游 handler 再次消费。
func CaptureMiddleware(r *capture.Recorder, nodeID string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			req := c.Request()
			ctx := req.Context()
			node := nodeID
			if node == "" {
				node = req.Header.Get("User-Identify")
			}
			if node == "" {
				node = "anonymous"
			}

			// Drain the request body so we can replay the original request downstream.
			var reqBuf bytes.Buffer
			if req.Body != nil {
				_, _ = io.Copy(&reqBuf, req.Body)
				req.Body = io.NopCloser(bytes.NewReader(reqBuf.Bytes()))
			}

			// Wrap the response writer so we capture body bytes too.
			rw := &captureWriter{ResponseWriter: c.Response().Writer, buf: &bytes.Buffer{}}
			c.Response().Writer = rw

			err := next(c)

			dur := time.Since(start)
			status := c.Response().Status

			entry := capture.Capture{
				NodeID:    node,
				Direction: "inbound",
				Method:    req.Method,
				Path:      req.URL.Path,
				URL:       req.URL.String(),
				Remote:    req.RemoteAddr,
				Status:    status,
				StartedAt: start,
				Duration:  dur,
				Header:    req.Header,
				Request:   bytes.NewReader(reqBuf.Bytes()),
				Response:  bytes.NewReader(rw.buf.Bytes()),
				Err:       err,
			}
			logCaptureIfNeeded(ctx, r, entry)
			return err
		}
	}
}

func logCaptureIfNeeded(ctx context.Context, r *capture.Recorder, entry capture.Capture) {
	go r.Record(ctx, entry)
}

// captureWriter wraps echo's ResponseWriter and tees the body to a buffer.
type captureWriter struct {
	http.ResponseWriter
	buf *bytes.Buffer
}

func (w *captureWriter) Write(b []byte) (int, error) {
	n, err := w.ResponseWriter.Write(b)
	if n > 0 {
		_, _ = w.buf.Write(b[:n])
	}
	return n, err
}

// CaptureOutbound wire adapter 发射出站流量时使用的辅助中间件。
// 当前为 no-op：wire client 通过自己的中间件记录条目，调用方无需干预请求生命周期。
func CaptureOutbound(_ *slog.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc { return next }
}

// TraceMiddleware 协议端 trace id 中间件。
//
// 从请求头读取 X-Trace-Id（或生成新值），存入 ctx 并写回响应头；记录 http_entry /
// http_exit 两条日志（入口 Debug、出口按状态分级）。下游 handler 可通过
// logging.TraceIDFromContext 取用。
func TraceMiddleware(l *slog.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			id := logging.ResolveTraceID(c.Request().Header.Get("X-Trace-Id"))
			ctx := c.Request().Context()
			ctx = logging.WithTraceID(ctx, id)
			c.SetRequest(c.Request().WithContext(ctx))
			c.Response().Header().Set("X-Trace-Id", id)

			req := c.Request()
			logger := logging.FromContext(ctx)
			logger.Debug("http request entry",
				slog.String("event", "http_entry"),
				slog.String("method", req.Method),
				slog.String("path", req.URL.Path),
				slog.String("remote", req.RemoteAddr),
			)

			start := time.Now()
			err := next(c)

			level := slog.LevelInfo
			if err != nil {
				level = slog.LevelWarn
			}
			logger.LogAttrs(ctx, level, "http request exit",
				slog.String("event", "http_exit"),
				slog.String("method", req.Method),
				slog.String("path", req.URL.Path),
				slog.Int("status", c.Response().Status),
				slog.Int64("duration_ms", time.Since(start).Milliseconds()),
			)
			if l != nil && err != nil {
				l.Warn("http handler error", slog.String("trace_id", id), slog.String("error", err.Error()))
			}
			return err
		}
	}
}
