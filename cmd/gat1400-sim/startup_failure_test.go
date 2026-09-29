package main

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
)

// TestFatalHandler_LoggerReady_EmitsStartupFailure 验证 main() 的 fatal 收尾
// 在 loggerReady=true 时会把错误以 event=startup_failure 写入根 logger。
//
// 我们不直接调 main()，而是把 fatal 收尾的逻辑抽成 emitFatal(ready, err, profile)
// 并在此测试中调用；这样不依赖 config.Load / 监听端口等副作用。
func TestFatalHandler_LoggerReady_EmitsStartupFailure(t *testing.T) {
	var buf bytes.Buffer
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(slog.New(slog.NewJSONHandler(noopWriter{}, nil))) })

	emitFatal(true, errFake("simulated startup failure"), "prod")

	out := buf.String()
	if !strings.Contains(out, `"event":"startup_failure"`) {
		t.Fatalf("expected startup_failure event in log, got: %s", out)
	}
	if !strings.Contains(out, `"error":"simulated startup failure"`) {
		t.Fatalf("expected error attribute, got: %s", out)
	}
	if !strings.Contains(out, `"profile":"prod"`) {
		t.Fatalf("expected profile attribute, got: %s", out)
	}
	// JSON 结构合法性。
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("non-JSON log line: %s err=%v", line, err)
		}
	}
}

// TestFatalHandler_LoggerNotReady_FallsBackToStderr 验证 logger 不可用时
// 错误走 stderr 兜底，MUST NOT 产生结构化日志（spec: 二者互斥）。
func TestFatalHandler_LoggerNotReady_FallsBackToStderr(t *testing.T) {
	var buf bytes.Buffer
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(slog.New(slog.NewJSONHandler(noopWriter{}, nil))) })

	emitFatal(false, errFake("config parse failed"), "prod")

	if buf.Len() != 0 {
		t.Fatalf("must not emit structured log when logger not ready, got: %s", buf.String())
	}
}

// errFake 构造一个占位 error，便于测试断言。
type errFake string

func (e errFake) Error() string { return string(e) }

// noopWriter 用作测试结束后的 logger sink，避免污染其它测试。
type noopWriter struct{}

func (noopWriter) Write(p []byte) (int, error) { return len(p), nil }
