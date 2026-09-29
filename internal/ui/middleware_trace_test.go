package ui

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"

	applogging "github.com/noroadzh/gat1400-simulator/internal/app/logging"
)

func captureHandler(c echo.Context) error {
	id, _ := c.Get("trace_id").(string)
	return c.String(http.StatusOK, id)
}

func ctxEchoHandler(c echo.Context) error {
	id := applogging.TraceIDFromContext(c.Request().Context())
	return c.String(http.StatusOK, id)
}

// withMiddleware wires TraceMiddleware around a single echo handler, returning
// an echo.Context that has already been routed through the middleware chain.
func withMiddleware(t *testing.T, base *slog.Logger, headers map[string]string, handler echo.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	e := echo.New()
	e.Use(TraceMiddleware(base))
	e.GET("/x", handler)
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func TestTraceMiddleware_GeneratesAndEchoes(t *testing.T) {
	var buf bytes.Buffer
	base := slog.New(slog.NewJSONHandler(&buf, nil))
	rec := withMiddleware(t, base, nil, captureHandler)

	if got := rec.Header().Get(HeaderTraceID); got == "" {
		t.Fatalf("expected %s to be set on response, got empty", HeaderTraceID)
	}
	if body := rec.Body.String(); body == "" {
		t.Fatalf("expected handler to receive trace_id via echo context, got empty body")
	}
	if body := rec.Body.String(); body != rec.Header().Get(HeaderTraceID) {
		t.Fatalf("expected echo context trace_id (%s) to match response header (%s)", body, rec.Header().Get(HeaderTraceID))
	}
	if !strings.Contains(buf.String(), `"event":"http_exit"`) {
		t.Fatalf("expected structured http_exit log entry, got: %s", buf.String())
	}
}

func TestTraceMiddleware_AcceptsIncoming(t *testing.T) {
	var buf bytes.Buffer
	base := slog.New(slog.NewJSONHandler(&buf, nil))
	const incoming = "abcdef0123456789abcdef0123456789"
	rec := withMiddleware(t, base, map[string]string{HeaderTraceID: incoming}, captureHandler)

	if got := rec.Header().Get(HeaderTraceID); got != incoming {
		t.Fatalf("expected echoed trace_id=%s, got %s", incoming, got)
	}
	if rec.Body.String() != incoming {
		t.Fatalf("expected handler to receive %s, got %s", incoming, rec.Body.String())
	}
}

func TestTraceMiddleware_PropagatesToRequestContext(t *testing.T) {
	base := slog.New(slog.NewJSONHandler(io.Discard, nil))
	rec := withMiddleware(t, base, nil, ctxEchoHandler)
	got := rec.Body.String()
	if got == "" {
		t.Fatalf("expected handler to see non-empty trace_id via request context")
	}
}

func TestTraceMiddleware_LoggerFromContextCarriesID(t *testing.T) {
	var buf bytes.Buffer
	base := slog.New(slog.NewJSONHandler(&buf, nil))
	e := echo.New()
	e.Use(TraceMiddleware(base))
	e.GET("/x", func(c echo.Context) error {
		applogging.FromContext(c.Request().Context(), base).Info("inside", slog.String("event", "demo"))
		return c.NoContent(http.StatusNoContent)
	})
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	// Split buf into newline-delimited JSON records and ensure both carry the
	// same trace_id attribute.
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected >=2 log lines, got %d: %s", len(lines), buf.String())
	}
	var firstTrace string
	for i, line := range lines {
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("line %d not JSON: %v / %s", i, err, line)
		}
		id, _ := m["trace_id"].(string)
		if id == "" {
			t.Fatalf("line %d missing trace_id: %s", i, line)
		}
		if i == 0 {
			firstTrace = id
		} else if id != firstTrace {
			t.Fatalf("line %d trace_id=%s differs from line 0 trace_id=%s", i, id, firstTrace)
		}
	}
}

func TestWithTraceID_NoopOnEmpty(t *testing.T) {
	ctx := context.Background()
	out := applogging.WithTraceID(ctx, "")
	if out != ctx {
		t.Fatalf("WithTraceID(ctx, \"\") should return the same ctx, got %v", out)
	}
}
