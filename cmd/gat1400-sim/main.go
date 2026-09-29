// Package main GA/T 1400 模拟器入口。
//
// 职责：
//   - 引导应用容器（配置、logger、节点服务、场景引擎、抓包记录器、HTTP API、Web BFF）
//   - 按分层架构组织依赖（app -> domain -> adapter，禁止反向引用）
//   - 启动优雅关闭 context
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/noroadzh/gat1400-simulator/internal/adapter/capture"
	"github.com/noroadzh/gat1400-simulator/internal/adapter/httpapi"
	scenarioadapter "github.com/noroadzh/gat1400-simulator/internal/adapter/scenario"
	"github.com/noroadzh/gat1400-simulator/internal/adapter/storage"
	"github.com/noroadzh/gat1400-simulator/internal/adapter/wire"
	"github.com/noroadzh/gat1400-simulator/internal/app/application"
	"github.com/noroadzh/gat1400-simulator/internal/app/config"
	"github.com/noroadzh/gat1400-simulator/internal/app/logging"
	"github.com/noroadzh/gat1400-simulator/internal/app/ports"
	"github.com/noroadzh/gat1400-simulator/internal/domain/ids"
	"github.com/noroadzh/gat1400-simulator/internal/ui"
)

var (
	// profile 默认值固定为 "info"，与项目旧行为一致；显式 --profile=prod 才会
	// 升级到 warn（生产 BUG 日志），--profile=test 才会下放到 debug。
	profileFlag = flag.String("profile", "info", "运行时环境：prod|dev|test|info|warn|debug（决定默认日志级别）")
)

func main() {
	// -healthcheck 子命令：由 Docker / Compose healthcheck 调用，
	// 直接对本地 BFF 端口做 TCP 探测，不依赖外部 shell / curl。
	if len(os.Args) > 1 && os.Args[1] == "-healthcheck" {
		addr := os.Getenv("GAT1400_CONTROL_LISTEN")
		if addr == "" {
			addr = ":14080"
		}
		if _, err := net.DialTimeout("tcp", "127.0.0.1"+addr, 3*time.Second); err != nil {
			fmt.Fprintf(os.Stderr, "healthcheck failed: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	flag.Parse()

	ready, err := run(*profileFlag)
	if err != nil {
		emitFatal(ready, err, *profileFlag)
		os.Exit(1)
	}
}

// emitFatal 把启动阶段的致命错误统一处理：
//   - ready=true：根 logger 可用，写 event=startup_failure；
//   - ready=false：根 logger 不可用，走 stderr 兜底。
//
// 抽出来便于测试断言 event 字段，也避免 main() 与 run() 双写同一错误。
func emitFatal(ready bool, err error, profile string) {
	if ready {
		slog.Default().Error("startup_failure",
			slog.String("event", "startup_failure"),
			slog.String("error", err.Error()),
			slog.String("profile", profile),
		)
		return
	}
	fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
}

// run 引导应用容器并在失败时返回原因。
//
// 返回值 ready 表示根 logger 是否已可用（配置加载 + logger 构造均成功）：
//   - ready == false：调用方必须走 stderr 兜底，用户至少能看到错误原因。
//   - ready == true：调用方应把错误以 event=startup_failure 写入根 logger。
//
// 该状态由 run() 自身返回值携带，不使用包级可变状态，避免跨函数读写。
func run(profile string) (ready bool, err error) {
	cfg, err := config.Load(
		"configs/default.yaml",
		os.Getenv("GAT1400_CONFIG"),
		"configs/local.yaml",
	)
	if err != nil {
		// 配置加载失败时 logger 尚未初始化，调用方走 stderr 兜底。
		return false, fmt.Errorf("load config: %w", err)
	}

	rootLogger, closer, err := logging.New(&cfg.Log, profile)
	if err != nil {
		// logger 构造失败，调用方仍走 stderr 兜底。
		return false, fmt.Errorf("init logger: %w", err)
	}
	defer func() { _ = closer() }()

	// logging.New already calls slog.SetDefault internally so legacy
	// slog.Default() call sites (e.g. internal/ui/ws.go) share the same root.
	// 此处 ready=true 意味着后续失败可由调用方统一落 event=startup_failure。

	rootCtx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	captureStore, err := storage.NewCaptureStore(rootCtx, cfg.Storage.Path)
	if err != nil {
		return true, fmt.Errorf("open capture store: %w", err)
	}
	defer captureStore.Close()

	nonceStore, err := storage.NewNonceStore(rootCtx, cfg.Storage.Path)
	if err != nil {
		return true, fmt.Errorf("open nonce store: %w", err)
	}
	defer nonceStore.Close()

	captureReader, err := storage.NewCaptureReader(cfg.Storage.Path)
	if err != nil {
		return true, fmt.Errorf("open capture reader: %w", err)
	}

	recorder := capture.NewRecorder(captureStore, rootLogger)
	idGen := ids.NewGenerator(uint32(cfg.Node.SiteCode), uint32(cfg.Node.IndustryCode))
	wireClient := wire.NewClient(rootLogger, nonceStore)
	wireClient.Configure(wire.Options{
		Username: cfg.Auth.Username,
		Password: cfg.Auth.Password,
		Realm:    cfg.Auth.Realm,
		Qop:      cfg.Auth.Qop,
		Timeout:  10 * time.Second,
	})

	nodeSvc := application.NewNodeService(rootLogger, idGen)
	factory := scenarioadapter.NewFactory(idGen, cfg.ScenarioSeed)
	dispatcher := scenarioadapter.NewOutboundDispatcher(wireClient, rootLogger, recorder)
	engine := scenarioadapter.NewEngine(rootLogger, factory, dispatcher, recorder, nodeSvc)
	engine.SetKeepaliveInterval(cfg.KeepaliveInterval)

	// The HTTP registry installs a sync hook on nodeSvc so that every node
	// materialised by the scenario engine gets a live HTTP listener.
	var nodeRegistry *httpapi.NodeRegistry
	nodeSvc.SetSyncHook(func(ctx context.Context) error {
		if nodeRegistry == nil {
			return nil
		}
		return nodeRegistry.Sync(ctx)
	})

	scenarioSvc := application.NewScenarioService(rootLogger, engine)

	// Load scenarios from disk. Auto-start happens AFTER the node registry is
	// wired so that the engine's materialisation can trigger listener setup.
	if cfg.ScenariosDir != "" {
		loader := scenarioadapter.NewLoader()
		loaded, lerrs := loader.LoadDir(cfg.ScenariosDir)
		for _, e := range lerrs {
			rootLogger.Warn("scenario load error", slog.String("error", e.Error()))
		}
		scenarioSvc.LoadAll(loaded)
		rootLogger.Info("scenarios loaded",
			slog.String("dir", cfg.ScenariosDir),
			slog.Int("count", len(loaded)),
			slog.Int("errors", len(lerrs)),
		)
	}

	apiServer := httpapi.NewServer(rootLogger, nodeSvc, scenarioSvc, recorder, nonceStore, idGen, httpapiConfig(cfg))
	nodeRegistry = httpapi.NewNodeRegistry(rootLogger, nodeSvc, scenarioSvc, recorder, nonceStore, idGen, httpapiConfig(cfg))
	bffServer := ui.NewServer(rootLogger, nodeSvc, scenarioSvc, recorder, captureReader, uiConfig(cfg))

	// Auto-start all scenarios that declared Schedule.AutoStart=true.
	// ScenarioService.AutoStart filters internally; each selected scenario gets
	// ScenarioService.Start → Engine.AutoStart → Engine.Start.
	if errs := scenarioSvc.AutoStart(rootCtx); len(errs) > 0 {
		for _, err := range errs {
			rootLogger.Warn("auto-start failed", slog.String("error", err.Error()))
		}
	}

	errCh := make(chan error, 2)
	go func() {
		if err := apiServer.Start(rootCtx, cfg.Protocol.Listen); err != nil && !errors.Is(err, context.Canceled) {
			errCh <- fmt.Errorf("protocol api: %w", err)
		}
	}()
	go func() {
		if err := bffServer.Start(rootCtx, cfg.Control.Listen); err != nil && !errors.Is(err, context.Canceled) {
			errCh <- fmt.Errorf("control bff: %w", err)
		}
	}()

	rootLogger.Info("gat1400-simulator started",
		slog.String("protocol_addr", cfg.Protocol.Listen),
		slog.String("control_addr", cfg.Control.Listen),
		slog.String("scenarios_dir", cfg.ScenariosDir),
		slog.String("profile", profile),
	)

	// Stop any running scenarios on shutdown.
	defer func() {
		for _, id := range scenarioSvc.Running() {
			_ = scenarioSvc.Stop(context.Background(), id)
		}
	}()

	select {
	case <-rootCtx.Done():
		rootLogger.Info("shutdown signal received")
	case err := <-errCh:
		cancel()
		// server 失败属于 startup 之后才发生的运行时错误，单独记一条 event=server_error
		// 让运维能区分「启动阶段失败」与「启动后崩溃」。
		rootLogger.Error("server error",
			slog.String("event", "server_error"),
			slog.String("error", err.Error()),
		)
		return true, err
	}

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()
	_ = apiServer.Shutdown(shutdownCtx)
	_ = bffServer.Shutdown(shutdownCtx)
	if nodeRegistry != nil {
		_ = nodeRegistry.Shutdown(shutdownCtx)
	}

	rootLogger.Info("gat1400-simulator stopped")
	return true, nil
}

// Compile-time assertions：编译期验证 application 包实现了 ports 接口。
var (
	_ ports.NodeService     = (*application.NodeService)(nil)
	_ ports.ScenarioService = (*application.ScenarioService)(nil)
)

// httpapiConfig adapts the global Config to the httpapi.Config shape.
func httpapiConfig(c *config.Config) *httpapi.Config {
	return &httpapi.Config{
		Auth: httpapi.AuthConfig{
			Realm:    c.Auth.Realm,
			Username: c.Auth.Username,
			Password: c.Auth.Password,
			Qop:      c.Auth.Qop,
		},
	}
}

// uiConfig adapts the global Config to the BFF Config shape.
//
// 关键：把协议端监听地址（`:9000`）转换成 BFF 可访问的 base URL（`http://127.0.0.1:9000`）。
// BFF 资源对象端点组（/api/control/resources/*）通过该 base URL 透传到协议端。
func uiConfig(c *config.Config) *ui.Config {
	u := &ui.Config{}
	u.Control.Listen = c.Control.Listen
	u.Protocol.Listen = c.Protocol.Listen
	u.Protocol.BaseURL = protocolBaseURL(c.Protocol.Listen)
	return u
}

// protocolBaseURL 把 ":14000" 形式监听地址转换为 BFF 可访问的 base URL。
func protocolBaseURL(listen string) string {
	addr := listen
	if addr == "" {
		addr = ":14000"
	}
	host := "127.0.0.1"
	if strings.HasPrefix(addr, ":") {
		return "http://" + host + addr
	}
	if strings.Contains(addr, ":") {
		return "http://" + addr
	}
	return "http://" + addr + ":14000"
}
