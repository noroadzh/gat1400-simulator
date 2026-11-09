package wire

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/noroadzh/gat1400-simulator/internal/domain/resource"
)

// Collection 提供 12 Kind 业务对象集合的推送、查询与分页列表。
type Collection struct {
	c *Client
}

// Collection 返回 Collection 子类型实例。
func (c *Client) Collection() *Collection {
	return &Collection{c: c}
}

// ListResult 分页列表结果。
type ListResult struct {
	Items    []map[string]any
	TotalNum int
}

// Push POST /VIID/<Collection>，body 为标准列表包装。
func (c *Collection) Push(ctx context.Context, baseURL, deviceID string, kind resource.Kind, body map[string]any) error {
	coll, err := collectionName(kind)
	if err != nil {
		return err
	}
	path := strings.TrimSuffix(baseURL, "/") + "/VIID/" + coll
	_, statusCode, err := c.c.PostJSON(ctx, path, deviceID, body)
	if err != nil {
		return fmt.Errorf("Collection.Push: %w", err)
	}
	if statusCode != http.StatusOK {
		return fmt.Errorf("Collection.Push: unexpected status %d", statusCode)
	}
	return nil
}

// GetByID GET /VIID/<Collection>/:id。
func (c *Collection) GetByID(ctx context.Context, baseURL, deviceID string, kind resource.Kind, id string) (map[string]any, error) {
	coll, err := collectionName(kind)
	if err != nil {
		return nil, err
	}
	path := strings.TrimSuffix(baseURL, "/") + "/VIID/" + coll + "/" + id
	resp, statusCode, err := c.c.GetJSON(ctx, path, deviceID)
	if err != nil {
		return nil, fmt.Errorf("Collection.GetByID: %w", err)
	}
	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("Collection.GetByID: unexpected status %d", statusCode)
	}
	if obj, ok := resp[string(kind)].(map[string]any); ok {
		return obj, nil
	}
	return resp, nil
}

// PutByID PUT /VIID/<Collection>/:id。
func (c *Collection) PutByID(ctx context.Context, baseURL, deviceID string, kind resource.Kind, id string, body map[string]any) error {
	coll, err := collectionName(kind)
	if err != nil {
		return err
	}
	path := strings.TrimSuffix(baseURL, "/") + "/VIID/" + coll + "/" + id
	_, statusCode, err := c.c.PutJSON(ctx, path, deviceID, body)
	if err != nil {
		return fmt.Errorf("Collection.PutByID: %w", err)
	}
	if statusCode != http.StatusOK {
		return fmt.Errorf("Collection.PutByID: unexpected status %d", statusCode)
	}
	return nil
}

// DeleteByID DELETE /VIID/<Collection>/:id。
func (c *Collection) DeleteByID(ctx context.Context, baseURL, deviceID string, kind resource.Kind, id string) error {
	coll, err := collectionName(kind)
	if err != nil {
		return err
	}
	path := strings.TrimSuffix(baseURL, "/") + "/VIID/" + coll + "/" + id
	_, statusCode, err := c.c.DeleteJSON(ctx, path, deviceID)
	if err != nil {
		return fmt.Errorf("Collection.DeleteByID: %w", err)
	}
	if statusCode != http.StatusOK {
		return fmt.Errorf("Collection.DeleteByID: unexpected status %d", statusCode)
	}
	return nil
}

// List GET /VIID/<Collection>，支持 PageRecordNum / RecordStartNo。
func (c *Collection) List(ctx context.Context, baseURL, deviceID string, kind resource.Kind, pageRecordNum, recordStartNo int) (ListResult, error) {
	coll, err := collectionName(kind)
	if err != nil {
		return ListResult{}, err
	}
	u, err := url.Parse(strings.TrimSuffix(baseURL, "/") + "/VIID/" + coll)
	if err != nil {
		return ListResult{}, fmt.Errorf("Collection.List: %w", err)
	}
	q := u.Query()
	if pageRecordNum > 0 {
		q.Set("PageRecordNum", strconv.Itoa(pageRecordNum))
	}
	if recordStartNo > 0 {
		q.Set("RecordStartNo", strconv.Itoa(recordStartNo))
	}
	u.RawQuery = q.Encode()

	resp, statusCode, err := c.c.GetJSON(ctx, u.String(), deviceID)
	if err != nil {
		return ListResult{}, fmt.Errorf("Collection.List: %w", err)
	}
	if statusCode != http.StatusOK {
		return ListResult{}, fmt.Errorf("Collection.List: unexpected status %d", statusCode)
	}

	listKey := string(kind) + "List"
	objKey := string(kind) + "Object"
	items := extractWrappedList(resp, listKey, objKey)
	total := 0
	switch v := resp["TotalNum"].(type) {
	case float64:
		total = int(v)
	case int:
		total = v
	}
	return ListResult{Items: items, TotalNum: total}, nil
}

func collectionName(kind resource.Kind) (string, error) {
	coll := resource.CollectionOf(kind)
	if coll == "" {
		return "", fmt.Errorf("Collection: unsupported kind %q", kind)
	}
	return coll, nil
}
