package e2e

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/noroadzh/gat1400-simulator/internal/adapter/capture"
	"github.com/noroadzh/gat1400-simulator/internal/adapter/httpapi"
	"github.com/noroadzh/gat1400-simulator/internal/adapter/storage"
	"github.com/noroadzh/gat1400-simulator/internal/app/application"
	"github.com/noroadzh/gat1400-simulator/internal/app/ports"
	"github.com/noroadzh/gat1400-simulator/internal/domain/ids"
	"github.com/noroadzh/gat1400-simulator/internal/domain/node"
)

// TestCaptureE2E_RecordsEveryRequest verifies that the CaptureMiddleware
// persists each request to the CaptureStore, and the BFF can serve them back.
func TestCaptureE2E_RecordsEveryRequest(t *testing.T) {
	ln := realListener(t)
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "capture-e2e.db")

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
		Auth: httpapi.AuthConfig{Realm: "viid", Username: "admin", Password: "admin", Qop: "auth"},
	})

	ts := httptest.NewUnstartedServer(srv)
	ts.Listener.Close()
	ts.Listener = ln
	ts.Start()
	t.Cleanup(ts.Close)

	// Seed a node.
	nodeID := "41000000005030312222"
	nodeSvc.UpsertNode(ctx, node.Node{
		ID: nodeID, Name: nodeID, Role: node.RoleDevice,
		Capabilities: []node.Capability{node.CapSystem, node.CapCollection},
	})

	// Send a POST with a JSON body.
	body := `{"PersonList":{"PersonObject":[{"PersonID":"P001","PersonName":"张三"}]}}`
	req, _ := http.NewRequest("POST", ts.URL+"/VIID/Persons", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/VIID+JSON")
	req.Header.Set("User-Identify", nodeID)
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer resp.Body.Close()
	io.ReadAll(resp.Body)

	// Give the async recorder a moment to persist.
	// (In production the recorder uses a buffered channel, so 100ms is safe.)
	reader, err := storage.NewCaptureReader(dbPath)
	if err != nil {
		t.Fatalf("NewCaptureReader: %v", err)
	}

	// The CaptureStore writes synchronously, but a small grace window ensures
	// any asynchronous buffering has flushed before we query.
	deadline := time.Now().Add(2 * time.Second)
	var entries []ports.CaptureEntry
	for time.Now().Before(deadline) {
		entries, err = reader.Query(ports.CaptureFilter{Limit: 100})
		if err != nil {
			t.Fatalf("Query: %v", err)
		}
		if len(entries) > 0 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if len(entries) == 0 {
		t.Fatal("no captures recorded within 2s")
	}

	// Find the Persons POST capture.
	var found bool
	for _, e := range entries {
		if e.Method == "POST" && strings.Contains(e.Path, "/VIID/Persons") {
			found = true
			if e.NodeID != nodeID {
				t.Errorf("NodeID = %q, want %q", e.NodeID, nodeID)
			}
			if e.Status != 200 {
				t.Errorf("status = %d, want 200", e.Status)
			}
			break
		}
	}
	if !found {
		t.Errorf("no Persons POST capture found in %d entries", len(entries))
	}
}
