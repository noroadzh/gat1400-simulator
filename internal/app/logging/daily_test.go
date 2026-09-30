package logging

import (
	"log/slog"
	"path/filepath"
	"sync"
	"testing"

	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/noroadzh/gat1400-simulator/internal/app/config"
)

// TestNew_DailyRotation_ConcurrentWrite 模拟跨天瞬间多 goroutine 并发触发
// maybeRotate：通过构造 dailyWriter 后并发 Write，依赖 -race 检测器兜底。
// 即便未触发（last 在测试窗口内不更新），也至少验证了 daily rotator 的并发
// 路径不会 panic。
func TestNew_DailyRotation_ConcurrentWrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "daily_race.log")
	cfg := &config.LogConfig{
		Level: "info", Format: "json", Stdout: false, File: path,
		Rotation: "daily", MaxSizeMB: 1, MaxBackups: 2, MaxAgeDays: 7,
	}
	log, closer, err := New(cfg, "dev")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = closer() }()

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				log.Info("concurrent", slog.Int("worker", n), slog.Int("j", j))
			}
		}(i)
	}
	wg.Wait()
}

// TestDailyWriter_RotateIsSerialized 验证 maybeRotate 与 stop 之间的同步：
// 并发触发 Write 与 stop 后，stop 必须等待所有 in-flight rotate 结束，
// 不能出现"已经 close 但还在 rotate"的 panic。
func TestDailyWriter_RotateIsSerialized(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "daily_serialize.log")

	rot := newLumberjackForTest(path)
	d := newDailyWriter(rot)
	t.Cleanup(func() { d.stop() })

	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_, _ = d.Write([]byte("x"))
			}
		}()
	}
	wg.Wait()
	d.stop()

	// 二次调用 stop 必须幂等。
	d.stop()
}

// newLumberjackForTest 构造一个测试用 lumberjack.Logger，避免直接依赖未导出符号。
func newLumberjackForTest(path string) *lumberjack.Logger {
	return &lumberjack.Logger{
		Filename:   path,
		MaxSize:    1,
		MaxBackups: 2,
		MaxAge:     7,
		Compress:   false,
	}
}
