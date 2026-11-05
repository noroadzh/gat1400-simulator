package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/noroadzh/gat1400-simulator/internal/adapter/capture"
	"github.com/noroadzh/gat1400-simulator/internal/adapter/storage"
	"github.com/noroadzh/gat1400-simulator/internal/app/application"
	"github.com/noroadzh/gat1400-simulator/internal/domain/ids"
	"github.com/noroadzh/gat1400-simulator/internal/domain/node"
	"github.com/noroadzh/gat1400-simulator/internal/domain/response"
)

// newTestServer 将协议服务端与临时 sqlite store 配接，
// 返回 Server、NodeService 与 ScenarioService，供进程内调用。
func newTestServer(t *testing.T) (*Server, *application.NodeService, *application.ScenarioService) {
	t.Helper()
	ctx := context.Background()

	dbPath := filepath.Join(t.TempDir(), "test.db")
	nonce, err := storage.NewNonceStore(ctx, dbPath)
	if err != nil {
		t.Fatalf("NewNonceStore: %v", err)
	}
	t.Cleanup(func() { _ = nonce.Close() })

	cap, err := storage.NewCaptureStore(ctx, dbPath)
	if err != nil {
		t.Fatalf("NewCaptureStore: %v", err)
	}
	t.Cleanup(func() { _ = cap.Close() })

	rec := capture.NewRecorder(cap, slog.New(slog.NewTextHandler(io.Discard, nil)))
	idGen := ids.NewGenerator(41000000, 130)
	nodeSvc := application.NewNodeService(slog.New(slog.NewTextHandler(io.Discard, nil)), idGen)
	scenSvc := application.NewScenarioService(slog.New(slog.NewTextHandler(io.Discard, nil)), nil)

	cfg := &Config{Auth: AuthConfig{
		Realm:    "test.realm",
		Username: "admin",
		Password: "admin",
		Qop:      "auth",
	}}
	srv := NewServer(slog.New(slog.NewTextHandler(io.Discard, nil)), nodeSvc, scenSvc, rec, nonce, idGen, cfg)
	return srv, nodeSvc, scenSvc
}

// decodeResponse 将响应 body 解码为 map[string]any，供测试断言使用。
func decodeResponse(t *testing.T, body io.Reader) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.NewDecoder(body).Decode(&m); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return m
}

// doRequest 对 Server 执行 httptest 请求，返回响应记录器。
func doRequest(srv *Server, req *http.Request) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	srv.e.ServeHTTP(rec, req)
	return rec
}

// mustJSONRequest 构造一个带 application/VIID+JSON Content-Type 的请求。
func mustJSONRequest(t *testing.T, method, target string, body any) *http.Request {
	t.Helper()
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		r = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, target, r)
	req.Header.Set("Content-Type", "application/VIID+JSON")
	return req
}

// mustChallenge 触发一次 401 挑战并返回 WWW-Authenticate 头的值。
// 每次调用都会由服务端签发一个新 nonce，供后续合法 Digest 请求使用。
func mustChallenge(t *testing.T, srv *Server, method, path string) string {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	rec := doRequest(srv, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("challenge request status = %d, want 401", rec.Code)
	}
	chal := rec.Header().Get("WWW-Authenticate")
	if chal == "" {
		t.Fatal("missing WWW-Authenticate challenge header")
	}
	return chal
}

// digestAuthHeader 从 WWW-Authenticate 挑战中提取 nonce，
// 构造一个结构合规的 Digest Authorization 头（response 字段不做完整性校验，
// 因为服务端 verifyAuthorization 只校验 nonce 的存在与新鲜度）。
func digestAuthHeader(challenge, uri string) string {
	nonce := extractNonceFromChallenge(challenge)
	return `Digest username="admin", realm="test.realm", nonce="` + nonce +
		`", uri="` + uri + `", response="x", nc=00000001, cnonce="x", qop=auth, algorithm=MD5`
}

// extractNonceFromChallenge 从 "Digest realm=..., nonce="abc", ..." 中提取 nonce 值。
func extractNonceFromChallenge(challenge string) string {
	const marker = `nonce="`
	i := strings.Index(challenge, marker)
	if i < 0 {
		return ""
	}
	rest := challenge[i+len(marker):]
	if j := strings.Index(rest, `"`); j >= 0 {
		return rest[:j]
	}
	return rest
}

// --- System 路由：Register / Keepalive / Time ---

func TestSystem_RegisterRequiresDigestAndMarksNodeOnline(t *testing.T) {
	srv, nodeSvc, _ := newTestServer(t)
	// Register a node so MarkSeen succeeds.
	if _, err := nodeSvc.UpsertNode(context.Background(), node.Node{
		ID: "41000000005030312222", Name: "cam-1", Role: node.RoleDevice,
	}); err != nil {
		t.Fatalf("UpsertNode: %v", err)
	}

	// First request without Authorization header must be rejected with 401
	// and carry a Digest challenge.
	req := mustJSONRequest(t, http.MethodPost, "/VIID/System/Register",
		map[string]any{"RegisterObject": map[string]any{"DeviceID": "41000000005030312222"}})
	rec := doRequest(srv, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("first call status = %d, want 401", rec.Code)
	}
	if !strings.Contains(rec.Header().Get("WWW-Authenticate"), "Digest ") {
		t.Fatalf("WWW-Authenticate missing Digest challenge")
	}

	// Second request with a valid nonce (echoed from the challenge) is accepted.
	req2 := mustJSONRequest(t, http.MethodPost, "/VIID/System/Register",
		map[string]any{"RegisterObject": map[string]any{"DeviceID": "41000000005030312222"}})
	req2.Header.Set("Authorization", digestAuthHeader(rec.Header().Get("WWW-Authenticate"), "/VIID/System/Register"))
	rec2 := doRequest(srv, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("second call status = %d, body=%s", rec2.Code, rec2.Body.String())
	}
	got, _ := nodeSvc.GetNode(context.Background(), "41000000005030312222")
	if got.Status != node.StatusOnline {
		t.Errorf("Status = %s, want online", got.Status)
	}
}

// TestNonceReplayRejected 验证 RFC 2617 §3 重放保护：
// 同一 nonce 的第二次请求必须被服务端拒绝（401 挑战）。
// GA/T 1400.4 §5.1 要求服务端按 (nonce, nc) 联合主键去重。
func TestNonceReplayRejected(t *testing.T) {
	srv, nodeSvc, _ := newTestServer(t)
	if _, err := nodeSvc.UpsertNode(context.Background(), node.Node{
		ID: "41000000005030312222", Name: "cam-1", Role: node.RoleDevice,
	}); err != nil {
		t.Fatalf("UpsertNode: %v", err)
	}

	// First request: obtain a fresh challenge and use its nonce.
	chal := mustChallenge(t, srv, http.MethodPost, "/VIID/System/Register")
	auth := digestAuthHeader(chal, "/VIID/System/Register")

	req := mustJSONRequest(t, http.MethodPost, "/VIID/System/Register",
		map[string]any{"RegisterObject": map[string]any{"DeviceID": "41000000005030312222"}})
	req.Header.Set("Authorization", auth)
	rec := doRequest(srv, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("first request status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}

	// Second request: replay the SAME nonce — MUST be rejected with 401.
	req2 := mustJSONRequest(t, http.MethodPost, "/VIID/System/Register",
		map[string]any{"RegisterObject": map[string]any{"DeviceID": "41000000005030312222"}})
	req2.Header.Set("Authorization", auth)
	rec2 := doRequest(srv, req2)
	if rec2.Code != http.StatusUnauthorized {
		t.Fatalf("replay status = %d, want 401, body=%s", rec2.Code, rec2.Body.String())
	}
	if !strings.Contains(rec2.Header().Get("WWW-Authenticate"), "Digest ") {
		t.Fatalf("replay must include a new Digest challenge")
	}
}

func TestSystem_RegisterInvalidBodyReturns400(t *testing.T) {
	srv, _, _ := newTestServer(t)
	// Obtain a valid nonce via a 401 challenge before sending the malformed body.
	chal := mustChallenge(t, srv, http.MethodPost, "/VIID/System/Register")
	req := httptest.NewRequest(http.MethodPost, "/VIID/System/Register",
		strings.NewReader(`{"foo":"bar"}`))
	req.Header.Set("Content-Type", "application/VIID+JSON")
	req.Header.Set("Authorization", digestAuthHeader(chal, "/VIID/System/Register"))
	rec := doRequest(srv, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	body := decodeResponse(t, rec.Body)
	if code, ok := body["StatusCode"].(float64); !ok || int(code) != int(response.CodeInvalid) {
		t.Errorf("StatusCode = %v, want %d, body=%v", body["StatusCode"], response.CodeInvalid, body)
	}
}

func TestSystem_RegisterUnknownDeviceReturns404(t *testing.T) {
	srv, _, _ := newTestServer(t)
	chal := mustChallenge(t, srv, http.MethodPost, "/VIID/System/Register")
	req := mustJSONRequest(t, http.MethodPost, "/VIID/System/Register",
		map[string]any{"RegisterObject": map[string]any{"DeviceID": "unknown"}})
	req.Header.Set("Authorization", digestAuthHeader(chal, "/VIID/System/Register"))
	rec := doRequest(srv, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestSystem_KeepaliveRequiresUserIdentify(t *testing.T) {
	srv, _, _ := newTestServer(t)
	req := httptest.NewRequest(http.MethodPost, "/VIID/System/Keepalive", nil)
	rec := doRequest(srv, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("missing User-Identify status = %d", rec.Code)
	}
}

func TestSystem_KeepaliveMarksSeen(t *testing.T) {
	srv, nodeSvc, _ := newTestServer(t)
	if _, err := nodeSvc.UpsertNode(context.Background(), node.Node{
		ID: "DEV-1", Name: "DEV-1", Role: node.RoleDevice,
	}); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/VIID/System/Keepalive", nil)
	req.Header.Set("User-Identify", "DEV-1")
	rec := doRequest(srv, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	got, _ := nodeSvc.GetNode(context.Background(), "DEV-1")
	if got.Status != node.StatusOnline {
		t.Errorf("Status = %s, want online", got.Status)
	}
}

func TestSystem_TimeReturnsRFC3339(t *testing.T) {
	srv, _, _ := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/VIID/System/Time", nil)
	rec := doRequest(srv, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	body := decodeResponse(t, rec.Body)
	rs, _ := body["ResponseStatus"].(map[string]any)
	if rs == nil || rs["StatusCode"].(float64) != float64(response.CodeOK) {
		t.Errorf("unexpected ResponseStatus: %v", rs)
	}
	if _, ok := body["Time"].(string); !ok {
		t.Errorf("expected Time string, got %T", body["Time"])
	}
}

// --- Collection 路由：Persons/Faces/Vehicles 等 CRUD ---

func TestCollection_PostPersonsRoundTrip(t *testing.T) {
	srv, _, _ := newTestServer(t)
	body := map[string]any{
		"PersonList": map[string]any{
			"PersonObject": []any{
				map[string]any{"PersonID": "p1", "Name": "Alice"},
				map[string]any{"PersonID": "p2", "Name": "Bob"},
			},
		},
	}
	req := mustJSONRequest(t, http.MethodPost, "/VIID/Persons", body)
	rec := doRequest(srv, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	got := decodeResponse(t, rec.Body)
	if got["ItemCount"].(float64) != 2 {
		t.Errorf("ItemCount = %v, want 2", got["ItemCount"])
	}

	// List should now return both persons.
	req2 := httptest.NewRequest(http.MethodGet, "/VIID/Persons", nil)
	rec2 := doRequest(srv, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("list status = %d", rec2.Code)
	}
	list := decodeResponse(t, rec2.Body)
	plist, _ := list["PersonList"].(map[string]any)
	if plist == nil {
		t.Fatalf("PersonList missing: %v", list)
	}
	objs, _ := plist["PersonObject"].([]any)
	if len(objs) != 2 {
		t.Errorf("PersonObject len = %d, want 2", len(objs))
	}

	// Get / Data / Info subresources.
	for _, path := range []string{"/VIID/Persons/p1", "/VIID/Persons/p1/Info", "/VIID/Persons/p1/Data"} {
		req3 := httptest.NewRequest(http.MethodGet, path, nil)
		rec3 := doRequest(srv, req3)
		if rec3.Code != http.StatusOK {
			t.Errorf("GET %s status = %d, body=%s", path, rec3.Code, rec3.Body.String())
		}
	}

	// Delete then 404.
	req4 := httptest.NewRequest(http.MethodDelete, "/VIID/Persons/p1", nil)
	rec4 := doRequest(srv, req4)
	if rec4.Code != http.StatusOK {
		t.Fatalf("delete status = %d", rec4.Code)
	}
	req5 := httptest.NewRequest(http.MethodGet, "/VIID/Persons/p1", nil)
	rec5 := doRequest(srv, req5)
	if rec5.Code != http.StatusNotFound {
		t.Errorf("after delete status = %d, want 404", rec5.Code)
	}
}

func TestCollection_PostMissingEnvelopeReturns400(t *testing.T) {
	srv, _, _ := newTestServer(t)
	req := mustJSONRequest(t, http.MethodPost, "/VIID/Persons", map[string]any{"oops": true})
	rec := doRequest(srv, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestCollection_PostPersonsRejectsSkippingID(t *testing.T) {
	srv, _, _ := newTestServer(t)
	body := map[string]any{
		"PersonList": map[string]any{
			"PersonObject": []any{
				map[string]any{"Name": "no-id"},  // missing PersonID; ignored
				map[string]any{"PersonID": "p1"},
			},
		},
	}
	req := mustJSONRequest(t, http.MethodPost, "/VIID/Persons", body)
	rec := doRequest(srv, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	got := decodeResponse(t, rec.Body)
	if got["ItemCount"].(float64) != 1 {
		t.Errorf("ItemCount = %v, want 1 (item without id dropped)", got["ItemCount"])
	}
}

// --- Cascade 路由：Subscribes / SubscribeNotifications / Dispositions ---

func TestSubscribe_CreateListDelete(t *testing.T) {
	srv, _, _ := newTestServer(t)

	req := mustJSONRequest(t, http.MethodPost, "/VIID/Subscribes",
		map[string]any{"SubscribeID": "S-1", "Title": "watch"})
	rec := doRequest(srv, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("create status = %d", rec.Code)
	}

	// List.
	reqList := httptest.NewRequest(http.MethodGet, "/VIID/Subscribes", nil)
	recList := doRequest(srv, reqList)
	if recList.Code != http.StatusOK {
		t.Fatalf("list status = %d", recList.Code)
	}
	list := decodeResponse(t, recList.Body)
	sl, _ := list["SubscribeList"].(map[string]any)
	if sl == nil {
		t.Fatalf("SubscribeList missing: %v", list)
	}
	objs, _ := sl["SubscribeObject"].([]any)
	if len(objs) != 1 {
		t.Errorf("SubscribeObject len = %d, want 1", len(objs))
	}

	// Update.
	reqUp := mustJSONRequest(t, http.MethodPut, "/VIID/Subscribes/S-1",
		map[string]any{"SubscribeID": "S-1", "Title": "watch-v2"})
	recUp := doRequest(srv, reqUp)
	if recUp.Code != http.StatusOK {
		t.Fatalf("update status = %d", recUp.Code)
	}

	// Delete.
	reqDel := httptest.NewRequest(http.MethodDelete, "/VIID/Subscribes/S-1", nil)
	recDel := doRequest(srv, reqDel)
	if recDel.Code != http.StatusOK {
		t.Fatalf("delete status = %d", recDel.Code)
	}

	// List after delete is empty.
	recList2 := doRequest(srv, httptest.NewRequest(http.MethodGet, "/VIID/Subscribes", nil))
	list2 := decodeResponse(t, recList2.Body)
	sl2, _ := list2["SubscribeList"].(map[string]any)
	objs2, _ := sl2["SubscribeObject"].([]any)
	if len(objs2) != 0 {
		t.Errorf("after delete SubscribeObject len = %d, want 0", len(objs2))
	}
}

func TestSubscribeNotifications_RecordedAndListed(t *testing.T) {
	srv, _, _ := newTestServer(t)

	for i := 0; i < 3; i++ {
		req := mustJSONRequest(t, http.MethodPost, "/VIID/SubscribeNotifications",
			map[string]any{"SubscribeID": "S-X", "Title": "match", "InfoIDs": []string{"a", "b"}})
		rec := doRequest(srv, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("create notification #%d status = %d", i, rec.Code)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/VIID/SubscribeNotifications?subscribeId=S-X", nil)
	rec := doRequest(srv, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status = %d", rec.Code)
	}
	body := decodeResponse(t, rec.Body)
	sl, _ := body["SubscribeNotificationList"].(map[string]any)
	if sl == nil {
		t.Fatalf("SubscribeNotificationList missing: %v", body)
	}
	objs, _ := sl["SubscribeNotificationObject"].([]any)
	if len(objs) != 3 {
		t.Errorf("SubscribeNotificationObject len = %d, want 3", len(objs))
	}
	for i, obj := range objs {
		m, _ := obj.(map[string]any)
		if m == nil || m["NotificationID"] == "" {
			t.Errorf("entry %d missing NotificationID: %v", i, m)
		}
	}
}

func TestDispositions_CreateAndUpdate(t *testing.T) {
	srv, _, _ := newTestServer(t)
	req := mustJSONRequest(t, http.MethodPost, "/VIID/Dispositions",
		map[string]any{"DispositionID": "D-1", "Title": "wanted"})
	rec := doRequest(srv, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("create status = %d", rec.Code)
	}

	reqUp := mustJSONRequest(t, http.MethodPut, "/VIID/Dispositions/D-1",
		map[string]any{"DispositionID": "D-1", "Title": "wanted-v2"})
	recUp := doRequest(srv, reqUp)
	if recUp.Code != http.StatusOK {
		t.Fatalf("update status = %d", recUp.Code)
	}

	reqDel := httptest.NewRequest(http.MethodDelete, "/VIID/Dispositions/D-1", nil)
	recDel := doRequest(srv, reqDel)
	if recDel.Code != http.StatusOK {
		t.Fatalf("delete status = %d", recDel.Code)
	}
}

// --- Catalog 路由：APE / APS / Tollgate / Lane（静态数据） ---

func TestCatalog_APEsAndTollgatesHaveEntries(t *testing.T) {
	srv, _, _ := newTestServer(t)
	for _, path := range []string{"/VIID/APEs", "/VIID/APSs", "/VIID/Tollgates", "/VIID/Lanes"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := doRequest(srv, req)
		if rec.Code != http.StatusOK {
			t.Errorf("%s status = %d", path, rec.Code)
		}
		body := decodeResponse(t, rec.Body)
		// Each catalog response carries a list with at least one entry.
		found := false
		for _, v := range body {
			if m, ok := v.(map[string]any); ok {
				if objs, ok := m["APEObject"].([]any); ok && len(objs) > 0 {
					found = true
				}
				if objs, ok := m["APSObject"].([]any); ok && len(objs) > 0 {
					found = true
				}
				if objs, ok := m["TollgateObject"].([]any); ok && len(objs) > 0 {
					found = true
				}
				if objs, ok := m["LaneObject"].([]any); ok && len(objs) > 0 {
					found = true
				}
			}
		}
		if !found {
			t.Errorf("%s returned empty: %v", path, body)
		}
	}
}

// --- Binder：application/VIID+JSON Content-Type 解析 ---

func TestBinder_AcceptsVIIDJSONContentType(t *testing.T) {
	srv, _, _ := newTestServer(t)
	body := map[string]any{
		"PersonList": map[string]any{
			"PersonObject": []any{
				map[string]any{"PersonID": "v1"},
			},
		},
	}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/VIID/Persons", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/VIID+JSON")
	rec := doRequest(srv, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
}

func TestBinder_FallsBackToJSON(t *testing.T) {
	srv, _, _ := newTestServer(t)
	body := map[string]any{
		"SubscribeID": "S-1",
	}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/VIID/Subscribes", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	rec := doRequest(srv, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
}

// --- 表 13 根资源 / VIIDServers / 通知实例 / 表 14 VIAS ---

func TestVIIDRootAndSystem(t *testing.T) {
	srv, _, _ := newTestServer(t)

	rec := doRequest(srv, httptest.NewRequest(http.MethodGet, "/VIID", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("/VIID status = %d", rec.Code)
	}
	body := decodeResponse(t, rec.Body)
	list, _ := body["ResourceURIList"].(map[string]any)
	uris, _ := list["ResourceURI"].([]any)
	if len(uris) < 5 {
		t.Fatalf("ResourceURI too short: %v", uris)
	}

	recSys := doRequest(srv, httptest.NewRequest(http.MethodGet, "/VIID/System", nil))
	if recSys.Code != http.StatusOK {
		t.Fatalf("/VIID/System status = %d", recSys.Code)
	}
	sysBody := decodeResponse(t, recSys.Body)
	sys, _ := sysBody["System"].(map[string]any)
	if sys == nil || sys["Name"] == "" {
		t.Fatalf("System missing: %v", sysBody)
	}
}

func TestVIIDServers_CRUD(t *testing.T) {
	srv, _, _ := newTestServer(t)

	recList := doRequest(srv, httptest.NewRequest(http.MethodGet, "/VIID/VIIDServers", nil))
	if recList.Code != http.StatusOK {
		t.Fatalf("list status = %d", recList.Code)
	}
	listBody := decodeResponse(t, recList.Body)
	sl, _ := listBody["VIIDServerList"].(map[string]any)
	objs, _ := sl["VIIDServerObject"].([]any)
	if len(objs) < 1 {
		t.Fatalf("seed VIIDServer missing: %v", listBody)
	}

	id := "41000000005030319999"
	req := mustJSONRequest(t, http.MethodPost, "/VIID/VIIDServers", map[string]any{
		"VIIDServerList": map[string]any{
			"VIIDServerObject": []any{
				map[string]any{"VIIDServerID": id, "ServerName": "peer"},
			},
		},
	})
	if rec := doRequest(srv, req); rec.Code != http.StatusOK {
		t.Fatalf("upsert status = %d", rec.Code)
	}

	recGet := doRequest(srv, httptest.NewRequest(http.MethodGet, "/VIID/VIIDServers/"+id, nil))
	if recGet.Code != http.StatusOK {
		t.Fatalf("get status = %d", recGet.Code)
	}

	recDel := doRequest(srv, httptest.NewRequest(http.MethodDelete, "/VIID/VIIDServers/"+id, nil))
	if recDel.Code != http.StatusOK {
		t.Fatalf("delete status = %d", recDel.Code)
	}
	recMiss := doRequest(srv, httptest.NewRequest(http.MethodGet, "/VIID/VIIDServers/"+id, nil))
	if recMiss.Code != http.StatusNotFound {
		t.Fatalf("get after delete status = %d, want 404", recMiss.Code)
	}
}

func TestSubscribeNotification_GetAndDeleteByID(t *testing.T) {
	srv, _, _ := newTestServer(t)

	req := mustJSONRequest(t, http.MethodPost, "/VIID/SubscribeNotifications",
		map[string]any{"SubscribeID": "S-Y", "Title": "hit"})
	rec := doRequest(srv, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("create status = %d", rec.Code)
	}
	created := decodeResponse(t, rec.Body)
	nid, _ := created["NotificationID"].(string)
	if nid == "" {
		t.Fatalf("NotificationID missing: %v", created)
	}

	recGet := doRequest(srv, httptest.NewRequest(http.MethodGet, "/VIID/SubscribeNotifications/"+nid, nil))
	if recGet.Code != http.StatusOK {
		t.Fatalf("get status = %d body=%s", recGet.Code, recGet.Body.String())
	}

	recDel := doRequest(srv, httptest.NewRequest(http.MethodDelete, "/VIID/SubscribeNotifications/"+nid, nil))
	if recDel.Code != http.StatusOK {
		t.Fatalf("delete status = %d", recDel.Code)
	}
	recMiss := doRequest(srv, httptest.NewRequest(http.MethodGet, "/VIID/SubscribeNotifications/"+nid, nil))
	if recMiss.Code != http.StatusNotFound {
		t.Fatalf("get after delete = %d, want 404", recMiss.Code)
	}
}

func TestDispositionNotification_GetAndDeleteByID(t *testing.T) {
	srv, _, _ := newTestServer(t)

	req := mustJSONRequest(t, http.MethodPost, "/VIID/DispositionNotifications",
		map[string]any{"DispositionID": "D-Y", "Title": "alarm"})
	rec := doRequest(srv, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("create status = %d", rec.Code)
	}
	created := decodeResponse(t, rec.Body)
	nid, _ := created["DispositionNotificationID"].(string)
	if nid == "" {
		t.Fatalf("DispositionNotificationID missing: %v", created)
	}

	recGet := doRequest(srv, httptest.NewRequest(http.MethodGet, "/VIID/DispositionNotifications/"+nid, nil))
	if recGet.Code != http.StatusOK {
		t.Fatalf("get status = %d", recGet.Code)
	}
	recDel := doRequest(srv, httptest.NewRequest(http.MethodDelete, "/VIID/DispositionNotifications/"+nid, nil))
	if recDel.Code != http.StatusOK {
		t.Fatalf("delete status = %d", recDel.Code)
	}
}

func TestCollection_BulkPutDeleteAndPagination(t *testing.T) {
	srv, _, _ := newTestServer(t)

	post := mustJSONRequest(t, http.MethodPost, "/VIID/Persons", map[string]any{
		"PersonList": map[string]any{
			"PersonObject": []any{
				map[string]any{"PersonID": "P1", "Name": "a"},
				map[string]any{"PersonID": "P2", "Name": "b"},
				map[string]any{"PersonID": "P3", "Name": "a"},
			},
		},
	})
	if rec := doRequest(srv, post); rec.Code != http.StatusOK {
		t.Fatalf("post = %d", rec.Code)
	}

	put := mustJSONRequest(t, http.MethodPut, "/VIID/Persons", map[string]any{
		"PersonList": map[string]any{
			"PersonObject": []any{map[string]any{"PersonID": "P2", "Name": "b2"}},
		},
	})
	if rec := doRequest(srv, put); rec.Code != http.StatusOK {
		t.Fatalf("put = %d", rec.Code)
	}

	list := doRequest(srv, httptest.NewRequest(http.MethodGet, "/VIID/Persons?Name=a&PageRecordNum=1&RecordStartNo=1", nil))
	if list.Code != http.StatusOK {
		t.Fatalf("list = %d", list.Code)
	}
	body := decodeResponse(t, list.Body)
	if body["TotalNum"].(float64) != 2 {
		t.Fatalf("TotalNum = %v want 2", body["TotalNum"])
	}
	pl := body["PersonList"].(map[string]any)
	objs := pl["PersonObject"].([]any)
	if len(objs) != 1 {
		t.Fatalf("page len = %d want 1", len(objs))
	}

	del := httptest.NewRequest(http.MethodDelete, "/VIID/Persons?PersonID=P1,P3", nil)
	if rec := doRequest(srv, del); rec.Code != http.StatusOK {
		t.Fatalf("delete = %d", rec.Code)
	}
	left := doRequest(srv, httptest.NewRequest(http.MethodGet, "/VIID/Persons", nil))
	leftBody := decodeResponse(t, left.Body)
	if leftBody["TotalNum"].(float64) != 1 {
		t.Fatalf("after delete TotalNum = %v", leftBody["TotalNum"])
	}
}

func TestCatalog_GetByID(t *testing.T) {
	srv, _, _ := newTestServer(t)
	rec := doRequest(srv, httptest.NewRequest(http.MethodGet, "/VIID/APEs/41000000005030312222", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("get ape = %d", rec.Code)
	}
	miss := doRequest(srv, httptest.NewRequest(http.MethodGet, "/VIID/APEs/no-such", nil))
	if miss.Code != http.StatusNotFound {
		t.Fatalf("miss = %d want 404", miss.Code)
	}
}

func TestObjectStore_PersistPerson(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "obj.db")
	store, err := storage.NewObjectStore(ctx, dbPath)
	if err != nil {
		t.Fatalf("NewObjectStore: %v", err)
	}
	defer func() { _ = store.Close() }()

	srv, _, _ := newTestServer(t)
	srv.AttachObjectStore(store, slog.New(slog.NewTextHandler(io.Discard, nil)))

	post := mustJSONRequest(t, http.MethodPost, "/VIID/Persons", map[string]any{
		"PersonList": map[string]any{
			"PersonObject": []any{map[string]any{"PersonID": "PX", "Name": "persist"}},
		},
	})
	if rec := doRequest(srv, post); rec.Code != http.StatusOK {
		t.Fatalf("post = %d", rec.Code)
	}

	srv2, _, _ := newTestServer(t)
	store2, err := storage.NewObjectStore(ctx, dbPath)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer func() { _ = store2.Close() }()
	srv2.AttachObjectStore(store2, slog.New(slog.NewTextHandler(io.Discard, nil)))
	get := doRequest(srv2, httptest.NewRequest(http.MethodGet, "/VIID/Persons/PX", nil))
	if get.Code != http.StatusOK {
		t.Fatalf("reload get = %d body=%s", get.Code, get.Body.String())
	}
}

func TestVIAS_TasksAndControls(t *testing.T) {
	srv, _, _ := newTestServer(t)

	recRoot := doRequest(srv, httptest.NewRequest(http.MethodGet, "/VIAS", nil))
	if recRoot.Code != http.StatusOK {
		t.Fatalf("/VIAS status = %d", recRoot.Code)
	}

	req := mustJSONRequest(t, http.MethodPost, "/VIAS/Tasks", map[string]any{
		"TaskList": map[string]any{
			"TaskObject": []any{map[string]any{"TaskName": "face-detect"}},
		},
	})
	rec := doRequest(srv, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("create task status = %d body=%s", rec.Code, rec.Body.String())
	}
	created := decodeResponse(t, rec.Body)
	ids, _ := created["TaskIDList"].([]any)
	if len(ids) != 1 {
		t.Fatalf("TaskIDList = %v", ids)
	}
	taskID, _ := ids[0].(string)

	recGet := doRequest(srv, httptest.NewRequest(http.MethodGet, "/VIAS/Tasks/"+taskID, nil))
	if recGet.Code != http.StatusOK {
		t.Fatalf("get task status = %d", recGet.Code)
	}

	ctrl := mustJSONRequest(t, http.MethodPost, "/VIAS/TaskControls", map[string]any{
		"TaskID": taskID, "Action": "Start",
	})
	if recCtrl := doRequest(srv, ctrl); recCtrl.Code != http.StatusOK {
		t.Fatalf("control status = %d", recCtrl.Code)
	}
	st := doRequest(srv, httptest.NewRequest(http.MethodGet, "/VIAS/TaskStatuses/"+taskID, nil))
	if st.Code != http.StatusOK {
		t.Fatalf("status get = %d", st.Code)
	}
	stBody := decodeResponse(t, st.Body)
	ts, _ := stBody["TaskStatus"].(map[string]any)
	if ts["Status"] != "Running" {
		t.Fatalf("Status = %v, want Running", ts["Status"])
	}

	img := mustJSONRequest(t, http.MethodPost, "/VIAS/Tasks/"+taskID+"/Images", map[string]any{
		"ImageList": map[string]any{
			"ImageObject": []any{map[string]any{"ImageID": "img-1", "Data": map[string]any{"fmt": "jpeg"}}},
		},
	})
	if recImg := doRequest(srv, img); recImg.Code != http.StatusOK {
		t.Fatalf("images upsert = %d", recImg.Code)
	}
	recImgGet := doRequest(srv, httptest.NewRequest(http.MethodGet, "/VIAS/Tasks/"+taskID+"/Images/img-1", nil))
	if recImgGet.Code != http.StatusOK {
		t.Fatalf("image get = %d", recImgGet.Code)
	}

	cap := doRequest(srv, httptest.NewRequest(http.MethodGet, "/VIAS/SystemCapability/AnalysisCapability", nil))
	if cap.Code != http.StatusOK {
		t.Fatalf("capability = %d", cap.Code)
	}
}
