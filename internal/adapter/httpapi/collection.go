package httpapi

import (
	"net/http"
	"sync"

	"github.com/labstack/echo/v4"

	"github.com/noroadzh/gat1400-simulator/internal/app/ports"
	"github.com/noroadzh/gat1400-simulator/internal/domain/resource"
	"github.com/noroadzh/gat1400-simulator/internal/domain/response"
)

// resourceRepo 内存内 key/value 存储，存放各 Kind 的资源实体。
// HTTP API 读写此处；data-service API 从此查询。
// 数据结构：Kind -> ResourceID -> 原始 payload map
// info/data 分别存放 §5.2 Info / Data 子资源，与父对象解耦。
type resourceRepo struct {
	mu   sync.RWMutex
	data map[resource.Kind]map[string]map[string]any
	info map[resource.Kind]map[string]map[string]any
	blob map[resource.Kind]map[string]map[string]any
}

func newResourceRepo() *resourceRepo {
	r := &resourceRepo{
		data: map[resource.Kind]map[string]map[string]any{},
		info: map[resource.Kind]map[string]map[string]any{},
		blob: map[resource.Kind]map[string]map[string]any{},
	}
	for _, k := range resource.AllKinds {
		r.data[k] = map[string]map[string]any{}
		r.info[k] = map[string]map[string]any{}
		r.blob[k] = map[string]map[string]any{}
	}
	return r
}

func (r *resourceRepo) store(kind resource.Kind, id string, payload map[string]any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.data[kind][id] = payload
}

func (r *resourceRepo) get(kind resource.Kind, id string) (map[string]any, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	v, ok := r.data[kind][id]
	return v, ok
}

func (r *resourceRepo) list(kind resource.Kind) []map[string]any {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]map[string]any, 0, len(r.data[kind]))
	for _, v := range r.data[kind] {
		out = append(out, v)
	}
	return out
}

func (r *resourceRepo) getInfo(kind resource.Kind, id string) (map[string]any, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	v, ok := r.info[kind][id]
	return v, ok
}

func (r *resourceRepo) putInfo(kind resource.Kind, id string, payload map[string]any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.info[kind][id] = payload
}

func (r *resourceRepo) deleteInfo(kind resource.Kind, id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.info[kind], id)
}

func (r *resourceRepo) getData(kind resource.Kind, id string) (map[string]any, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	v, ok := r.blob[kind][id]
	return v, ok
}

func (r *resourceRepo) putData(kind resource.Kind, id string, payload map[string]any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.blob[kind][id] = payload
}

func (r *resourceRepo) deleteData(kind resource.Kind, id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.blob[kind], id)
}

// registerCollectionRoutes 为每种 Kind 绑定 POST /VIID/<Collection>。
// 无 URI 映射的 Kind（kind.CollectionOf() 返回空）被跳过。
func (s *Server) registerCollectionRoutes() {
	for _, kind := range resource.AllKinds {
		coll := resource.CollectionOf(kind)
		if coll == "" {
			continue
		}
		s.e.POST("/VIID/"+coll, s.makeCollectionHandler(kind))
	}
}

// registerDataServiceRoutes 绑定 GET/PUT/DELETE 操作，以及 /{ID}/Info、/{ID}/Data 子资源。
func (s *Server) registerDataServiceRoutes() {
	for _, kind := range resource.AllKinds {
		coll := resource.CollectionOf(kind)
		if coll == "" {
			continue
		}
		// List
		s.e.GET("/VIID/"+coll, s.makeDataListHandler(kind))
		// Single
		s.e.GET("/VIID/"+coll+"/:id", s.makeDataGetHandler(kind))
		s.e.PUT("/VIID/"+coll+"/:id", s.makeDataPutHandler(kind))
		s.e.DELETE("/VIID/"+coll+"/:id", s.makeDataDeleteHandler(kind))
		// Info subresource
		s.e.GET("/VIID/"+coll+"/:id/Info", s.makeInfoGetHandler(kind))
		s.e.PUT("/VIID/"+coll+"/:id/Info", s.makeInfoPutHandler(kind))
		s.e.DELETE("/VIID/"+coll+"/:id/Info", s.makeInfoDeleteHandler(kind))
		// Data subresource
		s.e.GET("/VIID/"+coll+"/:id/Data", s.makeDataGetSubHandler(kind))
		s.e.POST("/VIID/"+coll+"/:id/Data", s.makeDataPutSubHandler(kind))
		s.e.PUT("/VIID/"+coll+"/:id/Data", s.makeDataPutSubHandler(kind))
		s.e.DELETE("/VIID/"+coll+"/:id/Data", s.makeDataDeleteSubHandler(kind))
	}
}

// makeCollectionHandler 接收标准 GA/T 1400.4 §5.2 列表信封：
//
//	{ "<Kind>List": { "<Kind>Object": [ {<obj>}, ... ] } }
//
// 并按 ID 字段存入内存 repo。
func (s *Server) makeCollectionHandler(kind resource.Kind) echo.HandlerFunc {
	return func(c echo.Context) error {
		var raw map[string]any
		if err := c.Bind(&raw); err != nil {
			return c.JSON(http.StatusBadRequest, response.Error("", response.CodeInvalid, "invalid body"))
		}
		listKey := string(kind) + "List"
		inner, ok := raw[listKey].(map[string]any)
		if !ok {
			return c.JSON(http.StatusBadRequest, response.Error("", response.CodeInvalid, "missing "+listKey))
		}
		arr, ok := inner[string(kind)+"Object"].([]any)
		if !ok {
			return c.JSON(http.StatusBadRequest, response.Error("", response.CodeInvalid, "missing "+string(kind)+"Object"))
		}
		idField := resource.IDOf(kind)
		var stored []string
		for _, item := range arr {
			obj, ok := item.(map[string]any)
			if !ok {
				continue
			}
			id, _ := obj[idField].(string)
			if id == "" {
				continue
			}
			s.repo.store(kind, id, obj)
			stored = append(stored, id)
		}
		// Notify any in-process subscribers. For now we only echo the resource
		// back to the caller via the standard response shell.
		for _, id := range stored {
			if err := ports.NotifyResource(c.Request().Context(), s.log, kind, id); err != nil {
				s.log.Warn("notify failed", "error", err)
			}
		}
		return c.JSON(http.StatusOK, map[string]any{
			"ResponseStatus": response.OK(""),
			"ItemCount":      len(stored),
		})
	}
}

// makeDataListHandler 返回列表处理器（GET /VIID/<Collection>）。
// GA/T 1400.4 §5.2 列表查询：响应为 <Kind>List 包裹 <Kind>Object 数组。
func (s *Server) makeDataListHandler(kind resource.Kind) echo.HandlerFunc {
	return func(c echo.Context) error {
		items := s.repo.list(kind)
		listKey := string(kind) + "List"
		objKey := string(kind) + "Object"
		return c.JSON(http.StatusOK, map[string]any{
			"ResponseStatus": response.OK(""),
			listKey:          map[string]any{objKey: items},
		})
	}
}

// makeDataGetHandler 返回单条查询处理器（GET /VIID/<Collection>/:id）。
// GA/T 1400.4 §5.2 单条查询：按 <Kind>ID 检索。
func (s *Server) makeDataGetHandler(kind resource.Kind) echo.HandlerFunc {
	return func(c echo.Context) error {
		id := c.Param("id")
		obj, ok := s.repo.get(kind, id)
		if !ok {
			return c.JSON(http.StatusNotFound, response.Error(id, response.CodeNotFound, "not found"))
		}
		return c.JSON(http.StatusOK, map[string]any{
			"ResponseStatus": response.OK(id),
			string(kind):     obj,
		})
	}
}

// makeDataPutHandler 返回更新处理器（PUT /VIID/<Collection>/:id）。
// GA/T 1400.4 §5.2 单条更新：body 为完整对象，按 URL 中的 <Kind>ID 替换。
func (s *Server) makeDataPutHandler(kind resource.Kind) echo.HandlerFunc {
	return func(c echo.Context) error {
		id := c.Param("id")
		var raw map[string]any
		if err := c.Bind(&raw); err != nil {
			return c.JSON(http.StatusBadRequest, response.Error(id, response.CodeInvalid, "invalid body"))
		}
		s.repo.store(kind, id, raw)
		return c.JSON(http.StatusOK, map[string]any{"ResponseStatus": response.OK(id)})
	}
}

// makeDataDeleteHandler 返回软删除处理器（DELETE /VIID/<Collection>/:id）。
// GA/T 1400.4 §5.2 单条删除：直接从内存 map 中删除 key（软删除）。
func (s *Server) makeDataDeleteHandler(kind resource.Kind) echo.HandlerFunc {
	return func(c echo.Context) error {
		id := c.Param("id")
		// Soft delete: forget parent + Info/Data 子资源。
		s.repo.mu.Lock()
		delete(s.repo.data[kind], id)
		delete(s.repo.info[kind], id)
		delete(s.repo.blob[kind], id)
		s.repo.mu.Unlock()
		return c.JSON(http.StatusOK, map[string]any{"ResponseStatus": response.OK(id)})
	}
}

// makeInfoGetHandler GET /VIID/<Collection>/:id/Info —— 返回独立 Info；未写入时回退父对象摘要。
func (s *Server) makeInfoGetHandler(kind resource.Kind) echo.HandlerFunc {
	return func(c echo.Context) error {
		id := c.Param("id")
		if _, ok := s.repo.get(kind, id); !ok {
			return c.JSON(http.StatusNotFound, response.Error(id, response.CodeNotFound, "not found"))
		}
		info, ok := s.repo.getInfo(kind, id)
		if !ok {
			info = map[string]any{
				resource.IDOf(kind): id,
				"Kind":              string(kind),
				"Source":            "derived",
			}
		}
		return c.JSON(http.StatusOK, map[string]any{
			"ResponseStatus": response.OK(id),
			"Info":           info,
		})
	}
}

func (s *Server) makeInfoPutHandler(kind resource.Kind) echo.HandlerFunc {
	return func(c echo.Context) error {
		id := c.Param("id")
		if _, ok := s.repo.get(kind, id); !ok {
			return c.JSON(http.StatusNotFound, response.Error(id, response.CodeNotFound, "not found"))
		}
		var body map[string]any
		if err := c.Bind(&body); err != nil {
			return c.JSON(http.StatusBadRequest, response.Error(id, response.CodeInvalid, "invalid body"))
		}
		if body == nil {
			body = map[string]any{}
		}
		body[resource.IDOf(kind)] = id
		s.repo.putInfo(kind, id, body)
		return c.JSON(http.StatusOK, map[string]any{"ResponseStatus": response.OK(id)})
	}
}

func (s *Server) makeInfoDeleteHandler(kind resource.Kind) echo.HandlerFunc {
	return func(c echo.Context) error {
		id := c.Param("id")
		s.repo.deleteInfo(kind, id)
		return c.JSON(http.StatusOK, map[string]any{"ResponseStatus": response.OK(id)})
	}
}

// makeDataGetSubHandler GET Data 子资源（与父对象分离的二进制/结构化载荷信封）。
func (s *Server) makeDataGetSubHandler(kind resource.Kind) echo.HandlerFunc {
	return func(c echo.Context) error {
		id := c.Param("id")
		if _, ok := s.repo.get(kind, id); !ok {
			return c.JSON(http.StatusNotFound, response.Error(id, response.CodeNotFound, "not found"))
		}
		data, ok := s.repo.getData(kind, id)
		if !ok {
			data = map[string]any{}
		}
		return c.JSON(http.StatusOK, map[string]any{
			"ResponseStatus": response.OK(id),
			"Data":           data,
		})
	}
}

func (s *Server) makeDataPutSubHandler(kind resource.Kind) echo.HandlerFunc {
	return func(c echo.Context) error {
		id := c.Param("id")
		if _, ok := s.repo.get(kind, id); !ok {
			return c.JSON(http.StatusNotFound, response.Error(id, response.CodeNotFound, "not found"))
		}
		var body map[string]any
		if err := c.Bind(&body); err != nil {
			return c.JSON(http.StatusBadRequest, response.Error(id, response.CodeInvalid, "invalid body"))
		}
		if body == nil {
			body = map[string]any{}
		}
		s.repo.putData(kind, id, body)
		return c.JSON(http.StatusOK, map[string]any{"ResponseStatus": response.OK(id)})
	}
}

func (s *Server) makeDataDeleteSubHandler(kind resource.Kind) echo.HandlerFunc {
	return func(c echo.Context) error {
		id := c.Param("id")
		s.repo.deleteData(kind, id)
		return c.JSON(http.StatusOK, map[string]any{"ResponseStatus": response.OK(id)})
	}
}