package httpapi

import (
	"net/http"
	"sync"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/noroadzh/gat1400-simulator/internal/domain/response"
)

// viasRepo 分析系统（表 14 `/VIAS/*`）内存存储。
type viasRepo struct {
	mu       sync.RWMutex
	tasks    map[string]map[string]any
	controls map[string]map[string]any
	rules    map[string]map[string]any
	// taskID -> kind -> id -> payload（VideoSlices / Images / VideoLabels）
	taskRes map[string]map[string]map[string]map[string]any
}

func newVIASRepo() *viasRepo {
	return &viasRepo{
		tasks:    map[string]map[string]any{},
		controls: map[string]map[string]any{},
		rules:    map[string]map[string]any{},
		taskRes:  map[string]map[string]map[string]map[string]any{},
	}
}

// ensureTaskBucket 调用方 MUST 已持有 r.mu 写锁。
func (r *viasRepo) ensureTaskBucket(taskID, kind string) map[string]map[string]any {
	if r.taskRes[taskID] == nil {
		r.taskRes[taskID] = map[string]map[string]map[string]any{}
	}
	if r.taskRes[taskID][kind] == nil {
		r.taskRes[taskID][kind] = map[string]map[string]any{}
	}
	return r.taskRes[taskID][kind]
}

func (r *viasRepo) listTaskRes(taskID, kind string) []map[string]any {
	r.mu.RLock()
	defer r.mu.RUnlock()
	bucket := r.taskRes[taskID][kind]
	out := make([]map[string]any, 0, len(bucket))
	for _, v := range bucket {
		out = append(out, v)
	}
	return out
}

func (r *viasRepo) getTaskRes(taskID, kind, rid string) (map[string]any, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	v, ok := r.taskRes[taskID][kind][rid]
	return v, ok
}

// registerVIASRoutes 注册 GA/T 1400.4 表 14 分析系统资源 URI（最小可用实现）。
func (s *Server) registerVIASRoutes() {
	g := s.e.Group("/VIAS")
	g.GET("", s.handleVIASRoot)
	g.GET("/System", s.handleVIASSystem)
	g.POST("/System/Register", s.handleVIASRegister, s.digestAuth())
	g.POST("/System/UnRegister", s.handleVIASUnRegister, s.digestAuth())
	g.POST("/System/Keepalive", s.handleKeepalive) // 复用 VIID 心跳逻辑
	g.GET("/System/Time", s.handleTime)

	g.GET("/Tasks", s.handleVIASTasksList)
	g.POST("/Tasks", s.handleVIASTasksCreate)
	g.GET("/Tasks/:id", s.handleVIASTaskGet)
	g.PUT("/Tasks/:id", s.handleVIASTaskPut)
	g.DELETE("/Tasks/:id", s.handleVIASTaskDelete)

	// 任务下资源
	for _, kind := range []string{"VideoSlices", "Images", "VideoLabels"} {
		k := kind
		g.GET("/Tasks/:id/"+k, s.makeVIASTaskResList(k))
		g.POST("/Tasks/:id/"+k, s.makeVIASTaskResUpsert(k))
		g.GET("/Tasks/:id/"+k+"/:rid", s.makeVIASTaskResGet(k))
		g.DELETE("/Tasks/:id/"+k+"/:rid", s.makeVIASTaskResDelete(k))
		if k == "VideoSlices" || k == "Images" {
			g.GET("/Tasks/:id/"+k+"/:rid/Data", s.makeVIASTaskResDataGet(k))
			g.POST("/Tasks/:id/"+k+"/:rid/Data", s.makeVIASTaskResDataPut(k))
			g.DELETE("/Tasks/:id/"+k+"/:rid/Data", s.makeVIASTaskResDataDelete(k))
		}
	}

	g.GET("/TaskStatuses", s.handleVIASTaskStatuses)
	g.GET("/TaskStatuses/:id", s.handleVIASTaskStatusGet)

	g.GET("/TaskControls", s.handleVIASTaskControlsList)
	g.POST("/TaskControls", s.handleVIASTaskControlsCreate)

	g.GET("/AnalysisRules", s.handleVIASRulesList)
	g.POST("/AnalysisRules", s.handleVIASRulesUpsert)
	g.GET("/AnalysisRules/:id", s.handleVIASRuleGet)
	g.DELETE("/AnalysisRules/:id", s.handleVIASRuleDelete)

	g.GET("/SystemCapability/AnalysisCapability", s.handleVIASCapability("Analysis"))
	g.GET("/SystemCapability/EnhancementAndRestorationCapability", s.handleVIASCapability("EnhancementAndRestoration"))
	g.GET("/SystemCapability/RetrievalCapability", s.handleVIASCapability("Retrieval"))
}

func (s *Server) handleVIASRoot(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]any{
		"ResponseStatus": response.OK(""),
		"ResourceURIList": map[string]any{"ResourceURI": []string{
			"/VIAS/System", "/VIAS/Tasks", "/VIAS/TaskStatuses", "/VIAS/TaskControls",
			"/VIAS/AnalysisRules",
			"/VIAS/SystemCapability/AnalysisCapability",
			"/VIAS/SystemCapability/EnhancementAndRestorationCapability",
			"/VIAS/SystemCapability/RetrievalCapability",
		}},
	})
}

func (s *Server) handleVIASSystem(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]any{
		"ResponseStatus": response.OK(""),
		"System": map[string]any{
			"Name":       "gat1400-simulator-vias",
			"Version":    "0.2.0",
			"Protocol":   "GA/T 1400.4-2017",
			"Role":       "analysis",
			"ServerTime": time.Now().UTC().Format(time.RFC3339),
		},
	})
}

func (s *Server) handleVIASRegister(c echo.Context) error {
	return s.handleRegister(c)
}

func (s *Server) handleVIASUnRegister(c echo.Context) error {
	return s.handleUnRegister(c)
}

func (s *Server) handleVIASTasksList(c echo.Context) error {
	s.vias.mu.RLock()
	defer s.vias.mu.RUnlock()
	out := make([]map[string]any, 0, len(s.vias.tasks))
	for _, v := range s.vias.tasks {
		out = append(out, v)
	}
	return c.JSON(http.StatusOK, map[string]any{
		"ResponseStatus": response.OK(""),
		"TaskList":       map[string]any{"TaskObject": out},
	})
}

func (s *Server) handleVIASTasksCreate(c echo.Context) error {
	var body map[string]any
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, response.Error("", response.CodeInvalid, "invalid body"))
	}
	list, _ := body["TaskList"].(map[string]any)
	arr, _ := list["TaskObject"].([]any)
	if len(arr) == 0 {
		// 单对象形态
		if body["TaskID"] != nil || body["TaskName"] != nil {
			arr = []any{body}
		}
	}
	ids := []string{}
	s.vias.mu.Lock()
	for _, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		id, _ := m["TaskID"].(string)
		if id == "" {
			id = s.ids.SubscribeID()
			m["TaskID"] = id
		}
		if m["Status"] == nil {
			m["Status"] = "Created"
		}
		m["CreatedAt"] = time.Now().UTC().Format(time.RFC3339)
		s.vias.tasks[id] = m
		ids = append(ids, id)
		s.persistVIASTask(id, m)
	}
	s.vias.mu.Unlock()
	return c.JSON(http.StatusOK, map[string]any{
		"ResponseStatus": response.OK(""),
		"ItemCount":      len(ids),
		"TaskIDList":     ids,
	})
}

func (s *Server) handleVIASTaskGet(c echo.Context) error {
	id := c.Param("id")
	s.vias.mu.RLock()
	obj, ok := s.vias.tasks[id]
	s.vias.mu.RUnlock()
	if !ok {
		return c.JSON(http.StatusNotFound, response.Error(id, response.CodeNotFound, "not found"))
	}
	return c.JSON(http.StatusOK, map[string]any{"ResponseStatus": response.OK(id), "Task": obj})
}

func (s *Server) handleVIASTaskPut(c echo.Context) error {
	id := c.Param("id")
	var body map[string]any
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, response.Error(id, response.CodeInvalid, "invalid body"))
	}
	if body == nil {
		body = map[string]any{}
	}
	body["TaskID"] = id
	s.vias.mu.Lock()
	s.vias.tasks[id] = body
	s.vias.mu.Unlock()
	s.persistVIASTask(id, body)
	return c.JSON(http.StatusOK, map[string]any{"ResponseStatus": response.OK(id)})
}

func (s *Server) handleVIASTaskDelete(c echo.Context) error {
	id := c.Param("id")
	s.vias.mu.Lock()
	delete(s.vias.tasks, id)
	delete(s.vias.taskRes, id)
	s.vias.mu.Unlock()
	s.persistVIASTaskDelete(id)
	return c.JSON(http.StatusOK, map[string]any{"ResponseStatus": response.OK(id)})
}

func singularTaskKind(kind string) string {
	switch kind {
	case "VideoSlices":
		return "VideoSlice"
	case "Images":
		return "Image"
	case "VideoLabels":
		return "VideoLabel"
	}
	return kind
}

func (s *Server) makeVIASTaskResList(kind string) echo.HandlerFunc {
	return func(c echo.Context) error {
		taskID := c.Param("id")
		out := s.vias.listTaskRes(taskID, kind)
		sk := singularTaskKind(kind)
		return c.JSON(http.StatusOK, map[string]any{
			"ResponseStatus": response.OK(taskID),
			sk + "List":      map[string]any{sk + "Object": out},
		})
	}
}

func (s *Server) makeVIASTaskResUpsert(kind string) echo.HandlerFunc {
	return func(c echo.Context) error {
		taskID := c.Param("id")
		var body map[string]any
		if err := c.Bind(&body); err != nil {
			return c.JSON(http.StatusBadRequest, response.Error(taskID, response.CodeInvalid, "invalid body"))
		}
		sk := singularTaskKind(kind)
		list, _ := body[sk+"List"].(map[string]any)
		arr, _ := list[sk+"Object"].([]any)
		n := 0
		s.vias.mu.Lock()
		bucket := s.vias.ensureTaskBucket(taskID, kind)
		for _, item := range arr {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			idField := sk + "ID"
			id, _ := m[idField].(string)
			if id == "" {
				id = s.ids.UUID()
				m[idField] = id
			}
			bucket[id] = m
			n++
		}
		s.vias.mu.Unlock()
		return c.JSON(http.StatusOK, map[string]any{"ResponseStatus": response.OK(taskID), "ItemCount": n})
	}
}

func (s *Server) makeVIASTaskResGet(kind string) echo.HandlerFunc {
	return func(c echo.Context) error {
		taskID, rid := c.Param("id"), c.Param("rid")
		obj, ok := s.vias.getTaskRes(taskID, kind, rid)
		if !ok {
			return c.JSON(http.StatusNotFound, response.Error(rid, response.CodeNotFound, "not found"))
		}
		sk := singularTaskKind(kind)
		return c.JSON(http.StatusOK, map[string]any{"ResponseStatus": response.OK(rid), sk: obj})
	}
}

func (s *Server) makeVIASTaskResDelete(kind string) echo.HandlerFunc {
	return func(c echo.Context) error {
		taskID, rid := c.Param("id"), c.Param("rid")
		s.vias.mu.Lock()
		bucket := s.vias.ensureTaskBucket(taskID, kind)
		delete(bucket, rid)
		s.vias.mu.Unlock()
		return c.JSON(http.StatusOK, map[string]any{"ResponseStatus": response.OK(rid)})
	}
}

func (s *Server) makeVIASTaskResDataGet(kind string) echo.HandlerFunc {
	return func(c echo.Context) error {
		taskID, rid := c.Param("id"), c.Param("rid")
		obj, ok := s.vias.getTaskRes(taskID, kind, rid)
		if !ok {
			return c.JSON(http.StatusNotFound, response.Error(rid, response.CodeNotFound, "not found"))
		}
		data, _ := obj["Data"].(map[string]any)
		if data == nil {
			data = map[string]any{}
		}
		return c.JSON(http.StatusOK, map[string]any{"ResponseStatus": response.OK(rid), "Data": data})
	}
}

func (s *Server) makeVIASTaskResDataPut(kind string) echo.HandlerFunc {
	return func(c echo.Context) error {
		taskID, rid := c.Param("id"), c.Param("rid")
		var body map[string]any
		if err := c.Bind(&body); err != nil {
			return c.JSON(http.StatusBadRequest, response.Error(rid, response.CodeInvalid, "invalid body"))
		}
		s.vias.mu.Lock()
		bucket := s.vias.ensureTaskBucket(taskID, kind)
		obj, ok := bucket[rid]
		if !ok {
			obj = map[string]any{singularTaskKind(kind) + "ID": rid}
			bucket[rid] = obj
		}
		obj["Data"] = body
		s.vias.mu.Unlock()
		return c.JSON(http.StatusOK, map[string]any{"ResponseStatus": response.OK(rid)})
	}
}

func (s *Server) makeVIASTaskResDataDelete(kind string) echo.HandlerFunc {
	return func(c echo.Context) error {
		taskID, rid := c.Param("id"), c.Param("rid")
		s.vias.mu.Lock()
		if bucket := s.vias.ensureTaskBucket(taskID, kind); bucket[rid] != nil {
			delete(bucket[rid], "Data")
		}
		s.vias.mu.Unlock()
		return c.JSON(http.StatusOK, map[string]any{"ResponseStatus": response.OK(rid)})
	}
}

func (s *Server) handleVIASTaskStatuses(c echo.Context) error {
	s.vias.mu.RLock()
	defer s.vias.mu.RUnlock()
	out := make([]map[string]any, 0, len(s.vias.tasks))
	for id, t := range s.vias.tasks {
		st, _ := t["Status"].(string)
		if st == "" {
			st = "Unknown"
		}
		out = append(out, map[string]any{"TaskID": id, "Status": st})
	}
	return c.JSON(http.StatusOK, map[string]any{
		"ResponseStatus":   response.OK(""),
		"TaskStatusList":   map[string]any{"TaskStatusObject": out},
	})
}

func (s *Server) handleVIASTaskStatusGet(c echo.Context) error {
	id := c.Param("id")
	s.vias.mu.RLock()
	t, ok := s.vias.tasks[id]
	s.vias.mu.RUnlock()
	if !ok {
		return c.JSON(http.StatusNotFound, response.Error(id, response.CodeNotFound, "not found"))
	}
	st, _ := t["Status"].(string)
	return c.JSON(http.StatusOK, map[string]any{
		"ResponseStatus": response.OK(id),
		"TaskStatus":     map[string]any{"TaskID": id, "Status": st},
	})
}

func (s *Server) handleVIASTaskControlsList(c echo.Context) error {
	s.vias.mu.RLock()
	defer s.vias.mu.RUnlock()
	out := make([]map[string]any, 0, len(s.vias.controls))
	for _, v := range s.vias.controls {
		out = append(out, v)
	}
	return c.JSON(http.StatusOK, map[string]any{
		"ResponseStatus":    response.OK(""),
		"TaskControlList":   map[string]any{"TaskControlObject": out},
	})
}

func (s *Server) handleVIASTaskControlsCreate(c echo.Context) error {
	var body map[string]any
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, response.Error("", response.CodeInvalid, "invalid body"))
	}
	list, _ := body["TaskControlList"].(map[string]any)
	arr, _ := list["TaskControlObject"].([]any)
	if len(arr) == 0 && body["TaskID"] != nil {
		arr = []any{body}
	}
	n := 0
	s.vias.mu.Lock()
	for _, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		taskID, _ := m["TaskID"].(string)
		action, _ := m["Action"].(string) // Start / Pause / Stop
		if taskID == "" {
			continue
		}
		if t, ok := s.vias.tasks[taskID]; ok {
			switch action {
			case "Start", "Resume":
				t["Status"] = "Running"
			case "Pause":
				t["Status"] = "Paused"
			case "Stop":
				t["Status"] = "Stopped"
			}
		}
		cid := s.ids.UUID()
		m["TaskControlID"] = cid
		m["AppliedAt"] = time.Now().UTC().Format(time.RFC3339)
		s.vias.controls[cid] = m
		n++
	}
	s.vias.mu.Unlock()
	return c.JSON(http.StatusOK, map[string]any{"ResponseStatus": response.OK(""), "ItemCount": n})
}

func (s *Server) handleVIASRulesList(c echo.Context) error {
	s.vias.mu.RLock()
	defer s.vias.mu.RUnlock()
	out := make([]map[string]any, 0, len(s.vias.rules))
	for _, v := range s.vias.rules {
		out = append(out, v)
	}
	return c.JSON(http.StatusOK, map[string]any{
		"ResponseStatus":    response.OK(""),
		"AnalysisRuleList":  map[string]any{"AnalysisRuleObject": out},
	})
}

func (s *Server) handleVIASRulesUpsert(c echo.Context) error {
	var body map[string]any
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, response.Error("", response.CodeInvalid, "invalid body"))
	}
	list, _ := body["AnalysisRuleList"].(map[string]any)
	arr, _ := list["AnalysisRuleObject"].([]any)
	n := 0
	s.vias.mu.Lock()
	for _, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		id, _ := m["AnalysisRuleID"].(string)
		if id == "" {
			id = s.ids.UUID()
			m["AnalysisRuleID"] = id
		}
		s.vias.rules[id] = m
		s.persistVIASRule(id, m)
		n++
	}
	s.vias.mu.Unlock()
	return c.JSON(http.StatusOK, map[string]any{"ResponseStatus": response.OK(""), "ItemCount": n})
}

func (s *Server) handleVIASRuleGet(c echo.Context) error {
	id := c.Param("id")
	s.vias.mu.RLock()
	obj, ok := s.vias.rules[id]
	s.vias.mu.RUnlock()
	if !ok {
		return c.JSON(http.StatusNotFound, response.Error(id, response.CodeNotFound, "not found"))
	}
	return c.JSON(http.StatusOK, map[string]any{"ResponseStatus": response.OK(id), "AnalysisRule": obj})
}

func (s *Server) handleVIASRuleDelete(c echo.Context) error {
	id := c.Param("id")
	s.vias.mu.Lock()
	delete(s.vias.rules, id)
	s.vias.mu.Unlock()
	s.persistVIASRuleDelete(id)
	return c.JSON(http.StatusOK, map[string]any{"ResponseStatus": response.OK(id)})
}

func (s *Server) handleVIASCapability(kind string) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]any{
			"ResponseStatus": response.OK(""),
			"Capability": map[string]any{
				"Type":        kind,
				"Supported":   true,
				"Vendor":      "gat1400-simulator",
				"Description": "simulated " + kind + " capability",
			},
		})
	}
}
