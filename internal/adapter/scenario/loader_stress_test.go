package scenario

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestLoadBidirectionalStressYAML(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// internal/adapter/scenario → repo root
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "../../.."))
	path := filepath.Join(root, "configs/scenarios/bidirectional-stress.yaml")

	s, err := NewLoader().LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	if s.ID != "bidirectional-stress" {
		t.Fatalf("id = %q", s.ID)
	}
	if s.Schedule.AutoStart {
		t.Fatal("AutoStart must be false")
	}
	if len(s.Actions) < 2 {
		t.Fatalf("expected actions, got %d", len(s.Actions))
	}
	if len(s.Resources) < 1 {
		t.Fatalf("expected resources, got %d", len(s.Resources))
	}
}
