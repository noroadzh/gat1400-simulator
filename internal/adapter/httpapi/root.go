package httpapi

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/noroadzh/gat1400-simulator/internal/domain/resource"
	"github.com/noroadzh/gat1400-simulator/internal/domain/response"
)

// registerRootRoutes 注册表 13 顶级与系统属性资源。
//
//	GET /VIID         — 顶级资源：返回本模拟器已挂载的资源 URI 目录
//	GET /VIID/System  — 系统属性资源：版本 / 能力摘要
func (s *Server) registerRootRoutes() {
	s.e.GET("/VIID", s.handleVIIDRoot)
	s.e.GET("/VIID/System", s.handleSystemRoot)
}

func (s *Server) handleVIIDRoot(c echo.Context) error {
	uris := []string{
		"/VIID/System",
		"/VIID/System/Register",
		"/VIID/System/UnRegister",
		"/VIID/System/Keepalive",
		"/VIID/System/Time",
		"/VIID/VIIDServers",
		"/VIID/APEs",
		"/VIID/APSs",
		"/VIID/Tollgates",
		"/VIID/Lanes",
		"/VIID/Subscribes",
		"/VIID/SubscribeNotifications",
		"/VIID/Dispositions",
		"/VIID/DispositionNotifications",
	}
	for _, k := range resource.AllKinds {
		if coll := resource.CollectionOf(k); coll != "" {
			uris = append(uris, "/VIID/"+coll)
		}
	}
	return c.JSON(http.StatusOK, map[string]any{
		"ResponseStatus": response.OK(""),
		"ResourceURIList": map[string]any{
			"ResourceURI": uris,
		},
	})
}

func (s *Server) handleSystemRoot(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]any{
		"ResponseStatus": response.OK(""),
		"System": map[string]any{
			"Name":           "gat1400-simulator",
			"Version":        "0.2.0",
			"Protocol":       "GA/T 1400.4-2017",
			"ContentType":    "application/VIID+JSON",
			"ServerTime":     time.Now().UTC().Format(time.RFC3339),
			"Capabilities":   []string{"system", "collection", "cascade", "catalog", "vias"},
			"VIIDServerCount": len(s.servers.list()),
		},
	})
}
