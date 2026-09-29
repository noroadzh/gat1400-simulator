package scenario

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/noroadzh/gat1400-simulator/internal/domain/node"
	"github.com/noroadzh/gat1400-simulator/internal/adapter/wire"
)

func newTestDispatcher(t *testing.T, handler http.HandlerFunc) (*OutboundDispatcher, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	client := wire.NewClient(logger, nil)
	d := NewOutboundDispatcher(client, logger, nil)
	return d, srv
}

// TestDispatchKeepalive_PathAndHeaders verifies DispatchKeepalive posts to the
// expected URI with the expected User-Identify header.
func TestDispatchKeepalive_PathAndHeaders(t *testing.T) {
	var gotPath, gotDeviceID string
	d, srv := newTestDispatcher(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotDeviceID = r.Header.Get("User-Identify")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ResponseStatus":{"StatusCode":0}}`))
	})

	target := node.Node{ID: "DEV00000000000000000001", HTTPListen: srv.URL[7:]} // strip "http://"
	if err := d.DispatchKeepalive(context.Background(), target); err != nil {
		t.Fatalf("DispatchKeepalive: %v", err)
	}
	if gotPath != "/VIID/System/Keepalive" {
		t.Fatalf("path = %q", gotPath)
	}
	if gotDeviceID != "DEV00000000000000000001" {
		t.Fatalf("User-Identify = %q", gotDeviceID)
	}
}

// TestDispatchUnregister_AllowsFailure verifies an Unregister against an
// unreachable URL returns an error without panicking. The engine treats such
// errors as best-effort.
func TestDispatchUnregister_AllowsFailure(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	client := wire.NewClient(logger, nil)
	d := NewOutboundDispatcher(client, logger, nil)

	target := node.Node{ID: "DEV00000000000000000001", HTTPListen: ":1"} // refuse
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	err := d.DispatchUnregister(ctx, target)
	if err == nil {
		t.Fatalf("expected error connecting to :1, got nil")
	}
}

// TestDispatchRegister_Success verifies DispatchRegister posts RegisterObject.
func TestDispatchRegister_Success(t *testing.T) {
	var gotPath string
	var bodyBytes []byte
	d, srv := newTestDispatcher(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		bodyBytes, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ResponseStatus":{"StatusCode":0}}`))
	})

	target := node.Node{ID: "DEV00000000000000000001", HTTPListen: srv.URL[7:]}
	body := map[string]any{
		"RegisterObject": map[string]any{"DeviceID": "DEV00000000000000000001"},
	}
	if err := d.DispatchRegister(context.Background(), target, body); err != nil {
		t.Fatalf("DispatchRegister: %v", err)
	}
	if gotPath != "/VIID/System/Register" {
		t.Fatalf("path = %q", gotPath)
	}
	if len(bodyBytes) == 0 {
		t.Fatal("body empty")
	}
	if !strings.Contains(string(bodyBytes), "RegisterObject") {
		t.Fatalf("body missing RegisterObject: %s", bodyBytes)
	}
}

// TestDispatchSubscribe_Path verifies DispatchSubscribe posts to /VIID/Subscribes.
func TestDispatchSubscribe_Path(t *testing.T) {
	var gotPath string
	d, srv := newTestDispatcher(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ResponseStatus":{"StatusCode":0}}`))
	})
	target := node.Node{ID: "DEV00000000000000000001", HTTPListen: srv.URL[7:]}
	if err := d.DispatchSubscribe(context.Background(), target, map[string]any{}); err != nil {
		t.Fatalf("DispatchSubscribe: %v", err)
	}
	if gotPath != "/VIID/Subscribes" {
		t.Fatalf("path = %q", gotPath)
	}
}

// TestDispatchSubscribeNotification_Path verifies DispatchSubscribeNotification
// posts to /VIID/SubscribeNotifications.
func TestDispatchSubscribeNotification_Path(t *testing.T) {
	var gotPath string
	d, srv := newTestDispatcher(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ResponseStatus":{"StatusCode":0}}`))
	})
	target := node.Node{ID: "DEV00000000000000000001", HTTPListen: srv.URL[7:]}
	if err := d.DispatchSubscribeNotification(context.Background(), target, map[string]any{"NotifyInfo": "x"}); err != nil {
		t.Fatalf("DispatchSubscribeNotification: %v", err)
	}
	if gotPath != "/VIID/SubscribeNotifications" {
		t.Fatalf("path = %q", gotPath)
	}
}

// TestDispatchDisposition_Path verifies DispatchDisposition posts to /VIID/Dispositions.
func TestDispatchDisposition_Path(t *testing.T) {
	var gotPath string
	d, srv := newTestDispatcher(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ResponseStatus":{"StatusCode":0}}`))
	})
	target := node.Node{ID: "DEV00000000000000000001", HTTPListen: srv.URL[7:]}
	if err := d.DispatchDisposition(context.Background(), target, map[string]any{}); err != nil {
		t.Fatalf("DispatchDisposition: %v", err)
	}
	if gotPath != "/VIID/Dispositions" {
		t.Fatalf("path = %q", gotPath)
	}
}

// TestDispatch_NilRecorderSafe verifies record() does not panic when recorder is nil.
func TestDispatch_NilRecorderSafe(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	client := wire.NewClient(logger, nil)
	d := NewOutboundDispatcher(client, logger, nil) // recorder == nil
	if d.recorder != nil {
		t.Fatal("expected nil recorder")
	}
	// Should not panic on any dispatch path.
	d2, srv := newTestDispatcher(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ResponseStatus":{"StatusCode":0}}`))
	})
	d2.recorder = nil
	target := node.Node{ID: "DEV00000000000000000001", HTTPListen: srv.URL[7:]}
	_ = d2.DispatchKeepalive(context.Background(), target)
	_ = d2.DispatchUnregister(context.Background(), target)
	_ = d2.DispatchSubscribe(context.Background(), target, map[string]any{})
}