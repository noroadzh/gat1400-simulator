## 任务

### 任务：定义 Scenario 领域类型

- [x] `internal/domain/scenario/scenario.go` —— `Scenario`、`ScheduleSpec`、`NodeSpec`、`ResourceSpec`、`SubscribeSpec`、`FaultSpec`

### 任务：YAML 加载器

- [x] `internal/adapter/scenario/loader.go` —— `LoadAll(dir) ([]Scenario, error)`
- [x] 使用 `gopkg.in/yaml.v3` 解析
- [x] 返回错误时带上文件路径上下文

### 任务：资源工厂

- [x] `internal/adapter/scenario/factory.go` —— `FakeFactory`、`StaticFactory`
- [x] `Make(kind, attrs)` 返回 Resource
- [x] 使用带种子的 `math/rand` 保证确定性

### 任务：故障注入器

- [x] `internal/adapter/scenario/faults.go` —— `ProbabilityFaultInjector`
- [x] `ShouldFault(target)` 返回 FaultDecision

### 任务：ScenarioEngine

- [x] `internal/adapter/scenario/engine.go` —— `ScenarioEngine` 结构体
- [x] `Start(ctx, s)`、`Stop(id)`、`AutoStart(ctx)`
- [x] `IsRunning(id)`、`Running() []string`

### 任务：组装 Runnable

- [x] `internal/adapter/scenario/runnable.go` —— 包装 `adapter/wire.Client` 与 ticker 循环
- [x] 每条 ResourceSpec 产出一个 goroutine，在 tick 时 POST

### 任务：示例场景

- [x] `configs/scenarios/single-device.yaml` —— 最小的 1 设备 + 1 平台
- [x] `configs/scenarios/small-town.yaml` —— 3 设备 + 1 平台（含级联）
- [x] `configs/scenarios/load-test.yaml` —— 10 设备 + 高资源推送速率

### 任务：单元测试

- [x] `loader_test.go` —— 合法 YAML、畸形 YAML、缺失文件
- [x] `factory_test.go` —— 确定性、Kind 覆盖
- [x] `faults_test.go` —— 概率正确性、可复现性
- [x] `engine_test.go` —— start/stop、autoStart、未知 id

### 验证

- `go test ./internal/adapter/scenario/...` → exit 0
- `configs/scenarios/*.yaml` 全部可被正常解析
- `engine.AutoStart` 仅启动 `autoStart: true` 的场景