package wire

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestVIAS_Paths(t *testing.T) {
	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/VIID+JSON")
		switch {
		case r.URL.Path == "/VIAS/System/Time":
			_, _ = w.Write([]byte(`{"Time":"2026-01-01T00:00:00Z"}`))
		case r.URL.Path == "/VIAS/Tasks" && r.Method == http.MethodPost:
			_, _ = w.Write([]byte(`{"Task":{"TaskID":"task-1"}}`))
		case r.URL.Path == "/VIAS/Tasks/task-1" && r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`{"Task":{"TaskID":"task-1"}}`))
		default:
			_, _ = w.Write([]byte(`{"ResponseStatus":{"StatusCode":0}}`))
		}
	}))
	defer srv.Close()

	v := NewClient(nil, nil).VIAS()
	ctx := t.Context()
	dev := "DEV1"

	if err := v.Register(ctx, srv.URL, RegisterObject{DeviceID: dev, Status: "ONLINE", Keepalive: 60}); err != nil {
		t.Fatalf("Register: %v", err)
	}
	if err := v.Keepalive(ctx, srv.URL, dev); err != nil {
		t.Fatalf("Keepalive: %v", err)
	}
	if _, err := v.ServerTime(ctx, srv.URL); err != nil {
		t.Fatalf("ServerTime: %v", err)
	}
	resp, err := v.CreateTask(ctx, srv.URL, dev, map[string]any{"TaskID": "task-1"})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	_ = resp
	if _, err := v.GetTask(ctx, srv.URL, dev, "task-1"); err != nil {
		t.Fatalf("GetTask: %v", err)
	}
	if err := v.TaskControl(ctx, srv.URL, dev, map[string]any{"Action": "Start", "TaskID": "task-1"}); err != nil {
		t.Fatalf("TaskControl: %v", err)
	}
	if err := v.UpsertAnalysisRule(ctx, srv.URL, dev, map[string]any{"AnalysisRuleID": "r1"}); err != nil {
		t.Fatalf("UpsertAnalysisRule: %v", err)
	}
	if _, err := v.AnalysisCapability(ctx, srv.URL, dev); err != nil {
		t.Fatalf("AnalysisCapability: %v", err)
	}
	if err := v.DeleteTask(ctx, srv.URL, dev, "task-1"); err != nil {
		t.Fatalf("DeleteTask: %v", err)
	}
	if err := v.UnRegister(ctx, srv.URL, dev); err != nil {
		t.Fatalf("UnRegister: %v", err)
	}

	mustContain := []string{
		"POST /VIAS/System/Register",
		"POST /VIAS/System/Keepalive",
		"GET /VIAS/System/Time",
		"POST /VIAS/Tasks",
		"GET /VIAS/Tasks/task-1",
		"POST /VIAS/TaskControls",
		"POST /VIAS/AnalysisRules",
		"GET /VIAS/SystemCapability/AnalysisCapability",
		"DELETE /VIAS/Tasks/task-1",
		"POST /VIAS/System/UnRegister",
	}
	joined := strings.Join(paths, "\n")
	for _, want := range mustContain {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing %q in %v", want, paths)
		}
	}
}
