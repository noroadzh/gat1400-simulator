package httpapi

import (
	"net/http"
	"sync"

	"github.com/labstack/echo/v4"

	"github.com/noroadzh/gat1400-simulator/internal/domain/response"
)

// viidServerRepo 内存存储联网服务（表 13 `/VIID/VIIDServers`）。
type viidServerRepo struct {
	mu   sync.RWMutex
	data map[string]map[string]any
}

func newVIIDServerRepo() *viidServerRepo {
	r := &viidServerRepo{data: map[string]map[string]any{}}
	// 种子：本模拟器自身
	r.data["41000000005030310001"] = map[string]any{
		"VIIDServerID":   "41000000005030310001",
		"ServerName":     "gat1400-simulator-local",
		"IPAddr":         "127.0.0.1",
		"Port":           14000,
		"IsOnline":       true,
		"OwnerApsID":     "41000000005030313001",
	}
	return r
}

func (r *viidServerRepo) list() []map[string]any {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]map[string]any, 0, len(r.data))
	for _, v := range r.data {
		out = append(out, v)
	}
	return out
}

func (r *viidServerRepo) get(id string) (map[string]any, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	v, ok := r.data[id]
	return v, ok
}

func (r *viidServerRepo) upsert(id string, obj map[string]any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.data[id] = obj
}

func (r *viidServerRepo) delete(id string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.data[id]; !ok {
		return false
	}
	delete(r.data, id)
	return true
}

func (s *Server) registerVIIDServerRoutes() {
	s.e.GET("/VIID/VIIDServers", s.handleVIIDServersList)
	s.e.POST("/VIID/VIIDServers", s.handleVIIDServersUpsert)
	s.e.PUT("/VIID/VIIDServers", s.handleVIIDServersUpsert)
	s.e.GET("/VIID/VIIDServers/:id", s.handleVIIDServerGet)
	s.e.PUT("/VIID/VIIDServers/:id", s.handleVIIDServerPut)
	s.e.DELETE("/VIID/VIIDServers/:id", s.handleVIIDServerDelete)
}

func (s *Server) handleVIIDServersList(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]any{
		"ResponseStatus": response.OK(""),
		"VIIDServerList": map[string]any{"VIIDServerObject": s.servers.list()},
	})
}

func (s *Server) handleVIIDServersUpsert(c echo.Context) error {
	var body map[string]any
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, response.Error("", response.CodeInvalid, "invalid body"))
	}
	list, _ := body["VIIDServerList"].(map[string]any)
	arr, _ := list["VIIDServerObject"].([]any)
	n := 0
	for _, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		id, _ := m["VIIDServerID"].(string)
		if id == "" {
			id = s.ids.DeviceID()
			m["VIIDServerID"] = id
		}
		s.servers.upsert(id, m)
		s.persistVIIDServer(id, m)
		n++
	}
	return c.JSON(http.StatusOK, map[string]any{"ResponseStatus": response.OK(""), "ItemCount": n})
}

func (s *Server) handleVIIDServerGet(c echo.Context) error {
	id := c.Param("id")
	obj, ok := s.servers.get(id)
	if !ok {
		return c.JSON(http.StatusNotFound, response.Error(id, response.CodeNotFound, "not found"))
	}
	return c.JSON(http.StatusOK, map[string]any{
		"ResponseStatus": response.OK(id),
		"VIIDServer":     obj,
	})
}

func (s *Server) handleVIIDServerPut(c echo.Context) error {
	id := c.Param("id")
	var body map[string]any
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, response.Error(id, response.CodeInvalid, "invalid body"))
	}
	if body == nil {
		body = map[string]any{}
	}
	body["VIIDServerID"] = id
	s.servers.upsert(id, body)
	s.persistVIIDServer(id, body)
	return c.JSON(http.StatusOK, map[string]any{"ResponseStatus": response.OK(id)})
}

func (s *Server) handleVIIDServerDelete(c echo.Context) error {
	id := c.Param("id")
	if !s.servers.delete(id) {
		return c.JSON(http.StatusNotFound, response.Error(id, response.CodeNotFound, "not found"))
	}
	s.persistVIIDServerDelete(id)
	return c.JSON(http.StatusOK, map[string]any{"ResponseStatus": response.OK(id)})
}
