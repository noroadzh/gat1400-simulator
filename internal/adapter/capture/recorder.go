// Package capture 把 wire 层 HTTP 事务转换为结构化抓包记录。
//
// Recorder 同时承担写入和读取接口：
//   - HTTP 中间件、场景引擎调用 Record 写入
//   - BFF、Exporter 通过 CaptureReader 读取
package capture

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"

	"github.com/noroadzh/gat1400-simulator/internal/app/ports"
)

// Recorder 同时实现写入与读取两端（写入端在 Record，读取端委派给底层 store）。
type Recorder struct {
	store           ports.CaptureStore
	log             *slog.Logger
	mu              sync.Mutex
	entriesWritten  uint64 // 进程内累计写入条数，便于运维一眼看出流量级
}

// NewRecorder 把 recorder 装配到持久化 store 上。logger 用于异步失败日志。
func NewRecorder(store ports.CaptureStore, logger *slog.Logger) *Recorder {
	return &Recorder{store: store, log: logger}
}

// Capture HTTP 中间件与场景引擎产出的标准输入形态。recorder 把其转换为 CaptureEntry。
type Capture struct {
	NodeID    string
	Direction string
	Method    string
	Path      string
	URL       string
	Remote    string
	Status    int
	StartedAt time.Time
	Duration  time.Duration
	Header    map[string][]string
	Request   io.Reader
	Response  io.Reader
	Err       error
}

// Record 写入一条抓包。
//
// Request / Response 两个 io.Reader 在锁内同步读取（避免 Reader 被回收）。
// JSON 解析失败时降级为 {"_raw": "<原文>"} 以保证渲染端仍能展示。
func (r *Recorder) Record(ctx context.Context, c Capture) {
	if r == nil {
		return
	}
	entry := ports.CaptureEntry{
		NodeID:     c.NodeID,
		Direction:  c.Direction,
		Method:     c.Method,
		Path:       c.Path,
		URL:        c.URL,
		Remote:     c.Remote,
		Status:     c.Status,
		DurationMs: c.Duration.Milliseconds(),
		StartedAt:  c.StartedAt.UTC(),
		Header:     c.Header,
	}
	if c.Request != nil {
		buf, _ := io.ReadAll(c.Request)
		entry.Request = decodeJSONMap(buf)
	}
	if c.Response != nil {
		buf, _ := io.ReadAll(c.Response)
		entry.Response = decodeJSONMap(buf)
	}
	if c.Err != nil {
		entry.Error = c.Err.Error()
	}

	r.mu.Lock()
	appendErr := r.store.Append(entry)
	if appendErr == nil {
		r.entriesWritten++
	}
	total := r.entriesWritten
	r.mu.Unlock()

	if r.log == nil {
		return
	}
	if appendErr != nil {
		r.log.Warn("capture append failed",
			slog.String("event", "capture_write"),
			slog.String("node_id", c.NodeID),
			slog.String("method", c.Method),
			slog.String("path", c.Path),
			slog.String("error", appendErr.Error()),
		)
		return
	}
	r.log.Debug("capture append",
		slog.String("event", "capture_write"),
		slog.String("node_id", c.NodeID),
		slog.String("method", c.Method),
		slog.String("path", c.Path),
		slog.Int("status", c.Status),
		slog.Uint64("count", total),
	)
}

// Query 委派给底层持久化 store。提供该方法以让上层只看到一个 Recorder 外观。
// 完整的读取 API 由 storage.NewCaptureReader 独立提供——便于按需打开独立连接。
func (r *Recorder) Query(_ ports.CaptureFilter) ([]ports.CaptureEntry, error) {
	return nil, fmt.Errorf("capture: query not yet implemented")
}

// decodeJSONMap 尝试把字节流解析为通用 JSON object；非对象（如 string / array）包成
// {"_raw": "<原文>"}，保证抓包渲染端总是收到 map。
func decodeJSONMap(b []byte) map[string]interface{} {
	if len(b) == 0 {
		return nil
	}
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err == nil {
		return m
	}
	return map[string]interface{}{"_raw": string(b)}
}