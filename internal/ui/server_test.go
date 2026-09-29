package ui

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
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
// 协议端指向 mockProtocolServer（可选）；默认使用 http://127.0.0.1:14000（不会被实际访问）。
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

	cfg := &Config{}
	cfg.Protocol.BaseURL = "http://127.0.0.1:14000"
	return NewServer(log, nodeSvc, scenSvc, rec, reader, cfg)
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
		"id":           42,       // wrong type
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

func TestAPIPathNotAffectedByFallback(t *testing.T) {
	// GET /api/control/nodes must return JSON, not the SPA fallback.
	srv := newTestBFF(t)
	req := httptest.NewRequest(http.MethodGet, "/api/control/nodes", nil)
	rec := httptest.NewRecorder()
	srv.e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Errorf("expected Content-Type application/json, got %s", ct)
	}
	body := rec.Body.String()
	if strings.Contains(body, `id="app"`) {
		t.Errorf("GET /api/control/nodes returned SPA HTML — SPA fallback leaked into API path")
	}
}

func TestWebSocketPathNotAffectedByFallback(t *testing.T) {
	// GET /ws/events with Upgrade: websocket must not return SPA fallback.
	srv := newTestBFF(t)
	req := httptest.NewRequest(http.MethodGet, "/ws/events", nil)
	req.Header.Set("Connection", "upgrade")
	req.Header.Set("Upgrade", "websocket")
	rec := httptest.NewRecorder()
	srv.e.ServeHTTP(rec, req)

	// Either 101 (if hub upgrades) or 400 (no WS handler in test) — neither should be 200 SPA HTML.
	if rec.Code == http.StatusOK && strings.Contains(rec.Body.String(), `id="app"`) {
		t.Errorf("GET /ws/events returned SPA fallback — WS path leaked into SPA catch-all")
	}
}

// --- Resource API 测试（薄透传到协议端）---

// mockProtocolServer 模拟协议端响应，用于 BFF 资源端点测试。
// 行为由 handler map 决定，未配置的请求返回 500。
func mockProtocolServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	// 计数表（GET /VIID/<Collection>）
	counts := map[string]int{
		"Persons":          3,
		"MotorVehicles":    0,
		"NonMotorVehicles": 0,
		"Faces":            0,
	}
	// GET /VIID/<Collection> 返回列表信封
	mux.HandleFunc("/VIID/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/VIID/")
		parts := strings.Split(path, "/")
		coll := parts[0]
		_, known := counts[coll]
		if !known {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if r.Method == http.MethodGet && len(parts) == 1 {
			kind := singularOf(coll)
			w.Header().Set("Content-Type", "application/VIID+JSON")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"ResponseStatus": map[string]any{"StatusCode": 0, "StatusString": "OK"},
				kind + "List":    map[string]any{kind + "Object": make([]any, counts[coll])},
			})
			return
		}
		// 单条 GET/DELETE
		if r.Method == http.MethodGet && len(parts) == 2 {
			// mock 约定：id == "missing" 返回 404（模拟协议端未命中）。
			if parts[1] == "missing" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"ResponseStatus": map[string]any{"StatusCode": 2, "StatusString": "NOTFOUND"},
				})
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"ResponseStatus": map[string]any{"StatusCode": 0, "StatusString": "OK"},
				singularOf(coll): map[string]any{"ID": parts[1]},
			})
			return
		}
		if r.Method == http.MethodDelete && len(parts) == 2 {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"ResponseStatus": map[string]any{"StatusCode": 0, "StatusString": "OK"},
			})
			return
		}
		// POST /VIID/<Collection>
		if r.Method == http.MethodPost && len(parts) == 1 {
			// 读取并 echo 客户端 body 内容到响应，便于测试验证 body 字节未丢失。
			body, _ := io.ReadAll(r.Body)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"ResponseStatus": map[string]any{"StatusCode": 0, "StatusString": "OK"},
				"ItemCount":      1,
				"ReceivedBody":   string(body),
			})
			return
		}
		// PUT /VIID/<Collection>/<id>
		if r.Method == http.MethodPut && len(parts) == 2 {
			body, _ := io.ReadAll(r.Body)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"ResponseStatus": map[string]any{"StatusCode": 0, "StatusString": "OK"},
				"ReceivedBody":   string(body),
			})
			return
		}
		// GET /VIID/<Collection>/<id>/Info
		if r.Method == http.MethodGet && len(parts) == 3 && parts[2] == "Info" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"ResponseStatus": map[string]any{"StatusCode": 0, "StatusString": "OK"},
				"Info":           map[string]any{"ID": parts[1]},
			})
			return
		}
		http.Error(w, "mock unsupported", http.StatusInternalServerError)
	})
	return httptest.NewServer(mux)
}

// singularOf 把复数 URI 段转为单数 Kind 名（与 AllKinds 命名一致）。
// 用于把 Persons -> Person 这种信封映射。
func singularOf(coll string) string {
	strip := map[string]string{
		"Persons":          "Person",
		"Faces":            "Face",
		"MotorVehicles":    "MotorVehicle",
		"NonMotorVehicles": "NonMotorVehicle",
		"Things":           "Thing",
		"Scenes":           "Scene",
		"VideoSlices":      "VideoSlice",
		"Images":           "Image",
		"Files":            "File",
		"Cases":            "Case",
		"VideoLabels":      "VideoLabel",
		"AnalysisRules":    "AnalysisRule",
	}
	return strip[coll]
}

// newBFFWithProtocol 装配 BFF，并指向给定的 mock 协议端 base URL。
func newBFFWithProtocol(t *testing.T, baseURL string) *Server {
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
	cfg := &Config{}
	cfg.Protocol.BaseURL = baseURL
	return NewServer(log, nodeSvc, scenSvc, rec, reader, cfg)
}

func TestResourceAPI_ListAllKinds(t *testing.T) {
	mock := mockProtocolServer(t)
	t.Cleanup(mock.Close)
	srv := newBFFWithProtocol(t, mock.URL)

	rec := do(t, srv, http.MethodGet, "/api/control/resources", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := decode(t, rec.Body)
	kinds, ok := body["kinds"].([]any)
	if !ok {
		t.Fatalf("kinds field missing or not array: %v", body)
	}
	if len(kinds) != 12 {
		t.Errorf("kinds length = %d, want 12", len(kinds))
	}
	// Persons 应有 count=3（mock 设置）
	for _, ki := range kinds {
		m := ki.(map[string]any)
		if m["kind"] == "Person" {
			if int(m["count"].(float64)) != 3 {
				t.Errorf("Person count = %v, want 3", m["count"])
			}
		}
	}
	// 检查 IDField 映射
	for _, ki := range kinds {
		m := ki.(map[string]any)
		switch m["kind"] {
		case "Person":
			if m["idField"] != "PersonID" {
				t.Errorf("Person idField = %v, want PersonID", m["idField"])
			}
		case "MotorVehicle":
			if m["idField"] != "MotorVehicleID" {
				t.Errorf("MotorVehicle idField = %v, want MotorVehicleID", m["idField"])
			}
		}
	}
}

func TestResourceAPI_GetByKind_Success(t *testing.T) {
	mock := mockProtocolServer(t)
	t.Cleanup(mock.Close)
	srv := newBFFWithProtocol(t, mock.URL)

	rec := do(t, srv, http.MethodGet, "/api/control/resources/Person/list/p1", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := decode(t, rec.Body)
	if body["ResponseStatus"].(map[string]any)["StatusCode"].(float64) != 0 {
		t.Errorf("ResponseStatus.StatusCode = %v, want 0", body["ResponseStatus"])
	}
}

func TestResourceAPI_GetByKind_NotFound(t *testing.T) {
	mock := mockProtocolServer(t)
	t.Cleanup(mock.Close)
	srv := newBFFWithProtocol(t, mock.URL)

	// mock 约定 /VIID/Persons/missing 返回 404。
	// BFF 应原样透传。
	rec := do(t, srv, http.MethodGet, "/api/control/resources/Person/list/missing", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 (protocol 404 must pass-through)", rec.Code)
	}
	body := decode(t, rec.Body)
	rs := body["ResponseStatus"].(map[string]any)
	if rs["StatusString"] != "NOTFOUND" {
		t.Errorf("StatusString = %v, want NOTFOUND", rs["StatusString"])
	}
}

func TestResourceAPI_CreateAndDelete_RoundTrip(t *testing.T) {
	mock := mockProtocolServer(t)
	t.Cleanup(mock.Close)
	srv := newBFFWithProtocol(t, mock.URL)

	// POST 批量写入
	body := map[string]any{
		"PersonList": map[string]any{
			"PersonObject": []any{
				map[string]any{"PersonID": "P0001", "Name": "Alice"},
			},
		},
	}
	rec := do(t, srv, http.MethodPost, "/api/control/resources/Person/list", body)
	if rec.Code != http.StatusOK {
		t.Fatalf("POST status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	resp := decode(t, rec.Body)
	if int(resp["ItemCount"].(float64)) != 1 {
		t.Errorf("ItemCount = %v, want 1", resp["ItemCount"])
	}
	// 关键：mock 把收到的 body 字符串原样 echo 进响应。验证 BFF 没吞 body 字节。
	if received, _ := resp["ReceivedBody"].(string); received == "" {
		t.Errorf("POST body lost in transit (mock received empty body)")
	}

	// DELETE
	rec2 := do(t, srv, http.MethodDelete, "/api/control/resources/Person/list/P0001", nil)
	if rec2.Code != http.StatusOK {
		t.Fatalf("DELETE status = %d, want 200", rec2.Code)
	}
}

func TestResourceAPI_InvalidKind_Returns400(t *testing.T) {
	mock := mockProtocolServer(t)
	t.Cleanup(mock.Close)
	srv := newBFFWithProtocol(t, mock.URL)

	rec := do(t, srv, http.MethodGet, "/api/control/resources/Foo/list", nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	body := decode(t, rec.Body)
	rs, ok := body["ResponseStatus"].(map[string]any)
	if !ok {
		t.Fatalf("missing ResponseStatus in body: %v", body)
	}
	if rs["StatusString"] != "INVALID" {
		t.Errorf("StatusString = %v, want INVALID", rs["StatusString"])
	}
}

func TestResourceAPI_Update_PassThrough(t *testing.T) {
	mock := mockProtocolServer(t)
	t.Cleanup(mock.Close)
	srv := newBFFWithProtocol(t, mock.URL)

	body := map[string]any{"PersonID": "p1", "Name": "AliceUpdated"}
	rec := do(t, srv, http.MethodPut, "/api/control/resources/Person/list/p1", body)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	resp := decode(t, rec.Body)
	if received, _ := resp["ReceivedBody"].(string); received == "" {
		t.Errorf("PUT body lost in transit")
	}
}

func TestResourceAPI_Info_PassThrough(t *testing.T) {
	mock := mockProtocolServer(t)
	t.Cleanup(mock.Close)
	srv := newBFFWithProtocol(t, mock.URL)

	rec := do(t, srv, http.MethodGet, "/api/control/resources/Person/list/p1/info", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("Info status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	body := decode(t, rec.Body)
	if _, ok := body["Info"]; !ok {
		t.Errorf("Info subresource field missing in upstream response: %v", body)
	}
}

func TestResourceAPI_UpstreamUnreachable_Returns502(t *testing.T) {
	// 用一个保证无监听的端口（:0 然后立即关闭拿到空闲端口）。
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := l.Addr().String()
	_ = l.Close()
	srv := newBFFWithProtocol(t, "http://"+addr)

	rec := do(t, srv, http.MethodGet, "/api/control/resources/Person/list", nil)
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502", rec.Code)
	}
	body := decode(t, rec.Body)
	rs := body["ResponseStatus"].(map[string]any)
	if rs["StatusString"] != "SERVER_ERROR" {
		t.Errorf("StatusString = %v, want SERVER_ERROR", rs["StatusString"])
	}
}

func TestResourceAPI_QueryStringForwardedToUpstream(t *testing.T) {
	// 协议端用 httptest 替代；断言 BFF 把 ?pageSize=&pageNum= 原样转发
	// 到上游 URL，否则协议端分页/过滤参数会静默丢失。
	var gotQuery string
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/VIID+JSON;charset=UTF-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"PersonList":{"PersonObject":[{"PersonID":"p1"}]}}`))
	}))
	defer mock.Close()

	srv := newTestBFF(t)
	srv.protocolClient.baseURL = mock.URL

	rec := do(t, srv, http.MethodGet,
		"/api/control/resources/Person/list?pageSize=20&pageNum=2", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	if gotQuery != "pageSize=20&pageNum=2" {
		t.Errorf("upstream RawQuery = %q, want %q", gotQuery, "pageSize=20&pageNum=2")
	}
}
