package node

import "errors"

// 哨兵错误。domain 层 MUST 仅返回（或者包装）以下错误，调用方使用 errors.Is 判定。
// 严禁抛出裸 fmt.Errorf，否则上层无法通过稳定的映射表把错误映射为协议响应。
//
// 注意：error message 保留英文形式（与历史日志、单元测试断言兼容），仅在本注释中提供中文说明。
var (
	ErrNotFound          = errors.New("node: not found")          // 按 ID 查找节点不存在
	ErrDuplicate         = errors.New("node: duplicate id")       // 创建时 ID 冲突
	ErrInvalidRole       = errors.New("node: invalid role")       // Role 不在已知枚举内
	ErrMissingID         = errors.New("node: missing id")         // Node.ID 为空
	ErrMissingName       = errors.New("node: missing name")       // Node.Name 为空
	ErrInvalidCapability = errors.New("node: invalid capability") // Capability 不在已知枚举内
)