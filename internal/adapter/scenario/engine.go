// Package scenario 场景引擎：把 YAML 场景文件转换为可执行的 goroutine，
// 按节点分发资源推送（Register → Resource Push → Keepalive → UnRegister）。
//
// Engine 的 Start/Stop/AutoStart 驱动节点行为，本身不持有 HTTP 服务（服务由 bootstrap 启动）。
// 详见 internal/adapter/scenario/loader.go 的 YAML 解析。
package scenario

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"sync"
	"time"

	"github.com/noroadzh/gat1400-simulator/internal/adapter/capture"
	"github.com/noroadzh/gat1400-simulator/internal/domain/node"
	"github.com/noroadzh/gat1400-simulator/internal/domain/scenario"
)

// NodeProvisioner 引擎把 scenario.NodeSpec 实例化为 Node 聚合所依赖的接口。
// application.NodeService 满足此接口——adapter 不直接依赖 application。
type NodeProvisioner interface {
	UpsertNode(ctx context.Context, n node.Node) (*node.Node, error)
	GetNode(ctx context.Context, id string) (*node.Node, error)
	SyncListeners(ctx context.Context) error
}

// Engine 场景包运行时。
//
// 每个运行中场景拥有：
//   - 每个 ResourceSpec 一个 pacing goroutine
//   - 每个 SubscriptionSpec 一个通知 goroutine
//   - 全部派生自单个可取消 context（场景 Stop 时 cancel）
type Engine struct {
	log        *slog.Logger
	factory    *Factory
	dispatcher *OutboundDispatcher
	recorder   *capture.Recorder
	provisioner NodeProvisioner

	mu       sync.Mutex
	running  map[string]*runHandle

	// keepaliveInterval 是每个设备节点周期调用 /VIID/System/Keepalive 的间隔。
	// 0 = 禁用（缺省场景用于控制节点）。
	keepaliveInterval time.Duration
}

type runHandle struct {
	scenario scenario.Scenario
	cancel   context.CancelFunc
	done     chan struct{}
}

// NewEngine 装配引擎。recorder 用于把出站 dispatch 写入抓包存储，供 Web 仪表盘展示。
func NewEngine(log *slog.Logger, factory *Factory, dispatcher *OutboundDispatcher, recorder *capture.Recorder, prov NodeProvisioner) *Engine {
	return &Engine{
		log:               log,
		factory:           factory,
		dispatcher:        dispatcher,
		recorder:          recorder,
		provisioner:       prov,
		running:           map[string]*runHandle{},
		keepaliveInterval: 30 * time.Second,
	}
}

// SetKeepaliveInterval 调整心跳周期。0 = 禁用。
func (e *Engine) SetKeepaliveInterval(d time.Duration) { e.keepaliveInterval = d }

// AutoStart 根据 Scenario.Schedule.AutoStart 决定是否启动场景。
// 与 Start 的差别：Start 无条件启动；AutoStart 仅在 Schedule.AutoStart=true
// 时内部调用 Start，否则直接返回 nil（no-op，不启动任何 goroutine）。
//
// 该方法为场景级 §5.1 Register 流程的语义化入口——上游控制平面
// 启动时统一调用 AutoStart，由场景包自身声明是否随进程启动，
// 不需要 main.go 写额外条件分支。
func (e *Engine) AutoStart(ctx context.Context, s scenario.Scenario) error {
	if !s.Schedule.AutoStart {
		return nil
	}
	return e.Start(ctx, s)
}

// Start 把场景中声明的节点物化（idempotent），并启动 pacing goroutine。
//
// 对同一 ID 重复调用是 no-op。
// 节点物化失败时整体失败，不会进入运行集。
func (e *Engine) Start(ctx context.Context, s scenario.Scenario) error {
	e.mu.Lock()
	if _, exists := e.running[s.ID]; exists {
		e.mu.Unlock()
		return nil
	}
	e.mu.Unlock()

	// Materialise the nodes once at start time. Mark them online so the
	// dashboard reflects the new lifetime.
	if err := e.materialise(ctx, s); err != nil {
		return fmt.Errorf("engine: materialise: %w", err)
	}

	runCtx, cancel := context.WithCancel(ctx)
	handle := &runHandle{scenario: s, cancel: cancel, done: make(chan struct{})}
	e.mu.Lock()
	e.running[s.ID] = handle
	e.mu.Unlock()

	go e.run(runCtx, handle)
	return nil
}

// Stop 取消场景的所有 goroutine 并等待其退出。
// 5 秒硬超时：超时后记 warn 日志但不影响调用方。
//
// 退出前会按需触发每个 device 节点的 /VIID/System/UnRegister（HTTP 层主动下线）。
// Unregister 在节点 HTTPListen 为空时安全跳过。
func (e *Engine) Stop(id string) error {
	e.mu.Lock()
	h, ok := e.running[id]
	if !ok {
		e.mu.Unlock()
		return nil
	}
	delete(e.running, id)
	e.mu.Unlock()

	// Best-effort unregister for every device-role node before tearing down goroutines.
	unregCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for _, ns := range h.scenario.Nodes {
		if ns.Role != node.RoleDevice {
			continue
		}
		if ns.Upstream == "" {
			continue
		}
		n, _ := e.provisioner.GetNode(unregCtx, ns.Ref)
		if n == nil || n.Upstream == "" {
			continue
		}
		// dispatcher 用 target.HTTPListen 推导目标 URL。
		target := node.Node{ID: n.ID, Name: n.Name, HTTPListen: n.Upstream}
		if err := e.dispatcher.DispatchUnregister(unregCtx, target); err != nil {
			e.log.Debug("unregister",
				slog.String("node", ns.Ref),
				slog.String("error", err.Error()))
		}
	}

	h.cancel()
	select {
	case <-h.done:
	case <-time.After(5 * time.Second):
		e.log.Warn("engine: scenario stop timeout", slog.String("id", id))
	}
	return nil
}

// IsRunning 报告场景是否处于运行态。
func (e *Engine) IsRunning(id string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	_, ok := e.running[id]
	return ok
}

// ListRunning 返回当前所有运行中场景的 ID。
func (e *Engine) ListRunning() []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := make([]string, 0, len(e.running))
	for id := range e.running {
		out = append(out, id)
	}
	return out
}

func (e *Engine) run(ctx context.Context, h *runHandle) {
	defer close(h.done)
	var wg sync.WaitGroup

	for _, rs := range h.scenario.Resources {
		if rs.Rate <= 0 {
			continue
		}
		wg.Add(1)
		go func(rs scenario.ResourceSpec) {
			defer wg.Done()
			e.paceResources(ctx, h.scenario, rs)
		}(rs)
	}

	// Subscribe goroutines: when an upstream node produces a resource of the
	// configured kind, the subscriber's /VIID/SubscribeNotifications endpoint
	// receives a synthetic notification.
	for _, sub := range h.scenario.Subscriptions {
		wg.Add(1)
		go func(s scenario.SubscriptionSpec) {
			defer wg.Done()
			e.dispatchNotifications(ctx, h.scenario, s)
		}(sub)
	}

	// Keepalive goroutines: each device-role node pings its upstream on a fixed
	// interval (with optional jitter). Disabled when keepaliveInterval <= 0.
	if e.keepaliveInterval > 0 {
		jitter := h.scenario.Exceptions.KeepaliveJitter
		for _, ns := range h.scenario.Nodes {
			if ns.Role != node.RoleDevice {
				continue
			}
			if ns.Upstream == "" {
				continue
			}
			wg.Add(1)
			go func(ns scenario.NodeSpec) {
				defer wg.Done()
				e.runKeepalive(ctx, h.scenario, ns, jitter)
			}(ns)
		}
	}

	// Optional auto-shutdown.
	if h.scenario.Schedule.Duration > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			t := time.NewTimer(h.scenario.Schedule.Duration)
			defer t.Stop()
			select {
			case <-ctx.Done():
			case <-t.C:
				e.log.Info("scenario auto-stop fired",
					slog.String("id", h.scenario.ID),
					slog.Duration("duration", h.scenario.Schedule.Duration),
				)
				_ = e.Stop(h.scenario.ID)
			}
		}()
	}

	wg.Wait()
}

// materialise creates Node aggregates for every NodeSpec in the scenario.
// Existing nodes are left untouched (UpsertNode is idempotent on the same ID).
func (e *Engine) materialise(ctx context.Context, s scenario.Scenario) error {
	for _, ns := range s.Nodes {
		id := ns.Ref
		if id == "" {
			continue
		}
		n := node.Node{
			ID:           id,
			Name:         defaultName(ns.Ref, ns.Role),
			Role:         ns.Role,
			HTTPListen:   ns.Listen,
			Upstream:     ns.Upstream,
			Capabilities: ns.Capabilities,
			Tags:         append([]string(nil), ns.Tags...),
		}
		if _, err := e.provisioner.UpsertNode(ctx, n); err != nil {
			return fmt.Errorf("node %s: %w", id, err)
		}
		e.log.Info("engine materialised node",
			slog.String("event", "engine_materialise"),
			slog.String("scenario_id", s.ID),
			slog.String("node_id", id),
			slog.String("role", string(ns.Role)),
		)
	}
	// Ensure every materialised node has a live HTTP listener.
	if err := e.provisioner.SyncListeners(ctx); err != nil {
		return fmt.Errorf("sync listeners: %w", err)
	}

	// Fire Register for device-role nodes whose HTTP listener is up.
	// Drop rate is controlled by Exceptions.RegisterDropRate (0..1).
	e.fireRegisters(s)

	return nil
}

// fireRegisters 异步对每个 device 节点发送 Register 请求；失败仅记日志。
// GA/T 1400.4 §5.1 Register：device 节点向其 upstream（VIID Source/Server）注册，
// 此处调 DispatchRegister 把构造好的 RegisterObject 投递到节点的 Upstream。
func (e *Engine) fireRegisters(s scenario.Scenario) {
	for _, ns := range s.Nodes {
		if ns.Role != node.RoleDevice {
			e.log.Debug("fireRegisters skip non-device",
				slog.String("node", ns.Ref),
				slog.String("role", string(ns.Role)))
			continue
		}
		if rand.Float64() < s.Exceptions.RegisterDropRate {
			e.log.Debug("register dropped",
				slog.String("node", ns.Ref))
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		n, err := e.provisioner.GetNode(ctx, ns.Ref)
		if err != nil || n == nil || n.Upstream == "" {
			cancel()
			continue
		}
		body := map[string]any{
			"RegisterObject": map[string]any{
				"DeviceID": n.ID,
				"DeviceName": n.Name,
			},
		}
		// dispatcher 用 target.HTTPListen 推导客户端目标 URL，
		// 因此把 Upstream 放进临时节点的 HTTPListen 字段。
		target := node.Node{ID: n.ID, Name: n.Name, HTTPListen: n.Upstream}
		if rerr := e.dispatcher.DispatchRegister(ctx, target, body); rerr != nil {
			e.log.Debug("register",
				slog.String("node", ns.Ref),
				slog.String("error", rerr.Error()))
		}
		cancel()
	}
}

// runKeepalive 在设备节点 Upstream 上周期发送 /VIID/System/Keepalive。
// GA/T 1400.4 §5.3 Keepalive：device 节点每 keepaliveInterval 一次发送心跳，
// 服务端按 User-Identify 头 MarkSeen 节点。
// runKeepalive 定期向上游发送心跳请求。
// GA/T 1400.4 §5.3 Keepalive：device 节点按 keepaliveInterval（缺省 30s）向 upstream 发送心跳，
// 失联超过心跳周期 × 3 视为离线；jitter 避免多节点同步打点。
func (e *Engine) runKeepalive(ctx context.Context, s scenario.Scenario, ns scenario.NodeSpec, jitter time.Duration) {
	t := e.keepaliveInterval
	if t <= 0 {
		return
	}
	if jitter > 0 {
		// 应用一次确定性的 jitter offset（基于节点 ref 哈希），避免所有节点同时打心跳。
		offset := time.Duration(hashRef(ns.Ref)%int64(jitter)) * time.Millisecond
		//nolint:staticcheck // intentional small sleeps at goroutine startup
		select {
		case <-time.After(offset):
		case <-ctx.Done():
			return
		}
	}
	ticker := time.NewTicker(t)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			nctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			n, err := e.provisioner.GetNode(nctx, ns.Ref)
			if err != nil || n == nil || n.Upstream == "" {
				cancel()
				continue
			}
			// dispatcher 用 target.HTTPListen 推导目标 URL，把 Upstream 放进去。
			target := node.Node{ID: n.ID, Name: n.Name, HTTPListen: n.Upstream}
			if kerr := e.dispatcher.DispatchKeepalive(nctx, target); kerr != nil {
				e.log.Debug("keepalive",
					slog.String("node", ns.Ref),
					slog.String("error", kerr.Error()))
			}
			cancel()
		}
	}
}

// hashRef 把节点 ref 字符串映射为 int64，用于把 jitter 分散到不同节点。
func hashRef(s string) int64 {
	var h int64 = 0
	for i := 0; i < len(s); i++ {
		h = h*131 + int64(s[i])
	}
	if h < 0 {
		h = -h
	}
	return h
}

func defaultName(ref string, role node.Role) string {
	if ref == "" {
		return string(role)
	}
	return string(role) + ":" + ref
}

// paceResources ticks at 1/rate seconds per emit. Each tick generates one
// resource and dispatches it to the node's upstream (if any) so the message
// traverses the full protocol stack.
func (e *Engine) paceResources(ctx context.Context, s scenario.Scenario, rs scenario.ResourceSpec) {
	if rs.Rate <= 0 {
		return
	}
	period := time.Second / time.Duration(rs.Rate)
	if period < 10*time.Millisecond {
		period = 10 * time.Millisecond
	}
	ticker := time.NewTicker(period)
	defer ticker.Stop()

	// Optional initial burst — fire-and-forget, no immediate delay.
	if rs.Burst > 0 {
		for i := 0; i < rs.Burst; i++ {
			if ctx.Err() != nil {
				return
			}
			e.emit(ctx, s, rs)
		}
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			e.emit(ctx, s, rs)
		}
	}
}

// emit 按资源类型构造一个样本对象并投递到目标端点。
// GA/T 1400.4 §5.2 Collection Push：通过 POST /VIID/<Kind>s 向上层注入
// Faces/Videos/Vehicles/... 资源对象（factory.Build 构造，dispatcher 投递）。
func (e *Engine) emit(ctx context.Context, s scenario.Scenario, rs scenario.ResourceSpec) {
	payload := e.factory.Build(rs.Kind)
	target, err := e.findTarget(ctx, s, rs.NodeRef)
	if err != nil {
		e.log.Debug("emit skipped", slog.String("reason", err.Error()))
		return
	}
	// Apply exception injection (latency spike).
	if s.Exceptions.LatencySpike > 0 {
		select {
		case <-time.After(s.Exceptions.LatencySpike):
		case <-ctx.Done():
			return
		}
	}
	e.log.Debug("pacing emit",
		slog.String("event", "engine_pacing_emit"),
		slog.String("scenario_id", s.ID),
		slog.String("node_id", rs.NodeRef),
		slog.String("kind", string(rs.Kind)),
	)
	if err := e.dispatcher.Dispatch(ctx, target, rs.Kind, payload); err != nil {
		e.log.Debug("dispatch", slog.String("err", err.Error()))
	}
}

// findTarget resolves a node reference to a concrete node.Node. The reference
// is matched against the scenario's NodeSpec.Ref values.
func (e *Engine) findTarget(ctx context.Context, s scenario.Scenario, ref string) (node.Node, error) {
	if ref == "" {
		return node.Node{}, errors.New("emit: empty node ref")
	}
	for _, ns := range s.Nodes {
		if ns.Ref == ref {
			// Pull the persisted node so HTTPListen reflects what main wired up.
			n, err := e.provisioner.GetNode(ctx, ref)
			if err == nil && n != nil {
				return *n, nil
			}
			return node.Node{ID: ref, HTTPListen: ns.Listen, Upstream: ns.Upstream}, nil
		}
	}
	return node.Node{}, fmt.Errorf("emit: unknown ref %q", ref)
}

// dispatchNotifications fans out subscribe notifications to subscriber nodes on
// the configured interval. The notification payload is a synthetic envelope
// generated by the factory; subscribers receive it on their
// /VIID/SubscribeNotifications endpoint.
// GA/T 1400.4 §5.4 SubscribeNotification：上层 server 推送触发通知到下层
// subscriber；本实现按 Interval 周期投递，SubscribeRejectRate 控制丢包率。
// dispatchNotifications 监听上游指定类型的资源，按订阅配置推送给 subscriber。
// GA/T 1400.4 §5.4 SubscribeNotification Push：当上游产生订阅范围内的资源时，
// 向 subscriber 的 /VIID/SubscribeNotifications 端点推送通知（dispatcher.DispatchNotification）。
func (e *Engine) dispatchNotifications(ctx context.Context, s scenario.Scenario, sub scenario.SubscriptionSpec) {
	interval := sub.Interval
	if interval <= 0 {
		interval = 5 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if s.Exceptions.SubscribeRejectRate > 0 && rand.Float64() < s.Exceptions.SubscribeRejectRate {
				continue
			}
			// Resolve subscriber and publisher.
			subTarget, err := e.findTarget(ctx, s, sub.SubscriberRef)
			if err != nil {
				continue
			}
			_, _ = e.findTarget(ctx, s, sub.PublisherRef)
			notification := map[string]any{
				"SubscribeID": "sub-" + sub.PublisherRef + "-" + string(sub.Topic),
				"Title":       string(sub.Topic),
				"TriggerTime": time.Now().UTC().Format(time.RFC3339),
				"InfoIDs":     []string{},
			}
			// Override receiveAddr when explicit; otherwise derive from subscriber's HTTPListen.
			target := subTarget
			if sub.ReceiveAddr != "" {
				// We synthesise a node with the explicit receiveAddr.
				target = node.Node{ID: subTarget.ID, HTTPListen: extractPort(sub.ReceiveAddr), Upstream: sub.ReceiveAddr}
			}
			url := target.Upstream
			if url == "" {
				// Fall back to /VIID/SubscribeNotifications on subscriber's own listener.
				base, _ := baseURL(target)
				url = base + "/VIID/SubscribeNotifications"
			}
			// We can't easily reuse the dispatcher without HTTP — so use a tiny
			// inline post. Errors are intentionally swallowed (this is background work).
			e.postNotification(ctx, url, sub.SubscriberRef, notification)
		}
	}
}

func (e *Engine) postNotification(ctx context.Context, url, nodeID string, body map[string]any) {
	// Resolve URL to a Node so the capture record attributes the request correctly.
	target := node.Node{
		ID:         nodeID,
		Upstream:   url,
		HTTPListen: extractPort(url),
	}
	if err := e.dispatcher.DispatchSubscribeNotification(ctx, target, body); err != nil {
		e.log.Debug("notification post",
			slog.String("node", nodeID),
			slog.String("error", err.Error()),
		)
	}
}

// extractPort parses "http://host:port/..." → ":port". Used when a subscriber
// specifies an explicit receiveAddr that differs from the node's listen addr.
func extractPort(u string) string {
	// Trim scheme.
	for _, prefix := range []string{"http://", "https://"} {
		if len(u) > len(prefix) && u[:len(prefix)] == prefix {
			u = u[len(prefix):]
			break
		}
	}
	if i := indexByte(u, '/'); i >= 0 {
		u = u[:i]
	}
	return ":" + lastSegment(u)
}

func indexByte(s string, b byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}
	return -1
}

func lastSegment(s string) string {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == ':' {
			return s[i+1:]
		}
	}
	return s
}
