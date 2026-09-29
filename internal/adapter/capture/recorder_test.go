package capture

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/noroadzh/gat1400-simulator/internal/adapter/storage"
	"github.com/noroadzh/gat1400-simulator/internal/app/ports"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func newTestStore(t *testing.T) (ports.CaptureStore, func()) {
	t.Helper()
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "capture.db")
	cap, err := storage.NewCaptureStore(ctx, dbPath)
	if err != nil {
		t.Fatalf("NewCaptureStore: %v", err)
	}
	return cap, func() { _ = cap.Close() }
}

func TestRecorder_RecordPersistsEntry(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()
	rec := NewRecorder(store, discardLogger())

	rec.Record(context.Background(), Capture{
		NodeID:    "node-1",
		Direction: "inbound",
		Method:    http.MethodPost,
		Path:      "/VIID/Persons",
		URL:       "/VIID/Persons",
		Remote:    "127.0.0.1:1234",
		Status:    http.StatusOK,
		Header:    http.Header{"User-Identify": []string{"node-1"}},
		Request:   bytes.NewReader([]byte(`{"hello":"world"}`)),
		Response:  bytes.NewReader([]byte(`{"ok":true}`)),
	})
}

func TestRecorder_RecordWithoutBodiesStillSucceeds(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()
	rec := NewRecorder(store, discardLogger())

	rec.Record(context.Background(), Capture{
		NodeID:    "node-1",
		Direction: "inbound",
		Method:    http.MethodGet,
		Path:      "/VIID/Persons",
		Status:    http.StatusOK,
	})
}

func TestRecorder_NilReceiverIsSafe(t *testing.T) {
	var r *Recorder
	r.Record(context.Background(), Capture{NodeID: "x"}) // must not panic
}

func TestRecorder_NilLoggerDoesNotPanic(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()
	rec := NewRecorder(store, nil)
	rec.Record(context.Background(), Capture{NodeID: "x", Method: http.MethodGet, Status: http.StatusOK})
}

func TestRecorder_DecodeJSONMapHandlesRawAndEmpty(t *testing.T) {
	cases := []struct {
		name      string
		in        []byte
		expectNil bool
	}{
		{"empty", nil, true},
		{"valid-json", []byte(`{"k":"v"}`), false},
		{"invalid-json-falls-back", []byte(`not-json`), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := decodeJSONMap(tc.in)
			if (got == nil) != tc.expectNil {
				t.Errorf("nil=%v expectNil=%v got=%v", got == nil, tc.expectNil, got)
			}
		})
	}
}

func TestRecorder_QueryIsCurrentlyUnimplemented(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()
	rec := NewRecorder(store, discardLogger())
	if _, err := rec.Query(ports.CaptureFilter{}); err == nil {
		t.Fatal("expected error for stub Query")
	}
}