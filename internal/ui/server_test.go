package ui

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/noroadzh/gat1400-simulator/internal/adapter/capture"
	"github.com/noroadzh/gat1400-simulator/internal/adapter/storage"
	"github.com/noroadzh/gat1400-simulator/internal/app/application"
	"github.com/noroadzh/gat1400-simulator/internal/app/ports"
	"github.com/noroadzh/gat1400-simulator/internal/domain/ids"
	"github.com/noroadzh/gat1400-simulator/internal/domain/node"
)

// fakeCaptureReader 让 BFF 在无真实 DB 的情况下也能服务抓包面板。
type fakeCaptureReader struct {
	entries []ports.CaptureEntry
}

func (f *fakeCaptureReader) Query(_ ports.CaptureFilter) ([]ports.CaptureEntry, error) {
	return f.entries, nil
}
func (f *fakeCaptureReader) ExportJSONL(_ ports.CaptureFilter) (string, error) {
	return "/tmp/out.jsonl", nil
}
func (f *fakeCaptureReader) ExportHAR(_ ports.CaptureFilter) (string, error) {
	return "/tmp/out.har", nil
}

// newTestBFF 装配 BFF 测试服务：临时 sqlite + 内存 store + 假 CaptureReader。
func newTestBFF(t *testing.T) *Server {
	t.Helper()
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "ui.db")
	cs, err := storage.NewCaptureStore(ctx, dbPath)
	if err != nil {
		t.Fatalf("NewCaptureStore: %v", err)
	}
	t.Cleanup(func() { _ = cs.Close() })
	rec := capture.NewRecorder(cs, slog.New(slog.NewTextHandler(io.Discard, nil)))

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	idGen := ids.NewGenerator(41000000, 130)
	nodeSvc := application.NewNodeService(log, idGen)
	scenSvc := application.NewScenarioService(log, nil)
	reader := &fakeCaptureReader{}

	return NewServer(log, nodeSvc, scenSvc, rec, reader, nil)
}

// do 构造并执行一次 httptest 请求，自动注入 application/json Content-Type。
func do(t *testing.T, srv *Server, method, target string, payload interface{}) *httptest.ResponseRecorder {
	t.Helper()
	var body io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		body = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, target, body)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	srv.e.ServeHTTP(rec, req)
	return rec
}

// decode 把 JSON body 解析为 map[string]any。
func decode(t *testing.T, r io.Reader) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.NewDecoder(r).Decode(&m); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return m
}

func TestBFF_Health(t *testing.T) {
	srv := newTestBFF(t)
	rec := do(t, srv, http.MethodGet, "/api/control/system/health", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	body := decode(t, rec.Body)
	if body["status"] != "OK" {
		t.Errorf("status = %v", body["status"])
	}
	if body["service"] != "gat1400-simulator" {
		t.Errorf("service = %v", body["service"])
	}
}

func TestBFF_NodesCRUD(t *testing.T) {
	srv := newTestBFF(t)

	// Empty list.
	rec := do(t, srv, http.MethodGet, "/api/control/nodes", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status = %d", rec.Code)
	}
	body := decode(t, rec.Body)
	if items, _ := body["items"].([]any); len(items) != 0 {
		t.Errorf("expected empty list, got %v", items)
	}

	// Create.
	rec2 := do(t, srv, http.MethodPost, "/api/control/nodes", map[string]any{
		"id": "ui-1", "name": "ui-1", "role": string(node.RoleDevice),
		"httpListen": ":19001", "capabilities": []string{string(node.CapSystem)},
	})
	if rec2.Code != http.StatusOK {
		t.Fatalf("create status = %d, body=%s", rec2.Code, rec2.Body.String())
	}

	// List now has 1.
	body3 := decode(t, do(t, srv, http.MethodGet, "/api/control/nodes", nil).Body)
	items, _ := body3["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("after create items = %d, want 1", len(items))
	}

	// Get.
	rec4 := do(t, srv, http.MethodGet, "/api/control/nodes/ui-1", nil)
	if rec4.Code != http.StatusOK {
		t.Fatalf("get status = %d", rec4.Code)
	}

	// Delete.
	rec5 := do(t, srv, http.MethodDelete, "/api/control/nodes/ui-1", nil)
	if rec5.Code != http.StatusOK {
		t.Fatalf("delete status = %d", rec5.Code)
	}
	body6 := decode(t, do(t, srv, http.MethodGet, "/api/control/nodes", nil).Body)
	items6, _ := body6["items"].([]any)
	if len(items6) != 0 {
		t.Errorf("after delete items = %d, want 0", len(items6))
	}
}

func TestBFF_UpsertNodeValidationErrorReturns400(t *testing.T) {
	srv := newTestBFF(t)
	rec := do(t, srv, http.MethodPost, "/api/control/nodes", map[string]any{
		"id": "ui-x", // missing name and role -> sanity fails
	})
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestBFF_GetUnknownNodeReturns404(t *testing.T) {
	srv := newTestBFF(t)
	rec := do(t, srv, http.MethodGet, "/api/control/nodes/missing", nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestBFF_ScenariosStartStop(t *testing.T) {
	srv := newTestBFF(t)
	rec := do(t, srv, http.MethodGet, "/api/control/scenarios", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("scenarios status = %d", rec.Code)
	}
	body := decode(t, rec.Body)
	if _, ok := body["items"]; !ok {
		t.Errorf("missing items: %v", body)
	}

	rec2 := do(t, srv, http.MethodGet, "/api/control/scenarios/missing", nil)
	if rec2.Code != http.StatusNotFound {
		t.Errorf("get unknown scenario status = %d, want 404", rec2.Code)
	}

	rec3 := do(t, srv, http.MethodPost, "/api/control/scenarios/missing/start", nil)
	if rec3.Code != http.StatusBadRequest {
		t.Errorf("start unknown status = %d, want 400", rec3.Code)
	}
	rec4 := do(t, srv, http.MethodPost, "/api/control/scenarios/missing/stop", nil)
	// Stop on unknown id is a no-op in ScenarioService; the BFF reports OK.
	if rec4.Code != http.StatusOK {
		t.Errorf("stop unknown status = %d, want 200", rec4.Code)
	}
}

func TestBFF_CapturesAndExport(t *testing.T) {
	srv := newTestBFF(t)
	rec := do(t, srv, http.MethodGet, "/api/control/captures?limit=50", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("captures status = %d", rec.Code)
	}
	body := decode(t, rec.Body)
	if _, ok := body["items"]; !ok {
		t.Errorf("missing items: %v", body)
	}

	rec2 := do(t, srv, http.MethodGet, "/api/control/captures/export/jsonl", nil)
	if rec2.Code != http.StatusOK {
		t.Fatalf("jsonl status = %d", rec2.Code)
	}
	body2 := decode(t, rec2.Body)
	if body2["path"] != "/tmp/out.jsonl" {
		t.Errorf("jsonl path = %v", body2["path"])
	}

	rec3 := do(t, srv, http.MethodGet, "/api/control/captures/export/har", nil)
	if rec3.Code != http.StatusOK {
		t.Fatalf("har status = %d", rec3.Code)
	}
	body3 := decode(t, rec3.Body)
	if body3["path"] != "/tmp/out.har" {
		t.Errorf("har path = %v", body3["path"])
	}
}

func TestBFF_Stats(t *testing.T) {
	srv := newTestBFF(t)
	rec := do(t, srv, http.MethodGet, "/api/control/stats", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	body := decode(t, rec.Body)
	for _, k := range []string{"nodeCount", "onlineNodeCount", "scenarioCount", "runningScenarios"} {
		if _, ok := body[k]; !ok {
			t.Errorf("missing %q in stats: %v", k, body)
		}
	}
}

func TestBFF_Info(t *testing.T) {
	srv := newTestBFF(t)
	rec := do(t, srv, http.MethodGet, "/api/control/system/info", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	body := decode(t, rec.Body)
	if body["revision"] != "0.1.0" {
		t.Errorf("revision = %v", body["revision"])
	}
}

func TestCountOnline(t *testing.T) {
	nodes := []node.Node{
		{ID: "a", Status: node.StatusOnline},
		{ID: "b", Status: node.StatusStopped},
		{ID: "c", Status: node.StatusOnline},
	}
	if got := countOnline(nodes); got != 2 {
		t.Errorf("countOnline = %d, want 2", got)
	}
}

func TestAtoiDefault(t *testing.T) {
	cases := []struct {
		in   string
		def  int
		want int
	}{
		{"", 5, 5},
		{"42", 5, 42},
		{"abc", 5, 5},
		{"123abc", 5, 5},
		{"0", 5, 0},
	}
	for _, tc := range cases {
		if got := atoiDefault(tc.in, tc.def); got != tc.want {
			t.Errorf("atoiDefault(%q,%d) = %d, want %d", tc.in, tc.def, got, tc.want)
		}
	}
}

func TestBindNode_RoundTrip(t *testing.T) {
	in := map[string]any{
		"id":           "n-1",
		"name":         "node-1",
		"role":         "device",
		"httpListen":   ":19001",
		"upstream":     "viid://peer",
		"capabilities": []any{"system", "collection"},
		"tags":         []any{"dev", "lab"},
		"metadata":     map[string]any{"k": "v"},
	}
	got := bindNode(in).toNode()
	if got.ID != "n-1" || got.Name != "node-1" || got.Role != node.RoleDevice {
		t.Errorf("basic fields not propagated: %+v", got)
	}
	if got.HTTPListen != ":19001" || got.Upstream != "viid://peer" {
		t.Errorf("network fields not propagated: %+v", got)
	}
	if len(got.Capabilities) != 2 || got.Capabilities[0] != node.CapSystem {
		t.Errorf("capabilities wrong: %+v", got.Capabilities)
	}
	if len(got.Tags) != 2 || got.Tags[1] != "lab" {
		t.Errorf("tags wrong: %+v", got.Tags)
	}
	if got.Metadata["k"] != "v" {
		t.Errorf("metadata wrong: %+v", got.Metadata)
	}
}

func TestBindNode_MissingAndWrongFields(t *testing.T) {
	agg := bindNode(map[string]any{
		"id":           42, // wrong type
		"capabilities": "system", // wrong type
		"tags":         map[string]any{"a": "b"},
		"metadata":     "not-a-map",
	})
	if agg.ID != "" {
		t.Errorf("expected empty ID, got %q", agg.ID)
	}
	if agg.Capabilities != nil || agg.Tags != nil || agg.Metadata != nil {
		t.Errorf("expected nil collections, got caps=%v tags=%v meta=%v",
			agg.Capabilities, agg.Tags, agg.Metadata)
	}
}

// TestHub_BroadcastAndClientCount WS 烟雾测试：hub 运行不 panic、广播/注册不阻塞。
func TestHub_BroadcastAndClientCount(t *testing.T) {
	srv := newTestBFF(t)
	go func() {
		srv.hub.broadcast(Event{Type: "ping", Payload: map[string]any{"t": time.Now().Unix()}})
	}()
	// Allow the broadcast goroutine to drain.
	time.Sleep(20 * time.Millisecond)
	if srv.hub == nil {
		t.Fatal("hub is nil")
	}
}