package ui

import (
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

// installProtocolExtraRoutes 注册 Catalog / VIIDServers / VIAS / Cascade 写操作透传。
func (s *Server) installProtocolExtraRoutes(api *echo.Group) {
	// Catalog
	for _, coll := range []string{"APEs", "APSs", "Tollgates", "Lanes"} {
		c := coll
		api.GET("/catalog/"+c, s.handleRawProxy("/VIID/"+c))
		api.POST("/catalog/"+c, s.handleRawProxy("/VIID/"+c))
		api.PUT("/catalog/"+c, s.handleRawProxy("/VIID/"+c))
		api.GET("/catalog/"+c+"/:id", s.handleRawProxyParam("/VIID/"+c))
		api.DELETE("/catalog/"+c+"/:id", s.handleRawProxyParam("/VIID/"+c))
	}

	// VIIDServers
	api.GET("/viid-servers", s.handleRawProxy("/VIID/VIIDServers"))
	api.POST("/viid-servers", s.handleRawProxy("/VIID/VIIDServers"))
	api.PUT("/viid-servers", s.handleRawProxy("/VIID/VIIDServers"))
	api.GET("/viid-servers/:id", s.handleRawProxyParam("/VIID/VIIDServers"))
	api.PUT("/viid-servers/:id", s.handleRawProxyParam("/VIID/VIIDServers"))
	api.DELETE("/viid-servers/:id", s.handleRawProxyParam("/VIID/VIIDServers"))

	// VIAS 子集
	api.GET("/vias", s.handleRawProxy("/VIAS"))
	api.GET("/vias/tasks", s.handleRawProxy("/VIAS/Tasks"))
	api.POST("/vias/tasks", s.handleRawProxy("/VIAS/Tasks"))
	api.GET("/vias/tasks/:id", s.handleRawProxyParam("/VIAS/Tasks"))
	api.PUT("/vias/tasks/:id", s.handleRawProxyParam("/VIAS/Tasks"))
	api.DELETE("/vias/tasks/:id", s.handleRawProxyParam("/VIAS/Tasks"))
	api.GET("/vias/task-statuses", s.handleRawProxy("/VIAS/TaskStatuses"))
	api.GET("/vias/task-statuses/:id", s.handleRawProxyParam("/VIAS/TaskStatuses"))
	api.GET("/vias/task-controls", s.handleRawProxy("/VIAS/TaskControls"))
	api.POST("/vias/task-controls", s.handleRawProxy("/VIAS/TaskControls"))
	api.GET("/vias/analysis-rules", s.handleRawProxy("/VIAS/AnalysisRules"))
	api.POST("/vias/analysis-rules", s.handleRawProxy("/VIAS/AnalysisRules"))
	api.GET("/vias/analysis-rules/:id", s.handleRawProxyParam("/VIAS/AnalysisRules"))
	api.DELETE("/vias/analysis-rules/:id", s.handleRawProxyParam("/VIAS/AnalysisRules"))
	api.GET("/vias/capability/:name", func(c echo.Context) error {
		name := c.Param("name")
		return s.handleRawProxy("/VIAS/SystemCapability/"+name)(c)
	})
}

// installSubscriptionRoutes 扩展：读 + 写 Cascade。
func (s *Server) installSubscriptionRoutes(api *echo.Group) {
	api.GET("/subscriptions", s.handleProxyVIID("/VIID/Subscribes"))
	api.POST("/subscriptions", s.handleRawProxy("/VIID/Subscribes"))
	api.PUT("/subscriptions", s.handleRawProxy("/VIID/Subscribes"))
	api.DELETE("/subscriptions", s.handleRawProxy("/VIID/Subscribes"))
	api.PUT("/subscriptions/:id", s.handleRawProxyParam("/VIID/Subscribes"))
	api.DELETE("/subscriptions/:id", s.handleRawProxyParam("/VIID/Subscribes"))

	api.GET("/dispositions", s.handleProxyVIID("/VIID/Dispositions"))
	api.POST("/dispositions", s.handleRawProxy("/VIID/Dispositions"))
	api.PUT("/dispositions", s.handleRawProxy("/VIID/Dispositions"))
	api.DELETE("/dispositions", s.handleRawProxy("/VIID/Dispositions"))
	api.PUT("/dispositions/:id", s.handleRawProxyParam("/VIID/Dispositions"))
	api.DELETE("/dispositions/:id", s.handleRawProxyParam("/VIID/Dispositions"))

	api.GET("/subscribe-notifications", s.handleProxyVIID("/VIID/SubscribeNotifications"))
	api.POST("/subscribe-notifications", s.handleRawProxy("/VIID/SubscribeNotifications"))
	api.DELETE("/subscribe-notifications", s.handleRawProxy("/VIID/SubscribeNotifications"))
	api.GET("/subscribe-notifications/:id", s.handleRawProxyParam("/VIID/SubscribeNotifications"))
	api.DELETE("/subscribe-notifications/:id", s.handleRawProxyParam("/VIID/SubscribeNotifications"))

	api.GET("/disposition-notifications", s.handleProxyVIID("/VIID/DispositionNotifications"))
	api.POST("/disposition-notifications", s.handleRawProxy("/VIID/DispositionNotifications"))
	api.DELETE("/disposition-notifications", s.handleRawProxy("/VIID/DispositionNotifications"))
	api.GET("/disposition-notifications/:id", s.handleRawProxyParam("/VIID/DispositionNotifications"))
	api.DELETE("/disposition-notifications/:id", s.handleRawProxyParam("/VIID/DispositionNotifications"))
}

// handleRawProxy 透传固定上游路径（方法与 query/body 原样转发）。
func (s *Server) handleRawProxy(upstreamPath string) echo.HandlerFunc {
	return func(c echo.Context) error {
		return s.proxyUpstream(c, upstreamPath)
	}
}

// handleRawProxyParam 透传上游路径 + /:id。
func (s *Server) handleRawProxyParam(upstreamBase string) echo.HandlerFunc {
	return func(c echo.Context) error {
		id := c.Param("id")
		return s.proxyUpstream(c, strings.TrimRight(upstreamBase, "/")+"/"+id)
	}
}

func (s *Server) proxyUpstream(c echo.Context, upstreamPath string) error {
	start := time.Now()
	if s.protocolClient == nil || s.protocolClient.baseURL == "" {
		return c.JSON(http.StatusBadGateway, map[string]any{"error": "protocol base URL not configured"})
	}
	uri := s.protocolClient.baseURL + upstreamPath
	if raw := c.Request().URL.RawQuery; raw != "" {
		uri += "?" + raw
	}
	var body io.Reader
	if c.Request().Body != nil && c.Request().Body != http.NoBody {
		body = c.Request().Body
	}
	req, err := http.NewRequestWithContext(c.Request().Context(), c.Request().Method, uri, body)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": err.Error()})
	}
	if ct := c.Request().Header.Get("Content-Type"); ct != "" {
		req.Header.Set("Content-Type", ct)
	} else if body != nil {
		req.Header.Set("Content-Type", "application/VIID+JSON")
	}
	req.Header.Set("Accept", "application/VIID+JSON")
	resp, err := s.protocolClient.http.Do(req)
	if err != nil {
		return c.JSON(http.StatusBadGateway, map[string]any{"error": "protocol server unreachable: " + err.Error()})
	}
	defer func() { _ = resp.Body.Close() }()
	out, err := io.ReadAll(resp.Body)
	if err != nil {
		return c.JSON(http.StatusBadGateway, map[string]any{"error": err.Error()})
	}
	s.log.Debug("bff.proxy", "path", upstreamPath, "status", resp.StatusCode, "dur", time.Since(start))
	ct := resp.Header.Get("Content-Type")
	if ct == "" {
		ct = "application/json"
	}
	return c.Blob(resp.StatusCode, ct, out)
}
