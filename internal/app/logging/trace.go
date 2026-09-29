package logging

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
)

// traceIDKey is the unexported context key under which the trace id is stored.
type traceIDKey struct{}

// NewTraceID 生成 16 字节的随机 hex 字符串作为 trace id。
func NewTraceID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// 不可能失败；保留兜底以防未来 rand.Reader 被替换
		return ""
	}
	return hex.EncodeToString(b[:])
}

// ResolveTraceID 把请求头传入的 trace id 规范化：非空则原样返回，否则生成新的。
// 用于 HTTP 中间件既支持上游透传，又在缺省时为新请求分配。
func ResolveTraceID(incoming string) string {
	if incoming != "" {
		return incoming
	}
	return NewTraceID()
}

// WithTraceID returns a copy of ctx that carries the given trace id.
// A nil or empty id is treated as a no-op (ctx is returned unchanged) so that
// callers do not accidentally poison downstream middleware with a blank id.
func WithTraceID(ctx context.Context, id string) context.Context {
	if id == "" {
		return ctx
	}
	return context.WithValue(ctx, traceIDKey{}, id)
}

// TraceIDFromContext extracts the trace id previously stored via WithTraceID.
// Returns "" when no id is present (e.g. when called from a non-HTTP code path).
func TraceIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v, ok := ctx.Value(traceIDKey{}).(string); ok {
		return v
	}
	return ""
}

// FromContext returns a logger derived from base with the trace id attached as
// a stable attribute. The returned logger is safe to use even when ctx has no
// trace id — it degrades to base in that case. A nil base falls back to
// slog.Default() so helpers without an explicit logger still work.
//
// 示例（推荐传入当前 logger）：
//
//	logger := logging.FromContext(c.Request().Context(), baseLogger)
func FromContext(ctx context.Context, base ...*slog.Logger) *slog.Logger {
	root := slog.Default()
	if len(base) > 0 && base[0] != nil {
		root = base[0]
	}
	id := TraceIDFromContext(ctx)
	if id == "" {
		return root
	}
	return root.With(slog.String("trace_id", id))
}
