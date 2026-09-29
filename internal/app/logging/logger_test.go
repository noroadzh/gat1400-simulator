package logging

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/noroadzh/gat1400-simulator/internal/app/config"
)

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input   string
		want    slog.Level
		wantErr bool
	}{
		{"debug", slog.LevelDebug, false},
		{"DEBUG", slog.LevelDebug, false},
		{"info", slog.LevelInfo, false},
		{"INFO", slog.LevelInfo, false},
		{"", slog.LevelInfo, false},
		{"warn", slog.LevelWarn, false},
		{"WARNING", slog.LevelWarn, false},
		{"error", slog.LevelError, false},
		{"ERROR", slog.LevelError, false},
		{"foo", slog.LevelInfo, true},
		{"  info  ", slog.LevelInfo, false},
	}
	for _, tt := range tests {
		got, err := ParseLevel(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseLevel(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}
		if got != tt.want {
			t.Errorf("ParseLevel(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestLevelFromProfile(t *testing.T) {
	tests := []struct {
		profile string
		want    slog.Level
		wantErr bool
	}{
		{"prod", slog.LevelWarn, false},
		{"PROD", slog.LevelWarn, false},
		{"dev", slog.LevelInfo, false},
		{"DEV", slog.LevelInfo, false},
		{"test", slog.LevelDebug, false},
		{"TEST", slog.LevelDebug, false},
		{"", slog.LevelInfo, false},
		{"unknown", slog.LevelInfo, true},
		{"staging", slog.LevelInfo, true},
	}
	for _, tt := range tests {
		got, err := LevelFromProfile(tt.profile)
		if (err != nil) != tt.wantErr {
			t.Errorf("LevelFromProfile(%q) error = %v, wantErr %v", tt.profile, err, tt.wantErr)
			continue
		}
		if got != tt.want {
			t.Errorf("LevelFromProfile(%q) = %v, want %v", tt.profile, got, tt.want)
		}
	}
}

func TestResolveLevel(t *testing.T) {
	tests := []struct {
		cfgLevel string
		profile  string
		want     slog.Level
		wantErr  bool
	}{
		{"debug", "prod", slog.LevelDebug, false},
		{"info", "prod", slog.LevelInfo, false},
		{"", "prod", slog.LevelWarn, false},
		{"", "dev", slog.LevelInfo, false},
		{"", "test", slog.LevelDebug, false},
		{"", "", slog.LevelInfo, false},
		{"", "unknown", slog.LevelInfo, true},
	}
	for _, tt := range tests {
		got, err := ResolveLevel(tt.cfgLevel, tt.profile)
		if (err != nil) != tt.wantErr {
			t.Errorf("ResolveLevel(%q, %q) error = %v, wantErr %v",
				tt.cfgLevel, tt.profile, err, tt.wantErr)
			continue
		}
		if got != tt.want {
			t.Errorf("ResolveLevel(%q, %q) = %v, want %v",
				tt.cfgLevel, tt.profile, got, tt.want)
		}
	}
}

func TestNew_StdoutOnly(t *testing.T) {
	// 通过 setStdoutSink 注入临时文件，避免替换进程级 os.Stdout 变量。
	tmp := t.TempDir()
	stdoutFile := filepath.Join(tmp, "stdout.log")
	f, err := os.Create(stdoutFile)
	if err != nil {
		t.Fatalf("create stdout file: %v", err)
	}
	restore := setStdoutSink(f)
	defer func() {
		restore()
		_ = f.Close()
	}()

	cfg := &config.LogConfig{Level: "info", Format: "json", Stdout: true, File: ""}
	log, closer, err := New(cfg, "dev")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	log.Info("test stdout")
	if err := closer(); err != nil {
		t.Errorf("closer: %v", err)
	}
	if err := closer(); err != nil {
		t.Errorf("closer second (idempotent): %v", err)
	}
	_ = f.Sync()

	data, err := os.ReadFile(stdoutFile)
	if err != nil {
		t.Fatalf("read stdout file: %v", err)
	}
	if !strings.Contains(string(data), "test stdout") {
		t.Errorf("stdout should contain 'test stdout', got %q", string(data))
	}
}

func TestNew_FileSink(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")

	cfg := &config.LogConfig{
		Level: "debug", Format: "json", Stdout: false, File: path,
		MaxSizeMB: 1, MaxBackups: 2, MaxAgeDays: 7, Compress: false,
	}
	log, closer, err := New(cfg, "dev")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	log.Info("file test")
	_ = closer()

	// 验证文件写入。
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if !strings.Contains(string(data), "file test") {
		t.Errorf("file should contain 'file test', got %q", string(data))
	}
}

func TestNew_Fanout(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fanout.log")
	stdoutFile := filepath.Join(dir, "stdout.log")

	f, err := os.Create(stdoutFile)
	if err != nil {
		t.Fatalf("create stdout file: %v", err)
	}
	restore := setStdoutSink(f)
	defer func() {
		restore()
		_ = f.Close()
	}()

	cfg := &config.LogConfig{Level: "info", Format: "json", Stdout: true, File: path}
	log, closer, err := New(cfg, "dev")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	log.Info("fanout test")
	_ = closer()
	_ = f.Sync()

	stdoutData, err := os.ReadFile(stdoutFile)
	if err != nil {
		t.Fatalf("read stdout file: %v", err)
	}
	if !strings.Contains(string(stdoutData), "fanout test") {
		t.Errorf("stdout should contain 'fanout test', got %q", string(stdoutData))
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if !strings.Contains(string(data), "fanout test") {
		t.Errorf("file should contain 'fanout test', got %q", string(data))
	}
}

func TestNew_DailyRotation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "daily.log")

	cfg := &config.LogConfig{
		Level: "info", Format: "json", Stdout: false, File: path,
		Rotation: "daily", MaxSizeMB: 1, MaxBackups: 3, MaxAgeDays: 7, Compress: false,
	}
	log, closer, err := New(cfg, "dev")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	log.Info("daily test")
	_ = closer()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if !strings.Contains(string(data), "daily test") {
		t.Errorf("file should contain 'daily test', got %q", string(data))
	}
}

func TestNew_CloseIdempotent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "close_idem.log")

	cfg := &config.LogConfig{Level: "info", Format: "json", Stdout: false, File: path}
	_, closer, err := New(cfg, "dev")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := closer(); err != nil {
		t.Errorf("first close: %v", err)
	}
	if err := closer(); err != nil {
		t.Errorf("second close (idempotent): %v", err)
	}
}

func TestNew_UnknownFormat(t *testing.T) {
	cfg := &config.LogConfig{Level: "info", Format: "unknown"}
	_, _, err := New(cfg, "dev")
	if err == nil {
		t.Error("expected error for unknown format")
	}
}

func TestNew_NilConfig(t *testing.T) {
	log, closer, err := New(nil, "dev")
	if err != nil {
		t.Fatalf("New(nil) error = %v", err)
	}
	log.Info("nil config ok")
	_ = closer()
}
