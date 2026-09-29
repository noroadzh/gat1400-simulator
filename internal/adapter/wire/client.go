// Package wire 实现场景引擎与同进程分发器使用的出站 HTTP 客户端。
//
// 主要职责：
//   - 自动处理 HTTP Digest 认证（RFC 2617，支持 qop=auth）
//   - 401 挑战时自动重试一次
//   - 在每个请求上设置 GAT 1400.4 协议要求的 User-Identify 头
//   - 统一 Content-Type 为 application/VIID+JSON
package wire

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/noroadzh/gat1400-simulator/internal/adapter/storage"
)

// Client 应用层使用的高层外观。所有 wire HTTP 调用都通过它进行。
type Client struct {
	httpClient *http.Client
	log        *slog.Logger
	nonce      *storage.NonceStore

	mu        sync.Mutex
	username  string
	password  string
	realm     string
	qop       string
}

// Options Client 运行时配置项。
type Options struct {
	Username string
	Password string
	Realm    string
	Qop      string
	Timeout  time.Duration
}

// NewClient 构造一个绑定到指定 nonce store 的客户端。
//
// nonce store 在以下两种用途下都会用到：
//   - 作为 client 时校验服务端签发的挑战（nonce 持久化保证进程重启不丢）
//   - 作为 server 时保存 digest challenge（实现 server-mode 触发）
//
// 两种行为对调用方透明。
func NewClient(log *slog.Logger, nonce *storage.NonceStore) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		log:        log,
		nonce:      nonce,
		username:   "admin",
		password:   "admin",
		realm:      "com.gat1400.simulator",
		qop:        "auth",
	}
}

// Configure 运行时更新 digest 凭据。
//
// 不会清空已签发 nonce 表中未过期的 nonce（这些已交给持久层管理），
// 仅影响后续 challenge 的应答计算。
func (c *Client) Configure(opts Options) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if opts.Username != "" {
		c.username = opts.Username
	}
	if opts.Password != "" {
		c.password = opts.Password
	}
	if opts.Realm != "" {
		c.realm = opts.Realm
	}
	if opts.Qop != "" {
		c.qop = opts.Qop
	}
	if opts.Timeout > 0 {
		c.httpClient.Timeout = opts.Timeout
	}
}

// PostJSON 发送 POST 请求（JSON body），返回解析后的 JSON 响应。
//
// 收到 401 挑战时会自动完成 Digest 握手并重试一次。
// body 必须可 JSON 编码；Content-Type 固定为 application/VIID+JSON 以满足 GA/T 1400.4 服务端实现。
func (c *Client) PostJSON(ctx context.Context, url, deviceID string, body any) (map[string]any, int, error) {
	raw, err := encodeJSON(body)
	if err != nil {
		return nil, 0, err
	}
	status, payload, err := c.do(ctx, http.MethodPost, url, deviceID, raw)
	if err != nil {
		return nil, status, err
	}
	return decodeJSONMap(payload), status, nil
}

// GetJSON 发送 GET 请求，返回解析后的 JSON 响应。
func (c *Client) GetJSON(ctx context.Context, url, deviceID string) (map[string]any, int, error) {
	status, payload, err := c.do(ctx, http.MethodGet, url, deviceID, nil)
	if err != nil {
		return nil, status, err
	}
	return decodeJSONMap(payload), status, nil
}

// do 内部助手：负责 Digest 握手与重试逻辑。
// RFC 2617 §3：构造请求 → 发送 → 若 401 则解析挑战 → 重构 Authorization 头 → 重试一次。
// 重试仅限一次，避免服务端故障下死循环。
func (c *Client) do(ctx context.Context, method, url, deviceID string, body []byte) (int, []byte, error) {
	req, err := c.buildRequest(ctx, method, url, deviceID, body)
	if err != nil {
		return 0, nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		// Digest challenge: build an Authorization header and retry once.
		challenge, ok := parseChallenge(resp.Header.Get("WWW-Authenticate"))
		if !ok {
			return resp.StatusCode, nil, errors.New("wire: 401 without Digest challenge")
		}
		auth, err := c.buildAuthorization(challenge, method, url, body)
		if err != nil {
			return resp.StatusCode, nil, err
		}
		req, err := c.buildRequest(ctx, method, url, deviceID, body)
		if err != nil {
			return resp.StatusCode, nil, err
		}
		req.Header.Set("Authorization", auth)
		resp, err = c.httpClient.Do(req)
		if err != nil {
			return 0, nil, err
		}
		defer resp.Body.Close()
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, err
	}
	return resp.StatusCode, respBody, nil
}

// buildRequest 构造标准 HTTP 请求：自动注入 User-Identify 头、Content-Type、Accept。
// RFC 2617 §3.2：GA/T 1400.4 要求 User-Identify 标识请求设备；Content-Type 固定 application/VIID+JSON。
func (c *Client) buildRequest(ctx context.Context, method, url, deviceID string, body []byte) (*http.Request, error) {
	var reader io.Reader
	if len(body) > 0 {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		return nil, err
	}
	if deviceID != "" {
		req.Header.Set("User-Identify", deviceID)
	}
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/VIID+JSON")
	}
	req.Header.Set("Accept", "application/VIID+JSON, application/json")
	return req, nil
}

// challenge 从 WWW-Authenticate: Digest 头解析出的字段集合。
type challenge struct {
	realm     string
	nonce     string
	algorithm string
	qop       string
	opaque    string
}

var (
	digestKV = regexp.MustCompile(`([a-zA-Z][a-zA-Z0-9_-]*)\s*=\s*("([^"]*)"|([^,]*))`)
)

// parseChallenge 从 WWW-Authenticate 头解析 RFC 2617 §3.2.1 字段。
//
//   - 引号包裹的值会被去掉引号
//   - algorithm 缺省为 MD5（RFC 默认）
//   - realm 或 nonce 任一为空都判定为解析失败
func parseChallenge(header string) (challenge, bool) {
	if !strings.HasPrefix(header, "Digest ") {
		return challenge{}, false
	}
	c := challenge{algorithm: "MD5"}
	for _, m := range digestKV.FindAllStringSubmatch(header, -1) {
		key := strings.ToLower(m[1])
		val := m[3]
		if val == "" {
			val = m[4]
		}
		val = strings.TrimSpace(val)
		switch key {
		case "realm":
			c.realm = val
		case "nonce":
			c.nonce = val
		case "algorithm":
			c.algorithm = val
		case "qop":
			c.qop = val
		case "opaque":
			c.opaque = val
		}
	}
	if c.realm == "" || c.nonce == "" {
		return challenge{}, false
	}
	return c, true
}

// buildAuthorization 构造 Digest Authorization 头的值。
// RFC 2617 §3.2.2.1：
//   - HA1 = MD5(username:realm:password)
//   - HA2 = MD5(method:uri)
//   - response = MD5(HA1:nonce:nc:cnonce:qop:HA2)（qop 模式）
// cnonce 使用 crypto/rand 生成，禁止使用 time.Now() 等可预测源。
func (c *Client) buildAuthorization(ch challenge, method, url string, body []byte) (string, error) {
	c.mu.Lock()
	username := c.username
	password := c.password
	preferredQop := c.qop
	c.mu.Unlock()

	uri := url
	if idx := strings.Index(uri, "://"); idx > 0 {
		rest := uri[idx+3:]
		if slash := strings.Index(rest, "/"); slash >= 0 {
			uri = rest[slash:]
		}
	}

	ha1 := md5hex(username + ":" + ch.realm + ":" + password)
	ha2 := md5hex(method + ":" + uri)
	nc := "00000001"
	cnonce := randomHex(16)

	qop := ch.qop
	if qop == "" {
		qop = preferredQop
	}

	var response string
	if qop != "" {
		response = md5hex(strings.Join([]string{ha1, ch.nonce, nc, cnonce, qop, ha2}, ":"))
	} else {
		response = md5hex(strings.Join([]string{ha1, ch.nonce, ha2}, ":"))
	}

	var b strings.Builder
	fmt.Fprintf(&b, `Digest username="%s", realm="%s", nonce="%s", uri="%s", response="%s", algorithm=%s, nc=%s, cnonce="%s"`,
		username, ch.realm, ch.nonce, uri, response, ch.algorithm, nc, cnonce)
	if qop != "" {
		fmt.Fprintf(&b, `, qop=%s`, qop)
	}
	if ch.opaque != "" {
		fmt.Fprintf(&b, `, opaque="%s"`, ch.opaque)
	}
	return b.String(), nil
}

// helpers ---------------------------------------------------------------

// md5hex 计算字符串的 MD5 并以小写 hex 返回。
func md5hex(s string) string {
	h := md5.Sum([]byte(s))
	return hex.EncodeToString(h[:])
}

// randomHex 从 crypto/rand 读取 n/2 字节并以 hex 编码返回（结果为 n 个 hex 字符）。
//
// 严禁使用 time.Now() 等可预测源——cnonce 一旦可预测，digest 重放保护失效。
// 若 r / 熵源读取失败（极罕见），返回全零字符串并由调用方决定是否重试。
func randomHex(n int) string {
	if n <= 0 || n%2 != 0 {
		return ""
	}
	buf := make([]byte, n/2)
	if _, err := rand.Read(buf); err != nil {
		return strings.Repeat("0", n)
	}
	return hex.EncodeToString(buf)
}

func encodeJSON(v any) ([]byte, error) {
	if v == nil {
		return nil, nil
	}
	return jsonMarshal(v)
}

// decodeJSONMap 把字节流解析为 map。失败时返回 {"_raw": "<原文本>"}，
// 这样上游 BFF 仍能展示非 JSON 响应（如 HTML 错误页）。
func decodeJSONMap(b []byte) map[string]any {
	if len(b) == 0 {
		return map[string]any{}
	}
	var m map[string]any
	if err := jsonUnmarshal(b, &m); err == nil {
		return m
	}
	return map[string]any{"_raw": string(b)}
}