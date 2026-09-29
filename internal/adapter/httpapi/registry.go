package httpapi

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"sync"

	"github.com/labstack/echo/v4"

	"github.com/noroadzh/gat1400-simulator/internal/adapter/capture"
	"github.com/noroadzh/gat1400-simulator/internal/adapter/storage"
	"github.com/noroadzh/gat1400-simulator/internal/app/application"
	"github.com/noroadzh/gat1400-simulator/internal/domain/ids"
	"github.com/noroadzh/gat1400-simulator/internal/domain/node"
)

// NodeRegistry 每个注册节点持有一个独立的 echo.Server，
// 使各节点可以在各自的 HTTPListen 地址上响应而不冲突。
// goroutine-safe（sync.Mutex 保护 servers map）。
type NodeRegistry struct {
	log      *slog.Logger
	nodeSvc  *application.NodeService
	scenarioSvc *application.ScenarioService
	recorder *capture.Recorder
	nonceStore *storage.NonceStore
	idGen    *ids.Generator
	cfg      *Config
	mu       sync.Mutex
	servers  map[string]*nodeServer
}

type nodeServer struct {
	echo   *echo.Echo
	listener net.Listener
	cancel context.CancelFunc
}

// NewNodeRegistry returns an empty registry. The registry must outlive the
// listeners it spawns; lifetime is owned by Start/Stop.
func NewNodeRegistry(log *slog.Logger, nodeSvc *application.NodeService, scenarioSvc *application.ScenarioService, recorder *capture.Recorder, nonceStore *storage.NonceStore, idGen *ids.Generator, cfg *Config) *NodeRegistry {
	return &NodeRegistry{
		log: log, nodeSvc: nodeSvc, scenarioSvc: scenarioSvc,
		recorder: recorder, nonceStore: nonceStore, idGen: idGen,
		cfg: &Config{Auth: cfg.Auth},
		servers: map[string]*nodeServer{},
	}
}

// Sync 遍历所有已注册节点，重新启动 HTTPListen 地址与当前不一致的节点的监听器。
// 返回绑定时遇到的第一个错误；已运行的节点不受影响。
//
// 典型调用：main.go 在节点变更后调用一次，使节点列表与监听器集合同步。
func (r *NodeRegistry) Sync(ctx context.Context) error {
	nodes, err := r.nodeSvc.ListNodes(ctx)
	if err != nil {
		return err
	}
	// Build the desired set.
	desired := map[string]string{}
	for _, n := range nodes {
		if n.HTTPListen == "" {
			continue
		}
		desired[n.ID] = n.HTTPListen
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	// Stop nodes that are no longer desired or whose address changed.
	for id, srv := range r.servers {
		want, ok := desired[id]
		if !ok || want != srv.listener.Addr().String() {
			srv.cancel()
			_ = srv.listener.Close()
			delete(r.servers, id)
		}
	}
	// Start nodes that aren't running yet.
	for id, addr := range desired {
		if _, ok := r.servers[id]; ok {
			continue
		}
		n, err := r.nodeSvc.GetNode(ctx, id)
		if err != nil {
			return fmt.Errorf("node %s: %w", id, err)
		}
		if err := r.startLocked(ctx, *n, addr); err != nil {
			return err
		}
	}
	return nil
}

// Start brings a single node listener up. Used by external callers (main)
// when they want fine-grained control. Idempotent.
func (r *NodeRegistry) Start(ctx context.Context, n node.Node) error {
	if n.HTTPListen == "" {
		return errors.New("registry: node has empty HTTPListen")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.servers[n.ID]; ok {
		return nil
	}
	return r.startLocked(ctx, n, n.HTTPListen)
}

// Stop terminates the listener for one node.
func (r *NodeRegistry) Stop(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	srv, ok := r.servers[id]
	if !ok {
		return nil
	}
	srv.cancel()
	delete(r.servers, id)
	return srv.listener.Close()
}

// Shutdown terminates every listener. Safe to call multiple times.
func (r *NodeRegistry) Shutdown(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for id, srv := range r.servers {
		srv.cancel()
		_ = srv.listener.Close()
		delete(r.servers, id)
	}
	return nil
}

func (r *NodeRegistry) startLocked(ctx context.Context, n node.Node, addr string) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen %s for %s: %w", addr, n.ID, err)
	}
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	// Install routes for this node — sharing the same Router as the protocol API
	// is fine because each node is a separate listener; the upstream Server
	// (which wires the same handler factory) is unrelated.
	InstallRoutes(e, r.log, r.nodeSvc, r.scenarioSvc, r.recorder, r.nonceStore, r.idGen, r.cfg, n.ID)

	runCtx, cancel := context.WithCancel(ctx)
	go func() {
		if err := e.Server.Serve(ln); err != nil && !errors.Is(err, net.ErrClosed) {
			r.log.Warn("node listener stopped",
				slog.String("node", n.ID),
				slog.String("addr", ln.Addr().String()),
				slog.String("error", err.Error()),
			)
		}
	}()
	r.servers[n.ID] = &nodeServer{echo: e, listener: ln, cancel: cancel}
	_ = runCtx
	r.log.Info("node listener up",
		slog.String("node", n.ID),
		slog.String("addr", ln.Addr().String()),
	)
	return nil
}
