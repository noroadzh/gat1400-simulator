package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
)

// ObjectStore 通用 JSON 对象持久化（资源 / Catalog / Cascade / VIAS / VIIDServers）。
// bucket 约定：resource:<Kind> | resource-info:<Kind> | resource-data:<Kind> |
// catalog:APE | catalog:APS | catalog:Tollgate | catalog:Lane |
// subscribe | disposition | notif:<sid> | disp-notif:<did> |
// viidserver | vias:task | vias:control | vias:rule | vias:taskres:<taskID>:<kind>
type ObjectStore struct {
	db        *sql.DB
	closeOnce sync.Once
	closeErr  error
}

// NewObjectStore 打开与 capture/nonce 同一 DB 文件。
func NewObjectStore(ctx context.Context, path string) (*ObjectStore, error) {
	_ = ctx
	db, err := open(path)
	if err != nil {
		return nil, err
	}
	return &ObjectStore{db: db}, nil
}

func (o *ObjectStore) Close() error {
	o.closeOnce.Do(func() { o.closeErr = o.db.Close() })
	return o.closeErr
}

// Put 写入或覆盖一条 JSON 对象。
func (o *ObjectStore) Put(bucket, id string, payload map[string]any) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("objectstore put marshal: %w", err)
	}
	_, err = o.db.Exec(`INSERT INTO viid_object(bucket, id, payload) VALUES(?,?,?)
		ON CONFLICT(bucket, id) DO UPDATE SET payload=excluded.payload`, bucket, id, string(b))
	if err != nil {
		return fmt.Errorf("objectstore put: %w", err)
	}
	return nil
}

// Get 读取单条；不存在返回 (nil, false, nil)。
func (o *ObjectStore) Get(bucket, id string) (map[string]any, bool, error) {
	var raw string
	err := o.db.QueryRow(`SELECT payload FROM viid_object WHERE bucket=? AND id=?`, bucket, id).Scan(&raw)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return nil, false, err
	}
	return m, true, nil
}

// Delete 删除单条；不存在仍返回 nil。
func (o *ObjectStore) Delete(bucket, id string) error {
	_, err := o.db.Exec(`DELETE FROM viid_object WHERE bucket=? AND id=?`, bucket, id)
	return err
}

// List 返回 bucket 下全部对象（无序）。
func (o *ObjectStore) List(bucket string) ([]map[string]any, error) {
	rows, err := o.db.Query(`SELECT id, payload FROM viid_object WHERE bucket=?`, bucket)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var id, raw string
		if err := rows.Scan(&id, &raw); err != nil {
			return nil, err
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(raw), &m); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// ListIDs 返回 bucket 下全部 id。
func (o *ObjectStore) ListIDs(bucket string) ([]string, error) {
	rows, err := o.db.Query(`SELECT id FROM viid_object WHERE bucket=?`, bucket)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// DeleteBucket 清空整个 bucket。
func (o *ObjectStore) DeleteBucket(bucket string) error {
	_, err := o.db.Exec(`DELETE FROM viid_object WHERE bucket=?`, bucket)
	return err
}
