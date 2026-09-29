// Package storage 实现模拟器基于 SQLite 的持久化适配器。
//
// 故意保持精简：仅暴露两个 store（nonce、capture），它们通过同一份 SQLite 文件共享连接。
// 使用 modernc.org/sqlite（纯 Go、无 CGO），便于跨平台交叉编译。
//
// 数据模型：
//   - capture 表：每次 HTTP 请求的完整快照（含 header / request / response）
//   - nonce 表：HTTP Digest 认证的 nonce 重放保护
//   - node 表：节点元数据快照
package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"

	"github.com/noroadzh/gat1400-simulator/internal/app/ports"
)

// open 打开（并自动迁移）指定路径的模拟器数据库。
// 同一进程内多次调用是安全的：*sql.DB 内部的连接池按路径复用。
//
// DSN 中启用：
//   - WAL 模式：提高并发写性能
//   - busy_timeout=5s：缓解瞬时锁竞争
//   - foreign_keys=ON：启用外键约束
func open(path string) (*sql.DB, error) {
	if path == "" {
		return nil, errors.New("storage: empty path")
	}
	if dir := filepath.Dir(path); dir != "" {
		if err := mkdirAll(dir); err != nil {
			return nil, err
		}
	}
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)", path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(8)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(time.Hour)
	if err := migrate(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

// migrate 创建标准 schema。DDL 全部幂等（CREATE ... IF NOT EXISTS），
// 保证启动流程的可复现性——无论是否首次启动，结果都一致。
func migrate(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS capture (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			node_id     TEXT NOT NULL,
			direction   TEXT NOT NULL,
			method      TEXT NOT NULL,
			path        TEXT NOT NULL,
			url         TEXT NOT NULL,
			remote      TEXT NOT NULL,
			status      INTEGER NOT NULL,
			duration_ms INTEGER NOT NULL,
			started_at  DATETIME NOT NULL,
			header      TEXT NOT NULL,
			request     TEXT NOT NULL,
			response    TEXT NOT NULL,
			error       TEXT NOT NULL DEFAULT ''
		);`,
		`CREATE INDEX IF NOT EXISTS idx_capture_started_at ON capture(started_at);`,
		`CREATE INDEX IF NOT EXISTS idx_capture_node ON capture(node_id);`,
		`CREATE TABLE IF NOT EXISTS nonce (
			value      TEXT PRIMARY KEY,
			issued_at  DATETIME NOT NULL,
			used_count INTEGER NOT NULL DEFAULT 0,
			expires_at DATETIME NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS node (
			id          TEXT PRIMARY KEY,
			name        TEXT NOT NULL,
			role        TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			http_listen TEXT NOT NULL DEFAULT '',
			upstream    TEXT NOT NULL DEFAULT '',
			capabilities TEXT NOT NULL DEFAULT '[]',
			status      TEXT NOT NULL DEFAULT 'stopped',
			last_seen_at DATETIME,
			tags        TEXT NOT NULL DEFAULT '[]',
			metadata    TEXT NOT NULL DEFAULT '{}',
			created_at  DATETIME NOT NULL,
			updated_at  DATETIME NOT NULL
		);`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return fmt.Errorf("storage: migrate: %w", err)
		}
	}
	return nil
}

// NewCaptureStore 返回一个 sqlite 后端的 CaptureStore。
// ctx 当前未使用，预留用于将来支持 graceful shutdown。
func NewCaptureStore(ctx context.Context, path string) (ports.CaptureStore, error) {
	db, err := open(path)
	if err != nil {
		return nil, err
	}
	return &captureStore{db: db}, nil
}

// NewNonceStore 返回一个 sqlite 后端的 NonceStore。
// 与 capture store 共享同一份 schema，所以全程序只需要一份数据库文件。
func NewNonceStore(ctx context.Context, path string) (*NonceStore, error) {
	db, err := open(path)
	if err != nil {
		return nil, err
	}
	return &NonceStore{db: db, ttl: 5 * time.Minute}, nil
}

// mkdirAll os.MkdirAll 的小封装，集中放到这里以便测试桩替换（更干净的测试边界）。
func mkdirAll(dir string) error {
	return mkdir(dir, 0o755)
}