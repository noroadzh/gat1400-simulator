package application

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/noroadzh/gat1400-simulator/internal/domain/ids"
	"github.com/noroadzh/gat1400-simulator/internal/domain/node"
	"github.com/noroadzh/gat1400-simulator/internal/domain/scenario"
)

// fakeEngine records Start/Stop calls without doing any work.
type fakeEngine struct {
	mu        sync.Mutex
	started   []string
	stopped   []string
	startErr  error
	stopErr   error
	running   map[string]bool
	listCalls int
}

func newFakeEngine() *fakeEngine { return &fakeEngine{running: map[string]bool{}} }

func (f *fakeEngine) Start(_ context.Context, s scenario.Scenario) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.startErr != nil {
		return f.startErr
	}
	f.started = append(f.started, s.ID)
	f.running[s.ID] = true
	return nil
}

func (f *fakeEngine) Stop(id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.stopErr != nil {
		return f.stopErr
	}
	f.stopped = append(f.stopped, id)
	delete(f.running, id)
	return nil
}

func (f *fakeEngine) IsRunning(id string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.running[id]
}

func (f *fakeEngine) ListRunning() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.listCalls++
	out := make([]string, 0, len(f.running))
	for id := range f.running {
		out = append(out, id)
	}
	return out
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestNodeService_UpsertStoresAndPreservesCreatedAt(t *testing.T) {
	svc := NewNodeService(discardLogger(), ids.NewGenerator(41000000, 130))

	n1 := node.Node{ID: "n1", Name: "n1", Role: node.RoleDevice}
	stored, err := svc.UpsertNode(context.Background(), n1)
	if err != nil {
		t.Fatalf("UpsertNode: %v", err)
	}
	if stored.CreatedAt.IsZero() {
		t.Fatal("CreatedAt not set")
	}
	first := stored.CreatedAt
	time.Sleep(2 * time.Millisecond)

	stored2, err := svc.UpsertNode(context.Background(), node.Node{
		ID: "n1", Name: "n1-renamed", Role: node.RoleDevice,
	})
	if err != nil {
		t.Fatalf("UpsertNode: %v", err)
	}
	if !stored2.CreatedAt.Equal(first) {
		t.Errorf("CreatedAt changed on update: was=%v now=%v", first, stored2.CreatedAt)
	}
	if stored2.Name != "n1-renamed" {
		t.Errorf("Name = %q, want n1-renamed", stored2.Name)
	}
	if stored2.UpdatedAt.Before(first) {
		t.Errorf("UpdatedAt must be >= CreatedAt: %v vs %v", stored2.UpdatedAt, first)
	}
}

func TestNodeService_UpsertRejectsInvalidNode(t *testing.T) {
	svc := NewNodeService(discardLogger(), ids.NewGenerator(41000000, 130))
	cases := []struct {
		name string
		n    node.Node
	}{
		{"missing-id", node.Node{Name: "x", Role: node.RoleDevice}},
		{"missing-name", node.Node{ID: "n1", Role: node.RoleDevice}},
		{"missing-role", node.Node{ID: "n1", Name: "n1"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := svc.UpsertNode(context.Background(), tc.n); err == nil {
				t.Fatal("expected sanity error")
			}
		})
	}
}

func TestNodeService_GetAndRemove(t *testing.T) {
	svc := NewNodeService(discardLogger(), ids.NewGenerator(41000000, 130))
	if _, err := svc.GetNode(context.Background(), "missing"); !errors.Is(err, node.ErrNotFound) {
		t.Fatalf("GetNode(missing) = %v, want ErrNotFound", err)
	}

	if _, err := svc.UpsertNode(context.Background(), node.Node{
		ID: "n1", Name: "n1", Role: node.RoleDevice,
	}); err != nil {
		t.Fatalf("UpsertNode: %v", err)
	}
	got, err := svc.GetNode(context.Background(), "n1")
	if err != nil {
		t.Fatalf("GetNode: %v", err)
	}
	if got.ID != "n1" || got.Name != "n1" {
		t.Errorf("unexpected node %+v", got)
	}
	if err := svc.RemoveNode(context.Background(), "n1"); err != nil {
		t.Fatalf("RemoveNode: %v", err)
	}
	if _, err := svc.GetNode(context.Background(), "n1"); !errors.Is(err, node.ErrNotFound) {
		t.Errorf("GetNode after Remove = %v, want ErrNotFound", err)
	}
	// Removing a non-existent id is a no-op.
	if err := svc.RemoveNode(context.Background(), "absent"); err != nil {
		t.Errorf("RemoveNode(absent) returned %v", err)
	}
}

func TestNodeService_MarkSeenFlipsStatus(t *testing.T) {
	svc := NewNodeService(discardLogger(), ids.NewGenerator(41000000, 130))
	if err := svc.MarkSeen(context.Background(), "ghost"); !errors.Is(err, node.ErrNotFound) {
		t.Errorf("MarkSeen(ghost) = %v, want ErrNotFound", err)
	}
	if _, err := svc.UpsertNode(context.Background(), node.Node{
		ID: "n1", Name: "n1", Role: node.RoleDevice,
	}); err != nil {
		t.Fatal(err)
	}
	if err := svc.MarkSeen(context.Background(), "n1"); err != nil {
		t.Fatalf("MarkSeen: %v", err)
	}
	got, _ := svc.GetNode(context.Background(), "n1")
	if got.Status != node.StatusOnline {
		t.Errorf("Status = %s, want online", got.Status)
	}
	if got.LastSeenAt.IsZero() {
		t.Error("LastSeenAt not set")
	}
}

func TestNodeService_ListReturnsSnapshot(t *testing.T) {
	svc := NewNodeService(discardLogger(), ids.NewGenerator(41000000, 130))
	for _, id := range []string{"a", "b", "c"} {
		if _, err := svc.UpsertNode(context.Background(), node.Node{
			ID: id, Name: id, Role: node.RoleDevice,
		}); err != nil {
			t.Fatal(err)
		}
	}
	got, err := svc.ListNodes(context.Background())
	if err != nil {
		t.Fatalf("ListNodes: %v", err)
	}
	if len(got) != 3 {
		t.Errorf("len = %d, want 3", len(got))
	}
}

func TestNodeService_SyncHookFiresAndIsIdempotent(t *testing.T) {
	svc := NewNodeService(discardLogger(), ids.NewGenerator(41000000, 130))
	calls := 0
	svc.SetSyncHook(func(_ context.Context) error {
		calls++
		return nil
	})
	if err := svc.SyncListeners(context.Background()); err != nil {
		t.Fatalf("SyncListeners: %v", err)
	}
	if err := svc.SyncListeners(context.Background()); err != nil {
		t.Fatalf("SyncListeners: %v", err)
	}
	if calls != 2 {
		t.Errorf("calls = %d, want 2", calls)
	}
	// SyncListeners without a hook is a no-op.
	svc2 := NewNodeService(discardLogger(), ids.NewGenerator(41000000, 130))
	if err := svc2.SyncListeners(context.Background()); err != nil {
		t.Errorf("SyncListeners without hook returned %v", err)
	}
}

func TestScenarioService_StartStopLifecycle(t *testing.T) {
	eng := newFakeEngine()
	svc := NewScenarioService(discardLogger(), eng)

	svc.Register(scenario.Scenario{ID: "s1", Name: "one"})

	if err := svc.Start(context.Background(), "s1"); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if !svc.IsRunning("s1") {
		t.Fatal("expected s1 to be running")
	}
	if got := svc.Running(); len(got) != 1 || got[0] != "s1" {
		t.Errorf("Running = %v, want [s1]", got)
	}

	// Starting again is a no-op.
	if err := svc.Start(context.Background(), "s1"); err != nil {
		t.Fatalf("Start again: %v", err)
	}
	if len(eng.started) != 1 {
		t.Errorf("engine.Start calls = %d, want 1", len(eng.started))
	}

	if err := svc.Stop(context.Background(), "s1"); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if svc.IsRunning("s1") {
		t.Error("expected s1 to be stopped")
	}

	// Stopping again is a no-op.
	if err := svc.Stop(context.Background(), "s1"); err != nil {
		t.Errorf("Stop again: %v", err)
	}
}

func TestScenarioService_StartUnknownScenarioReturnsError(t *testing.T) {
	svc := NewScenarioService(discardLogger(), newFakeEngine())
	if err := svc.Start(context.Background(), "missing"); err == nil {
		t.Fatal("expected error for unknown scenario")
	}
}

func TestScenarioService_AutoStartRespectsFlag(t *testing.T) {
	eng := newFakeEngine()
	svc := NewScenarioService(discardLogger(), eng)
	svc.Register(scenario.Scenario{ID: "a", Schedule: scenario.ScheduleSpec{AutoStart: true}})
	svc.Register(scenario.Scenario{ID: "b"})
	svc.Register(scenario.Scenario{ID: "c", Schedule: scenario.ScheduleSpec{AutoStart: true}})

	if errs := svc.AutoStart(context.Background()); len(errs) != 0 {
		t.Fatalf("AutoStart errors: %v", errs)
	}
	if len(eng.started) != 2 {
		t.Errorf("engine.Start calls = %d, want 2", len(eng.started))
	}
	if !svc.IsRunning("a") || svc.IsRunning("b") || !svc.IsRunning("c") {
		t.Errorf("running state wrong: a=%v b=%v c=%v",
			svc.IsRunning("a"), svc.IsRunning("b"), svc.IsRunning("c"))
	}
}

func TestScenarioService_StartRollsBackOnEngineFailure(t *testing.T) {
	eng := newFakeEngine()
	eng.startErr = errors.New("boom")
	svc := NewScenarioService(discardLogger(), eng)
	svc.Register(scenario.Scenario{ID: "x"})

	if err := svc.Start(context.Background(), "x"); err == nil {
		t.Fatal("expected error from engine failure")
	}
	if svc.IsRunning("x") {
		t.Error("running flag should be rolled back on engine failure")
	}
}

func TestScenarioService_NilEngineIsNoOp(t *testing.T) {
	svc := NewScenarioService(discardLogger(), nil)
	svc.Register(scenario.Scenario{ID: "n"})
	if err := svc.Start(context.Background(), "n"); err != nil {
		t.Errorf("Start with nil engine: %v", err)
	}
	if !svc.IsRunning("n") {
		t.Error("expected n to be tracked as running even without engine")
	}
}

func TestScenarioService_LoadAllAndList(t *testing.T) {
	svc := NewScenarioService(discardLogger(), nil)
	svc.LoadAll(map[string]scenario.Scenario{
		"a": {ID: "a", Name: "alpha"},
		"b": {ID: "b", Name: "beta"},
	})
	if got := svc.ListScenarios(context.Background()); len(got) != 2 {
		t.Errorf("ListScenarios len = %d, want 2", len(got))
	}
	if _, err := svc.Get("missing"); err == nil {
		t.Error("Get(missing) should return error")
	}
	if got, err := svc.Get("a"); err != nil || got.Name != "alpha" {
		t.Errorf("Get(a) = %+v, err=%v", got, err)
	}
}