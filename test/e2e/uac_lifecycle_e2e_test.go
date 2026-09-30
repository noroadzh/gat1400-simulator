package e2e

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/noroadzh/gat1400-simulator/internal/adapter/capture"
	scenarioadapter "github.com/noroadzh/gat1400-simulator/internal/adapter/scenario"
	"github.com/noroadzh/gat1400-simulator/internal/adapter/storage"
	"github.com/noroadzh/gat1400-simulator/internal/adapter/wire"
	"github.com/noroadzh/gat1400-simulator/internal/app/application"
	"github.com/noroadzh/gat1400-simulator/internal/domain/ids"
	"github.com/noroadzh/gat1400-simulator/internal/domain/node"
	"github.com/noroadzh/gat1400-simulator/internal/domain/resource"
	scenariodomain "github.com/noroadzh/gat1400-simulator/internal/domain/scenario"
)

// TestUAC_Lifecycle_FullChain runs the full device lifecycle against a real
// protocol server: Register → Keepalive → Resource push → UnRegister.
func TestUAC_Lifecycle_FullChain(t *testing.T) {
	fx := startRealServer(t, realListener(t))
	ctx := context.Background()
	deviceID := "41000000001320000001"

	// Seed the device so the server accepts its Register.
	seedNode(t, fx, deviceID)

	// Wire client as UAC, pointing at the real server.
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	client := wire.NewClient(log, fx.nonce)
	client.Configure(wire.Options{
		Username: "admin",
		Password: "admin",
		Realm:    "com.gat1400.simulator",
		Qop:      "auth",
		Timeout:  5 * time.Second,
	})
	sys := client.System()

	// 1. Register
	if err := sys.Register(ctx, fx.ts.URL, wire.RegisterObject{
		DeviceID:  deviceID,
		Status:    "ONLINE",
		Keepalive: 60,
	}); err != nil {
		t.Fatalf("Register: %v", err)
	}

	// Node should be marked online by the server.
	got, err := fx.nodeSvc.GetNode(ctx, deviceID)
	if err != nil {
		t.Fatalf("GetNode after Register: %v", err)
	}
	if got == nil {
		t.Fatalf("node %s not registered on server", deviceID)
	}

	// 2. Keepalive
	if err := sys.Keepalive(ctx, fx.ts.URL, deviceID); err != nil {
		t.Fatalf("Keepalive: %v", err)
	}

	// 3. Person resource push via dispatcher
	cs, err := storage.NewCaptureStore(ctx, t.TempDir()+"/capture.db")
	if err != nil {
		t.Fatalf("NewCaptureStore: %v", err)
	}
	t.Cleanup(func() { _ = cs.Close() })
	rec := capture.NewRecorder(cs, log)
	disp := scenarioadapter.NewOutboundDispatcher(client, log, rec)
	target := node.Node{ID: deviceID, HTTPListen: fx.ts.URL[len("http://"):]}
	err = disp.Dispatch(ctx, target, resource.KindPerson, map[string]any{
		"PersonList": []map[string]any{
			{"PersonID": deviceID, "Name": "e2e"},
		},
	})
	if err != nil {
		t.Fatalf("Dispatch Person: %v", err)
	}

	// 4. UnRegister
	if err := sys.UnRegister(ctx, fx.ts.URL, deviceID); err != nil {
		t.Fatalf("UnRegister: %v", err)
	}
	got, _ = fx.nodeSvc.GetNode(ctx, deviceID)
	if got != nil && got.Status == node.StatusOnline {
		t.Fatalf("node still online after UnRegister: %+v", got)
	}
}

// TestUAC_Cascade_FullChain covers Subscribe → List → Disposition → Delete on
// the real protocol server.
func TestUAC_Cascade_FullChain(t *testing.T) {
	fx := startRealServer(t, realListener(t))
	ctx := context.Background()
	deviceID := "41000000001320000002"

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	client := wire.NewClient(log, fx.nonce)
	client.Configure(wire.Options{
		Username: "admin",
		Password: "admin",
		Realm:    "com.gat1400.simulator",
		Qop:      "auth",
		Timeout:  5 * time.Second,
	})
	cas := client.Cascade()

	// 1. SubscribeCreate
	subID, err := cas.SubscribeCreate(ctx, fx.ts.URL, deviceID, map[string]any{
		"SubscribeList": []map[string]any{
			{
				"SubscribeID":     "123456789012",
				"Title":           "e2e-sub",
				"SubscribeReason": "test",
			},
		},
	})
	if err != nil {
		t.Fatalf("SubscribeCreate: %v", err)
	}
	if subID == "" {
		t.Fatal("SubscribeCreate returned empty ID")
	}

	// 2. SubscribeList
	list, err := cas.SubscribeList(ctx, fx.ts.URL, deviceID)
	if err != nil {
		t.Fatalf("SubscribeList: %v", err)
	}
	if len(list) == 0 {
		t.Fatal("SubscribeList empty after SubscribeCreate")
	}

	// 3. DispositionCreate
	dispID, err := cas.DispositionCreate(ctx, fx.ts.URL, deviceID, map[string]any{
		"DispositionList": []map[string]any{
			{
				"DispositionID":   "91000000001000000001",
				"DispositionCode": "010000",
				"Title":           "e2e-disp",
			},
		},
	})
	if err != nil {
		t.Fatalf("DispositionCreate: %v", err)
	}
	if dispID == "" {
		t.Fatal("DispositionCreate returned empty ID")
	}

	// 4. SubscribeDelete
	if err := cas.SubscribeDelete(ctx, fx.ts.URL, deviceID, subID); err != nil {
		t.Fatalf("SubscribeDelete: %v", err)
	}

	// 5. SubscribeNotification push
	if err := cas.SubscribeNotificationPush(ctx, fx.ts.URL, deviceID, map[string]any{
		"SubscribeID": subID,
		"Title":       "e2e",
		"TriggerTime": time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		t.Fatalf("SubscribeNotificationPush: %v", err)
	}
}

// TestUAC_EngineLifecycle_RegisterAndUnregister exercises the scenario engine's
// full device lifecycle: Start fires Register + keepalives; Stop fires UnRegister.
func TestUAC_EngineLifecycle_RegisterAndUnregister(t *testing.T) {
	var mu sync.Mutex
	var registers, keepalives, unregisters int
	recorder := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		switch r.URL.Path {
		case "/VIID/System/Register":
			registers++
		case "/VIID/System/Keepalive":
			keepalives++
		case "/VIID/System/UnRegister":
			unregisters++
		}
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ResponseStatus":{"StatusCode":0}}`))
	})
	upstream := startMockUpstream(t, recorder)

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	client := wire.NewClient(log, nil)
	client.Configure(wire.Options{Timeout: 3 * time.Second})
	ctx := context.Background()
	cs, err := storage.NewCaptureStore(ctx, t.TempDir()+"/capture.db")
	if err != nil {
		t.Fatalf("NewCaptureStore: %v", err)
	}
	t.Cleanup(func() { _ = cs.Close() })
	rec := capture.NewRecorder(cs, log)
	disp := scenarioadapter.NewOutboundDispatcher(client, log, rec)

	nodeSvc := application.NewNodeService(log, ids.NewGenerator(41000000, 30))
	factory := scenarioadapter.NewFactory(ids.NewGenerator(41000000, 30), 42)
	engine := scenarioadapter.NewEngine(log, factory, disp, rec, nodeSvc)
	engine.SetKeepaliveInterval(100 * time.Millisecond)

	sc := scenariodomain.Scenario{
		ID: "e2e-lifecycle",
		Nodes: []scenariodomain.NodeSpec{
			{Ref: "dev-1", Role: node.RoleDevice, Listen: ":0", Upstream: upstream},
		},
		Resources: []scenariodomain.ResourceSpec{},
	}

	if err := engine.Start(ctx, sc); err != nil {
		t.Fatalf("engine.Start: %v", err)
	}

	// Give the engine time to fire Register and a couple of keepalives.
	time.Sleep(400 * time.Millisecond)
	if err := engine.Stop("e2e-lifecycle"); err != nil {
		t.Fatalf("engine.Stop: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if registers < 1 {
		t.Fatalf("expected >=1 Register, got %d", registers)
	}
	if keepalives < 1 {
		t.Fatalf("expected >=1 Keepalive, got %d", keepalives)
	}
	if unregisters < 1 {
		t.Fatalf("expected >=1 UnRegister, got %d", unregisters)
	}
}

// startMockUpstream returns a plain HTTP URL string backed by h.
func startMockUpstream(t *testing.T, h http.HandlerFunc) string {
	t.Helper()
	srv := httptestServer(t, h)
	return srv.URL
}

// httptestServer wraps httptest.NewServer with t.Cleanup.
func httptestServer(t *testing.T, h http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return srv
}

var _ = json.Marshal // keep encoding/json import when unused paths trimmed