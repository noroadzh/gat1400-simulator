// Package scenario 描述模拟器使用的场景包（Scenario）。
//
// Scenario 编码以下信息：
//   - 场景引擎如何按节奏（pace）生成资源
//   - 构建哪些订阅关系
//   - 启用哪些异常注入
//
// 场景以 YAML 文件形式存放在 configs/scenarios 目录下，由 loader 在启动时批量加载。
package scenario

import (
	"time"

	"github.com/noroadzh/gat1400-simulator/internal/domain/node"
	"github.com/noroadzh/gat1400-simulator/internal/domain/resource"
)

// Scenario YAML 根聚合。一个 YAML 文件对应一个 Scenario。
type Scenario struct {
	ID          string         `json:"id"          yaml:"id"`
	Name        string         `json:"name"        yaml:"name"`
	Description string         `json:"description" yaml:"description"`
	Tags        []string       `json:"tags"        yaml:"tags"`
	Nodes       []NodeSpec     `json:"nodes"       yaml:"nodes"`
	Subscriptions []SubscriptionSpec `json:"subscriptions" yaml:"subscriptions"`
	Resources   []ResourceSpec `json:"resources"   yaml:"resources"`
	Exceptions  ExceptionSpec  `json:"exceptions"  yaml:"exceptions"`
	Schedule    ScheduleSpec   `json:"schedule"    yaml:"schedule"`
}

// NodeSpec 描述场景启动时需要真实化的一个节点（角色、监听地址、向上游、能力位、标签）。
type NodeSpec struct {
	Ref          string            `json:"ref"          yaml:"ref"`
	Role         node.Role         `json:"role"         yaml:"role"`
	Listen       string            `json:"listen"       yaml:"listen"`
	Upstream     string            `json:"upstream"     yaml:"upstream"`
	Capabilities []node.Capability `json:"capabilities" yaml:"capabilities"`
	Tags         []string          `json:"tags"         yaml:"tags"`
}

// SubscriptionSpec 有向边（订阅方 -> 发布方），针对某个 Topic（资源 Kind）。
type SubscriptionSpec struct {
	SubscriberRef string                  `json:"subscriber" yaml:"subscriber"`
	PublisherRef  string                  `json:"publisher"  yaml:"publisher"`
	Topic         resource.Kind           `json:"topic"      yaml:"topic"`
	Interval      time.Duration           `json:"interval"   yaml:"interval"`
	ReceiveAddr   string                  `json:"receiveAddr" yaml:"receiveAddr"`
}

// ResourceSpec 描述某节点上某种类资源的按节奏产出流。
//
// Rate：每秒产出事件数；Burst：可选初始突发；Seed：随机种子（保证可复现）。
type ResourceSpec struct {
	NodeRef  string        `json:"node"     yaml:"node"`
	Kind     resource.Kind `json:"kind"     yaml:"kind"`
	Rate     int           `json:"rate"     yaml:"rate"`     // events per second
	Burst    int           `json:"burst"    yaml:"burst"`    // optional initial burst
	Seed     int64         `json:"seed"     yaml:"seed"`
}

// ExceptionSpec 异常注入开关。默认全部为 0 / 关闭。
type ExceptionSpec struct {
	RegisterDropRate   float64       `json:"registerDropRate"   yaml:"registerDropRate"` // 0..1
	KeepaliveJitter    time.Duration `json:"keepaliveJitter"    yaml:"keepaliveJitter"`
	SubscribeRejectRate float64      `json:"subscribeRejectRate" yaml:"subscribeRejectRate"`
	LatencySpike       time.Duration `json:"latencySpike"       yaml:"latencySpike"`
}

// ScheduleSpec 控制场景自动启动生命周期。Duration=0 表示永久运行。
type ScheduleSpec struct {
	AutoStart bool          `json:"autoStart" yaml:"autoStart"`
	Duration  time.Duration `json:"duration"  yaml:"duration"` // 0 = run forever
}