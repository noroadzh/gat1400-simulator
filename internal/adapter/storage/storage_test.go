package storage

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/noroadzh/gat1400-simulator/internal/app/ports"
)

// tempDB 在临时目录下创建一个测试用 SQLite 数据库路径。
func tempDB(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	return filepath.Join(dir, "sim.db")
}

// TestNonceStore_IssueAndConsume 验证 nonce 签发、消费、未知 nonce 报错。
func TestNonceStore_IssueAndConsume(t *testing.T) {
	path := tempDB(t)
	ctx := context.Background()
	ns, err := NewNonceStore(ctx, path)
	if err != nil {
		t.Fatalf("NewNonceStore: %v", err)
	}
	t.Cleanup(func() { _ = ns.Close() })

	if err := ns.Issue("n-1"); err != nil {
		t.Fatalf("Issue: %v", err)
	}
	if err := ns.Consume("n-1"); err != nil {
		t.Fatalf("Consume: %v", err)
	}
	if err := ns.Consume("n-2"); err != ErrNonceUnknown {
		t.Fatalf("Consume(n-2) = %v, want ErrNonceUnknown", err)
	}
}

// TestNonceStore_ConsumeAllowsExactlyOneUse 验证 nonce 一次性使用语义。
// GA/T 1400.4 §5.1 要求服务端按 (nonce, nc) 联合主键去重：第二次 Consume
// 同一 nonce 必须在服务层返回 ErrNonceUnknown，从而触发 401 挑战。
func TestNonceStore_ConsumeAllowsExactlyOneUse(t *testing.T) {
	path := tempDB(t)
	ctx := context.Background()
	ns, err := NewNonceStore(ctx, path)
	if err != nil {
		t.Fatalf("NewNonceStore: %v", err)
	}
	t.Cleanup(func() { _ = ns.Close() })

	if err := ns.Issue("n-1"); err != nil {
		t.Fatalf("Issue: %v", err)
	}
	// First consume MUST succeed.
	if err := ns.Consume("n-1"); err != nil {
		t.Fatalf("first Consume: %v", err)
	}
	// Second consume of the SAME nonce MUST return ErrNonceUnknown — replay protection.
	if err := ns.Consume("n-1"); err != ErrNonceUnknown {
		t.Fatalf("second Consume = %v, want ErrNonceUnknown", err)
	}
	// Unknown nonce is also rejected.
	if err := ns.Consume("unknown"); err != ErrNonceUnknown {
		t.Fatalf("Consume(unknown) = %v", err)
	}
}

// TestNonceStore_ExpiresAfterTTL 验证 TTL 过期后 nonce 不可消费。
func TestNonceStore_ExpiresAfterTTL(t *testing.T) {
	path := tempDB(t)
	ctx := context.Background()
	ns, err := NewNonceStore(ctx, path)
	if err != nil {
		t.Fatalf("NewNonceStore: %v", err)
	}
	// Inject a tiny TTL by mutating the field directly — acceptable for tests
	// because the field is unexported and lives in the same package.
	ns.ttl = 10 * time.Millisecond
	t.Cleanup(func() { _ = ns.Close() })

	if err := ns.Issue("n-1"); err != nil {
		t.Fatalf("Issue: %v", err)
	}
	time.Sleep(50 * time.Millisecond)
	if err := ns.Consume("n-1"); err != ErrNonceUnknown {
		t.Fatalf("Consume after expiry = %v, want ErrNonceUnknown", err)
	}
	if err := ns.Purge(); err != nil {
		t.Fatalf("Purge: %v", err)
	}
}

// TestCaptureStore_AppendAndQuery 验证抓包写入、查询、排序与 JSON 往返。
func TestCaptureStore_AppendAndQuery(t *testing.T) {
	path := tempDB(t)
	ctx := context.Background()
	cs, err := NewCaptureStore(ctx, path)
	if err != nil {
		t.Fatalf("NewCaptureStore: %v", err)
	}
	t.Cleanup(func() { _ = cs.Close() })

	cr, err := NewCaptureReader(path)
	if err != nil {
		t.Fatalf("NewCaptureReader: %v", err)
	}
	t.Cleanup(func() { _ = cr.Close() })

	now := time.Now().UTC().Truncate(time.Second)
	for i := 0; i < 5; i++ {
		entry := ports.CaptureEntry{
			NodeID:     "node-A",
			Direction:  "inbound",
			Method:     "POST",
			Path:       "/VIID/Persons",
			URL:        "http://x/VIID/Persons",
			Remote:     "127.0.0.1:1234",
			Status:     200,
			DurationMs: 10,
			StartedAt:  now.Add(time.Duration(i) * time.Second),
			Header:     map[string][]string{"Content-Type": {"application/VIID+JSON"}},
			Request:    map[string]any{"PersonID": "p1"},
			Response:   map[string]any{"ResponseStatus": map[string]any{"StatusCode": 0}},
		}
		if err := cs.Append(entry); err != nil {
			t.Fatalf("Append[%d]: %v", i, err)
		}
	}

	got, err := cr.Query(ports.CaptureFilter{NodeID: "node-A"})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(got) != 5 {
		t.Fatalf("len(got) = %d, want 5", len(got))
	}
	// Newest first.
	if !got[0].StartedAt.After(got[len(got)-1].StartedAt) && !got[0].StartedAt.Equal(got[len(got)-1].StartedAt) {
		t.Fatalf("ordering broken: first=%v last=%v", got[0].StartedAt, got[len(got)-1].StartedAt)
	}
	// round-trip JSON
	if got[0].Request["PersonID"] != "p1" {
		t.Fatalf("request payload not roundtripped: %v", got[0].Request)
	}
	if _, ok := got[0].Response["ResponseStatus"]; !ok {
		t.Fatalf("response payload not roundtripped: %v", got[0].Response)
	}
}

// TestCaptureStore_QueryFilters 验证 NodeID/Direction/Status/Method/Path/From/To 过滤条件。
func TestCaptureStore_QueryFilters(t *testing.T) {
	path := tempDB(t)
	ctx := context.Background()
	cs, err := NewCaptureStore(ctx, path)
	if err != nil {
		t.Fatalf("NewCaptureStore: %v", err)
	}
	t.Cleanup(func() { _ = cs.Close() })
	cr, err := NewCaptureReader(path)
	if err != nil {
		t.Fatalf("NewCaptureReader: %v", err)
	}
	t.Cleanup(func() { _ = cr.Close() })

	now := time.Now().UTC()
	mk := func(node, dir, method, path_ string, status int, ts time.Time) ports.CaptureEntry {
		return ports.CaptureEntry{
			NodeID: node, Direction: dir, Method: method, Path: path_, URL: "http://x" + path_,
			Remote: "127.0.0.1", Status: status, StartedAt: ts,
		}
	}
	cs.Append(mk("A", "inbound", "POST", "/VIID/Persons", 200, now))
	cs.Append(mk("A", "outbound", "POST", "/VIID/Persons", 500, now.Add(1*time.Second)))
	cs.Append(mk("B", "inbound", "GET", "/VIID/Time", 200, now.Add(2*time.Second)))

	if got, _ := cr.Query(ports.CaptureFilter{NodeID: "A"}); len(got) != 2 {
		t.Fatalf("NodeID filter: got %d", len(got))
	}
	if got, _ := cr.Query(ports.CaptureFilter{Direction: "inbound"}); len(got) != 2 {
		t.Fatalf("Direction filter: got %d", len(got))
	}
	if got, _ := cr.Query(ports.CaptureFilter{Status: 500}); len(got) != 1 {
		t.Fatalf("Status filter: got %d", len(got))
	}
	if got, _ := cr.Query(ports.CaptureFilter{Method: "GET"}); len(got) != 1 {
		t.Fatalf("Method filter: got %d", len(got))
	}
	if got, _ := cr.Query(ports.CaptureFilter{Path: "Persons"}); len(got) != 2 {
		t.Fatalf("Path filter: got %d", len(got))
	}
	if got, _ := cr.Query(ports.CaptureFilter{From: now.Add(1 * time.Second)}); len(got) != 2 {
		t.Fatalf("From filter: got %d", len(got))
	}
	if got, _ := cr.Query(ports.CaptureFilter{To: now.Add(500 * time.Millisecond)}); len(got) != 1 {
		t.Fatalf("To filter: got %d", len(got))
	}
}

// TestCaptureReader_ExportJSONL 验证 JSONL 导出文件格式与内容。
func TestCaptureReader_ExportJSONL(t *testing.T) {
	path := tempDB(t)
	ctx := context.Background()
	cs, _ := NewCaptureStore(ctx, path)
	t.Cleanup(func() { _ = cs.Close() })
	cr, _ := NewCaptureReader(path)
	t.Cleanup(func() { _ = cr.Close() })

	now := time.Now().UTC()
	cs.Append(ports.CaptureEntry{NodeID: "A", Direction: "inbound", Method: "POST", Path: "/x", URL: "http://x/x", Status: 200, StartedAt: now})

	tmp, err := cr.ExportJSONL(ports.CaptureFilter{})
	if err != nil {
		t.Fatalf("ExportJSONL: %v", err)
	}
	defer os.Remove(tmp)
	b, err := os.ReadFile(tmp)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if len(b) == 0 {
		t.Fatalf("empty jsonl")
	}
	var e ports.CaptureEntry
	if err := json.Unmarshal(b, &e); err != nil {
		t.Fatalf("first line not parseable: %v", err)
	}
	if e.NodeID != "A" {
		t.Fatalf("jsonl entry nodeId = %q", e.NodeID)
	}
}

// TestCaptureReader_ExportHAR 验证 HAR 1.2 文档格式与字段完整性。
func TestCaptureReader_ExportHAR(t *testing.T) {
	path := tempDB(t)
	ctx := context.Background()
	cs, _ := NewCaptureStore(ctx, path)
	t.Cleanup(func() { _ = cs.Close() })
	cr, _ := NewCaptureReader(path)
	t.Cleanup(func() { _ = cr.Close() })

	cs.Append(ports.CaptureEntry{NodeID: "A", Direction: "inbound", Method: "POST", Path: "/x", URL: "http://x/x", Status: 200, StartedAt: time.Now().UTC()})

	tmp, err := cr.ExportHAR(ports.CaptureFilter{})
	if err != nil {
		t.Fatalf("ExportHAR: %v", err)
	}
	defer os.Remove(tmp)
	b, err := os.ReadFile(tmp)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var doc struct {
		Log struct {
			Version  string           `json:"version"`
			Creator  map[string]any   `json:"creator"`
			Entries  []map[string]any `json:"entries"`
		} `json:"log"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatalf("HAR parse: %v\n%s", err, b)
	}
	if doc.Log.Version != "1.2" {
		t.Fatalf("HAR version = %q", doc.Log.Version)
	}
	if len(doc.Log.Entries) != 1 {
		t.Fatalf("HAR entries = %d", len(doc.Log.Entries))
	}
}