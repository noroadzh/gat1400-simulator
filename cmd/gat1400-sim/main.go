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
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/noroadzh/gat1400-simulator/internal/adapter/capture"
	"github.com/noroadzh/gat1400-simulator/internal/adapter/httpapi"
	scenarioadapter "github.com/noroadzh/gat1400-simulator/internal/adapter/scenario"
	"github.com/noroadzh/gat1400-simulator/internal/adapter/storage"
	"github.com/noroadzh/gat1400-simulator/internal/adapter/wire"
	"github.com/noroadzh/gat1400-simulator/internal/app/application"
	"github.com/noroadzh/gat1400-simulator/internal/app/config"
	"github.com/noroadzh/gat1400-simulator/internal/app/ports"
	"github.com/noroadzh/gat1400-simulator/internal/domain/ids"
	"github.com/noroadzh/gat1400-simulator/internal/ui"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	rootCtx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	cfg, err := config.Load(
		"configs/default.yaml",
		os.Getenv("GAT1400_CONFIG"),
		"configs/local.yaml",
	)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	captureStore, err := storage.NewCaptureStore(rootCtx, cfg.Storage.Path)
	if err != nil {
		return fmt.Errorf("open capture store: %w", err)
	}
	defer captureStore.Close()

	nonceStore, err := storage.NewNonceStore(rootCtx, cfg.Storage.Path)
	if err != nil {
		return fmt.Errorf("open nonce store: %w", err)
	}
	defer nonceStore.Close()

	captureReader, err := storage.NewCaptureReader(cfg.Storage.Path)
	if err != nil {
		return fmt.Errorf("open capture reader: %w", err)
	}

	recorder := capture.NewRecorder(captureStore, logger)
	idGen := ids.NewGenerator(uint32(cfg.Node.SiteCode), uint32(cfg.Node.IndustryCode))
	wireClient := wire.NewClient(logger, nonceStore)
	wireClient.Configure(wire.Options{
		Username: cfg.Auth.Username,
		Password: cfg.Auth.Password,
		Realm:    cfg.Auth.Realm,
		Qop:      cfg.Auth.Qop,
		Timeout:  10 * time.Second,
	})

	nodeSvc := application.NewNodeService(logger, idGen)
	factory := scenarioadapter.NewFactory(idGen, cfg.ScenarioSeed)
	dispatcher := scenarioadapter.NewOutboundDispatcher(wireClient, logger, recorder)
	engine := scenarioadapter.NewEngine(logger, factory, dispatcher, recorder, nodeSvc)
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

	scenarioSvc := application.NewScenarioService(logger, engine)

	// Load scenarios from disk. Auto-start happens AFTER the node registry is
	// wired so that the engine's materialisation can trigger listener setup.
	if cfg.ScenariosDir != "" {
		loader := scenarioadapter.NewLoader()
		loaded, lerrs := loader.LoadDir(cfg.ScenariosDir)
		for _, e := range lerrs {
			logger.Warn("scenario load error", slog.String("error", e.Error()))
		}
		scenarioSvc.LoadAll(loaded)
		logger.Info("scenarios loaded",
			slog.String("dir", cfg.ScenariosDir),
			slog.Int("count", len(loaded)),
			slog.Int("errors", len(lerrs)),
		)
	}

	apiServer := httpapi.NewServer(logger, nodeSvc, scenarioSvc, recorder, nonceStore, idGen, httpapiConfig(cfg))
	nodeRegistry = httpapi.NewNodeRegistry(logger, nodeSvc, scenarioSvc, recorder, nonceStore, idGen, httpapiConfig(cfg))
	bffServer := ui.NewServer(logger, nodeSvc, scenarioSvc, recorder, captureReader, uiConfig(cfg))

	// Auto-start all scenarios that declared Schedule.AutoStart=true.
	// ScenarioService.AutoStart filters internally; each selected scenario gets
	// ScenarioService.Start → Engine.AutoStart → Engine.Start.
	if errs := scenarioSvc.AutoStart(rootCtx); len(errs) > 0 {
		for _, err := range errs {
			logger.Warn("auto-start failed", slog.String("error", err.Error()))
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

	logger.Info("gat1400-simulator started",
		slog.String("protocol_addr", cfg.Protocol.Listen),
		slog.String("control_addr", cfg.Control.Listen),
		slog.String("scenarios_dir", cfg.ScenariosDir),
	)

	// Stop any running scenarios on shutdown.
	defer func() {
		for _, id := range scenarioSvc.Running() {
			_ = scenarioSvc.Stop(context.Background(), id)
		}
	}()

	select {
	case <-rootCtx.Done():
		logger.Info("shutdown signal received")
	case err := <-errCh:
		cancel()
		return err
	}

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()
	_ = apiServer.Shutdown(shutdownCtx)
	_ = bffServer.Shutdown(shutdownCtx)
	if nodeRegistry != nil {
		_ = nodeRegistry.Shutdown(shutdownCtx)
	}

	logger.Info("gat1400-simulator stopped")
	return nil
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

func uiConfig(c *config.Config) *ui.Config {
	u := &ui.Config{}
	u.Control.Listen = c.Control.Listen
	return u
}