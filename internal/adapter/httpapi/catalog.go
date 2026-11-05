package httpapi

import (
	"net/http"
	"sync"

	"github.com/labstack/echo/v4"

	"github.com/noroadzh/gat1400-simulator/internal/domain/response"
)

// catalogRepo 内存存储 APE / APS / Tollgate / Lane 目录记录。
// 支持 GET 列表 + POST 批量写入 + DELETE 按主键删除（动态编目）。
type catalogRepo struct {
	mu        sync.RWMutex
	apes      map[string]map[string]any
	apss      map[string]map[string]any
	tollgates map[string]map[string]any
	lanes     map[string]map[string]any
}

func newCatalogRepo() *catalogRepo {
	r := &catalogRepo{
		apes:      map[string]map[string]any{},
		apss:      map[string]map[string]any{},
		tollgates: map[string]map[string]any{},
		lanes:     map[string]map[string]any{},
	}
	// 种子数据：便于开箱即用的仪表盘浏览。
	r.apes["41000000005030312222"] = map[string]any{
		"APEID": "41000000005030312222", "APEName": "Sim-IPC-001", "Manufacturer": "SimCorp",
		"Model": "SIM-IPC", "Owner": "sim", "Area": `{"Province":"410000"}`,
		"IPAddress": "127.0.0.1", "Port": 14000,
	}
	r.apss["41000000005030313001"] = map[string]any{
		"APSID": "41000000005030313001", "APSName": "Sim-APS-001", "Manufacturer": "SimCorp",
		"Model": "SIM-APS", "IPAddress": "127.0.0.1", "Port": 14000,
	}
	r.tollgates["41000000005030314001"] = map[string]any{
		"TollgateID": "41000000005030314001", "TollgateName": "Sim-Tollgate-001",
		"Longitude": 116.397, "Latitude": 39.908,
	}
	r.lanes["41000000005030314101"] = map[string]any{
		"LaneID": "41000000005030314101", "LaneName": "Lane-1", "LaneNo": 1,
		"TollgateID": "41000000005030314001", "Direction": "East",
	}
	return r
}

func mapValues(m map[string]map[string]any) []map[string]any {
	out := make([]map[string]any, 0, len(m))
	for _, v := range m {
		out = append(out, v)
	}
	return out
}

func (r *catalogRepo) apesList() []map[string]any {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return mapValues(r.apes)
}
func (r *catalogRepo) apssList() []map[string]any {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return mapValues(r.apss)
}
func (r *catalogRepo) tollsList() []map[string]any {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return mapValues(r.tollgates)
}
func (r *catalogRepo) lanesList() []map[string]any {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return mapValues(r.lanes)
}

func (r *catalogRepo) upsertMany(bucket map[string]map[string]any, idField string, items []map[string]any) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := 0
	for _, item := range items {
		id, _ := item[idField].(string)
		if id == "" {
			continue
		}
		bucket[id] = item
		n++
	}
	return n
}

func (r *catalogRepo) deleteOne(bucket map[string]map[string]any, id string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := bucket[id]; !ok {
		return false
	}
	delete(bucket, id)
	return true
}

// registerCatalogRoutes 绑定目录路由（/VIID/APEs /APSs /Tollgates /Lanes）。
// GA/T 1400.4 §5.5 Catalog：支持列表、批量写入、按 ID 删除。
func (s *Server) registerCatalogRoutes() {
	s.e.GET("/VIID/APEs", s.handleAPEs)
	s.e.POST("/VIID/APEs", s.handleAPEsUpsert)
	s.e.PUT("/VIID/APEs", s.handleAPEsUpsert)
	s.e.DELETE("/VIID/APEs/:id", s.handleAPEDelete)

	s.e.GET("/VIID/APSs", s.handleAPSs)
	s.e.POST("/VIID/APSs", s.handleAPSsUpsert)
	s.e.DELETE("/VIID/APSs/:id", s.handleAPSDelete)

	s.e.GET("/VIID/Tollgates", s.handleTollgates)
	s.e.POST("/VIID/Tollgates", s.handleTollgatesUpsert)
	s.e.DELETE("/VIID/Tollgates/:id", s.handleTollgateDelete)

	s.e.GET("/VIID/Lanes", s.handleLanes)
	s.e.POST("/VIID/Lanes", s.handleLanesUpsert)
	s.e.DELETE("/VIID/Lanes/:id", s.handleLaneDelete)
}

func (s *Server) handleAPEs(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]any{
		"ResponseStatus": response.OK(""),
		"APEList":        map[string]any{"APEObject": s.catalog.apesList()},
	})
}

func (s *Server) handleAPSs(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]any{
		"ResponseStatus": response.OK(""),
		"APSList":        map[string]any{"APSObject": s.catalog.apssList()},
	})
}

func (s *Server) handleTollgates(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]any{
		"ResponseStatus": response.OK(""),
		"TollgateList":   map[string]any{"TollgateObject": s.catalog.tollsList()},
	})
}

func (s *Server) handleLanes(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]any{
		"ResponseStatus": response.OK(""),
		"LaneList":       map[string]any{"LaneObject": s.catalog.lanesList()},
	})
}

func extractCatalogObjects(body map[string]any, listKey, objKey string) []map[string]any {
	list, _ := body[listKey].(map[string]any)
	if list == nil {
		return nil
	}
	arr, _ := list[objKey].([]any)
	out := make([]map[string]any, 0, len(arr))
	for _, item := range arr {
		if m, ok := item.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

func (s *Server) handleAPEsUpsert(c echo.Context) error {
	var body map[string]any
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, response.Error("", response.CodeInvalid, "invalid body"))
	}
	n := s.catalog.upsertMany(s.catalog.apes, "APEID", extractCatalogObjects(body, "APEList", "APEObject"))
	return c.JSON(http.StatusOK, map[string]any{"ResponseStatus": response.OK(""), "ItemCount": n})
}

func (s *Server) handleAPSsUpsert(c echo.Context) error {
	var body map[string]any
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, response.Error("", response.CodeInvalid, "invalid body"))
	}
	n := s.catalog.upsertMany(s.catalog.apss, "APSID", extractCatalogObjects(body, "APSList", "APSObject"))
	return c.JSON(http.StatusOK, map[string]any{"ResponseStatus": response.OK(""), "ItemCount": n})
}

func (s *Server) handleTollgatesUpsert(c echo.Context) error {
	var body map[string]any
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, response.Error("", response.CodeInvalid, "invalid body"))
	}
	n := s.catalog.upsertMany(s.catalog.tollgates, "TollgateID", extractCatalogObjects(body, "TollgateList", "TollgateObject"))
	return c.JSON(http.StatusOK, map[string]any{"ResponseStatus": response.OK(""), "ItemCount": n})
}

func (s *Server) handleLanesUpsert(c echo.Context) error {
	var body map[string]any
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, response.Error("", response.CodeInvalid, "invalid body"))
	}
	n := s.catalog.upsertMany(s.catalog.lanes, "LaneID", extractCatalogObjects(body, "LaneList", "LaneObject"))
	return c.JSON(http.StatusOK, map[string]any{"ResponseStatus": response.OK(""), "ItemCount": n})
}

func (s *Server) handleAPEDelete(c echo.Context) error {
	id := c.Param("id")
	if !s.catalog.deleteOne(s.catalog.apes, id) {
		return c.JSON(http.StatusNotFound, response.Error(id, response.CodeNotFound, "not found"))
	}
	return c.JSON(http.StatusOK, map[string]any{"ResponseStatus": response.OK(id)})
}

func (s *Server) handleAPSDelete(c echo.Context) error {
	id := c.Param("id")
	if !s.catalog.deleteOne(s.catalog.apss, id) {
		return c.JSON(http.StatusNotFound, response.Error(id, response.CodeNotFound, "not found"))
	}
	return c.JSON(http.StatusOK, map[string]any{"ResponseStatus": response.OK(id)})
}

func (s *Server) handleTollgateDelete(c echo.Context) error {
	id := c.Param("id")
	if !s.catalog.deleteOne(s.catalog.tollgates, id) {
		return c.JSON(http.StatusNotFound, response.Error(id, response.CodeNotFound, "not found"))
	}
	return c.JSON(http.StatusOK, map[string]any{"ResponseStatus": response.OK(id)})
}

func (s *Server) handleLaneDelete(c echo.Context) error {
	id := c.Param("id")
	if !s.catalog.deleteOne(s.catalog.lanes, id) {
		return c.JSON(http.StatusNotFound, response.Error(id, response.CodeNotFound, "not found"))
	}
	return c.JSON(http.StatusOK, map[string]any{"ResponseStatus": response.OK(id)})
}
