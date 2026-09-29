// Package ports 定义应用层（application）所依赖的端口接口。
//
// 六边形架构（Hexagonal Architecture）的关键层：
//   - 所有 concrete adapter（HTTP server、SQLite、Wire client）实现在这些接口后面
//   - domain 和 application 包严格禁止 import concrete adapter
//   - main.go 负责将 concrete adapter 绑定到这些接口
//
// 好处：
//   - 业务逻辑（domain + application）可独立测试（mock 掉 port）
//   - adapter 可随时替换（如把 SQLite 换成 PostgreSQL）
package ports

import (
	"context"

	"github.com/noroadzh/gat1400-simulator/internal/domain/node"
	"github.com/noroadzh/gat1400-simulator/internal/domain/resource"
	"github.com/noroadzh/gat1400-simulator/internal/domain/scenario"
)

// NodeService 应用层管理模拟节点的门面接口。
// HTTP API、Web BFF 与场景引擎共同使用。
type NodeService interface {
	ListNodes(ctx context.Context) ([]node.Node, error)
	GetNode(ctx context.Context, id string) (*node.Node, error)
	UpsertNode(ctx context.Context, n node.Node) (*node.Node, error)
	RemoveNode(ctx context.Context, id string) error
	MarkSeen(ctx context.Context, id string) error
}

// ScenarioService 负责场景包生命周期（加载/启动/停止/查询）。
// 通过 resource factory 产出资源，通过 wire client 按需分发。
type ScenarioService interface {
	ListScenarios(ctx context.Context) []scenario.Scenario
	Start(ctx context.Context, id string) error
	Stop(ctx context.Context, id string) error
	IsRunning(id string) bool
}

// ResourceSink 接收场景引擎产生的资源实体。
// HTTP API 注册此 sink 接收外部客户端的资源写入；recorder 注册此 sink 记录 wire 交易。
type ResourceSink interface {
	OnResource(ctx context.Context, kind resource.Kind, payload any) error
}

// Notifier HTTP API 与场景引擎向上游推送通知的抽象。
// wire adapter 将其翻译为 outbound HTTP 调用；同进程 dispatcher 直接本地投递。
type Notifier interface {
	Dispatch(ctx context.Context, target node.Node, payload any) error
}