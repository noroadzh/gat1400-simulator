// Package ui 承载 Web BFF（控制平面）。
//
// 职责：
//   - 向 Vue 仪表盘暴露小规模 JSON API
//   - 提供 /ws 端点用于实时事件推送
//   - 完全不了解协议层 wire 格式——只通过 app/ports 接口与下层交互
package ui

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/noroadzh/gat1400-simulator/internal/adapter/capture"
	"github.com/noroadzh/gat1400-simulator/internal/app/application"
	"github.com/noroadzh/gat1400-simulator/internal/app/ports"
	"github.com/noroadzh/gat1400-simulator/internal/domain/node"
)

// Server is the BFF echo server.
type Server struct {
	e              *echo.Echo
	log            *slog.Logger
	nodeSvc        *application.NodeService
	scenarioSvc    *application.ScenarioService
	recorder       *capture.Recorder
	captureRead    ports.CaptureReader
	hub            *Hub
	srv            *http.Server
	protocolClient *protocolClient
}

// Config is the configuration envelope used by the BFF.
type Config struct {
	Control struct {
		Listen string `yaml:"listen"`
	} `yaml:"control"`
	// Protocol 描述协议端（UAS）监听地址与对外 base URL；
	// BFF 资源对象控制面端点组会把请求透传到该 base URL。
	Protocol struct {
		Listen  string `yaml:"listen"`
		BaseURL string `yaml:"baseUrl"`
	} `yaml:"protocol"`
}

// NewServer 装配 BFF。captureRead 供仪表盘抓包面板查询；recorder 保留用于实时录制与广播副作用。
func NewServer(l *slog.Logger, nodeSvc *application.NodeService, scenarioSvc *application.ScenarioService, recorder *capture.Recorder, captureRead ports.CaptureReader, cfg *Config) *Server {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	hub := newHub()
	protocolBase := ""
	if cfg != nil {
		protocolBase = cfg.Protocol.BaseURL
	}
	s := &Server{
		e: e, log: l,
		nodeSvc: nodeSvc, scenarioSvc: scenarioSvc,
		recorder:       recorder,
		captureRead:    captureRead,
		hub:            hub,
		protocolClient: newProtocolClient(protocolBase, l),
	}
	go hub.run()
	s.installRoutes()
	return s
}

// Start binds and runs the BFF.
func (s *Server) Start(ctx context.Context, addr string) error {
	if addr == "" {
		addr = ":14080"
	}
	s.srv = &http.Server{Addr: addr, Handler: s.e}
	errCh := make(chan error, 1)
	go func() {
		s.log.Info("control bff listening", slog.String("addr", addr))
		if err := s.srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errCh:
		return err
	}
}

// Shutdown gracefully terminates the listener.
func (s *Server) Shutdown(ctx context.Context) error {
	if s.srv == nil {
		return nil
	}
	return s.srv.Shutdown(ctx)
}

// installRoutes 注册控制面路由（节点管理 / 场景控制 / 抓包查询 / 统计 / WebSocket）。
//
// 自 Change 3 起，静态资源由独立的 nginx 容器服务；本服务仅保留 JSON API 与 WebSocket。
func (s *Server) installRoutes() {
	api := s.e.Group("/api/control")
	api.GET("/system/health", s.handleHealth)
	api.GET("/system/info", s.handleInfo)
	api.GET("/nodes", s.handleNodes)
	api.GET("/nodes/:id", s.handleNodeGet)
	api.POST("/nodes", s.handleUpsertNode)
	api.DELETE("/nodes/:id", s.handleDeleteNode)

	api.GET("/scenarios", s.handleScenarios)
	api.GET("/scenarios/:id", s.handleScenarioGet)
	api.POST("/scenarios/:id/start", s.handleScenarioStart)
	api.POST("/scenarios/:id/stop", s.handleScenarioStop)

	api.GET("/captures", s.handleCaptures)
	api.GET("/captures/export/jsonl", s.handleCaptureExportJSONL)
	api.GET("/captures/export/har", s.handleCaptureExportHAR)

	api.GET("/stats", s.handleStats)

	s.installResourceRoutes(api)

	ws := s.e.Group("/ws")
	ws.GET("/events", s.handleEvents)
}

// --- handlers ---

func (s *Server) handleHealth(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]any{
		"status":   "OK",
		"service":  "gat1400-simulator",
		"revision": "0.1.0",
	})
}

func (s *Server) handleInfo(c echo.Context) error {
	nodes, _ := s.nodeSvc.ListNodes(c.Request().Context())
	scenarios := s.scenarioSvc.ListScenarios(c.Request().Context())
	return c.JSON(http.StatusOK, map[string]any{
		"revision":         "0.1.0",
		"nodeCount":        len(nodes),
		"scenarioCount":    len(scenarios),
		"runningScenarios": s.scenarioSvc.Running(),
	})
}

func (s *Server) handleNodes(c echo.Context) error {
	list, err := s.nodeSvc.ListNodes(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]any{"items": list})
}

func (s *Server) handleNodeGet(c echo.Context) error {
	id := c.Param("id")
	n, err := s.nodeSvc.GetNode(c.Request().Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]any{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, n)
}

func (s *Server) handleUpsertNode(c echo.Context) error {
	var body map[string]any
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": err.Error()})
	}
	agg := bindNode(body)
	n, err := s.nodeSvc.UpsertNode(c.Request().Context(), agg.toNode())
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, n)
}

func (s *Server) handleDeleteNode(c echo.Context) error {
	id := c.Param("id")
	if err := s.nodeSvc.RemoveNode(c.Request().Context(), id); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]any{"status": "OK"})
}

func (s *Server) handleScenarios(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]any{"items": s.scenarioSvc.ListScenarios(c.Request().Context())})
}

func (s *Server) handleScenarioGet(c echo.Context) error {
	id := c.Param("id")
	sc, err := s.scenarioSvc.Get(id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]any{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, sc)
}

func (s *Server) handleScenarioStart(c echo.Context) error {
	id := c.Param("id")
	if err := s.scenarioSvc.Start(c.Request().Context(), id); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": err.Error()})
	}
	s.hub.broadcast(Event{Type: "scenario.started", Payload: map[string]any{"id": id}})
	return c.JSON(http.StatusOK, map[string]any{"status": "OK", "running": true})
}

func (s *Server) handleScenarioStop(c echo.Context) error {
	id := c.Param("id")
	if err := s.scenarioSvc.Stop(c.Request().Context(), id); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": err.Error()})
	}
	s.hub.broadcast(Event{Type: "scenario.stopped", Payload: map[string]any{"id": id}})
	return c.JSON(http.StatusOK, map[string]any{"status": "OK", "running": false})
}

func (s *Server) handleCaptures(c echo.Context) error {
	limit := atoiDefault(c.QueryParam("limit"), 200)
	filter := ports.CaptureFilter{
		NodeID:    c.QueryParam("nodeId"),
		Direction: c.QueryParam("direction"),
		Path:      c.QueryParam("path"),
		Limit:     limit,
	}
	entries, err := s.captureRead.Query(filter)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]any{"items": entries})
}

func (s *Server) handleCaptureExportJSONL(c echo.Context) error {
	path, err := s.captureRead.ExportJSONL(ports.CaptureFilter{Path: c.QueryParam("path"), Limit: 1000})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]any{"path": path})
}

func (s *Server) handleCaptureExportHAR(c echo.Context) error {
	path, err := s.captureRead.ExportHAR(ports.CaptureFilter{Path: c.QueryParam("path"), Limit: 1000})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]any{"path": path})
}

func (s *Server) handleStats(c echo.Context) error {
	nodes, _ := s.nodeSvc.ListNodes(c.Request().Context())
	scenarios := s.scenarioSvc.ListScenarios(c.Request().Context())
	running := 0
	for _, sc := range scenarios {
		if s.scenarioSvc.IsRunning(sc.ID) {
			running++
		}
	}
	return c.JSON(http.StatusOK, map[string]any{
		"nodeCount":        len(nodes),
		"onlineNodeCount":  countOnline(nodes),
		"scenarioCount":    len(scenarios),
		"runningScenarios": running,
	})
}

func (s *Server) handleEvents(c echo.Context) error {
	return s.hub.serve(c)
}

func countOnline(nodes []node.Node) int {
	count := 0
	for _, n := range nodes {
		if n.Status == node.StatusOnline {
			count++
		}
	}
	return count
}

func atoiDefault(s string, def int) int {
	if s == "" {
		return def
	}
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return def
		}
		n = n*10 + int(r-'0')
	}
	return n
}

// nodeAggregate is an internal type used by the BFF.
type nodeAggregate struct {
	ID           string
	Name         string
	Role         node.Role
	Description  string
	HTTPListen   string
	Upstream     string
	Capabilities []node.Capability
	Tags         []string
	Metadata     map[string]any
}

func (n nodeAggregate) toNode() node.Node {
	return node.Node{
		ID:           n.ID,
		Name:         n.Name,
		Role:         n.Role,
		HTTPListen:   n.HTTPListen,
		Upstream:     n.Upstream,
		Capabilities: n.Capabilities,
		Tags:         n.Tags,
		Metadata:     n.Metadata,
	}
}

// bindNode 把通用 JSON map 转换为 nodeAggregate。缺失字段安全降为零值，验证交给 application service。
func bindNode(in map[string]any) nodeAggregate {
	return nodeAggregate{
		ID:           asString(in["id"]),
		Name:         asString(in["name"]),
		Role:         asRole(in["role"]),
		Description:  asString(in["description"]),
		HTTPListen:   asString(in["httpListen"]),
		Upstream:     asString(in["upstream"]),
		Capabilities: asCaps(in["capabilities"]),
		Tags:         asStrings(in["tags"]),
		Metadata:     asMap(in["metadata"]),
	}
}

func asString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func asRole(v any) node.Role {
	if s, ok := v.(string); ok {
		return node.Role(s)
	}
	return ""
}

func asCaps(v any) []node.Capability {
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]node.Capability, 0, len(arr))
	for _, item := range arr {
		if s, ok := item.(string); ok {
			out = append(out, node.Capability(s))
		}
	}
	return out
}

func asStrings(v any) []string {
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, item := range arr {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func asMap(v any) map[string]any {
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return nil
}
