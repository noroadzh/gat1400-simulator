package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// mkdir os.MkdirAll 的可替换桩，供测试替换为伪文件系统。
var mkdir = os.MkdirAll

// tempFile 在 OS 临时目录下创建唯一文件名（pattern 必须包含 *.xxx 后缀）。
// 返回已打开的 *os.File，调用方必须 Close。
func tempFile(pattern string) (*os.File, error) {
	dir := filepath.Join(os.TempDir(), "gat1400")
	if err := mkdir(dir, 0o755); err != nil {
		return nil, err
	}
	return os.CreateTemp(dir, pattern)
}

// encodeJSONAny 把任意 JSON 对象渲染为字符串。
//   - nil 返回空字符串
//   - 序列化失败返回空字符串（不向上抛错，因为 HAR 字段允许缺失）
//
// 仅用于 HAR 导出的 postData / content 内联字段。
func encodeJSONAny(v any) string {
	if v == nil {
		return ""
	}
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}
