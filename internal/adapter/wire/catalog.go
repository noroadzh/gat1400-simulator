package wire

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

// Catalog 提供 GA/T 1400 Catalog / VIIDServers 高层 API。
type Catalog struct {
	c *Client
}

// Catalog 返回 Catalog 子类型实例。
func (c *Client) Catalog() *Catalog {
	return &Catalog{c: c}
}

// ListAPEs GET /VIID/APEs。
func (c *Catalog) ListAPEs(ctx context.Context, baseURL, deviceID string) ([]map[string]any, error) {
	return c.list(ctx, baseURL, deviceID, "APEs", "APEList", "APEObject")
}

// GetAPE GET /VIID/APEs/:id。
func (c *Catalog) GetAPE(ctx context.Context, baseURL, deviceID, id string) (map[string]any, error) {
	return c.getByID(ctx, baseURL, deviceID, "APEs", id, "APE")
}

// UpsertAPEs POST /VIID/APEs。
func (c *Catalog) UpsertAPEs(ctx context.Context, baseURL, deviceID string, body map[string]any) error {
	return c.upsert(ctx, baseURL, deviceID, "APEs", body)
}

// DeleteAPE DELETE /VIID/APEs/:id。
func (c *Catalog) DeleteAPE(ctx context.Context, baseURL, deviceID, id string) error {
	return c.deleteByID(ctx, baseURL, deviceID, "APEs", id)
}

// ListAPSs GET /VIID/APSs。
func (c *Catalog) ListAPSs(ctx context.Context, baseURL, deviceID string) ([]map[string]any, error) {
	return c.list(ctx, baseURL, deviceID, "APSs", "APSList", "APSObject")
}

// GetAPS GET /VIID/APSs/:id。
func (c *Catalog) GetAPS(ctx context.Context, baseURL, deviceID, id string) (map[string]any, error) {
	return c.getByID(ctx, baseURL, deviceID, "APSs", id, "APS")
}

// UpsertAPSs POST /VIID/APSs。
func (c *Catalog) UpsertAPSs(ctx context.Context, baseURL, deviceID string, body map[string]any) error {
	return c.upsert(ctx, baseURL, deviceID, "APSs", body)
}

// DeleteAPS DELETE /VIID/APSs/:id。
func (c *Catalog) DeleteAPS(ctx context.Context, baseURL, deviceID, id string) error {
	return c.deleteByID(ctx, baseURL, deviceID, "APSs", id)
}

// ListTollgates GET /VIID/Tollgates。
func (c *Catalog) ListTollgates(ctx context.Context, baseURL, deviceID string) ([]map[string]any, error) {
	return c.list(ctx, baseURL, deviceID, "Tollgates", "TollgateList", "TollgateObject")
}

// GetTollgate GET /VIID/Tollgates/:id。
func (c *Catalog) GetTollgate(ctx context.Context, baseURL, deviceID, id string) (map[string]any, error) {
	return c.getByID(ctx, baseURL, deviceID, "Tollgates", id, "Tollgate")
}

// UpsertTollgates POST /VIID/Tollgates。
func (c *Catalog) UpsertTollgates(ctx context.Context, baseURL, deviceID string, body map[string]any) error {
	return c.upsert(ctx, baseURL, deviceID, "Tollgates", body)
}

// DeleteTollgate DELETE /VIID/Tollgates/:id。
func (c *Catalog) DeleteTollgate(ctx context.Context, baseURL, deviceID, id string) error {
	return c.deleteByID(ctx, baseURL, deviceID, "Tollgates", id)
}

// ListLanes GET /VIID/Lanes。
func (c *Catalog) ListLanes(ctx context.Context, baseURL, deviceID string) ([]map[string]any, error) {
	return c.list(ctx, baseURL, deviceID, "Lanes", "LaneList", "LaneObject")
}

// GetLane GET /VIID/Lanes/:id。
func (c *Catalog) GetLane(ctx context.Context, baseURL, deviceID, id string) (map[string]any, error) {
	return c.getByID(ctx, baseURL, deviceID, "Lanes", id, "Lane")
}

// UpsertLanes POST /VIID/Lanes。
func (c *Catalog) UpsertLanes(ctx context.Context, baseURL, deviceID string, body map[string]any) error {
	return c.upsert(ctx, baseURL, deviceID, "Lanes", body)
}

// DeleteLane DELETE /VIID/Lanes/:id。
func (c *Catalog) DeleteLane(ctx context.Context, baseURL, deviceID, id string) error {
	return c.deleteByID(ctx, baseURL, deviceID, "Lanes", id)
}

// ListVIIDServers GET /VIID/VIIDServers。
func (c *Catalog) ListVIIDServers(ctx context.Context, baseURL, deviceID string) ([]map[string]any, error) {
	return c.list(ctx, baseURL, deviceID, "VIIDServers", "VIIDServerList", "VIIDServerObject")
}

// GetVIIDServer GET /VIID/VIIDServers/:id。
func (c *Catalog) GetVIIDServer(ctx context.Context, baseURL, deviceID, id string) (map[string]any, error) {
	return c.getByID(ctx, baseURL, deviceID, "VIIDServers", id, "VIIDServer")
}

// UpsertVIIDServers POST /VIID/VIIDServers。
func (c *Catalog) UpsertVIIDServers(ctx context.Context, baseURL, deviceID string, body map[string]any) error {
	return c.upsert(ctx, baseURL, deviceID, "VIIDServers", body)
}

// PutVIIDServer PUT /VIID/VIIDServers/:id。
func (c *Catalog) PutVIIDServer(ctx context.Context, baseURL, deviceID, id string, body map[string]any) error {
	path := strings.TrimSuffix(baseURL, "/") + "/VIID/VIIDServers/" + id
	_, statusCode, err := c.c.PutJSON(ctx, path, deviceID, body)
	if err != nil {
		return fmt.Errorf("PutVIIDServer: %w", err)
	}
	if statusCode != http.StatusOK {
		return fmt.Errorf("PutVIIDServer: unexpected status %d", statusCode)
	}
	return nil
}

// DeleteVIIDServer DELETE /VIID/VIIDServers/:id。
func (c *Catalog) DeleteVIIDServer(ctx context.Context, baseURL, deviceID, id string) error {
	return c.deleteByID(ctx, baseURL, deviceID, "VIIDServers", id)
}

func (c *Catalog) list(ctx context.Context, baseURL, deviceID, coll, listKey, objectKey string) ([]map[string]any, error) {
	path := strings.TrimSuffix(baseURL, "/") + "/VIID/" + coll
	resp, statusCode, err := c.c.GetJSON(ctx, path, deviceID)
	if err != nil {
		return nil, fmt.Errorf("Catalog.List %s: %w", coll, err)
	}
	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("Catalog.List %s: unexpected status %d", coll, statusCode)
	}
	return extractWrappedList(resp, listKey, objectKey), nil
}

func (c *Catalog) getByID(ctx context.Context, baseURL, deviceID, coll, id, singular string) (map[string]any, error) {
	path := strings.TrimSuffix(baseURL, "/") + "/VIID/" + coll + "/" + id
	resp, statusCode, err := c.c.GetJSON(ctx, path, deviceID)
	if err != nil {
		return nil, fmt.Errorf("Catalog.Get %s: %w", coll, err)
	}
	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("Catalog.Get %s: unexpected status %d", coll, statusCode)
	}
	if obj, ok := resp[singular].(map[string]any); ok {
		return obj, nil
	}
	// 部分实现对单条直接返回对象字段名不同；回退整包。
	return resp, nil
}

func (c *Catalog) upsert(ctx context.Context, baseURL, deviceID, coll string, body map[string]any) error {
	path := strings.TrimSuffix(baseURL, "/") + "/VIID/" + coll
	_, statusCode, err := c.c.PostJSON(ctx, path, deviceID, body)
	if err != nil {
		return fmt.Errorf("Catalog.Upsert %s: %w", coll, err)
	}
	if statusCode != http.StatusOK {
		return fmt.Errorf("Catalog.Upsert %s: unexpected status %d", coll, statusCode)
	}
	return nil
}

func (c *Catalog) deleteByID(ctx context.Context, baseURL, deviceID, coll, id string) error {
	path := strings.TrimSuffix(baseURL, "/") + "/VIID/" + coll + "/" + id
	_, statusCode, err := c.c.DeleteJSON(ctx, path, deviceID)
	if err != nil {
		return fmt.Errorf("Catalog.Delete %s: %w", coll, err)
	}
	if statusCode != http.StatusOK {
		return fmt.Errorf("Catalog.Delete %s: unexpected status %d", coll, statusCode)
	}
	return nil
}
