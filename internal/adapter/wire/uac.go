package wire

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

// System 提供 GA/T 1400 System 类（Register/UnRegister/Keepalive/Time）高层 API。
type System struct {
	c *Client
}

// Cascade 提供 GA/T 1400 Cascade 类（订阅/布控/通知）高层 API。
type Cascade struct {
	c *Client
}

// System 返回 System 子类型实例。
func (c *Client) System() *System {
	return &System{c: c}
}

// Cascade 返回 Cascade 子类型实例。
func (c *Client) Cascade() *Cascade {
	return &Cascade{c: c}
}

// RegisterObject 是 /VIID/System/Register 请求体。
type RegisterObject struct {
	DeviceID     string `json:"DeviceID"`
	Status       string `json:"Status"`
	Keepalive    int    `json:"Keepalive"`
	DeviceName   string `json:"DeviceName"`
	Manufacturer string `json:"Manufacturer"`
	Model        string `json:"Model"`
	Firmware     string `json:"Firmware"`
	Transport    string `json:"Transport"`
	Port         int    `json:"Port"`
}

// UnRegisterObject 是 /VIID/System/UnRegister 请求体。
type UnRegisterObject struct {
	DeviceID string `json:"DeviceID"`
}

// Register 提交设备注册请求到 baseURL/VIID/System/Register。
// GA/T 1400.4 §5.1 Register：上层（VIID Server）调用本方法注册下层（VIID Source）。
func (s *System) Register(ctx context.Context, baseURL string, obj RegisterObject) error {
	path := strings.TrimSuffix(baseURL, "/") + "/VIID/System/Register"
	// GA/T 1400 服务端期望 body 形如 {"RegisterObject":{...}}
	body := map[string]any{
		"RegisterObject": obj,
	}
	_, statusCode, err := s.c.PostJSON(ctx, path, obj.DeviceID, body)
	if err != nil {
		return fmt.Errorf("Register: %w", err)
	}
	if statusCode != http.StatusOK {
		return fmt.Errorf("Register: unexpected status %d", statusCode)
	}
	return nil
}

// UnRegister 提交设备注销请求到 baseURL/VIID/System/UnRegister。
// GA/T 1400.4 §5.1 UnRegister：注销已注册设备。
func (s *System) UnRegister(ctx context.Context, baseURL, deviceID string) error {
	path := strings.TrimSuffix(baseURL, "/") + "/VIID/System/UnRegister"
	// GA/T 1400 服务端期望 body 形如 {"UnRegisterObject":{"DeviceID":"..."}}
	body := map[string]any{
		"UnRegisterObject": UnRegisterObject{DeviceID: deviceID},
	}
	_, statusCode, err := s.c.PostJSON(ctx, path, deviceID, body)
	if err != nil {
		return fmt.Errorf("UnRegister: %w", err)
	}
	if statusCode != http.StatusOK {
		return fmt.Errorf("UnRegister: unexpected status %d", statusCode)
	}
	return nil
}

// Keepalive 发送心跳到 baseURL/VIID/System/Keepalive。deviceID 用于 User-Identify 头。
// GA/T 1400.4 §5.3 Keepalive：注册后每 90s 一次心跳；心跳端点不强制鉴权。
func (s *System) Keepalive(ctx context.Context, baseURL, deviceID string) error {
	path := strings.TrimSuffix(baseURL, "/") + "/VIID/System/Keepalive"
	_, statusCode, err := s.c.PostJSON(ctx, path, deviceID, nil)
	if err != nil {
		return fmt.Errorf("Keepalive: %w", err)
	}
	if statusCode != http.StatusOK {
		return fmt.Errorf("Keepalive: unexpected status %d", statusCode)
	}
	return nil
}

// ServerTime 从 baseURL/VIID/System/Time 获取服务端时间，返回 RFC3339 时间字符串。
// GA/T 1400.4 §5.3 Time：上层主动同步服务端时间。
func (s *System) ServerTime(ctx context.Context, baseURL string) (string, error) {
	path := strings.TrimSuffix(baseURL, "/") + "/VIID/System/Time"
	resp, statusCode, err := s.c.GetJSON(ctx, path, "")
	if err != nil {
		return "", fmt.Errorf("ServerTime: %w", err)
	}
	if statusCode != http.StatusOK {
		return "", fmt.Errorf("ServerTime: unexpected status %d", statusCode)
	}
	t, ok := resp["Time"].(string)
	if !ok || t == "" {
		return "", fmt.Errorf("ServerTime: missing or invalid Time field in response")
	}
	return t, nil
}

// SubscribeCreate 发送 POST /VIID/Subscribes 创建订阅，返回请求 body 中首个 SubscribeID。
// GA/T 1400.4 §5.4 Subscribe Create：上层订阅下层资源。
func (c *Cascade) SubscribeCreate(ctx context.Context, baseURL, deviceID string, body map[string]any) (string, error) {
	path := strings.TrimSuffix(baseURL, "/") + "/VIID/Subscribes"
	_, statusCode, err := c.c.PostJSON(ctx, path, deviceID, body)
	if err != nil {
		return "", fmt.Errorf("SubscribeCreate: %w", err)
	}
	if statusCode != http.StatusOK {
		return "", fmt.Errorf("SubscribeCreate: unexpected status %d", statusCode)
	}
	return parseSubscribeIDFromBody(body)
}

// DispositionCreate 发送 POST /VIID/Dispositions 创建布控，返回请求 body 中首个 DispositionID。
// GA/T 1400.4 §5.4 Disposition Create：上层对下层发起布控（人脸/车牌等检索任务）。
func (c *Cascade) DispositionCreate(ctx context.Context, baseURL, deviceID string, body map[string]any) (string, error) {
	path := strings.TrimSuffix(baseURL, "/") + "/VIID/Dispositions"
	_, statusCode, err := c.c.PostJSON(ctx, path, deviceID, body)
	if err != nil {
		return "", fmt.Errorf("DispositionCreate: %w", err)
	}
	if statusCode != http.StatusOK {
		return "", fmt.Errorf("DispositionCreate: unexpected status %d", statusCode)
	}
	return parseDispositionIDFromBody(body)
}

// SubscribeDelete 发送 POST /VIID/Subscribes（含 DeleteOperate）删除订阅。
// GA/T 1400.4 §5.4 Subscribe Delete：上层撤销订阅；body 含 DeleteOperate 子对象。
func (c *Cascade) SubscribeDelete(ctx context.Context, baseURL, deviceID, subscribeID string) error {
	path := strings.TrimSuffix(baseURL, "/") + "/VIID/Subscribes"
	body := map[string]any{
		"DeleteOperate": map[string]string{
			"SubscribeID": subscribeID,
		},
	}
	_, statusCode, err := c.c.PostJSON(ctx, path, deviceID, body)
	if err != nil {
		return fmt.Errorf("SubscribeDelete: %w", err)
	}
	if statusCode != http.StatusOK {
		return fmt.Errorf("SubscribeDelete: unexpected status %d", statusCode)
	}
	return nil
}

// SubscribeList 发送 GET /VIID/Subscribes 返回订阅列表。
// GA/T 1400.4 §5.4 Subscribe List：响应形如 {"SubscribeList":{"SubscribeObject":[...]}}。
func (c *Cascade) SubscribeList(ctx context.Context, baseURL, deviceID string) ([]map[string]any, error) {
	path := strings.TrimSuffix(baseURL, "/") + "/VIID/Subscribes"
	resp, _, err := c.c.GetJSON(ctx, path, deviceID)
	if err != nil {
		return nil, fmt.Errorf("SubscribeList: %w", err)
	}
	wrap, ok := resp["SubscribeList"].(map[string]any)
	if !ok {
		return nil, nil
	}
	arr, ok := wrap["SubscribeObject"].([]any)
	if !ok {
		return nil, nil
	}
	result := make([]map[string]any, 0, len(arr))
	for _, item := range arr {
		if m, ok := item.(map[string]any); ok {
			result = append(result, m)
		}
	}
	return result, nil
}

// SubscribeNotificationPush 推送通知到订阅端点。
// GA/T 1400.4 §5.4 SubscribeNotification Push：下层向上层推送订阅触发通知。
func (c *Cascade) SubscribeNotificationPush(ctx context.Context, baseURL, deviceID string, body map[string]any) error {
	path := strings.TrimSuffix(baseURL, "/") + "/VIID/SubscribeNotifications"
	_, statusCode, err := c.c.PostJSON(ctx, path, deviceID, body)
	if err != nil {
		return fmt.Errorf("SubscribeNotificationPush: %w", err)
	}
	if statusCode != http.StatusOK {
		return fmt.Errorf("SubscribeNotificationPush: unexpected status %d", statusCode)
	}
	return nil
}

// --- 解析 helper ---

// parseSubscribeIDFromBody 从请求体中提取首个 SubscribeID。
// GA/T 1400 中 SubscribeID 由发起方（UAC）在请求体中生成，服务端成功响应即表示接受。
// 兼容 SubscribeList 元素为 map[string]any 或 any 两种入参形态。
func parseSubscribeIDFromBody(body map[string]any) (string, error) {
	first, err := firstListElement(body, "SubscribeList")
	if err != nil {
		return "", fmt.Errorf("parseSubscribeIDFromBody: %w", err)
	}
	id, ok := first["SubscribeID"].(string)
	if !ok || id == "" {
		return "", fmt.Errorf("parseSubscribeIDFromBody: missing SubscribeID")
	}
	return id, nil
}

// parseDispositionIDFromBody 从请求体中提取首个 DispositionID。
// 兼容 DispositionList 元素为 map[string]any 或 any 两种入参形态。
func parseDispositionIDFromBody(body map[string]any) (string, error) {
	first, err := firstListElement(body, "DispositionList")
	if err != nil {
		return "", fmt.Errorf("parseDispositionIDFromBody: %w", err)
	}
	id, ok := first["DispositionID"].(string)
	if !ok || id == "" {
		return "", fmt.Errorf("parseDispositionIDFromBody: missing DispositionID")
	}
	return id, nil
}

// firstListElement 提取给定 key（其值为对象列表）的首个元素。
// 支持 []any 和 []map[string]any 两种形态。
func firstListElement(body map[string]any, key string) (map[string]any, error) {
	v, ok := body[key]
	if !ok {
		return nil, fmt.Errorf("missing %s list", key)
	}
	switch list := v.(type) {
	case []any:
		if len(list) == 0 {
			return nil, fmt.Errorf("empty %s list", key)
		}
		m, ok := list[0].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("first element of %s is not a map", key)
		}
		return m, nil
	case []map[string]any:
		if len(list) == 0 {
			return nil, fmt.Errorf("empty %s list", key)
		}
		return list[0], nil
	default:
		return nil, fmt.Errorf("missing or empty %s list", key)
	}
}
