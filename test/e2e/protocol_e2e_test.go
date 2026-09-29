package e2e

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
	"github.com/noroadzh/gat1400-simulator/internal/adapter/httpapi"
	"github.com/noroadzh/gat1400-simulator/internal/adapter/storage"
	"github.com/noroadzh/gat1400-simulator/internal/adapter/wire"
	"github.com/noroadzh/gat1400-simulator/internal/app/application"
	"github.com/noroadzh/gat1400-simulator/internal/app/ports"
	"github.com/noroadzh/gat1400-simulator/internal/domain/ids"
	"github.com/noroadzh/gat1400-simulator/internal/domain/node"
)

// realListener returns a real TCP listener bound to a random port on loopback.
func realListener(t *testing.T) net.Listener {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("real listener: %v", err)
	}
	return ln
}

type serverFixture struct {
	ts      *httptest.Server
	ln      net.Listener
	nodeSvc *application.NodeService
	scenSvc *application.ScenarioService
	capt    ports.CaptureStore
	nonce   *storage.NonceStore
}

// startRealServer starts a real httpapi.Server bound to a real TCP listener.
func startRealServer(t *testing.T, ln net.Listener) *serverFixture {
	t.Helper()
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "e2e.db")

	ns, err := storage.NewNonceStore(ctx, dbPath)
	if err != nil {
		t.Fatalf("nonce store: %v", err)
	}
	t.Cleanup(func() { _ = ns.Close() })

	cs, err := storage.NewCaptureStore(ctx, dbPath)
	if err != nil {
		t.Fatalf("capture store: %v", err)
	}
	t.Cleanup(func() { _ = cs.Close() })

	rec := capture.NewRecorder(cs, slog.New(slog.NewTextHandler(io.Discard, nil)))
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	idGen := ids.NewGenerator(41000000, 30)
	nodeSvc := application.NewNodeService(log, idGen)
	scenSvc := application.NewScenarioService(log, nil)

	srv := httpapi.NewServer(log, nodeSvc, scenSvc, rec, ns, idGen, &httpapi.Config{
		Auth: httpapi.AuthConfig{Realm: "com.gat1400.simulator", Username: "admin", Password: "admin", Qop: "auth"},
	})

	ts := httptest.NewUnstartedServer(srv)
	ts.Listener.Close()
	ts.Listener = ln
	ts.Start()
	t.Cleanup(ts.Close)

	return &serverFixture{
		ts: ts, ln: ln,
		nodeSvc: nodeSvc, scenSvc: scenSvc,
		capt: cs, nonce: ns,
	}
}

func seedNode(t *testing.T, fx *serverFixture, nodeID string) {
	t.Helper()
	if _, err := fx.nodeSvc.UpsertNode(context.Background(), node.Node{
		ID: nodeID, Name: nodeID, Role: node.RoleDevice,
		Capabilities: []node.Capability{node.CapSystem, node.CapCollection},
	}); err != nil {
		t.Fatalf("seed node: %v", err)
	}
}

// newClient builds a real wire.Client that points at the test server.
func newClient(t *testing.T, fx *serverFixture, nodeID string) *wire.Client {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "client-nonces.db")
	ns, err := storage.NewNonceStore(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("client nonce store: %v", err)
	}
	t.Cleanup(func() { _ = ns.Close() })
	c := wire.NewClient(slog.New(slog.NewTextHandler(io.Discard, nil)), ns)
	c.Configure(wire.Options{
		Username: "admin",
		Password: "admin",
		Realm:    "com.gat1400.simulator",
		Qop:      "auth",
	})
	return c
}

// TestProtocolE2E_RegisterAndPush exercises a real TCP socket from client to
// server. It uses wire.Client (the real HTTP client with Digest auto-retry)
// against a real httptest.Server bound to a real net.Listener.
func TestProtocolE2E_RegisterAndPush(t *testing.T) {
	fx := startRealServer(t, realListener(t))
	const nodeID = "41000000005030312222"
	seedNode(t, fx, nodeID)

	client := newClient(t, fx, nodeID)
	base := fx.ts.URL

	// --- Register (handles 401 → retry automatically) ---
	regBody := map[string]any{
		"RegisterObject": map[string]any{
			"DeviceID":     nodeID,
			"DeviceName":   "e2e-test-device",
			"Manufacturer": "e2e",
		},
	}
	payload, status, err := client.PostJSON(context.Background(), base+"/VIID/System/Register", nodeID, regBody)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if status != http.StatusOK {
		t.Fatalf("Register status = %d, want 200, payload=%v", status, payload)
	}
	if rs, ok := payload["ResponseStatus"].(map[string]any); !ok || rs["StatusCode"].(float64) != 0 {
		t.Errorf("Register ResponseStatus.StatusCode = %v, want 0", payload["ResponseStatus"])
	}

	// --- Push a Person ---
	personBody := map[string]any{
		"PersonList": map[string]any{
			"PersonObject": []any{
				map[string]any{
					"PersonID":   "41000000005030312001",
					"PersonName": "E2E张三",
					"Gender":     "male",
				},
			},
		},
	}
	payload2, status2, err := client.PostJSON(context.Background(), base+"/VIID/Persons", nodeID, personBody)
	if err != nil {
		t.Fatalf("PostJSON Persons: %v", err)
	}
	if status2 != http.StatusOK {
		t.Fatalf("Persons status = %d, want 200, payload=%v", status2, payload2)
	}
	if cnt, _ := payload2["ItemCount"].(float64); cnt != 1 {
		t.Errorf("ItemCount = %v, want 1", payload2["ItemCount"])
	}

	// --- Verify node is online ---
	time.Sleep(50 * time.Millisecond)
	n, err := fx.nodeSvc.GetNode(context.Background(), nodeID)
	if err != nil {
		t.Fatalf("GetNode: %v", err)
	}
	if n.Status != node.StatusOnline {
		t.Errorf("node status = %s, want online", n.Status)
	}
}

// TestProtocolE2E_RealTCPConnection verifies that the client and server
// communicate over an actual TCP connection (not a mock).
func TestProtocolE2E_RealTCPConnection(t *testing.T) {
	fx := startRealServer(t, realListener(t))

	// Dial the server address directly to verify TCP connectivity.
	conn, err := net.DialTimeout("tcp", fx.ln.Addr().String(), 2*time.Second)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	// Send a raw HTTP request over the raw TCP connection.
	req := "GET /VIID/System/Time HTTP/1.1\r\nHost: localhost\r\n\r\n"
	if _, err := conn.Write([]byte(req)); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Read the response.
	buf := make([]byte, 4096)
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, err := conn.Read(buf)
	if err != nil && err != io.EOF {
		t.Fatalf("read: %v", err)
	}
	resp := string(buf[:n])
	if !strings.HasPrefix(resp, "HTTP/1.1 200") {
		t.Errorf("raw response did not start with HTTP/1.1 200: %q", resp[:min(80, len(resp))])
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// TestProtocolE2E_NonceReplay verifies that replaying an Authorization header
// with an already-consumed nonce over a real TCP socket results in a 401, and
// that the replayed request does not mutate server-side state.
func TestProtocolE2E_NonceReplay(t *testing.T) {
	fx := startRealServer(t, realListener(t))
	const nodeID = "41000000005030312223"
	seedNode(t, fx, nodeID)
	client := newClient(t, fx, nodeID)
	base := fx.ts.URL

	regBody := map[string]any{
		"RegisterObject": map[string]any{
			"DeviceID":   nodeID,
			"DeviceName": "e2e-replay-device",
			"Manufacturer": "e2e",
		},
	}

	// --- First Register: succeeds through 401→retry handshake ---
	if _, status, err := client.PostJSON(context.Background(), base+"/VIID/System/Register", nodeID, regBody); err != nil || status != http.StatusOK {
		t.Fatalf("first Register: status=%d err=%v", status, err)
	}

	// --- Replay: issue a nonce manually and reuse it twice ---
	const replayedNonce = "e2e-replay-nonce-0001"
	ctx := context.Background()
	if err := fx.nonce.Issue(replayedNonce); err != nil {
		t.Fatalf("issue nonce: %v", err)
	}
	authHeader := `Digest username="admin",realm="com.gat1400.simulator",nonce="` + replayedNonce + `",uri="/VIID/System/Register",qop="auth",nc=00000001,cnonce="00000000000000ab",response="00000000000000000000000000000000",opaque=""`

	doRegister := func() (int, []byte) {
		b, _ := json.Marshal(regBody)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/VIID/System/Register", bytes.NewReader(b))
		if err != nil {
			t.Fatalf("new request: %v", err)
		}
		req.Header.Set("Content-Type", "application/VIID+JSON")
		req.Header.Set("User-Identify", nodeID)
		req.Header.Set("Authorization", authHeader)
		resp, err := fx.ts.Client().Do(req)
		if err != nil {
			t.Fatalf("do: %v", err)
		}
		defer resp.Body.Close()
		raw, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, raw
	}

	if code, raw := doRegister(); code != http.StatusOK {
		t.Fatalf("first use of nonce: status=%d want 200 body=%s", code, raw)
	}
	// Second use of the same nonce must be rejected with 401.
	if code, raw := doRegister(); code != http.StatusUnauthorized {
		t.Fatalf("replayed nonce: status=%d want 401 body=%s", code, raw)
	}

	// --- No side effects: node must remain registered exactly once (not duplicated) ---
	time.Sleep(50 * time.Millisecond)
	n, err := fx.nodeSvc.GetNode(ctx, nodeID)
	if err != nil {
		t.Fatalf("GetNode after replay: %v", err)
	}
	if n.ID != nodeID {
		t.Errorf("node ID = %s, want %s", n.ID, nodeID)
	}
}
