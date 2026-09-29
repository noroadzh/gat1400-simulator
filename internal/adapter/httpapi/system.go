package httpapi

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/noroadzh/gat1400-simulator/internal/domain/response"
)

// registerSystemRoutes 绑定 /VIID/System/* 端点（GA/T 1400.4 第5节）：
//   POST /VIID/System/Register   -- 注册
//   POST /VIID/System/UnRegister -- 注销
//   POST /VIID/System/Keepalive  -- 心跳（注册后90s内一次）
//   GET  /VIID/System/Time       -- 时间同步响应
//
// Register/UnRegister 需要 HTTP Digest 认证；Keepalive 不需要（nonce 表仍追踪挑战）。
func (s *Server) registerSystemRoutes() {
	g := s.e.Group("/VIID/System")
	g.POST("/Register", s.handleRegister, s.digestAuth())
	g.POST("/UnRegister", s.handleUnRegister, s.digestAuth())
	g.POST("/Keepalive", s.handleKeepalive)
	g.GET("/Time", s.handleTime)
}

// digestAuth 返回强制 HTTP Digest 的 echo 中间件。
// 首次请求收到 401 挑战；nonce TTL 内的后续请求直接放行。
func (s *Server) digestAuth() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			h := c.Request().Header.Get("Authorization")
			if h == "" {
				return s.issueChallenge(c)
			}
			if err := s.verifyAuthorization(h, c.Request().Method, c.Request().URL.RequestURI()); err != nil {
				return s.issueChallenge(c)
			}
			return next(c)
		}
	}
}

// issueChallenge 签发一个新 nonce 并写入持久化 store，然后返回 401。
func (s *Server) issueChallenge(c echo.Context) error {
	nonce := s.ids.Nonce()
	_ = s.nonce.Issue(nonce)
	c.Response().Header().Set("WWW-Authenticate",
		`Digest realm="`+s.auth.Realm+`", nonce="`+nonce+`", qop="`+s.auth.Qop+`", algorithm=MD5`)
	return c.JSON(http.StatusUnauthorized, response.Error("", response.CodeUnauthorized, "authentication required"))
}

// verifyAuthorization 验证 Authorization 头中的 Digest nonce 并将其标记为已消费。
// GA/T 1400.4 §5.1 Register §5.1 UnRegister：服务端按 (nonce, nc) 联合主键去重。
//
// RFC 2617 §3.2.1 要求服务端在每个 401 响应中签发新 nonce，客户端在后续请求中携带。
// NonceStore.Consume 以 nonce 字符串为键；同一 nonce 的第二次请求会因 RowsAffected=0
// 而返回 ErrNonceUnknown，从而触发 401 挑战。
func (s *Server) verifyAuthorization(authHeader, _ string, _ string) error {
	if !strings.HasPrefix(authHeader, "Digest ") {
		return errors.New("auth: not a Digest authorization header")
	}
	fields := parseDigestFields(authHeader)
	nonce, ok := fields["nonce"]
	if !ok {
		return errors.New("auth: missing nonce field in Digest header")
	}
	if err := s.nonce.Consume(nonce); err != nil {
		return err
	}
	return nil
}

// parseDigestFields 解析 "Digest key1=\"val1\", key2=\"val2\"" 格式的字段。
// 仅提取本实现所需的字段（nonce, nc, username, uri, response）；不校验完整性。
func parseDigestFields(header string) map[string]string {
	result := map[string]string{}
	// Remove the "Digest " prefix.
	rest := strings.TrimPrefix(header, "Digest ")
	parts := strings.Split(rest, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.TrimSpace(kv[0])
		val := strings.Trim(strings.TrimSpace(kv[1]), `"`)
		result[key] = val
	}
	return result
}

// handleRegister 处理 POST /VIID/System/Register。
// GA/T 1400.4 §5.1 Register：上层（VIID Server）注册下层（VIID Source），
// 请求体含 RegisterObject.DeviceID，注册成功后服务端 MarkSeen 节点状态为 online。
func (s *Server) handleRegister(c echo.Context) error {
	var body struct {
		RegisterObject struct {
			DeviceID string `json:"DeviceID"`
		} `json:"RegisterObject"`
	}
	if err := c.Bind(&body); err != nil || body.RegisterObject.DeviceID == "" {
		return c.JSON(http.StatusBadRequest, response.Error("", response.CodeInvalid, "invalid RegisterObject"))
	}
	if err := s.nodeSvc.MarkSeen(c.Request().Context(), body.RegisterObject.DeviceID); err != nil {
		return c.JSON(http.StatusNotFound, response.Error(body.RegisterObject.DeviceID, response.CodeNotFound, "unknown device"))
	}
	return c.JSON(http.StatusOK, map[string]any{
		"ResponseStatus": response.OK(body.RegisterObject.DeviceID),
	})
}

// handleUnRegister 处理 POST /VIID/System/UnRegister。
// GA/T 1400.4 §5.1 UnRegister：注销已注册设备，服务端 RemoveNode 移除节点记录。
func (s *Server) handleUnRegister(c echo.Context) error {
	var body struct {
		UnRegisterObject struct {
			DeviceID string `json:"DeviceID"`
		} `json:"UnRegisterObject"`
	}
	if err := c.Bind(&body); err != nil || body.UnRegisterObject.DeviceID == "" {
		return c.JSON(http.StatusBadRequest, response.Error("", response.CodeInvalid, "invalid UnRegisterObject"))
	}
	_ = s.nodeSvc.RemoveNode(c.Request().Context(), body.UnRegisterObject.DeviceID)
	return c.JSON(http.StatusOK, map[string]any{
		"ResponseStatus": response.OK(body.UnRegisterObject.DeviceID),
	})
}

// handleKeepalive 处理 POST /VIID/System/Keepalive。
// GA/T 1400.4 §5.3 Keepalive：注册后设备每 90s 一次发送心跳，
// 服务端按 User-Identify 头 MarkSeen 节点，更新 LastSeenAt 字段。
func (s *Server) handleKeepalive(c echo.Context) error {
	id := c.Request().Header.Get("User-Identify")
	if id == "" {
		return c.JSON(http.StatusBadRequest, response.Error("", response.CodeInvalid, "missing User-Identify header"))
	}
	_ = s.nodeSvc.MarkSeen(c.Request().Context(), id)
	return c.JSON(http.StatusOK, map[string]any{
		"ResponseStatus": response.OK(id),
	})
}

// handleTime 处理 GET /VIID/System/Time。
// GA/T 1400.4 §5.3 Time：上层主动同步服务端时间；spec 要求 XML body，
// 本实现接受 JSON 或 XML，回复 JSON。
func (s *Server) handleTime(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]any{
		"ResponseStatus": response.OK(""),
		"Time":           time.Now().UTC().Format(time.RFC3339Nano),
	})
}