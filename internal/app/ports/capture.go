package ports

import "time"

// CaptureEntry 跨切面抓包记录，HTTP API、场景引擎、BFF 共用。
// adapter 负责持久化；recorder 提供读取 API 供 Web 仪表盘使用。
// Direction 取值："inbound"（入站） | "outbound"（出站）
type CaptureEntry struct {
	ID         int64                  `json:"id"`
	NodeID     string                 `json:"nodeId"`
	Direction  string                 `json:"direction"` // inbound | outbound
	Method     string                 `json:"method"`
	Path       string                 `json:"path"`
	URL        string                 `json:"url"`
	Remote     string                 `json:"remote"`
	Status     int                    `json:"Status"`
	DurationMs int64                  `json:"durationMs"`
	StartedAt  time.Time              `json:"startedAt"`
	Header     map[string][]string    `json:"header"`
	Request    map[string]interface{} `json:"request,omitempty"`
	Response   map[string]interface{} `json:"response,omitempty"`
	Error      string                 `json:"error,omitempty"`
}

// CaptureFilter 仪表盘与导出器使用的抓包查询过滤器。零值字段表示"不限"。
type CaptureFilter struct {
	NodeID    string    `json:"nodeId"`
	Direction string    `json:"direction"`
	Status    int       `json:"status"`
	Method    string    `json:"method"`
	Path      string    `json:"path"`
	From      time.Time `json:"from"`
	To        time.Time `json:"to"`
	Limit     int       `json:"limit"`
}

// CaptureReader BFF 与导出器使用的抓包读取接口。
type CaptureReader interface {
	Query(filter CaptureFilter) ([]CaptureEntry, error)
	ExportJSONL(filter CaptureFilter) (string, error)
	ExportHAR(filter CaptureFilter) (string, error)
}

// CaptureStore HTTP 中间件与场景引擎使用的抓包写入接口。实现必须并发安全。
type CaptureStore interface {
	Append(entry CaptureEntry) error
	Close() error
}