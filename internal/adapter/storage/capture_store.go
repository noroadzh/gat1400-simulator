package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/noroadzh/gat1400-simulator/internal/app/ports"
)

// captureStore ports.CaptureStore 接口的 sqlite 实现。
type captureStore struct{ db *sql.DB }

// Append 持久化一条 CaptureEntry。
//
// Header / Request / Response 三个 map 字段通过 JSON 编码后再写入 TEXT 列；
// 列顺序与 struct 字段顺序解耦（INSERT 中按位置绑定）。
//
// 异常路径：marshal 失败立即返回错误，不写入半截数据。
func (s *captureStore) Append(e ports.CaptureEntry) error {
	header, err := json.Marshal(e.Header)
	if err != nil {
		return fmt.Errorf("capture: encode header: %w", err)
	}
	req, err := json.Marshal(e.Request)
	if err != nil {
		return fmt.Errorf("capture: encode request: %w", err)
	}
	resp, err := json.Marshal(e.Response)
	if err != nil {
		return fmt.Errorf("capture: encode response: %w", err)
	}
	_, err = s.db.Exec(`INSERT INTO capture
		(node_id, direction, method, path, url, remote, status, duration_ms, started_at, header, request, response, error)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		e.NodeID, e.Direction, e.Method, e.Path, e.URL, e.Remote, e.Status, e.DurationMs, e.StartedAt,
		string(header), string(req), string(resp), e.Error)
	return err
}

func (s *captureStore) Close() error { return s.db.Close() }