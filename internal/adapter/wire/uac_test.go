package wire

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestSystem_Register_401Retry verifies Register performs Digest 401 → auto-retry.
func TestSystem_Register_401Retry(t *testing.T) {
	const (
		user  = "admin"
		pass  = "admin123"
		realm = "gat1400"
	)

	var attempts int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		hdr := r.Header.Get("Authorization")
		if hdr == "" {
			w.Header().Set("WWW-Authenticate",
				`Digest realm="`+realm+`", nonce="N-1", qop="auth", algorithm=MD5`)
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"ResponseStatus":{"StatusCode":6}}`))
			return
		}
		// Second attempt must have Authorization header
		if !strings.Contains(hdr, `username="`+user+`"`) {
			t.Errorf("second attempt missing username in Authorization: %s", hdr)
		}
		w.Header().Set("Content-Type", "application/VIID+JSON")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ResponseStatus":{"StatusCode":0,"StatusString":"OK"}}`))
	}))
	defer srv.Close()

	c := NewClient(nil, nil)
	c.Configure(Options{Username: user, Password: pass, Qop: "auth"})

	err := c.System().Register(t.Context(), srv.URL, RegisterObject{
		DeviceID: "DEV00000000000000000001",
		Status:   "ONLINE",
		Keepalive: 60,
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
}

// TestSystem_Keepalive_Success verifies Keepalive hits correct path and returns OK.
func TestSystem_Keepalive_Success(t *testing.T) {
	var gotPath, gotDeviceID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotDeviceID = r.Header.Get("User-Identify")
		w.Header().Set("Content-Type", "application/VIID+JSON")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ResponseStatus":{"StatusCode":0}}`))
	}))
	defer srv.Close()

	c := NewClient(nil, nil)
	err := c.System().Keepalive(t.Context(), srv.URL, "DEV00000000000000000001")
	if err != nil {
		t.Fatalf("Keepalive: %v", err)
	}
	if gotPath != "/VIID/System/Keepalive" {
		t.Fatalf("path = %q, want /VIID/System/Keepalive", gotPath)
	}
	if gotDeviceID != "DEV00000000000000000001" {
		t.Fatalf("User-Identify = %q", gotDeviceID)
	}
}

// TestSystem_ServerTime_RFC3339 verifies ServerTime returns RFC3339 with "T".
func TestSystem_ServerTime_RFC3339(t *testing.T) {
	wantTime := time.Now().UTC().Format(time.RFC3339)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/VIID+JSON")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"Time":"` + wantTime + `"}`))
	}))
	defer srv.Close()

	c := NewClient(nil, nil)
	got, err := c.System().ServerTime(t.Context(), srv.URL)
	if err != nil {
		t.Fatalf("ServerTime: %v", err)
	}
	if !strings.Contains(got, "T") {
		t.Fatalf("ServerTime = %q, want RFC3339 with T separator", got)
	}
}

// TestSystem_UnRegister_OK verifies UnRegister hits correct path with DeviceID.
func TestSystem_UnRegister_OK(t *testing.T) {
	var gotPath, gotDeviceID string
	var bodyBytes []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotDeviceID = r.Header.Get("User-Identify")
		bodyBytes, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/VIID+JSON")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ResponseStatus":{"StatusCode":0}}`))
	}))
	defer srv.Close()

	c := NewClient(nil, nil)
	err := c.System().UnRegister(t.Context(), srv.URL, "DEV00000000000000000001")
	if err != nil {
		t.Fatalf("UnRegister: %v", err)
	}
	if gotPath != "/VIID/System/UnRegister" {
		t.Fatalf("path = %q, want /VIID/System/UnRegister", gotPath)
	}
	if gotDeviceID != "DEV00000000000000000001" {
		t.Fatalf("User-Identify = %q", gotDeviceID)
	}
	var body map[string]any
	if err := json.Unmarshal(bodyBytes, &body); err != nil {
		t.Fatalf("body not JSON: %v", err)
	}
	unRegObj, ok := body["UnRegisterObject"].(map[string]any)
	if !ok {
		t.Fatalf("body missing UnRegisterObject: %v", body)
	}
	if unRegObj["DeviceID"] != "DEV00000000000000000001" {
		t.Fatalf("UnRegisterObject.DeviceID = %v", unRegObj["DeviceID"])
	}
}

// TestCascade_SubscribeCreate_List_Delete verifies the full Subscribe lifecycle.
func TestCascade_SubscribeCreate_List_Delete(t *testing.T) {
	const subscribeID = "123456789012"

	var (
		subsMux sync.Mutex
		subs    []string
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/VIID/Subscribes":
			switch r.Method {
			case http.MethodGet:
				// SubscribeList: 返回真实的包裹对象格式
				subsMux.Lock()
				resp := `{"SubscribeList":{"SubscribeObject":[`
				for i, sub := range subs {
					if i > 0 {
						resp += ","
					}
					resp += `{"SubscribeID":"` + sub + `"}`
				}
				resp += `]}}`
				subsMux.Unlock()
				w.Header().Set("Content-Type", "application/VIID+JSON")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(resp))
			case http.MethodDelete:
				// SubscribeDelete
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"ResponseStatus":{"StatusCode":0}}`))
			default:
				// SubscribeCreate: 解析 body 并存储
				var req map[string]any
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					t.Fatalf("bad JSON: %v", err)
				}
				if subList, ok := req["SubscribeList"].([]any); ok {
					for _, item := range subList {
						if m, ok := item.(map[string]any); ok {
							if id, ok := m["SubscribeID"].(string); ok {
								subsMux.Lock()
								subs = append(subs, id)
								subsMux.Unlock()
							}
						}
					}
				}
				w.Header().Set("Content-Type", "application/VIID+JSON")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"SubscribeList":{"SubscribeObject":[{"SubscribeID":"` + subscribeID + `"}]}}`))
			}
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := NewClient(nil, nil)
	sys := c.Cascade()
	ctx := t.Context()
	deviceID := "DEV00000000000000000001"

	// Create
	gotID, err := sys.SubscribeCreate(ctx, srv.URL, deviceID, map[string]any{
		"SubscribeList": []map[string]any{
			{"SubscribeID": subscribeID},
		},
	})
	if err != nil {
		t.Fatalf("SubscribeCreate: %v", err)
	}
	if gotID != subscribeID {
		t.Fatalf("SubscribeID = %q, want %q", gotID, subscribeID)
	}

	// List
	list, err := sys.SubscribeList(ctx, srv.URL, deviceID)
	if err != nil {
		t.Fatalf("SubscribeList: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("SubscribeList length = %d, want 1", len(list))
	}
	if list[0]["SubscribeID"] != subscribeID {
		t.Fatalf("SubscribeList[0].SubscribeID = %v", list[0]["SubscribeID"])
	}

	// Delete
	err = sys.SubscribeDelete(ctx, srv.URL, deviceID, subscribeID)
	if err != nil {
		t.Fatalf("SubscribeDelete: %v", err)
	}
}

// TestCascade_DispositionCreate verifies DispositionCreate returns DispositionID.
func TestCascade_DispositionCreate(t *testing.T) {
	const dispositionID = "DISP00000001"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/VIID/Dispositions" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/VIID+JSON")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"DispositionList":[{"DispositionID":"` + dispositionID + `"}]}`))
	}))
	defer srv.Close()

	c := NewClient(nil, nil)
	gotID, err := c.Cascade().DispositionCreate(t.Context(), srv.URL, "DEV00000000000000000001", map[string]any{
		"DispositionList": []map[string]any{
			{"DispositionID": dispositionID},
		},
	})
	if err != nil {
		t.Fatalf("DispositionCreate: %v", err)
	}
	if gotID != dispositionID {
		t.Fatalf("DispositionID = %q, want %q", gotID, dispositionID)
	}
}

// TestCascade_SubscribeNotificationPush verifies notification POST hits correct path.
func TestCascade_SubscribeNotificationPush(t *testing.T) {
	var gotPath string
	var bodyBytes []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		bodyBytes, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/VIID+JSON")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ResponseStatus":{"StatusCode":0}}`))
	}))
	defer srv.Close()

	c := NewClient(nil, nil)
	err := c.Cascade().SubscribeNotificationPush(t.Context(), srv.URL, "DEV00000000000000000001", map[string]any{
		"NotifyInfo": "test",
	})
	if err != nil {
		t.Fatalf("SubscribeNotificationPush: %v", err)
	}
	if gotPath != "/VIID/SubscribeNotifications" {
		t.Fatalf("path = %q, want /VIID/SubscribeNotifications", gotPath)
	}
	var body map[string]any
	if err := json.Unmarshal(bodyBytes, &body); err != nil {
		t.Fatalf("body not JSON: %v", err)
	}
	if body["NotifyInfo"] != "test" {
		t.Fatalf("NotifyInfo = %v", body["NotifyInfo"])
	}
}
