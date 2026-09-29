// Package scenario 分层纪律：dispatcher 是 engine 的私有子包，
// 负责把节点行为派发为 wire.Client 调用（HTTP POST Register/Push/Keepalive）。
// dispatcher 不应被 engine 之外的任何包导入。
package scenario

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"time"

	"github.com/noroadzh/gat1400-simulator/internal/adapter/capture"
	"github.com/noroadzh/gat1400-simulator/internal/adapter/wire"
	"github.com/noroadzh/gat1400-simulator/internal/domain/node"
	"github.com/noroadzh/gat1400-simulator/internal/domain/resource"
)

// Dispatcher 引擎用于把生成的资源推送到对端 VIID 服务端的出站接口。
// 故意保持极简，便于单元测试时用 fake 替换，无需引入 HTTP client。
type Dispatcher interface {
	Dispatch(ctx context.Context, target node.Node, kind resource.Kind, payload map[string]any) error
}

// OutboundDispatcher 封装 wire HTTP 客户端。
//
// goroutine-safe：wire client 自行维护连接池与 digest 状态。
// 若 recorder 非空，每次 dispatch（成功或失败）都会写入抓包存储，供仪表盘展示出站流量。
type OutboundDispatcher struct {
	client   *wire.Client
	log      *slog.Logger
	recorder *capture.Recorder
	timeout  time.Duration
}

// NewOutboundDispatcher 返回一个把 payload POST 到 target 节点 VIID Collection URI 的 Dispatcher。
//
// base URL 从 target.HTTPListen 推导（如 ":14101" → "http://127.0.0.1:14101"）。
// recorder 可为 nil；非空时每次 dispatch 都会被记录。
func NewOutboundDispatcher(client *wire.Client, log *slog.Logger, recorder *capture.Recorder) *OutboundDispatcher {
	return &OutboundDispatcher{client: client, log: log, recorder: recorder, timeout: 5 * time.Second}
}

// DispatchRegister POSTs a RegisterObject to /VIID/System/Register on the target.
func (d *OutboundDispatcher) DispatchRegister(ctx context.Context, target node.Node, body map[string]any) error {
	url, err := baseURL(target)
	if err != nil {
		return err
	}
	url += "/VIID/System/Register"
	start := time.Now()
	cctx, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()
	resp, status, err := d.client.PostJSON(cctx, url, target.ID, body)
	dur := time.Since(start)
	record(d, ctx, target, "POST", "/VIID/System/Register", url, body, resp, status, dur, err)
	return err
}

// DispatchKeepalive POSTs an empty body to /VIID/System/Keepalive.
func (d *OutboundDispatcher) DispatchKeepalive(ctx context.Context, target node.Node) error {
	url, err := baseURL(target)
	if err != nil {
		return err
	}
	url += "/VIID/System/Keepalive"
	start := time.Now()
	cctx, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()
	resp, status, err := d.client.PostJSON(cctx, url, target.ID, nil)
	dur := time.Since(start)
	record(d, ctx, target, "POST", "/VIID/System/Keepalive", url, nil, resp, status, dur, err)
	return err
}

// DispatchUnregister POSTs to /VIID/System/UnRegister.
func (d *OutboundDispatcher) DispatchUnregister(ctx context.Context, target node.Node) error {
	url, err := baseURL(target)
	if err != nil {
		return err
	}
	url += "/VIID/System/UnRegister"
	body := map[string]any{
		"UnRegisterObject": map[string]any{"DeviceID": target.ID},
	}
	start := time.Now()
	cctx, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()
	resp, status, err := d.client.PostJSON(cctx, url, target.ID, body)
	dur := time.Since(start)
	record(d, ctx, target, "POST", "/VIID/System/UnRegister", url, body, resp, status, dur, err)
	return err
}

// DispatchSubscribe POSTs to /VIID/Subscribes.
func (d *OutboundDispatcher) DispatchSubscribe(ctx context.Context, target node.Node, body map[string]any) error {
	url, err := baseURL(target)
	if err != nil {
		return err
	}
	url += "/VIID/Subscribes"
	start := time.Now()
	cctx, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()
	resp, status, err := d.client.PostJSON(cctx, url, "", body)
	dur := time.Since(start)
	record(d, ctx, target, "POST", "/VIID/Subscribes", url, body, resp, status, dur, err)
	return err
}

// DispatchSubscribeNotification POSTs to /VIID/SubscribeNotifications.
func (d *OutboundDispatcher) DispatchSubscribeNotification(ctx context.Context, target node.Node, body map[string]any) error {
	url, err := baseURL(target)
	if err != nil {
		return err
	}
	url += "/VIID/SubscribeNotifications"
	start := time.Now()
	cctx, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()
	resp, status, err := d.client.PostJSON(cctx, url, "", body)
	dur := time.Since(start)
	record(d, ctx, target, "POST", "/VIID/SubscribeNotifications", url, body, resp, status, dur, err)
	return err
}

// DispatchDisposition POSTs to /VIID/Dispositions.
func (d *OutboundDispatcher) DispatchDisposition(ctx context.Context, target node.Node, body map[string]any) error {
	url, err := baseURL(target)
	if err != nil {
		return err
	}
	url += "/VIID/Dispositions"
	start := time.Now()
	cctx, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()
	resp, status, err := d.client.PostJSON(cctx, url, "", body)
	dur := time.Since(start)
	record(d, ctx, target, "POST", "/VIID/Dispositions", url, body, resp, status, dur, err)
	return err
}

// Dispatch POSTs payload to /VIID/<Collection> on the target.
func (d *OutboundDispatcher) Dispatch(ctx context.Context, target node.Node, kind resource.Kind, payload map[string]any) error {
	base, err := baseURL(target)
	if err != nil {
		return err
	}
	coll := resource.CollectionOf(kind)
	if coll == "" {
		return fmt.Errorf("dispatcher: no collection for %s", kind)
	}
	url := base + "/VIID/" + coll
	deviceID := target.ID

	// Marshal the body once so we can both POST it and copy it into the
	// capture record.
	bodyBytes, _ := json.Marshal(payload)
	start := time.Now()
	cctx, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()

	resp, status, err := d.client.PostJSON(cctx, url, deviceID, json.RawMessage(bodyBytes))
	dur := time.Since(start)
	record(d, ctx, target, "POST", "/VIID/"+coll, url, payload, resp, status, dur, err)
	if err != nil {
		return err
	}
	return nil
}

// baseURL converts target.HTTPListen into a fully-qualified http:// base URL.
// Empty listen addresses default to "127.0.0.1" on the port side.
func baseURL(target node.Node) (string, error) {
	addr := target.HTTPListen
	if addr == "" {
		return "", fmt.Errorf("dispatcher: node %s has empty HTTPListen", target.ID)
	}
	// 完整 URL（如 Upstream 的 http://host:port）直接拆 scheme+host。
	if u, perr := url.Parse(addr); perr == nil && u.Host != "" {
		scheme := u.Scheme
		if scheme == "" {
			scheme = "http"
		}
		return scheme + "://" + u.Host, nil
	}
	host := "127.0.0.1"
	port := ""
	if addr[0] == ':' {
		port = addr[1:]
	} else {
		for i := len(addr) - 1; i >= 0; i-- {
			if addr[i] == ':' {
				host = addr[:i]
				port = addr[i+1:]
				break
			}
		}
	}
	if host == "" {
		host = "127.0.0.1"
	}
	return fmt.Sprintf("http://%s:%s", host, port), nil
}

// record 统一记录出站抓包数据（成功/失败均记录）。
func record(d *OutboundDispatcher, ctx context.Context, target node.Node, method, path, url string, reqBody any, resp map[string]any, status int, dur time.Duration, err error) {
	if d.recorder == nil {
		return
	}
	reqBytes, _ := json.Marshal(reqBody)
	respBytes, _ := json.Marshal(resp)
	entry := capture.Capture{
		NodeID:    target.ID,
		Direction: "outbound",
		Method:    method,
		Path:      path,
		URL:       url,
		Remote:    target.Upstream,
		Status:    status,
		StartedAt: time.Now().Add(-dur),
		Duration:  dur,
		Header: map[string][]string{
			"Content-Type":  {"application/VIID+JSON"},
			"User-Identify": {target.ID},
		},
		Request:  io.NopCloser(bytes.NewReader(reqBytes)),
		Response: bytes.NewReader(respBytes),
		Err:      err,
	}
	d.recorder.Record(ctx, entry)
}
