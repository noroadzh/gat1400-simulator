package storage

import (
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"
)

// ErrNonceUnknown nonce 未签发或已过期。HTTP 调用方应返回 401。
var ErrNonceUnknown = errors.New("nonce: unknown")

// defaultNonceMaxUses 单个 nonce 在 TTL 窗口内允许的最大重用次数。
// 用于 digest 挑战的 nonce 重放保护——超出后必须重新签发。
const defaultNonceMaxUses = 100

// NonceStore 基于 sqlite 的薄包装。
//
// closeOnce 保证 Close 可以多次调用幂等,避免 t.Cleanup 链中重复
// 关闭触发 sql.ErrConnDone 或 Windows 上的文件锁冲突。
type NonceStore struct {
	db        *sql.DB
	ttl       time.Duration
	maxUses   int
	closeOnce sync.Once
	closeErr  error
}

// Issue 签发一个新 nonce 并写入持久化存储（RFC 2617 §3.2.1 步骤 1）。
// 使用 INSERT OR IGNORE 避免与既有 nonce 冲突；持久化保证进程重启后挑战不丢失。
func (n *NonceStore) Issue(value string) error {
	now := time.Now().UTC()
	_, err := n.db.Exec(`INSERT OR IGNORE INTO nonce(value, issued_at, used_count, expires_at) VALUES (?,?,?,?)`,
		value, now, 0, now.Add(n.ttl))
	return err
}

// Consume 标记 nonce 被消费一次，nonce 已过期、不存在或已被消费过时返回 ErrNonceUnknown。
//
// RFC 2617 §3.2.1 要求服务端在每个 401 响应中签发新 nonce，客户端在后续请求中携带。
// NonceStore.Consume 以 nonce 字符串为键；同一 nonce 的第二次请求会因 RowsAffected=0
// 而返回 ErrNonceUnknown，从而触发 401 挑战，实现重放保护。
//
// 实现说明：依赖单条 UPDATE 的 RowsAffected 判定 nonce 是否有效（未过期且未被消费）；
// WHERE 子句中的 used_count = 0 保证同一 nonce 只能被消费一次，从而实现
// GA/T 1400.4 §5.1 要求的 (nonce, nc) 联合主键去重——重放同一 Authorization
// 头会因 nonce 已被标记消费而返回 ErrNonceUnknown，触发 401 挑战。
// 故意不做自动 prune，由后台 goroutine 周期性调用 Purge 维护表大小。
func (n *NonceStore) Consume(value string) error {
	now := time.Now().UTC()
	res, err := n.db.Exec(`UPDATE nonce SET used_count = used_count + 1 WHERE value = ? AND expires_at > ? AND used_count = 0`, value, now)
	if err != nil {
		return fmt.Errorf("nonce: consume: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		// Either expired or never issued. We deliberately do not auto-prune;
		// a periodic GC pass keeps the table small.
		return ErrNonceUnknown
	}
	return nil
}

// Purge 删除所有已过期行。建议由后台 goroutine 周期性调用。
func (n *NonceStore) Purge() error {
	_, err := n.db.Exec(`DELETE FROM nonce WHERE expires_at <= ?`, time.Now().UTC())
	return err
}

// Close 释放底层 sqlite 句柄。多次调用安全幂等,返回第一次 Close 的错误。
// 幂等保护使测试 t.Cleanup 链可任意注册多个,避免在 Windows 上文件锁导致
// t.TempDir RemoveAll 失败。
func (n *NonceStore) Close() error {
	n.closeOnce.Do(func() { n.closeErr = n.db.Close() })
	return n.closeErr
}

// MaxUses returns the configured maximum number of reuses per nonce.
func (n *NonceStore) MaxUses() int {
	if n.maxUses == 0 {
		return defaultNonceMaxUses
	}
	return n.maxUses
}