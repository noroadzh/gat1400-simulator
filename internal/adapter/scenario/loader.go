// Package scenario 承载场景引擎 adapter。
//
// 职责：
//   - 把 configs/scenarios 下的 YAML 场景文件解析为 scenario.Scenario
//   - 按节奏（pacing）生成资源 goroutine
//   - 编排订阅关系（Subscriber -> Publisher）
//
// 分层纪律：
//   - adapter 是唯一知道 goroutine、timer、HTTP dispatch 的地方
//   - domain 层保持传输无关性，便于独立测试
package scenario

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/noroadzh/gat1400-simulator/internal/domain/scenario"
)

// Loader 扫描 YAML 目录并解析为 scenario.Scenario 切片。
//
// 设计原则：
//   - goroutine-safe（无内部可变状态，天然可并发调用）
//   - 宽容：解析错误返回（文件名 + 错误），其他文件仍可加载
//   - 多文件同名 ID 时 last-wins
type Loader struct{}

// NewLoader 构造一个空状态的 loader。
func NewLoader() *Loader { return &Loader{} }

// LoadDir 遍历 dir（不递归子目录），解析每个 .yaml/.yml 文件。
// 非 YAML 文件静默忽略；返回的 map 以场景 ID 为 key，多文件共存时 last-wins。
func (l *Loader) LoadDir(dir string) (map[string]scenario.Scenario, []error) {
	out := map[string]scenario.Scenario{}
	var errs []error
	if dir == "" {
		return out, []error{fmt.Errorf("scenarios: empty directory")}
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return out, []error{fmt.Errorf("scenarios: read %s: %w", dir, err)}
	}
	var paths []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			continue
		}
		paths = append(paths, filepath.Join(dir, name))
	}
	sort.Strings(paths)
	for _, p := range paths {
		s, err := l.LoadFile(p)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", filepath.Base(p), err))
			continue
		}
		out[s.ID] = s
	}
	return out, errs
}

// LoadFile 解析单个 YAML 文件。
// 磁盘格式与 scenario.Scenario 字段一一对应；YAML 注释保留。
// ID 缺省时回退到文件名（不含扩展名）。
func (l *Loader) LoadFile(path string) (scenario.Scenario, error) {
	var s scenario.Scenario
	b, err := os.ReadFile(path)
	if err != nil {
		return s, fmt.Errorf("read: %w", err)
	}
	if err := yaml.Unmarshal(b, &s); err != nil {
		return s, fmt.Errorf("yaml: %w", err)
	}
	if s.ID == "" {
		s.ID = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}
	return s, nil
}
