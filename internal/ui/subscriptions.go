package ui

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

// installSubscriptionRoutes 注册订阅 / 布控控制面端点，透传协议端 Cascade 路由。
//
//	GET /api/control/subscriptions              → GET /VIID/Subscribes
//	GET /api/control/dispositions               → GET /VIID/Dispositions
//	GET /api/control/subscribe-notifications    → GET /VIID/SubscribeNotifications
//	GET /api/control/disposition-notifications  → GET /VIID/DispositionNotifications
func (s *Server) installSubscriptionRoutes(api *echo.Group) {
	api.GET("/subscriptions", s.handleProxyVIID("/VIID/Subscribes"))
	api.GET("/dispositions", s.handleProxyVIID("/VIID/Dispositions"))
	api.GET("/subscribe-notifications", s.handleProxyVIID("/VIID/SubscribeNotifications"))
	api.GET("/disposition-notifications", s.handleProxyVIID("/VIID/DispositionNotifications"))
}

func (s *Server) handleProxyVIID(upstreamPath string) echo.HandlerFunc {
	return func(c echo.Context) error {
		start := time.Now()
		if s.protocolClient == nil || s.protocolClient.baseURL == "" {
			return c.JSON(http.StatusBadGateway, map[string]any{
				"error": "protocol base URL not configured",
			})
		}
		uri := s.protocolClient.baseURL + upstreamPath
		if raw := c.Request().URL.RawQuery; raw != "" {
			uri += "?" + raw
		}
		req, err := http.NewRequestWithContext(c.Request().Context(), http.MethodGet, uri, nil)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]any{"error": err.Error()})
		}
		resp, err := s.protocolClient.http.Do(req)
		if err != nil {
			s.log.Warn("bff.cascade.upstream_failed",
				slog.String("path", upstreamPath),
				slog.String("error", err.Error()))
			return c.JSON(http.StatusBadGateway, map[string]any{
				"error": "protocol server unreachable: " + err.Error(),
			})
		}
		defer func() { _ = resp.Body.Close() }()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return c.JSON(http.StatusBadGateway, map[string]any{"error": err.Error()})
		}

		// 规范化为控制面友好结构：{ items: [...] }
		var raw map[string]any
		if err := json.Unmarshal(body, &raw); err != nil {
			c.Response().WriteHeader(resp.StatusCode)
			_, _ = c.Response().Write(body)
			return nil
		}
		items := extractCascadeItems(raw)
		s.log.Debug("bff.cascade.proxy",
			slog.String("path", upstreamPath),
			slog.Int("count", len(items)),
			slog.Duration("dur", time.Since(start)))
		return c.JSON(http.StatusOK, map[string]any{
			"items":          items,
			"ResponseStatus": raw["ResponseStatus"],
		})
	}
}

func extractCascadeItems(raw map[string]any) []any {
	candidates := []struct {
		listKey string
		objKey  string
	}{
		{"SubscribeList", "SubscribeObject"},
		{"DispositionList", "DispositionObject"},
		{"SubscribeNotificationList", "SubscribeNotificationObject"},
		{"DispositionNotificationList", "DispositionNotificationObject"},
	}
	for _, c := range candidates {
		if list, ok := raw[c.listKey].(map[string]any); ok {
			if arr, ok := list[c.objKey].([]any); ok {
				return arr
			}
		}
	}
	return []any{}
}
