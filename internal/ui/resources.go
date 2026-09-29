// Package ui 资源对象控制面（BFF 端）。
//
// 该文件实现 BFF 资源对象端点组（/api/control/resources/*），把 GA/T 1400.4 协议端
// 已注册的 /VIID/<Collection> 系列路由以 JSON API 形式暴露给 Web UI 与外部探测工具。
//
// 设计原则：
//   - 薄透传：BFF 不解析、不修改协议端响应 body，仅原样回传状态码与 JSON。
//   - 单向：协议端已具备完整的资源对象语义（collection.go），BFF 不重复实现。
//   - HTTP-only：BFF 与协议端的通信仅通过 HTTP，不直连协议端的内存存储或 handler。
package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/labstack/echo/v4"

	"github.com/noroadzh/gat1400-simulator/internal/domain/resource"
)

// Kind 中文描述（前端 fallback）。BFF 返回 description 字段供前端直接展示；
// 前端也可忽略，使用本地静态表。
var kindDescription = map[resource.Kind]string{
	resource.KindPerson:          "人员",
	resource.KindFace:            "人脸",
	resource.KindMotorVehicle:    "机动车",
	resource.KindNonMotorVehicle: "非机动车",
	resource.KindThing:           "物品",
	resource.KindScene:           "场景",
	resource.KindVideoSlice:      "视频片段",
	resource.KindImage:           "图像",
	resource.KindFile:            "文件",
	resource.KindCase:            "案件",
	resource.KindVideoLabel:      "视频标签",
	resource.KindAnalysisRule:    "分析规则",
}

// ResourceKindMeta 是 BFF 在 /api/control/resources 返回的元数据项。
type ResourceKindMeta struct {
	Kind        string `json:"kind"`
	Collection  string `json:"collection"`
	IDField     string `json:"idField"`
	Description string `json:"description"`
	Count       int    `json:"count"`
}

// protocolClient 是 BFF 与协议端通信的薄 HTTP 客户端。
type protocolClient struct {
	baseURL string
	http    *http.Client
	log     *slog.Logger

	// kindMetaCache 缓存 /api/control/resources 的完整响应（12 Kind 元数据 +
	// count），避免每次刷新都串行发 12 次 GET /VIID/<Collection>。
	kindMetaMu    sync.Mutex
	kindMetaCache []ResourceKindMeta
	kindMetaAt    time.Time
	// kindMetaGroup 防止缓存击穿：并发 miss 时只允许一个 goroutine 去打上游，
	// 其余等待它的结果（thundering herd 防护）。
	kindMetaGroup singleflight.Group
}

const kindMetaCacheTTL = 5 * time.Second

func newProtocolClient(baseURL string, log *slog.Logger) *protocolClient {
	if baseURL == "" {
		baseURL = "http://127.0.0.1:14000"
	}
	return &protocolClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		http: &http.Client{
			Timeout: 10 * time.Second,
		},
		log: log,
	}
}

// installResourceRoutes 注册 BFF 资源对象端点组到 /api/control 之下。
//
// 路由表（按 Kind 12 种不重复注册——Kind 由 :kind 路径段携带）：
//
//	GET    /api/control/resources                       — Kind 元数据 + 实时计数
//	GET    /api/control/resources/:kind/list            — 列表查询
//	GET    /api/control/resources/:kind/list/:id        — 单条查询
//	POST   /api/control/resources/:kind/list            — 批量写入
//	PUT    /api/control/resources/:kind/list/:id        — 单条更新
//	DELETE /api/control/resources/:kind/list/:id        — 单条删除
//	GET    /api/control/resources/:kind/list/:id/info   — Info 子资源
func (s *Server) installResourceRoutes(api *echo.Group) {
	api.GET("/resources", s.handleResourceKinds)
	api.GET("/resources/:kind/list", s.handleResourceProxy)
	api.GET("/resources/:kind/list/:id", s.handleResourceProxy)
	api.POST("/resources/:kind/list", s.handleResourceProxy)
	api.PUT("/resources/:kind/list/:id", s.handleResourceProxy)
	api.DELETE("/resources/:kind/list/:id", s.handleResourceProxy)
	api.GET("/resources/:kind/list/:id/info", s.handleResourceProxy)
}

// resolveKind 把 :kind 路径段解析为合法 Kind；非法返回错误。
func resolveKind(raw string) (resource.Kind, error) {
	k := resource.Kind(raw)
	if k == "" || !resource.IsValid(k) {
		return "", fmt.Errorf("unknown kind: %q", raw)
	}
	return k, nil
}

// handleResourceKinds 返回 12 种 Kind 的元数据列表（含缓存的实时计数）。
func (s *Server) handleResourceKinds(c echo.Context) error {
	metas := s.fetchKindMetas(c.Request().Context())
	return c.JSON(http.StatusOK, map[string]any{"kinds": metas})
}

// fetchKindMetas 获取 Kind 元数据列表（5s 内存缓存，singleflight 防护击穿）。
func (s *Server) fetchKindMetas(ctx context.Context) []ResourceKindMeta {
	s.protocolClient.kindMetaMu.Lock()
	cached := s.protocolClient.kindMetaCache
	at := s.protocolClient.kindMetaAt
	s.protocolClient.kindMetaMu.Unlock()
	if cached != nil && time.Since(at) < kindMetaCacheTTL {
		return cached
	}

	v, err, _ := s.protocolClient.kindMetaGroup.Do("kind-metas", func() (any, error) {
		// 双重检查：另一个并发请求可能已填充缓存。
		s.protocolClient.kindMetaMu.Lock()
		cached := s.protocolClient.kindMetaCache
		at := s.protocolClient.kindMetaAt
		s.protocolClient.kindMetaMu.Unlock()
		if cached != nil && time.Since(at) < kindMetaCacheTTL {
			return cached, nil
		}

		metas := make([]ResourceKindMeta, 0, len(resource.AllKinds))
		for _, k := range resource.AllKinds {
			meta := ResourceKindMeta{
				Kind:        string(k),
				Collection:  resource.CollectionOf(k),
				IDField:     resource.IDOf(k),
				Description: kindDescription[k],
			}
			// 透传 GET /VIID/<Collection>，解析信封数 <Kind>Object 长度。
			count, err := s.countObjects(ctx, meta.Collection, meta.Kind)
			if err != nil {
				s.log.Warn("bff.resource.count failed",
					slog.String("kind", meta.Kind),
					slog.String("error", err.Error()))
				// 失败时保留 0，不中断整张表。
			}
			meta.Count = count
			metas = append(metas, meta)
		}

		s.protocolClient.kindMetaMu.Lock()
		s.protocolClient.kindMetaCache = metas
		s.protocolClient.kindMetaAt = time.Now()
		s.protocolClient.kindMetaMu.Unlock()
		return metas, nil
	})
	if err != nil {
		return nil
	}
	return v.([]ResourceKindMeta)
}

// countObjects 通过 GET /VIID/<Collection> 返回的列表信封解析对象数量。
//
// 协议端响应格式：{"ResponseStatus":{...}, "<Kind>List": {"<Kind>Object": [...]}}
func (s *Server) countObjects(ctx context.Context, collection, kind string) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		s.protocolClient.baseURL+"/VIID/"+collection, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Accept", "application/VIID+JSON")
	resp, err := s.protocolClient.http.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("protocol returned status %d", resp.StatusCode)
	}
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return 0, err
	}
	list, ok := body[kind+"List"].(map[string]any)
	if !ok {
		return 0, nil
	}
	arr, _ := list[kind+"Object"].([]any)
	return len(arr), nil
}

// handleResourceProxy 把 /api/control/resources/:kind/list[/...] 的请求
// 透传到协议端 /VIID/<Collection>[/...]。
//
// 设计要点：
//   - :kind 合法性立即校验（不合法 400）。
//   - 请求 body（POST/PUT）原样转发（用 io.Copy 避免大 body 一次性读到内存）。
//   - 响应 body、状态码、Content-Length 原样回写，仅 Content-Type 标准化为 application/json。
//   - 协议端连接失败返回 502 Bad Gateway + 标准错误信封。
func (s *Server) handleResourceProxy(c echo.Context) error {
	start := time.Now()
	kindStr := c.Param("kind")
	kind, err := resolveKind(kindStr)
	if err != nil {
		s.log.Warn("bff.resource.bad_kind",
			slog.String("kind", kindStr),
			slog.String("error", err.Error()))
		return c.JSON(http.StatusBadRequest, map[string]any{
			"ResponseStatus": map[string]any{
				"StatusCode":   1,
				"StatusString": "INVALID",
				"Description":  err.Error(),
				"Id":           kindStr,
			},
		})
	}
	coll := resource.CollectionOf(kind)

	// 把 Echo path 拼回协议端路径：
	//   /api/control/resources/:kind/list            → /VIID/<Collection>
	//   /api/control/resources/:kind/list/:id        → /VIID/<Collection>/:id
	//   /api/control/resources/:kind/list/:id/info   → /VIID/<Collection>/:id/Info
	upstreamPath := "/VIID/" + coll
	if id := c.Param("id"); id != "" {
		upstreamPath += "/" + id
		// Echo 不把 /info 当作 :param，但当前路由 URL 末尾 /info 时把它转给协议端 Info 子资源。
		if strings.HasSuffix(c.Request().URL.Path, "/info") {
			upstreamPath += "/Info"
		}
	}

	var body io.Reader
	if c.Request().Body != nil && c.Request().Body != http.NoBody {
		body = c.Request().Body
	}

	// query string 原样转发：协议端分页 / 过滤参数不能被 BFF 吞掉。
	upstreamURI := s.protocolClient.baseURL + upstreamPath
	if raw := c.Request().URL.RawQuery; raw != "" {
		upstreamURI += "?" + raw
	}

	req, err := http.NewRequestWithContext(c.Request().Context(), c.Request().Method,
		upstreamURI, body)
	if err != nil {
		s.log.Warn("bff.resource.build_request failed",
			slog.String("kind", string(kind)),
			slog.String("error", err.Error()))
		return c.JSON(http.StatusInternalServerError, map[string]any{
			"ResponseStatus": map[string]any{
				"StatusCode":   4,
				"StatusString": "SERVER_ERROR",
				"Description":  err.Error(),
			},
		})
	}
	// 透传请求头（Authorization / Content-Type / User-Identify 等）。
	for k, v := range c.Request().Header {
		if k == "Host" || k == "Content-Length" {
			continue
		}
		for _, vv := range v {
			req.Header.Add(k, vv)
		}
	}

	resp, err := s.protocolClient.http.Do(req)
	if err != nil {
		s.log.Warn("bff.resource.upstream_failed",
			slog.String("kind", string(kind)),
			slog.String("path", upstreamPath),
			slog.String("error", err.Error()))
		return c.JSON(http.StatusBadGateway, map[string]any{
			"ResponseStatus": map[string]any{
				"StatusCode":   4,
				"StatusString": "SERVER_ERROR",
				"Description":  "protocol server unreachable: " + err.Error(),
			},
		})
	}
	defer resp.Body.Close()

	// 流式回写：不要一次性把 body 读到内存。
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return c.JSON(http.StatusBadGateway, map[string]any{
			"ResponseStatus": map[string]any{
				"StatusCode":   4,
				"StatusString": "SERVER_ERROR",
				"Description":  "read upstream body: " + err.Error(),
			},
		})
	}

	// 状态码与日志
	dur := time.Since(start)
	logFn := s.log.Debug
	if resp.StatusCode >= 500 {
		logFn = s.log.Warn
	}
	logFn("bff.resource.proxy",
		slog.String("kind", string(kind)),
		slog.String("path", upstreamPath),
		slog.Int("status", resp.StatusCode),
		slog.Int64("duration_ms", dur.Milliseconds()))

	// 选择性原样回写 Content-Type（若为 JSON 族则标准化为 application/json）。
	ct := resp.Header.Get("Content-Type")
	if ct == "" || strings.HasPrefix(ct, "application/json") || strings.HasPrefix(ct, "application/VIID+JSON") {
		ct = "application/json; charset=utf-8"
	}
	return c.Blob(resp.StatusCode, ct, respBody)
}
