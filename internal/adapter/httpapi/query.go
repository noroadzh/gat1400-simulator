package httpapi

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/noroadzh/gat1400-simulator/internal/domain/resource"
)

// reservedQueryKeys 不参与对象字段过滤的分页/控制参数。
var reservedQueryKeys = map[string]struct{}{
	"PageRecordNum": {}, "RecordStartNo": {}, "pageNum": {}, "pageSize": {},
}

// parsePage 解析 GA/T 风格分页：RecordStartNo（从 1 起）+ PageRecordNum。
// 缺省：start=1，pageSize=0 表示不分页（返回全部）。
func parsePage(c echo.Context) (start, pageSize int) {
	start = 1
	if v := c.QueryParam("RecordStartNo"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			start = n
		}
	}
	if v := c.QueryParam("PageRecordNum"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			pageSize = n
		}
	}
	// BFF 兼容别名
	if pageSize == 0 {
		if v := c.QueryParam("pageSize"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				pageSize = n
			}
		}
	}
	if v := c.QueryParam("pageNum"); v != "" && pageSize > 0 {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			start = (n-1)*pageSize + 1
		}
	}
	return start, pageSize
}

// filterObjects 按 query 中与对象字段同名的键做精确匹配（字符串化比较）。
func filterObjects(c echo.Context, items []map[string]any) []map[string]any {
	filters := map[string]string{}
	for k, vals := range c.QueryParams() {
		if _, skip := reservedQueryKeys[k]; skip {
			continue
		}
		if len(vals) == 0 || vals[0] == "" {
			continue
		}
		filters[k] = vals[0]
	}
	if len(filters) == 0 {
		return items
	}
	out := make([]map[string]any, 0, len(items))
	for _, obj := range items {
		ok := true
		for k, want := range filters {
			got, exists := obj[k]
			if !exists || stringify(got) != want {
				ok = false
				break
			}
		}
		if ok {
			out = append(out, obj)
		}
	}
	return out
}

func stringify(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		if t == float64(int64(t)) {
			return strconv.FormatInt(int64(t), 10)
		}
		return strconv.FormatFloat(t, 'f', -1, 64)
	case int:
		return strconv.Itoa(t)
	case int64:
		return strconv.FormatInt(t, 10)
	case bool:
		return strconv.FormatBool(t)
	case nil:
		return ""
	default:
		b, err := json.Marshal(t)
		if err != nil {
			return fmt.Sprint(t)
		}
		return string(b)
	}
}

// paginate 对已过滤列表切片；返回页内数据与总数。
func paginate(items []map[string]any, start, pageSize int) (page []map[string]any, total int) {
	total = len(items)
	if pageSize <= 0 {
		return items, total
	}
	idx := start - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= total {
		return []map[string]any{}, total
	}
	end := idx + pageSize
	if end > total {
		end = total
	}
	return items[idx:end], total
}

// collectIDsFromQuery 从查询串提取待删/待操作 ID。
// 支持：<Kind>ID=a,b | IDList=a,b | <Kind>IDList=a,b | 重复同名参数。
func collectIDsFromQuery(c echo.Context, kind resource.Kind) []string {
	idField := resource.IDOf(kind)
	candidates := []string{
		c.QueryParam(idField),
		c.QueryParam(idField + "List"),
		c.QueryParam("IDList"),
	}
	seen := map[string]struct{}{}
	var out []string
	add := func(raw string) {
		for _, p := range strings.Split(raw, ",") {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			if _, ok := seen[p]; ok {
				continue
			}
			seen[p] = struct{}{}
			out = append(out, p)
		}
	}
	for _, raw := range candidates {
		add(raw)
	}
	// 重复参数：?PersonID=a&PersonID=b
	for _, v := range c.QueryParams()[idField] {
		add(v)
	}
	return out
}

// collectIDsFromBody 从 body 取 <Kind>IDList / IDList / DeleteOperate。
func collectIDsFromBody(body map[string]any, kind resource.Kind) []string {
	if body == nil {
		return nil
	}
	idField := resource.IDOf(kind)
	if ids, ok := extractDeleteIDs(body, idField+"List"); ok {
		return ids
	}
	if ids, ok := extractDeleteIDs(body, "IDList"); ok {
		return ids
	}
	return nil
}
