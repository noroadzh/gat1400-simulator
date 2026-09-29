// Package application 承载用例编排器（use-case orchestrator）。
//
// 分层纪律（必须遵守）：
//   - 仅依赖 domain 与 port 接口（ports.go）
//   - 禁止 import 任何 concrete adapter（internal/adapter/*）
//   - concrete adapter 在 main.go 中通过 DI 注入
//
// 保证：
//   - domain 可独立单元测试
//   - adapter 可替换而不影响业务逻辑（如 SQLite → PostgreSQL）
package application

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/noroadzh/gat1400-simulator/internal/domain/ids"
	"github.com/noroadzh/gat1400-simulator/internal/domain/node"
	"github.com/noroadzh/gat1400-simulator/internal/domain/scenario"
)

// NodeService ports.NodeService 的内存实现。goroutine-safe（读写锁保护所有操作）。
type NodeService struct {
	log *slog.Logger
	ids *ids.Generator
	mu  sync.RWMutex
	items map[string]*node.Node
	// syncListeners 每次 upsert 后触发，给 adapter 一个绑定/解绑 HTTP 监听器的机会。
	// 存为 func 而非接口，是为了避免把 adapter 包引入 application 层。
	syncListeners func(ctx context.Context) error
}

// NewNodeService 构造一个空的节点服务。不预置任何节点（由外部 YAML 场景或 API 注入）。
func NewNodeService(log *slog.Logger, gen *ids.Generator) *NodeService {
	return &NodeService{log: log, ids: gen, items: map[string]*node.Node{}}
}

// ListNodes 返回所有已注册节点的稳定快照。
func (s *NodeService) ListNodes(_ context.Context) ([]node.Node, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]node.Node, 0, len(s.items))
	for _, n := range s.items {
		out = append(out, *n)
	}
	return out, nil
}

// GetNode 返回请求节点的副本，不存在时返回 node.ErrNotFound。
func (s *NodeService) GetNode(_ context.Context, id string) (*node.Node, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	n, ok := s.items[id]
	if !ok {
		return nil, node.ErrNotFound
	}
	cp := *n
	return &cp, nil
}

// UpsertNode 插入或替换节点，应用 Sanity 校验。
func (s *NodeService) UpsertNode(_ context.Context, n node.Node) (*node.Node, error) {
	if err := n.Sanity(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	if existing, ok := s.items[n.ID]; ok {
		n.CreatedAt = existing.CreatedAt
	} else {
		n.CreatedAt = now
	}
	n.UpdatedAt = now
	if n.Status == "" {
		n.Status = node.StatusStopped
	}
	cp := n
	s.items[n.ID] = &cp
	return &cp, nil
}

// RemoveNode 按 id 删除节点。幂等：缺失 id 返回 nil。
func (s *NodeService) RemoveNode(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.items, id)
	return nil
}

// MarkSeen 更新节点的最后Seen时间，并将状态置为 online。
//
// 当服务端收到带 User-Identify 头的请求时，会调用此方法。
// 若节点不存在，返回 node.ErrNotFound（HTTP 中间件将其映射为 404）。
func (s *NodeService) MarkSeen(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	n, ok := s.items[id]
	if !ok {
		return node.ErrNotFound
	}
	now := time.Now().UTC()
	n.LastSeenAt = now
	n.Status = node.StatusOnline
	n.UpdatedAt = now
	return nil
}

// SetSyncHook 安装一个回调，在每次 UpsertNode 后触发。
//
// 用于 HTTP adapter registry：每当节点 Upsert 时，同步绑定/解绑对应的 echo HTTP 监听器。
// 多节点可在不同端口同时提供 VIID 服务。
func (s *NodeService) SetSyncHook(fn func(ctx context.Context) error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.syncListeners = fn
}

// SyncListeners 调用已注册的 hook；未注册时返回 nil。
func (s *NodeService) SyncListeners(ctx context.Context) error {
	s.mu.RLock()
	fn := s.syncListeners
	s.mu.RUnlock()
	if fn == nil {
		return nil
	}
	return fn(ctx)
}

// ScenarioService 负责场景包生命周期（加载/启动/停止/查询）。
//
// 实际 pacing（按节奏发送资源）委托给 ScenarioEngineAPI 实例。
// application 对传输层中立：可接入 HTTP、gRPC 等不同传输。
type ScenarioService struct {
	log    *slog.Logger
	engine ScenarioEngineAPI
	mu     sync.RWMutex
	scenarios map[string]*scenario.Scenario
	running   map[string]bool
}

// ScenarioEngineAPI 引擎必须满足的最小接口。注入点在 main.go。
// 具体实现在 internal/adapter/scenario/engine.go。
type ScenarioEngineAPI interface {
	Start(ctx context.Context, s scenario.Scenario) error
	Stop(id string) error
	IsRunning(id string) bool
	ListRunning() []string
}

// NewScenarioService 构造场景服务。
//
// engine 为 nil 时，Start/Stop 成为 no-op（仍追踪元数据）。
// 这样可以独立测试 BFF API 而不启动真正的场景引擎。
func NewScenarioService(log *slog.Logger, engine ScenarioEngineAPI) *ScenarioService {
	return &ScenarioService{
		log:       log,
		engine:    engine,
		scenarios: map[string]*scenario.Scenario{},
		running:   map[string]bool{},
	}
}

// Register 存储（或替换）场景元数据。可重复调用，最后一次拷贝生效。
func (s *ScenarioService) Register(sc scenario.Scenario) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.scenarios[sc.ID] = &sc
}

// LoadAll 批量注册场景（用于 YAML loader 一次性加载所有场景文件）。
// 冲突时后写入的覆盖先写入的。
func (s *ScenarioService) LoadAll(items map[string]scenario.Scenario) []error {
	s.mu.Lock()
	defer s.mu.Unlock()
	var errs []error
	for id, sc := range items {
		s.scenarios[id] = ptrOf(sc)
	}
	return errs
}

// AutoStart 遍历所有已注册场景，自动启动 Schedule.AutoStart=true 的场景。
// 在 main.go 启动流程中被调用。
func (s *ScenarioService) AutoStart(ctx context.Context) []error {
	s.mu.RLock()
	copies := make([]scenario.Scenario, 0, len(s.scenarios))
	for _, sc := range s.scenarios {
		copies = append(copies, *sc)
	}
	s.mu.RUnlock()
	var errs []error
	for _, sc := range copies {
		if sc.Schedule.AutoStart {
			if err := s.Start(ctx, sc.ID); err != nil {
				errs = append(errs, err)
			}
		}
	}
	return errs
}

// Get 返回单个场景的快照（不存在时返回 errScenarioUnknown）。
func (s *ScenarioService) Get(id string) (scenario.Scenario, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sc, ok := s.scenarios[id]
	if !ok {
		return scenario.Scenario{}, fmt.Errorf("scenario: %w: %s", errScenarioUnknown, id)
	}
	return *sc, nil
}

// ListScenarios 返回所有已加载场景的快照。
func (s *ScenarioService) ListScenarios(_ context.Context) []scenario.Scenario {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]scenario.Scenario, 0, len(s.scenarios))
	for _, v := range s.scenarios {
		out = append(out, *v)
	}
	return out
}

// Start 把场景加入运行集，并让引擎启动 pacing goroutine。
// 幂等：同一 ID 调用两次仅启动一次。
func (s *ScenarioService) Start(ctx context.Context, id string) error {
	s.mu.Lock()
	sc, ok := s.scenarios[id]
	if !ok {
		s.mu.Unlock()
		return fmt.Errorf("scenario: %w: %s", errScenarioUnknown, id)
	}
	if s.running[id] {
		s.mu.Unlock()
		return nil
	}
	s.running[id] = true
	s.mu.Unlock()

	if s.engine == nil {
		return nil
	}
	if err := s.engine.Start(ctx, *sc); err != nil {
		s.mu.Lock()
		delete(s.running, id)
		s.mu.Unlock()
		return err
	}
	s.log.Info("scenario started", slog.String("id", id))
	return nil
}

// Stop 把场景移出运行集，并让引擎 cancel 对应的 goroutine。
// 幂等：不在运行集的 ID 直接返回 nil。
func (s *ScenarioService) Stop(_ context.Context, id string) error {
	s.mu.Lock()
	if !s.running[id] {
		s.mu.Unlock()
		return nil
	}
	delete(s.running, id)
	s.mu.Unlock()

	if s.engine != nil {
		_ = s.engine.Stop(id)
	}
	s.log.Info("scenario stopped", slog.String("id", id))
	return nil
}

// IsRunning 报告场景是否处于活跃状态。
func (s *ScenarioService) IsRunning(id string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running[id]
}

// Running 返回所有活跃场景的 ID。
func (s *ScenarioService) Running() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]string, 0, len(s.running))
	for id := range s.running {
		out = append(out, id)
	}
	return out
}

func ptrOf[T any](v T) *T { return &v }

var errScenarioUnknown = errors.New("unknown")
