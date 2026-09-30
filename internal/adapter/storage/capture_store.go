package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/noroadzh/gat1400-simulator/internal/app/ports"
)

// captureStore ports.CaptureStore 接口的 sqlite 实现。
//
// closeOnce 保证 Close 可以多次调用幂等,避免 t.Cleanup 链中重复
// 关闭触发 sql.ErrConnDone 或 Windows 上的文件锁冲突。
type captureStore struct {
	db        *sql.DB
	closeOnce sync.Once
	closeErr  error
}

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

// Close 释放底层 sqlite 句柄。多次调用安全幂等,返回第一次 Close 的错误。
// 幂等保护使测试 t.Cleanup 链可任意注册多个,避免在 Windows 上文件锁导致
// t.TempDir RemoveAll 失败。
func (s *captureStore) Close() error {
	s.closeOnce.Do(func() { s.closeErr = s.db.Close() })
	return s.closeErr
}