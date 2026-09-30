// Package logging 统一模拟器日志体系。
//
// 设计要点（spec：logging）：
//   - 配置化 level / format / stdout / file / rotation / maxSizeMB / maxBackups /
//     maxAgeDays / compress；profile (prod|dev|test) 提供默认值，yaml 显式 level 优先。
//   - 多 sink（stdout + 本地文件）独立开关，按 io.MultiWriter fanout。
//   - 文件 sink 用 lumberjack（按大小轮转）+ 可选按天滚动（rotation=daily）。
//   - 调用 slog.SetDefault 同步，使遗留的 slog.Default() 也走到同一 handler。
//   - New 返回 (logger, closer, error)；closer 在文件 sink 未启用时为 no-op 幂等函数。
package logging

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/noroadzh/gat1400-simulator/internal/app/config"
)

// Profile 启动参数预设。
const (
	ProfileProd  = "prod"
	ProfileDev   = "dev"
	ProfileTest  = "test"
	ProfileEmpty = ""
)

// DefaultProfiles 各抽象 profile 对应的默认级别（spec Requirement: --profile 切换）。
// 仅列 spec 约定的三个环境名；字面 level（debug/info/warn/error）在 LevelFromProfile 中另行处理。
var DefaultProfiles = map[string]slog.Level{
	ProfileProd: slog.LevelWarn,
	ProfileDev:  slog.LevelInfo,
	ProfileTest: slog.LevelDebug,
}

// Format JSON / Text。
const (
	FormatJSON = "json"
	FormatText = "text"
)

// Rotation 文件轮转模式。
const (
	RotationSize  = "size"
	RotationDaily = "daily"
	// RotationDay 是 RotationDaily 的兼容别名，接受历史配置里的 day 写法。
	RotationDay = "day"
)

// ParseLevel 把字符串解析为 slog.Level；未知值返回 error。
// 接受：debug | info | warn | error（大小写不敏感）；空字符串按 info 处理。
func ParseLevel(s string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug, nil
	case "info", "":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, fmt.Errorf("logging: unknown level %q", s)
	}
}

// LevelFromProfile 把 --profile 映射为默认 slog.Level。
//
// 支持两种形式：
//   - 抽象 profile 名：prod / dev / test（spec 强约定）
//   - 字面 level：debug / info / warn / error（运维直接指定级别）
//
// 空字符串返回 (LevelInfo, nil)；未知值返回 ErrInvalidProfile。
func LevelFromProfile(p string) (slog.Level, error) {
	p = strings.TrimSpace(p)
	if p == "" {
		return slog.LevelInfo, nil
	}
	lower := strings.ToLower(p)
	if lvl, ok := DefaultProfiles[lower]; ok {
		return lvl, nil
	}
	// 字面 level 形式
	switch lower {
	case "debug", "info", "warn", "warning", "error":
		return ParseLevel(p)
	}
	return slog.LevelInfo, ErrInvalidProfile
}

// ResolveLevel 决定最终生效的 level：
//  1. cfg.Level 非空时使用 cfg.Level（yaml 显式优先）；
//  2. 否则使用 profile 默认（prod=Warn / dev=Info / test=Debug）；
//  3. profile 未知时返回 ErrInvalidProfile，由调用方退出非零。
func ResolveLevel(cfgLevel, profile string) (slog.Level, error) {
	if cfgLevel = strings.TrimSpace(cfgLevel); cfgLevel != "" {
		return ParseLevel(cfgLevel)
	}
	return LevelFromProfile(profile)
}

// ErrInvalidProfile 表示 --profile 取值不在 prod/dev/test 之内。
var ErrInvalidProfile = errors.New("logging: invalid profile")

// New 根据配置构造根 logger。
//
// 行为：
//   - stdout 与 file 可独立开关（cfg.Stdout / cfg.File）。
//   - file 非空时按 cfg.Rotation 选择按大小（size）或按天（daily）轮转。
//   - 构造成功后调用 slog.SetDefault(logger)。
//   - 返回的 closer 在 file 未启用时为 no-op；调用多次幂等。
//
// stdoutSink 是 cfg.Stdout=true 时使用的目标 writer。生产路径下为 os.Stdout；
// 测试可在 New 前通过 setStdoutSink 替换为 io.Writer，避免依赖进程级 fd。
var stdoutSink io.Writer = os.Stdout

// setStdoutSink 替换 stdout sink。仅供同包测试使用；调用方必须在测试结束前用
// 返回的 restore 函数恢复，否则全局状态污染其它测试。
func setStdoutSink(w io.Writer) func() {
	prev := stdoutSink
	stdoutSink = w
	return func() { stdoutSink = prev }
}

func New(cfg *config.LogConfig, profile string) (*slog.Logger, func() error, error) {
	if cfg == nil {
		cfg = &config.LogConfig{}
	}
	level, err := ResolveLevel(cfg.Level, profile)
	if err != nil {
		return nil, nil, err
	}

	writer, closer, err := buildSink(cfg)
	if err != nil {
		return nil, nil, err
	}

	// 选 handler：JSON 默认；text 可选。
	opts := &slog.HandlerOptions{Level: level}
	var handler slog.Handler
	switch strings.ToLower(strings.TrimSpace(cfg.Format)) {
	case FormatText:
		handler = slog.NewTextHandler(writer, opts)
	case "", FormatJSON:
		handler = slog.NewJSONHandler(writer, opts)
	default:
		_ = closer()
		return nil, nil, fmt.Errorf("logging: unknown format %q", cfg.Format)
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)
	return logger, closer, nil
}

// buildSink 构造 fanout writer 与关闭器。
func buildSink(cfg *config.LogConfig) (io.Writer, func() error, error) {
	var writers []io.Writer
	var closer func() error = func() error { return nil }
	anySink := false

	if cfg.Stdout {
		writers = append(writers, stdoutSink)
		anySink = true
	}

	if path := strings.TrimSpace(cfg.File); path != "" {
		dir := filepath.Dir(path)
		if dir != "" && dir != "." {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return nil, nil, fmt.Errorf("logging: create dir %s: %w", dir, err)
			}
		}
		rot := &lumberjack.Logger{
			Filename:   path,
			MaxSize:    cfg.MaxSizeMB,
			MaxBackups: cfg.MaxBackups,
			MaxAge:     cfg.MaxAgeDays,
			Compress:   cfg.Compress,
		}
		var w io.Writer = rot
		var daily *dailyWriter
		rotMode := strings.ToLower(strings.TrimSpace(cfg.Rotation))
		if rotMode == RotationDaily || rotMode == RotationDay {
			daily = newDailyWriter(rot)
			w = daily
		}
		writers = append(writers, w)
		closer = (&rotatorCloser{rot: rot, daily: daily}).Close
		anySink = true
	}

	if !anySink {
		// 既不要 stdout 也不要 file —— 退回到 stdout 避免日志黑洞（spec 规定默认行为）。
		writers = append(writers, stdoutSink)
	}

	var w io.Writer
	switch len(writers) {
	case 0:
		w = io.Discard
	case 1:
		w = writers[0]
	default:
		w = io.MultiWriter(writers...)
	}
	return w, closer, nil
}

// rotatorCloser 把 lumberjack 与 dailyWriter 合并关闭。daily 退出后台 goroutine。
type rotatorCloser struct {
	once  sync.Once
	rot   *lumberjack.Logger
	daily *dailyWriter
}

func (c *rotatorCloser) Close() error {
	var err error
	c.once.Do(func() {
		if c.daily != nil {
			c.daily.stop()
		}
		if c.rot != nil {
			err = c.rot.Close()
		}
	})
	return err
}

// dailyWriter 在跨过本地日期 00:00 时调用 rot.Rotate()。
//
// maybeRotate 由 Write 与后台 ticker 两条路径并发触发，d.last 的读改写与
// rot.Rotate() 都在 mu 下串行化，避免两条路径在跨天瞬间重复滚动。
//
// stopOnce 保证 stop 可以多次调用幂等,避免在 t.Cleanup 链中重复关闭触发
// 竞争或文件锁冲突（尤其在 Windows CI 环境下）。
type dailyWriter struct {
	rot      *lumberjack.Logger
	mu       sync.Mutex
	last     time.Time
	stopC    chan struct{}
	doneC    chan struct{}
	stopOnce sync.Once
}

func newDailyWriter(rot *lumberjack.Logger) *dailyWriter {
	d := &dailyWriter{
		rot:   rot,
		last:  startOfDayLocal(time.Now()),
		stopC: make(chan struct{}),
		doneC: make(chan struct{}),
	}
	go d.loop()
	return d
}

func (d *dailyWriter) Write(p []byte) (int, error) {
	d.maybeRotate()
	return d.rot.Write(p)
}

func (d *dailyWriter) maybeRotate() {
	d.mu.Lock()
	defer d.mu.Unlock()
	today := startOfDayLocal(time.Now())
	if today.After(d.last) {
		_ = d.rot.Rotate()
		d.last = today
	}
}

func (d *dailyWriter) loop() {
	defer close(d.doneC)
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	for {
		select {
		case <-d.stopC:
			return
		case <-t.C:
			d.maybeRotate()
		}
	}
}

func (d *dailyWriter) stop() {
	d.stopOnce.Do(func() {
		close(d.stopC)
		<-d.doneC
	})
}

func startOfDayLocal(t time.Time) time.Time {
	tt := t.Local()
	return time.Date(tt.Year(), tt.Month(), tt.Day(), 0, 0, 0, 0, tt.Location())
}
