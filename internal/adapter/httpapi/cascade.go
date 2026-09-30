package httpapi

import (
	"net/http"
	"sync"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/noroadzh/gat1400-simulator/internal/domain/response"
)

// extractDeleteIDs 从 body 中抽取删除 ID 列表。
// 支持两种 body 形态：
//   a) { "SubscribeIDList": ["id1", "id2"] }
//   b) { "DeleteOperate": { "SubscribeIDList": [...] } }
//
// 仅当 listKey 存在且为非空数组时返回 true，调用方应进入删除分支。
func extractDeleteIDs(body map[string]any, listKey string) ([]string, bool) {
	// Case (a): top-level array.
	if arr, ok := body[listKey].([]any); ok && len(arr) > 0 {
		out := make([]string, 0, len(arr))
		for _, v := range arr {
			if s, ok := v.(string); ok {
				out = append(out, s)
			}
		}
		return out, len(out) > 0
	}
	// Case (b): nested DeleteOperate object.
	if op, ok := body["DeleteOperate"].(map[string]any); ok {
		if arr, ok := op[listKey].([]any); ok && len(arr) > 0 {
			out := make([]string, 0, len(arr))
			for _, v := range arr {
				if s, ok := v.(string); ok {
					out = append(out, s)
				}
			}
			return out, len(out) > 0
		}
	}
	return nil, false
}

// subscribeRepo 内存存储 Subscribe / SubscribeNotification / Disposition 记录。
// 生产环境会替换为 sqlite；引导阶段用内存 map 简化接线。
type subscribeRepo struct {
	mu       sync.RWMutex
	subs     map[string]map[string]any // SubscribeID -> raw payload
	notifs   map[string][]map[string]any
	disp     map[string]map[string]any
	dispNotifs map[string][]map[string]any
}

func newSubscribeRepo() *subscribeRepo {
	return &subscribeRepo{
		subs:        map[string]map[string]any{},
		notifs:      map[string][]map[string]any{},
		disp:        map[string]map[string]any{},
		dispNotifs:  map[string][]map[string]any{},
	}
}

func (r *subscribeRepo) store(sc map[string]any) (string, bool) {
	id, _ := sc["SubscribeID"].(string)
	if id == "" {
		return "", false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.subs[id] = sc
	return id, true
}

func (r *subscribeRepo) storeNotification(id string, payload map[string]any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.notifs[id] = append(r.notifs[id], payload)
}

func (r *subscribeRepo) list() []map[string]any {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]map[string]any, 0, len(r.subs))
	for _, v := range r.subs {
		out = append(out, v)
	}
	return out
}

func (r *subscribeRepo) listNotifs(id string) []map[string]any {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]map[string]any(nil), r.notifs[id]...)
}

func (r *subscribeRepo) delete(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.subs, id)
	delete(r.notifs, id)
}

func (r *subscribeRepo) storeDisposition(sc map[string]any) (string, bool) {
	id, _ := sc["DispositionID"].(string)
	if id == "" {
		return "", false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.disp[id] = sc
	return id, true
}

func (r *subscribeRepo) listDispositions() []map[string]any {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]map[string]any, 0, len(r.disp))
	for _, v := range r.disp {
		out = append(out, v)
	}
	return out
}

func (r *subscribeRepo) deleteDisposition(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.disp, id)
}

// registerCascadeRoutes 绑定订阅/通知/布控路由（GA/T 1400.4 第5.4节）。
func (s *Server) registerCascadeRoutes() {
	// Subscribes
	s.e.POST("/VIID/Subscribes", s.handleSubscribeCreate)
	s.e.PUT("/VIID/Subscribes", s.handleSubscribeBulk)
	s.e.DELETE("/VIID/Subscribes", s.handleSubscribeBulkDelete)
	s.e.GET("/VIID/Subscribes", s.handleSubscribeList)
	s.e.PUT("/VIID/Subscribes/:id", s.handleSubscribeUpdate)
	s.e.DELETE("/VIID/Subscribes/:id", s.handleSubscribeDelete)

	// Notifications (inbound — publishers push notifications here)
	s.e.POST("/VIID/SubscribeNotifications", s.handleNotificationCreate)
	s.e.DELETE("/VIID/SubscribeNotifications", s.handleNotificationDelete)
	s.e.GET("/VIID/SubscribeNotifications", s.handleNotificationList)

	// Dispositions (布控)
	s.e.POST("/VIID/Dispositions", s.handleDispositionCreate)
	s.e.PUT("/VIID/Dispositions", s.handleDispositionBulk)
	s.e.DELETE("/VIID/Dispositions", s.handleDispositionBulkDelete)
	s.e.GET("/VIID/Dispositions", s.handleDispositionList)
	s.e.PUT("/VIID/Dispositions/:id", s.handleDispositionUpdate)
	s.e.DELETE("/VIID/Dispositions/:id", s.handleDispositionDelete)

	// Disposition notifications
	s.e.POST("/VIID/DispositionNotifications", s.handleDispositionNotificationCreate)
	s.e.DELETE("/VIID/DispositionNotifications", s.handleDispositionNotificationDelete)
	s.e.GET("/VIID/DispositionNotifications", s.handleDispositionNotificationList)
}

// --- Subscribes ---

// handleSubscribeCreate 处理 POST /VIID/Subscribes，支持两种请求形态。
// GA/T 1400.4 §5.4 Subscribe Create：创建订阅（标准形态）或按 body 删除（双形态）。
//
// 创建形态：请求体为 SubscribeObject，服务端存储后返回 SubscribeID。
// 删除形态：请求体含 SubscribeIDList（顶层或嵌套于 DeleteOperate），
// 服务端删除对应记录后返回 OK。URL 路径删除（DELETE /VIID/Subscribes/:id）同时保留。
func (s *Server) handleSubscribeCreate(c echo.Context) error {
	var body map[string]any
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, response.Error("", response.CodeInvalid, "invalid body"))
	}
	// Body delete branch: POST 含 SubscribeIDList 或 DeleteOperate 时进入删除路径。
	if ids, ok := extractDeleteIDs(body, "SubscribeIDList"); ok {
		for _, id := range ids {
			s.subRepo.delete(id)
		}
		return c.JSON(http.StatusOK, map[string]any{"ResponseStatus": response.OK("")})
	}
	// Create branch.
	id, ok := s.subRepo.store(body)
	if !ok {
		// Generate an ID when missing so the simulator can accept arbitrary clients.
		body["SubscribeID"] = s.ids.SubscribeID()
		id, _ = s.subRepo.store(body)
	}
	return c.JSON(http.StatusOK, map[string]any{
		"ResponseStatus": response.OK(id),
		"SubscribeID":    id,
	})
}

// handleSubscribeUpdate 处理 PUT /VIID/Subscribes/:id。
// GA/T 1400.4 §5.4 Subscribe Update：替换订阅内容，按 URL 中的 SubscribeID 索引。
func (s *Server) handleSubscribeUpdate(c echo.Context) error {
	id := c.Param("id")
	var body map[string]any
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, response.Error(id, response.CodeInvalid, "invalid body"))
	}
	body["SubscribeID"] = id
	s.subRepo.mu.Lock()
	s.subRepo.subs[id] = body
	s.subRepo.mu.Unlock()
	return c.JSON(http.StatusOK, map[string]any{"ResponseStatus": response.OK(id)})
}

// handleSubscribeDelete 处理 DELETE /VIID/Subscribes/:id。
// GA/T 1400.4 §5.4 Subscribe Delete：按 SubscribeID 删除单条订阅；
// 与 POST + body 列表删除是两种等价入口。
func (s *Server) handleSubscribeDelete(c echo.Context) error {
	id := c.Param("id")
	s.subRepo.delete(id)
	return c.JSON(http.StatusOK, map[string]any{"ResponseStatus": response.OK(id)})
}

// handleSubscribeBulk 处理 PUT /VIID/Subscribes。
// GA/T 1400.4 §5.4 Subscribe Batch Update：批量更新订阅（请求体 SubscribeObject 列表）。
func (s *Server) handleSubscribeBulk(c echo.Context) error {
	var body map[string]any
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, response.Error("", response.CodeInvalid, "invalid body"))
	}
	if arr, ok := body["SubscribeObject"].([]any); ok {
		for _, item := range arr {
			if m, ok := item.(map[string]any); ok {
				s.subRepo.store(m)
			}
		}
	}
	return c.JSON(http.StatusOK, map[string]any{"ResponseStatus": response.OK("")})
}

// handleSubscribeBulkDelete 处理 DELETE /VIID/Subscribes。
// GA/T 1400.4 §5.4 Subscribe Batch Delete：按 SubscribeIDList 批量删除。
func (s *Server) handleSubscribeBulkDelete(c echo.Context) error {
	var body map[string]any
	_ = c.Bind(&body)
	if arr, ok := body["SubscribeIDList"].([]any); ok {
		for _, item := range arr {
			if id, ok := item.(string); ok {
				s.subRepo.delete(id)
			}
		}
	}
	return c.JSON(http.StatusOK, map[string]any{"ResponseStatus": response.OK("")})
}

// handleSubscribeList 处理 GET /VIID/Subscribes。
// GA/T 1400.4 §5.4 Subscribe List：返回全部订阅列表（SubscribeList 包裹）。
func (s *Server) handleSubscribeList(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]any{
		"ResponseStatus": response.OK(""),
		"SubscribeList":  map[string]any{"SubscribeObject": s.subRepo.list()},
	})
}

// --- Subscribe Notifications ---

// handleNotificationCreate 处理 POST /VIID/SubscribeNotifications。
// GA/T 1400.4 §5.4 SubscribeNotification Push：上层 publisher 推送触发通知到下层 subscriber。
func (s *Server) handleNotificationCreate(c echo.Context) error {
	var body map[string]any
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, response.Error("", response.CodeInvalid, "invalid body"))
	}
	id, _ := body["SubscribeID"].(string)
	if id == "" {
		id = "global"
	}
	notif := map[string]any{
		"NotificationID": s.ids.UUID(),
		"Title":          body["Title"],
		"TriggerTime":    time.Now().UTC().Format(time.RFC3339),
		"InfoIDs":        body["InfoIDs"],
		"Raw":            body,
	}
	s.subRepo.storeNotification(id, notif)
	return c.JSON(http.StatusOK, map[string]any{
		"ResponseStatus": response.OK(id),
		"NotificationID": notif["NotificationID"],
	})
}

// handleNotificationDelete 处理 DELETE /VIID/SubscribeNotifications。
// GA/T 1400.4 §5.4 SubscribeNotification Delete：清空通知列表（占位实现）。
func (s *Server) handleNotificationDelete(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]any{"ResponseStatus": response.OK("")})
}

// handleNotificationList 处理 GET /VIID/SubscribeNotifications。
// GA/T 1400.4 §5.4 SubscribeNotification List：按 subscribeId 返回通知列表。
func (s *Server) handleNotificationList(c echo.Context) error {
	id := c.QueryParam("subscribeId")
	if id == "" {
		id = "global"
	}
	return c.JSON(http.StatusOK, map[string]any{
		"ResponseStatus":             response.OK(""),
		"SubscribeNotificationList": map[string]any{"SubscribeNotificationObject": s.subRepo.listNotifs(id)},
	})
}

// --- Dispositions (布控) ---

// handleDispositionCreate 处理 POST /VIID/Dispositions，支持两种请求形态。
// GA/T 1400.4 §5.4 Disposition Create：创建布控（标准形态）或按 body 删除（双形态）。
//
// 创建形态：请求体为 DispositionObject，服务端存储后返回 DispositionID。
// 删除形态：请求体含 DispositionIDList（顶层或嵌套于 DeleteOperate），
// 服务端删除对应记录后返回 OK。URL 路径删除（DELETE /VIID/Dispositions/:id）同时保留。
func (s *Server) handleDispositionCreate(c echo.Context) error {
	var body map[string]any
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, response.Error("", response.CodeInvalid, "invalid body"))
	}
	// Body delete branch: POST 含 DispositionIDList 或 DeleteOperate 时进入删除路径。
	if ids, ok := extractDeleteIDs(body, "DispositionIDList"); ok {
		for _, id := range ids {
			s.subRepo.deleteDisposition(id)
		}
		return c.JSON(http.StatusOK, map[string]any{"ResponseStatus": response.OK("")})
	}
	// Create branch.
	id, ok := s.subRepo.storeDisposition(body)
	if !ok {
		body["DispositionID"] = s.ids.SubscribeID()
		id, _ = s.subRepo.storeDisposition(body)
	}
	return c.JSON(http.StatusOK, map[string]any{
		"ResponseStatus": response.OK(id),
		"DispositionID":  id,
	})
}

// handleDispositionUpdate 处理 PUT /VIID/Dispositions/:id。
// GA/T 1400.4 §5.4 Disposition Update：按 URL 中的 DispositionID 替换布控内容。
func (s *Server) handleDispositionUpdate(c echo.Context) error {
	id := c.Param("id")
	var body map[string]any
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, response.Error(id, response.CodeInvalid, "invalid body"))
	}
	body["DispositionID"] = id
	s.subRepo.mu.Lock()
	s.subRepo.disp[id] = body
	s.subRepo.mu.Unlock()
	return c.JSON(http.StatusOK, map[string]any{"ResponseStatus": response.OK(id)})
}

// handleDispositionDelete 处理 DELETE /VIID/Dispositions/:id。
// GA/T 1400.4 §5.4 Disposition Delete：按 DispositionID 删除单条布控。
func (s *Server) handleDispositionDelete(c echo.Context) error {
	id := c.Param("id")
	s.subRepo.deleteDisposition(id)
	return c.JSON(http.StatusOK, map[string]any{"ResponseStatus": response.OK(id)})
}

// handleDispositionBulk 处理 PUT /VIID/Dispositions。
// GA/T 1400.4 §5.4 Disposition Batch Update：批量更新布控（请求体 DispositionObject 列表）。
func (s *Server) handleDispositionBulk(c echo.Context) error {
	var body map[string]any
	_ = c.Bind(&body)
	if arr, ok := body["DispositionObject"].([]any); ok {
		for _, item := range arr {
			if m, ok := item.(map[string]any); ok {
				s.subRepo.storeDisposition(m)
			}
		}
	}
	return c.JSON(http.StatusOK, map[string]any{"ResponseStatus": response.OK("")})
}

// handleDispositionBulkDelete 处理 DELETE /VIID/Dispositions。
// GA/T 1400.4 §5.4 Disposition Batch Delete：按 DispositionIDList 批量删除。
func (s *Server) handleDispositionBulkDelete(c echo.Context) error {
	var body map[string]any
	_ = c.Bind(&body)
	if arr, ok := body["DispositionIDList"].([]any); ok {
		for _, item := range arr {
			if id, ok := item.(string); ok {
				s.subRepo.deleteDisposition(id)
			}
		}
	}
	return c.JSON(http.StatusOK, map[string]any{"ResponseStatus": response.OK("")})
}

// handleDispositionList 处理 GET /VIID/Dispositions。
// GA/T 1400.4 §5.4 Disposition List：返回全部布控列表（DispositionList 包裹）。
func (s *Server) handleDispositionList(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]any{
		"ResponseStatus":  response.OK(""),
		"DispositionList": map[string]any{"DispositionObject": s.subRepo.listDispositions()},
	})
}

// handleDispositionNotificationCreate 处理 POST /VIID/DispositionNotifications。
// GA/T 1400.4 §5.4 DispositionNotification Push：布控命中时上层推送通知到下层 subscriber。
func (s *Server) handleDispositionNotificationCreate(c echo.Context) error {
	var body map[string]any
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, response.Error("", response.CodeInvalid, "invalid body"))
	}
	id, _ := body["DispositionID"].(string)
	if id == "" {
		id = "global"
	}
	notif := map[string]any{
		"DispositionNotificationID": s.ids.UUID(),
		"TriggerTime":              time.Now().UTC().Format(time.RFC3339),
		"Raw":                      body,
	}
	s.subRepo.mu.Lock()
	s.subRepo.dispNotifs[id] = append(s.subRepo.dispNotifs[id], notif)
	s.subRepo.mu.Unlock()
	return c.JSON(http.StatusOK, map[string]any{
		"ResponseStatus":            response.OK(id),
		"DispositionNotificationID": notif["DispositionNotificationID"],
	})
}

// handleDispositionNotificationDelete 处理 DELETE /VIID/DispositionNotifications。
// GA/T 1400.4 §5.4 DispositionNotification Delete：清空布控通知（占位实现）。
func (s *Server) handleDispositionNotificationDelete(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]any{"ResponseStatus": response.OK("")})
}

// handleDispositionNotificationList 处理 GET /VIID/DispositionNotifications。
// GA/T 1400.4 §5.4 DispositionNotification List：按 dispositionId 返回通知列表。
func (s *Server) handleDispositionNotificationList(c echo.Context) error {
	id := c.QueryParam("dispositionId")
	if id == "" {
		id = "global"
	}
	s.subRepo.mu.RLock()
	items := append([]map[string]any(nil), s.subRepo.dispNotifs[id]...)
	s.subRepo.mu.RUnlock()
	return c.JSON(http.StatusOK, map[string]any{
		"ResponseStatus":                response.OK(""),
		"DispositionNotificationList": map[string]any{"DispositionNotificationObject": items},
	})
}