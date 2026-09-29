package ports

import (
	"context"
	"log/slog"

	"github.com/noroadzh/gat1400-simulator/internal/domain/resource"
)

// NotifyResource 下游场景可挂载的钩子，用于接收场景引擎产生的资源事件。
// 默认实现是 no-op；scenario-engine adapter 在启动时注入真正的 dispatcher。
// 注意：这是包级变量（非接口），仅用于解耦加载顺序。
var NotifyResource = func(_ context.Context, _ *slog.Logger, _ resource.Kind, _ string) error { return nil }