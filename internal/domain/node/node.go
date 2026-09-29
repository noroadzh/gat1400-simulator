// Package node 定义模拟器使用的核心节点聚合（Node Aggregate）。
//
// 重要设计：GA/T 1400 系列不像 GB/T 28181 那样将通信双方固定为 server/client 二元结构。
// 任意节点可以同时既"提供"又"消费" VIID REST 接口。
// - Role（节点身份）表达的是协议层面的实体身份（采集设备/小平台/大平台/上级），
//   并不直接决定其能做什么。
// - Capability（能力位）才决定该节点实际"挂载"了哪些路由组。
//   换言之，Role 是协议声明，Capability 是当前状态。
package node

import "time"

// Role 节点对外声明的"身份标签"。决定场景引擎与控制台如何展示该节点。
//
// 注意：Role 仅是协议实体身份，不决定能调用哪些路由——路由可见性由 Capability + 监听端口共同决定。
type Role string

const (
	RoleDevice        Role = "device"         // 采集设备 / APE / APS，主体提供 System + Collection
	RolePlatformSmall Role = "platform-small" // 小平台（视图库能力相对较弱）
	RolePlatformLarge Role = "platform-large" // 大平台 / 上级平台
)

// Capability 节点当前对外提供（或订阅）的 REST 路由组。
//
// 该字段仅做声明式记录，是否真正可达由 HTTP 路由器决定（路由器实际注册的中间件与处理函数才权威）。
type Capability string

const (
	CapSystem       Capability = "system"        // /VIID/System/* —— 注册、心跳、时间同步
	CapCollection   Capability = "collection"    // 资源集合接口（Persons/Faces/Vehicles...）
	CapDataService  Capability = "data-service"  // 数据服务接口（订阅推送接收）
	CapCascade      Capability = "cascade"       // 级联相关（订阅管理、告警上报）
	CapControl      Capability = "control"       // 本机控制面（BFF/控制台接口）
	CapCaptureAgent Capability = "capture-agent" // 对外采集代理（作为 client 调用上级）
)

// Status 节点运行态。状态变迁由 NodeService 统一管理，外部禁止直接修改。
type Status string

const (
	StatusStopped Status = "stopped" // 未启动或已停止
	StatusOnline  Status = "online"  // 在线，最近一次心跳未超时
	StatusOffline Status = "offline" // 离线，心跳超时
	StatusError   Status = "error"   // 异常（启动失败、协议错误等）
)

// Node 核心节点聚合。零值无效；构造后请调用 Sanity() 做字段完整性校验。
type Node struct {
	ID            string         `json:"id"            yaml:"id"`
	Name          string         `json:"name"          yaml:"name"`
	Role          Role           `json:"role"          yaml:"role"`
	Description   string         `json:"description"   yaml:"description"`
	HTTPListen    string         `json:"httpListen"    yaml:"httpListen"`
	Upstream      string         `json:"upstream"      yaml:"upstream"`        // peer VIID base URL when acting as client
	Capabilities  []Capability   `json:"capabilities"  yaml:"capabilities"`
	Status        Status         `json:"Status"        yaml:"Status"`
	LastSeenAt    time.Time      `json:"lastSeenAt"    yaml:"lastSeenAt"`
	Tags          []string       `json:"tags"          yaml:"tags"`
	CreatedAt     time.Time      `json:"createdAt"     yaml:"createdAt"`
	UpdatedAt     time.Time      `json:"updatedAt"     yaml:"updatedAt"`
	Metadata      map[string]any `json:"metadata,omitempty" yaml:"metadata"`
}

// HasCapability 报告节点是否声明了指定能力。
//
// 注意：该方法只判断"声明"，不判断"实际可达"——如需判断实际可达性，请向 NodeService 查询。
func (n Node) HasCapability(c Capability) bool {
	for _, x := range n.Capabilities {
		if x == c {
			return true
		}
	}
	return false
}

// Sanity 对 Node 聚合做必填字段与枚举合法性校验。
//
// 不校验 ID 编码规则（20 位数字等）——编码合法性由 internal/domain/ids 包负责。
// 这样 domain 层只关注语义与枚举，不耦合协议字节约束，保持纯净。
func (n Node) Sanity() error {
	if n.ID == "" {
		return ErrMissingID
	}
	if n.Name == "" {
		return ErrMissingName
	}
	switch n.Role {
	case RoleDevice, RolePlatformSmall, RolePlatformLarge:
	default:
		return ErrInvalidRole
	}
	return nil
}