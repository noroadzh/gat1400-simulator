package httpapi

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/noroadzh/gat1400-simulator/internal/domain/response"
)

// catalogRepo 内存存储 APE / APS / Tollgate / Lane 目录记录。
// 模拟器用注册节点预填充，仪表盘可直接浏览。
type catalogRepo struct {
	apes      []map[string]any
	apss      []map[string]any
	tollgates []map[string]any
	lanes     []map[string]any
}

func newCatalogRepo() *catalogRepo {
	return &catalogRepo{
		apes: []map[string]any{
			{"APEID": "41000000005030312222", "APEName": "Sim-IPC-001", "Manufacturer": "SimCorp", "Model": "SIM-IPC", "Owner": "sim", "Area": "{\"Province\":\"410000\"}", "IPAddress": "127.0.0.1", "Port": 14000},
		},
		apss: []map[string]any{
			{"APSID": "41000000005030313001", "APSName": "Sim-APS-001", "Manufacturer": "SimCorp", "Model": "SIM-APS", "IPAddress": "127.0.0.1", "Port": 14000},
		},
		tollgates: []map[string]any{
			{"TollgateID": "41000000005030314001", "TollgateName": "Sim-Tollgate-001", "Longitude": 116.397, "Latitude": 39.908},
		},
		lanes: []map[string]any{
			{"LaneID": "41000000005030314101", "LaneName": "Lane-1", "LaneNo": 1, "TollgateID": "41000000005030314001", "Direction": "East"},
		},
	}
}

func (r *catalogRepo) apesList() []map[string]any  { return r.apes }
func (r *catalogRepo) apssList() []map[string]any  { return r.apss }
func (r *catalogRepo) tollsList() []map[string]any { return r.tollgates }
func (r *catalogRepo) lanesList() []map[string]any { return r.lanes }

// registerCatalogRoutes 绑定目录只读路由（/VIID/APEs /APSs /Tollgates /Lanes）。
// GA/T 1400.4 §5.5 Catalog：APEs/APSs/Tollgates/Lanes 四类设备编目树，
// 用于浏览已注册前端/平台/卡口/车道节点。
func (s *Server) registerCatalogRoutes() {
	s.e.GET("/VIID/APEs", s.handleAPEs)
	s.e.PUT("/VIID/APEs", s.handleAPEs)
	s.e.GET("/VIID/APSs", s.handleAPSs)
	s.e.GET("/VIID/Tollgates", s.handleTollgates)
	s.e.GET("/VIID/Lanes", s.handleLanes)
}

// handleAPEs 处理 GET /VIID/APEs。
// GA/T 1400.4 §5.5 APE（前端设备）目录：响应 APEList 包裹 APEObject 数组。
func (s *Server) handleAPEs(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]any{
		"ResponseStatus": response.OK(""),
		"APEList":        map[string]any{"APEObject": s.catalog.apesList()},
	})
}

// handleAPSs 处理 GET /VIID/APSs。
// GA/T 1400.4 §5.5 APS（应用平台）目录。
func (s *Server) handleAPSs(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]any{
		"ResponseStatus": response.OK(""),
		"APSList":        map[string]any{"APSObject": s.catalog.apssList()},
	})
}

// handleTollgates 处理 GET /VIID/Tollgates。
// GA/T 1400.4 §5.5 Tollgate（卡口）目录。
func (s *Server) handleTollgates(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]any{
		"ResponseStatus":  response.OK(""),
		"TollgateList":   map[string]any{"TollgateObject": s.catalog.tollsList()},
	})
}

// handleLanes 处理 GET /VIID/Lanes。
// GA/T 1400.4 §5.5 Lane（车道）目录，按 TollgateID 归属。
func (s *Server) handleLanes(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]any{
		"ResponseStatus": response.OK(""),
		"LaneList":       map[string]any{"LaneObject": s.catalog.lanesList()},
	})
}