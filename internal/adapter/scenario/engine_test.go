package scenario

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
	"github.com/noroadzh/gat1400-simulator/internal/domain/resource"
	"github.com/noroadzh/gat1400-simulator/internal/domain/scenario"
)

type fakeProvisioner struct {
	mu    sync.Mutex
	nodes map[string]*node.Node
	fail  bool
}

func (f *fakeProvisioner) UpsertNode(_ context.Context, n node.Node) (*node.Node, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.fail {
		return nil, errors.New("upsert failed")
	}
	if f.nodes == nil {
		f.nodes = map[string]*node.Node{}
	}
	cp := n
	f.nodes[n.ID] = &cp
	return &cp, nil
}

func (f *fakeProvisioner) GetNode(_ context.Context, id string) (*node.Node, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	n, ok := f.nodes[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return n, nil
}

func (f *fakeProvisioner) SyncListeners(context.Context) error { return nil }

func newTestEngine(t *testing.T, prov NodeProvisioner) *Engine {
	t.Helper()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	idGen := ids.NewGenerator(41000000, 30)
	factory := NewFactory(idGen, 42)
	disp := NewOutboundDispatcher(nil, log, nil)
	eng := NewEngine(log, factory, disp, nil, prov)
	eng.SetKeepaliveInterval(0) // 测试默认关闭 keepalive，避免外呼
	return eng
}

func TestEngine_StartIdempotentAndStop(t *testing.T) {
	prov := &fakeProvisioner{}
	eng := newTestEngine(t, prov)
	sc := scenario.Scenario{
		ID:   "s1",
		Name: "test",
		Nodes: []scenario.NodeSpec{{
			Ref: "dev-1", Role: node.RoleDevice, Listen: ":0",
			Capabilities: []node.Capability{node.CapSystem},
		}},
		Schedule: scenario.ScheduleSpec{AutoStart: false},
	}
	ctx := context.Background()
	if err := eng.Start(ctx, sc); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if !eng.IsRunning("s1") {
		t.Fatal("expected running")
	}
	// 幂等
	if err := eng.Start(ctx, sc); err != nil {
		t.Fatalf("Start again: %v", err)
	}
	if err := eng.Stop("s1"); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if eng.IsRunning("s1") {
		t.Fatal("expected stopped")
	}
	// Stop 不存在场景 no-op
	if err := eng.Stop("missing"); err != nil {
		t.Fatalf("Stop missing: %v", err)
	}
}

func TestEngine_StartFailsOnMaterialise(t *testing.T) {
	prov := &fakeProvisioner{fail: true}
	eng := newTestEngine(t, prov)
	sc := scenario.Scenario{
		ID: "s-fail",
		Nodes: []scenario.NodeSpec{{
			Ref: "dev-x", Role: node.RoleDevice,
		}},
	}
	if err := eng.Start(context.Background(), sc); err == nil {
		t.Fatal("expected materialise error")
	}
	if eng.IsRunning("s-fail") {
		t.Fatal("must not be running after failure")
	}
}

func TestEngine_AutoStart(t *testing.T) {
	prov := &fakeProvisioner{}
	eng := newTestEngine(t, prov)
	sc := scenario.Scenario{
		ID: "auto-off",
		Nodes: []scenario.NodeSpec{{
			Ref: "dev-2", Role: node.RolePlatformSmall,
		}},
		Schedule: scenario.ScheduleSpec{AutoStart: false},
	}
	if err := eng.AutoStart(context.Background(), sc); err != nil {
		t.Fatalf("AutoStart off: %v", err)
	}
	if eng.IsRunning("auto-off") {
		t.Fatal("AutoStart=false must be no-op")
	}

	sc.ID = "auto-on"
	sc.Schedule.AutoStart = true
	if err := eng.AutoStart(context.Background(), sc); err != nil {
		t.Fatalf("AutoStart on: %v", err)
	}
	if !eng.IsRunning("auto-on") {
		t.Fatal("expected running")
	}
	_ = eng.Stop("auto-on")
}

func TestEngine_ListRunningAfterStop(t *testing.T) {
	prov := &fakeProvisioner{}
	eng := newTestEngine(t, prov)
	sc := scenario.Scenario{
		ID: "pace",
		Nodes: []scenario.NodeSpec{{
			Ref: "collector", Role: node.RolePlatformSmall, Listen: ":0",
		}},
		// Rate=0：不触发 outbound Dispatch（测试用 nil wire.Client）。
		Resources: []scenario.ResourceSpec{{
			NodeRef: "collector", Kind: resource.KindPerson, Rate: 0,
		}},
	}
	ctx := context.Background()
	if err := eng.Start(ctx, sc); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if got := eng.ListRunning(); len(got) != 1 || got[0] != "pace" {
		t.Fatalf("ListRunning = %v", got)
	}
	time.Sleep(10 * time.Millisecond)
	if err := eng.Stop("pace"); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if len(eng.ListRunning()) != 0 {
		t.Fatal("expected empty running list")
	}
}
