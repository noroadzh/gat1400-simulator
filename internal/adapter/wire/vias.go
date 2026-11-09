package wire

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

// VIAS 提供分析系统子集高层 API。
type VIAS struct {
	c *Client
}

// VIAS 返回 VIAS 子类型实例。
func (c *Client) VIAS() *VIAS {
	return &VIAS{c: c}
}

// Register POST /VIAS/System/Register。
func (v *VIAS) Register(ctx context.Context, baseURL string, obj RegisterObject) error {
	path := strings.TrimSuffix(baseURL, "/") + "/VIAS/System/Register"
	body := map[string]any{"RegisterObject": obj}
	_, statusCode, err := v.c.PostJSON(ctx, path, obj.DeviceID, body)
	if err != nil {
		return fmt.Errorf("VIAS.Register: %w", err)
	}
	if statusCode != http.StatusOK {
		return fmt.Errorf("VIAS.Register: unexpected status %d", statusCode)
	}
	return nil
}

// UnRegister POST /VIAS/System/UnRegister。
func (v *VIAS) UnRegister(ctx context.Context, baseURL, deviceID string) error {
	path := strings.TrimSuffix(baseURL, "/") + "/VIAS/System/UnRegister"
	body := map[string]any{"UnRegisterObject": UnRegisterObject{DeviceID: deviceID}}
	_, statusCode, err := v.c.PostJSON(ctx, path, deviceID, body)
	if err != nil {
		return fmt.Errorf("VIAS.UnRegister: %w", err)
	}
	if statusCode != http.StatusOK {
		return fmt.Errorf("VIAS.UnRegister: unexpected status %d", statusCode)
	}
	return nil
}

// Keepalive POST /VIAS/System/Keepalive。
func (v *VIAS) Keepalive(ctx context.Context, baseURL, deviceID string) error {
	path := strings.TrimSuffix(baseURL, "/") + "/VIAS/System/Keepalive"
	_, statusCode, err := v.c.PostJSON(ctx, path, deviceID, nil)
	if err != nil {
		return fmt.Errorf("VIAS.Keepalive: %w", err)
	}
	if statusCode != http.StatusOK {
		return fmt.Errorf("VIAS.Keepalive: unexpected status %d", statusCode)
	}
	return nil
}

// ServerTime GET /VIAS/System/Time。
func (v *VIAS) ServerTime(ctx context.Context, baseURL string) (string, error) {
	path := strings.TrimSuffix(baseURL, "/") + "/VIAS/System/Time"
	resp, statusCode, err := v.c.GetJSON(ctx, path, "")
	if err != nil {
		return "", fmt.Errorf("VIAS.ServerTime: %w", err)
	}
	if statusCode != http.StatusOK {
		return "", fmt.Errorf("VIAS.ServerTime: unexpected status %d", statusCode)
	}
	t, ok := resp["Time"].(string)
	if !ok || t == "" {
		return "", fmt.Errorf("VIAS.ServerTime: missing Time")
	}
	return t, nil
}

// CreateTask POST /VIAS/Tasks。
func (v *VIAS) CreateTask(ctx context.Context, baseURL, deviceID string, body map[string]any) (map[string]any, error) {
	path := strings.TrimSuffix(baseURL, "/") + "/VIAS/Tasks"
	resp, statusCode, err := v.c.PostJSON(ctx, path, deviceID, body)
	if err != nil {
		return nil, fmt.Errorf("VIAS.CreateTask: %w", err)
	}
	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("VIAS.CreateTask: unexpected status %d", statusCode)
	}
	return resp, nil
}

// ListTasks GET /VIAS/Tasks。
func (v *VIAS) ListTasks(ctx context.Context, baseURL, deviceID string) (map[string]any, error) {
	path := strings.TrimSuffix(baseURL, "/") + "/VIAS/Tasks"
	resp, statusCode, err := v.c.GetJSON(ctx, path, deviceID)
	if err != nil {
		return nil, fmt.Errorf("VIAS.ListTasks: %w", err)
	}
	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("VIAS.ListTasks: unexpected status %d", statusCode)
	}
	return resp, nil
}

// GetTask GET /VIAS/Tasks/:id。
func (v *VIAS) GetTask(ctx context.Context, baseURL, deviceID, id string) (map[string]any, error) {
	path := strings.TrimSuffix(baseURL, "/") + "/VIAS/Tasks/" + id
	resp, statusCode, err := v.c.GetJSON(ctx, path, deviceID)
	if err != nil {
		return nil, fmt.Errorf("VIAS.GetTask: %w", err)
	}
	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("VIAS.GetTask: unexpected status %d", statusCode)
	}
	return resp, nil
}

// PutTask PUT /VIAS/Tasks/:id。
func (v *VIAS) PutTask(ctx context.Context, baseURL, deviceID, id string, body map[string]any) error {
	path := strings.TrimSuffix(baseURL, "/") + "/VIAS/Tasks/" + id
	_, statusCode, err := v.c.PutJSON(ctx, path, deviceID, body)
	if err != nil {
		return fmt.Errorf("VIAS.PutTask: %w", err)
	}
	if statusCode != http.StatusOK {
		return fmt.Errorf("VIAS.PutTask: unexpected status %d", statusCode)
	}
	return nil
}

// DeleteTask DELETE /VIAS/Tasks/:id。
func (v *VIAS) DeleteTask(ctx context.Context, baseURL, deviceID, id string) error {
	path := strings.TrimSuffix(baseURL, "/") + "/VIAS/Tasks/" + id
	_, statusCode, err := v.c.DeleteJSON(ctx, path, deviceID)
	if err != nil {
		return fmt.Errorf("VIAS.DeleteTask: %w", err)
	}
	if statusCode != http.StatusOK {
		return fmt.Errorf("VIAS.DeleteTask: unexpected status %d", statusCode)
	}
	return nil
}

// TaskControl POST /VIAS/TaskControls。
func (v *VIAS) TaskControl(ctx context.Context, baseURL, deviceID string, body map[string]any) error {
	path := strings.TrimSuffix(baseURL, "/") + "/VIAS/TaskControls"
	_, statusCode, err := v.c.PostJSON(ctx, path, deviceID, body)
	if err != nil {
		return fmt.Errorf("VIAS.TaskControl: %w", err)
	}
	if statusCode != http.StatusOK {
		return fmt.Errorf("VIAS.TaskControl: unexpected status %d", statusCode)
	}
	return nil
}

// UpsertAnalysisRule POST /VIAS/AnalysisRules。
func (v *VIAS) UpsertAnalysisRule(ctx context.Context, baseURL, deviceID string, body map[string]any) error {
	path := strings.TrimSuffix(baseURL, "/") + "/VIAS/AnalysisRules"
	_, statusCode, err := v.c.PostJSON(ctx, path, deviceID, body)
	if err != nil {
		return fmt.Errorf("VIAS.UpsertAnalysisRule: %w", err)
	}
	if statusCode != http.StatusOK {
		return fmt.Errorf("VIAS.UpsertAnalysisRule: unexpected status %d", statusCode)
	}
	return nil
}

// ListAnalysisRules GET /VIAS/AnalysisRules。
func (v *VIAS) ListAnalysisRules(ctx context.Context, baseURL, deviceID string) (map[string]any, error) {
	path := strings.TrimSuffix(baseURL, "/") + "/VIAS/AnalysisRules"
	resp, statusCode, err := v.c.GetJSON(ctx, path, deviceID)
	if err != nil {
		return nil, fmt.Errorf("VIAS.ListAnalysisRules: %w", err)
	}
	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("VIAS.ListAnalysisRules: unexpected status %d", statusCode)
	}
	return resp, nil
}

// DeleteAnalysisRule DELETE /VIAS/AnalysisRules/:id。
func (v *VIAS) DeleteAnalysisRule(ctx context.Context, baseURL, deviceID, id string) error {
	path := strings.TrimSuffix(baseURL, "/") + "/VIAS/AnalysisRules/" + id
	_, statusCode, err := v.c.DeleteJSON(ctx, path, deviceID)
	if err != nil {
		return fmt.Errorf("VIAS.DeleteAnalysisRule: %w", err)
	}
	if statusCode != http.StatusOK {
		return fmt.Errorf("VIAS.DeleteAnalysisRule: unexpected status %d", statusCode)
	}
	return nil
}

// AnalysisCapability GET /VIAS/SystemCapability/AnalysisCapability。
func (v *VIAS) AnalysisCapability(ctx context.Context, baseURL, deviceID string) (map[string]any, error) {
	return v.capability(ctx, baseURL, deviceID, "AnalysisCapability")
}

// EnhancementCapability GET /VIAS/SystemCapability/EnhancementAndRestorationCapability。
func (v *VIAS) EnhancementCapability(ctx context.Context, baseURL, deviceID string) (map[string]any, error) {
	return v.capability(ctx, baseURL, deviceID, "EnhancementAndRestorationCapability")
}

// RetrievalCapability GET /VIAS/SystemCapability/RetrievalCapability。
func (v *VIAS) RetrievalCapability(ctx context.Context, baseURL, deviceID string) (map[string]any, error) {
	return v.capability(ctx, baseURL, deviceID, "RetrievalCapability")
}

func (v *VIAS) capability(ctx context.Context, baseURL, deviceID, name string) (map[string]any, error) {
	path := strings.TrimSuffix(baseURL, "/") + "/VIAS/SystemCapability/" + name
	resp, statusCode, err := v.c.GetJSON(ctx, path, deviceID)
	if err != nil {
		return nil, fmt.Errorf("VIAS.%s: %w", name, err)
	}
	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("VIAS.%s: unexpected status %d", name, statusCode)
	}
	return resp, nil
}
