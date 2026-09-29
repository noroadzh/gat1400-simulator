package ports

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/noroadzh/gat1400-simulator/internal/domain/node"
	"github.com/noroadzh/gat1400-simulator/internal/domain/resource"
)

func TestNotifyResource_DefaultNoopReturnsNil(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	if err := NotifyResource(context.Background(), log, resource.KindPerson, "p1"); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

// fakeNodeService 编译期验证 ports.NodeService 接口被最小实现满足。
// 当接口重构时捕获退化（structurally satisfied 模式）。
type fakeNodeService struct{}

func (f *fakeNodeService) ListNodes(_ context.Context) ([]node.Node, error) { return nil, nil }
func (f *fakeNodeService) GetNode(_ context.Context, _ string) (*node.Node, error) {
	return nil, nil
}
func (f *fakeNodeService) UpsertNode(_ context.Context, _ node.Node) (*node.Node, error) {
	return nil, nil
}
func (f *fakeNodeService) RemoveNode(_ context.Context, _ string) error { return nil }
func (f *fakeNodeService) MarkSeen(_ context.Context, _ string) error   { return nil }

func TestNodeService_InterfaceSatisfiedByMinimalImpl(t *testing.T) {
	var _ NodeService = (*fakeNodeService)(nil)
}

// fakeNotifier 验证 Notifier 接口契约。
type fakeNotifier struct{ called bool }

func (f *fakeNotifier) Dispatch(_ context.Context, _ node.Node, _ any) error {
	f.called = true
	return nil
}

func TestNotifier_InterfaceContract(t *testing.T) {
	var n Notifier = &fakeNotifier{}
	_ = n.Dispatch(context.Background(), node.Node{}, nil)
	if n.(*fakeNotifier).called != true {
		t.Fatal("Dispatch did not invoke implementation")
	}
}

// fakeResourceSink 满足 ResourceSink 接口。
type fakeResourceSink struct{ called bool }

func (f *fakeResourceSink) OnResource(_ context.Context, _ resource.Kind, _ any) error {
	f.called = true
	return nil
}

func TestResourceSink_InterfaceContract(t *testing.T) {
	var s ResourceSink = &fakeResourceSink{}
	_ = s.OnResource(context.Background(), resource.KindFace, nil)
	if s.(*fakeResourceSink).called != true {
		t.Fatal("OnResource did not invoke implementation")
	}
}