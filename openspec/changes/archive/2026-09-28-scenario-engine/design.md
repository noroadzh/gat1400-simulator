# 设计：场景引擎

## Scenario YAML 格式

```yaml
id: small-town-v1
name: 小镇 —— 3 路摄像头 + 1 个平台
schedule:
  autoStart: true
  interval: 30s

nodes:
  - id: alpha-1
    role: device
    httpListen: ":19101"
    upstream: "http://localhost:19001"
    capabilities: [system, collection]
    tags: [entrance]

  - id: alpha-2
    role: device
    httpListen: ":19102"
    upstream: "http://localhost:19001"
    capabilities: [system, collection]

  - id: alpha-3
    role: device
    httpListen: ":19103"
    upstream: "http://localhost:19001"
    capabilities: [system, collection]

  - id: alpha-platform
    role: platform-large
    httpListen: ":19001"
    capabilities: [system, collection, cascade]

resources:
  - kind: Person
    source: alpha-1
    interval: 5s
    count: 100
    factory: fake
    attributes:
      gender: ["male", "female"]
      ageRange: [18, 70]

  - kind: Face
    source: alpha-1
    parent: Person
    interval: 5s

  - kind: Vehicle
    source: alpha-2
    interval: 3s
    count: 50

subscribes:
  - id: sub-1
    source: alpha-platform
    destination: alpha-platform
    resource: Person
    criteria:
      gender: male
    window: 24h

faults:
  - target: alpha-1
    type: delay
    probability: 0.1
    duration: 2s

  - target: alpha-2
    type: drop
    probability: 0.05
```

## 引擎结构

```go
type ScenarioEngine struct {
    log       *slog.Logger
    factory   ResourceFactory
    faults    FaultInjector
    scenarios map[string]Scenario
    runnables map[string]Runnable
    running   map[string]bool
}

type Runnable interface {
    Start(ctx context.Context) error
    Stop() error
}
```

引擎并不直接持有 HTTP server。每个 `Node` 声明产出一个 "runnable"，它会：
1. 启动一个按 `Upstream` 配置的 HTTP 客户端（`adapter/wire.Client`）
2. 启动时调用 `Register`
3. 为每条 `Resource` 声明调度一个 `time.Ticker`
4. 每 tick 时通过 `ResourceFactory` 生成 Resource，并由客户端 POST 出去

## 资源工厂

```go
type ResourceFactory interface {
    Make(kind Kind, attrs Metadata) (Resource, error)
}
```

`FakeFactory` 产出随机值：
- Person：随机 ID（UUID）、随机性别、随机年龄（18–90）、随机民族
- Face：随机 face ID、父 Person 引用、伪 JPEG 字节（或空）
- Vehicle：随机号牌、随机颜色、随机品牌
- Plate：随机的 6 位中国车牌
- Image：随机 JPEG 头字节（或空）

## 故障注入器

```go
type FaultInjector interface {
    ShouldFault(target string) (FaultDecision, error)
}

type FaultDecision struct {
    Skip      bool
    Delay     time.Duration
    Reorder   bool
    Malformed bool
}
```

`ProbabilityFaultInjector` 为每个 target 使用独立的种子 RNG，从而保证故障行为在多次运行间可复现（便于 golden sample 测试）。

## 生命周期

1. `engine.LoadAll(scenariosDir)` → 把 YAML 文件读到内存
3. `engine.Scenarios()` → 返回已加载的场景列表
2. `engine.Start(ctx, s)` → 启动默认的 ConfigNode，再启动所有启用的 Client 节点
4. `engine.Stop(id)` → 优雅停止 ConfigNode 与 Clients

引擎 MUST NOT 识别图中的环路。YAML 中的环是用户失误。