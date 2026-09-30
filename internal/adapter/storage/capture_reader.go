package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/noroadzh/gat1400-simulator/internal/app/ports"
)

// captureReader ports.CaptureReader 接口的 sqlite 实现。
// 通过 NewCaptureReader 暴露，对外只暴露 reader 的 API。
//
// closeOnce 保证 Close 可以多次调用幂等,避免 t.Cleanup 链中重复
// 关闭触发 sql.ErrConnDone 或 Windows 上的文件锁冲突。
type captureReader struct {
	db        *sql.DB
	closeOnce sync.Once
	closeErr  error
}

// NewCaptureReader 返回一个 sqlite 后端的 CaptureReader。
// 与其他 store 共享同一数据库连接池。
func NewCaptureReader(path string) (ports.CaptureReader, error) {
	db, err := open(path)
	if err != nil {
		return nil, err
	}
	return &captureReader{db: db}, nil
}

// Query 返回符合过滤器的最新抓包记录，按 id DESC 排序（"最新优先"），
// 便于仪表盘列表视图保持一致。
//
// Limit 规则：
//   - 未指定（<=0）：200
//   - 超过 1000：截断为 200，防止单次响应过大
//   - 1..1000：原样使用
func (r *captureReader) Query(filter ports.CaptureFilter) ([]ports.CaptureEntry, error) {
	var (
		clauses []string
		args    []any
	)
	if filter.NodeID != "" {
		clauses = append(clauses, "node_id = ?")
		args = append(args, filter.NodeID)
	}
	if filter.Direction != "" {
		clauses = append(clauses, "direction = ?")
		args = append(args, filter.Direction)
	}
	if filter.Status != 0 {
		clauses = append(clauses, "status = ?")
		args = append(args, filter.Status)
	}
	if filter.Method != "" {
		clauses = append(clauses, "method = ?")
		args = append(args, filter.Method)
	}
	if filter.Path != "" {
		clauses = append(clauses, "path LIKE ?")
		args = append(args, "%"+filter.Path+"%")
	}
	if !filter.From.IsZero() {
		clauses = append(clauses, "started_at >= ?")
		args = append(args, filter.From.UTC())
	}
	if !filter.To.IsZero() {
		clauses = append(clauses, "started_at <= ?")
		args = append(args, filter.To.UTC())
	}
	where := ""
	if len(clauses) > 0 {
		where = "WHERE " + strings.Join(clauses, " AND ")
	}
	limit := filter.Limit
	if limit <= 0 || limit > 1000 {
		limit = 200
	}
	q := fmt.Sprintf(`SELECT id, node_id, direction, method, path, url, remote, status, duration_ms, started_at, header, request, response, error
		FROM capture %s ORDER BY id DESC LIMIT ?`, where)
	args = append(args, limit)

	rows, err := r.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var out []ports.CaptureEntry
	for rows.Next() {
		var (
			e        ports.CaptureEntry
			header   string
			request  string
			response string
			errStr   string
			started  time.Time
		)
		if err := rows.Scan(&e.ID, &e.NodeID, &e.Direction, &e.Method, &e.Path, &e.URL, &e.Remote,
			&e.Status, &e.DurationMs, &started, &header, &request, &response, &errStr); err != nil {
			return nil, err
		}
		e.StartedAt = started.UTC()
		_ = json.Unmarshal([]byte(header), &e.Header)
		_ = json.Unmarshal([]byte(request), &e.Request)
		_ = json.Unmarshal([]byte(response), &e.Response)
		e.Error = errStr
		out = append(out, e)
	}
	return out, rows.Err()
}

// ExportJSONL 把过滤后的抓包以 JSON Lines 格式写入临时文件，返回文件路径。
// 调用方负责删除（或依赖 OS 重启时清理 /tmp）。
func (r *captureReader) ExportJSONL(filter ports.CaptureFilter) (string, error) {
	entries, err := r.Query(filter)
	if err != nil {
		return "", err
	}
	tmp, err := tempFile("gat1400-capture-*.jsonl")
	if err != nil {
		return "", err
	}
	for _, e := range entries {
		b, err := json.Marshal(e)
		if err != nil {
			_ = tmp.Close()
			return "", err
		}
		if _, err := tmp.Write(b); err != nil {
			_ = tmp.Close()
			return "", err
		}
		if _, err := tmp.Write([]byte("\n")); err != nil {
			_ = tmp.Close()
			return "", err
		}
	}
	return tmp.Name(), tmp.Close()
}

// ExportHAR 把过滤后的抓包渲染成 HAR 1.2 文档（HTTP Archive 格式）。
// 输出可直接被 Chrome DevTools / Charles 等工具打开复现。
func (r *captureReader) ExportHAR(filter ports.CaptureFilter) (string, error) {
	entries, err := r.Query(filter)
	if err != nil {
		return "", err
	}
	log := struct {
		Log struct {
			Version string `json:"version"`
			Creator struct {
				Name    string `json:"name"`
				Version string `json:"version"`
			} `json:"creator"`
			Entries []map[string]any `json:"entries"`
		} `json:"log"`
	}{}
	log.Log.Version = "1.2"
	log.Log.Creator.Name = "gat1400-simulator"
	log.Log.Creator.Version = "0.1.0"
	for _, e := range entries {
		entry := map[string]any{
			"startedDateTime": e.StartedAt.Format(time.RFC3339Nano),
			"time":             e.DurationMs,
			"request": map[string]any{
				"method":      e.Method,
				"url":         e.URL,
				"httpVersion": "HTTP/1.1",
				"headers":     flattenHeader(e.Header),
				"queryString": []any{},
				"postData": map[string]any{
					"mimeType": "application/VIID+JSON",
					"text":     encodeJSONAny(e.Request),
				},
				"headersSize": -1,
				"bodySize":    -1,
			},
			"response": map[string]any{
				"status":      e.Status,
				"statusText":  fmt.Sprintf("%d", e.Status),
				"httpVersion": "HTTP/1.1",
				"headers":     flattenHeader(e.Header),
				"content": map[string]any{
					"mimeType": "application/VIID+JSON",
					"text":     encodeJSONAny(e.Response),
				},
				"headersSize": -1,
				"bodySize":    -1,
			},
			"cache": map[string]any{},
		}
		if e.Error != "" {
			entry["_error"] = e.Error
		}
		log.Log.Entries = append(log.Log.Entries, entry)
	}
	tmp, err := tempFile("gat1400-capture-*.har")
	if err != nil {
		return "", err
	}
	enc := json.NewEncoder(tmp)
	if err := enc.Encode(log); err != nil {
		_ = tmp.Close()
		return "", err
	}
	return tmp.Name(), tmp.Close()
}

func flattenHeader(h map[string][]string) []map[string]string {
	out := make([]map[string]string, 0, len(h))
	for k, vs := range h {
		for _, v := range vs {
			out = append(out, map[string]string{"name": k, "value": v})
		}
	}
	return out
}

// Close 释放底层 sqlite 句柄。多次调用安全幂等,返回第一次 Close 的错误。
// 幂等保护使测试 t.Cleanup 链可任意注册多个,避免在 Windows 上文件锁导致
// t.TempDir RemoveAll 失败。
func (r *captureReader) Close() error {
	r.closeOnce.Do(func() { r.closeErr = r.db.Close() })
	return r.closeErr
}
