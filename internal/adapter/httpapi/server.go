// Package httpapi 向其他节点暴露 GA/T 1400.4 REST 接口。
//
// 同时安装抓包中间件，把每个入站/出站事务记录到持久化存储。
// HTTP 框架使用 labstack/echo/v4。
package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/noroadzh/gat1400-simulator/internal/adapter/capture"
	"github.com/noroadzh/gat1400-simulator/internal/adapter/storage"
	"github.com/noroadzh/gat1400-simulator/internal/app/application"
	"github.com/noroadzh/gat1400-simulator/internal/domain/ids"
)

// Server is the protocol-level echo server.
type Server struct {
	e           *echo.Echo
	log         *slog.Logger
	nodeSvc     *application.NodeService
	scenarioSvc *application.ScenarioService
	recorder    *capture.Recorder
	nonce       *storage.NonceStore
	ids         *ids.Generator
	auth        AuthConfig
	repo        *resourceRepo
	subRepo     *subscribeRepo
	catalog     *catalogRepo
	srv         *http.Server
}

// AuthConfig is the digest credentials presented to clients.
type AuthConfig struct {
	Realm    string
	Username string
	Password string
	Qop      string
}

// Config is the configuration envelope passed from main.go. It mirrors the
// fields actually consulted by the protocol server.
type Config struct {
	Auth AuthConfig
}

// NewServer wires the protocol server with the application services. The
// returned server only exposes the *default* listener (the well-known port for
// the simulated VIID Server / control node). Per-node listeners are owned by a
// NodeRegistry created separately and reachable via NewNodeRegistry.
func NewServer(log *slog.Logger, nodeSvc *application.NodeService, scenarioSvc *application.ScenarioService, recorder *capture.Recorder, nonce *storage.NonceStore, idGen *ids.Generator, cfg *Config) *Server {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	s := &Server{
		e: e, log: log,
		nodeSvc: nodeSvc, scenarioSvc: scenarioSvc,
		recorder: recorder, nonce: nonce, ids: idGen,
		auth:    cfg.Auth,
		repo:    newResourceRepo(),
		subRepo: newSubscribeRepo(),
		catalog: newCatalogRepo(),
	}
	s.installMiddleware()
	s.installRoutes()
	return s
}

// Start binds the listener and blocks until ctx is cancelled.
func (s *Server) Start(ctx context.Context, addr string) error {
	if addr == "" {
		addr = ":14000"
	}
	s.srv = &http.Server{
		Addr:    addr,
		Handler: s.e,
	}
	errCh := make(chan error, 1)
	go func() {
		s.log.Info("protocol api listening", slog.String("addr", addr))
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

// installMiddleware registers the capture middleware. Digest auth is added
// per-route (only /VIID/System/* require it per the protocol).
func (s *Server) installMiddleware() {
	s.e.Use(TraceMiddleware(s.log))
	s.e.Use(CaptureMiddleware(s.recorder, ""))
	// GAT 1400.4 carries application/VIID+JSON which echo's default binder
	// does not understand; map it to the JSON binder so handlers can call
	// c.Bind() uniformly.
	s.e.Binder = viidBinder{base: s.e.Binder}
}

// viidBinder 扩展 echo.DefaultBinder，使其识别 application/VIID+JSON 为 JSON 内容类型。
// 其他 MIME 类型透传给底层 binder。
type viidBinder struct {
	base echo.Binder
}

func (b viidBinder) Bind(params interface{}, c echo.Context) error {
	ct := strings.Split(c.Request().Header.Get(echo.HeaderContentType), ";")[0]
	if strings.EqualFold(ct, "application/VIID+JSON") {
		body, err := io.ReadAll(c.Request().Body)
		if err != nil {
			return err
		}
		if len(body) == 0 {
			return nil
		}
		return json.Unmarshal(body, params)
	}
	if b.base == nil {
		base := echo.DefaultBinder{}
		return base.Bind(params, c)
	}
	return b.base.Bind(params, c)
}

// installRoutes 注册全部 GA/T 1400.4 端点。
// 具体 handler 实现在 collection.go / system.go / cascade.go / catalog.go。
func (s *Server) installRoutes() {
	s.registerSystemRoutes()
	s.registerCollectionRoutes()
	s.registerDataServiceRoutes()
	s.registerCascadeRoutes()
	s.registerCatalogRoutes()
}

// ServeHTTP implements http.Handler so the server can be wrapped in
// httptest.NewServer for integration tests.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.e.ServeHTTP(w, r)
}

// InstallRoutes 把 GA/T 1400.4 端点挂载到任意 echo 实例。
// 供 NodeRegistry 给每个节点分配独立 HTTP 监听器，同时复用 handler 实现与抓包配线。
func InstallRoutes(e *echo.Echo, log *slog.Logger, nodeSvc *application.NodeService, scenarioSvc *application.ScenarioService, recorder *capture.Recorder, nonce *storage.NonceStore, idGen *ids.Generator, cfg *Config, nodeID string) {
	s := &Server{
		e: e, log: log,
		nodeSvc: nodeSvc, scenarioSvc: scenarioSvc,
		recorder: recorder, nonce: nonce, ids: idGen,
		auth:    cfg.Auth,
		repo:    newResourceRepo(),
		subRepo: newSubscribeRepo(),
		catalog: newCatalogRepo(),
	}
	e.Use(TraceMiddleware(log))
	e.Use(CaptureMiddleware(recorder, nodeID))
	s.installRoutes()
}