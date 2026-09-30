## ADDED Requirements

### Requirement: Scenario 必须可以从 YAML 加载

`LoadAll(dir)` 方法必须读取目录下全部 `*.yaml` 文件并解析为 `Scenario` 值。YAML 畸形的文件必须导致报错。

#### Scenario: 合法的 YAML 文件
WHEN 对包含一个合法 YAML 文件的目录调用 `LoadAll(dir)`
THEN 必须返回含一个元素的切片。

#### Scenario: 畸形的 YAML
WHEN 对包含一个畸形 YAML 文件的目录调用 `LoadAll(dir)`
THEN 必须返回 error。

### Requirement: Scenario 必须支持四类以上资源 Kind

每个 Scenario 必须能声明如下 Kind 的资源：Person、Face、Vehicle、Plate、NonMotorVehicle、Image、Object。

### Requirement: 每个 Node 必须恰好产出一个 runnable

对 YAML 中的每条 `NodeSpec`，引擎必须恰好产出一个 `Runnable`。Runnable 的 `Start` 必须为每条资源声明最多启动一个 goroutine。

### Requirement: ScenarioEngine.Start 必须遵循 autoStart

当 `ScheduleSpec.AutoStart=true` 时，调用 `engine.AutoStart(ctx)` 必须启动该场景。为 false 时，该场景不得被自动启动。

### Requirement: ResourceFactory 必须产出确定性的 ID

对给定的 `seed` 与 `seq`，工厂每次调用必须产出相同的 `Resource.ID`。这对 golden sample 测试至关重要。

### Requirement: FaultInjector 必须是概率化且可复现的

对给定的 seed，同一 target 在多次运行中必须产生相同的故障决策（按 target 使用显式种子的 `math/rand`）。

### Requirement: 引擎必须支持无引擎运行

当以 nil 的 `Runnable` 工厂构造（如测试用）时，`Start` 必须把场景标记为 running，但不得启动任何 goroutine。`IsRunning(id)` 仍必须返回 true。

### Requirement: 引擎必须支持对未注册场景的 Start/Stop

`Start(unknownID)` 必须返回 `ErrScenarioNotFound`。

---

## ADDED Architecture Decisions

### Decision: YAML 是场景的唯一事实来源

场景定义在 YAML 文件中，而不是 Go 代码里。这让运维可以在不重编译的情况下替换场景。

### Decision: 引擎不持有 HTTP server

每个节点的 HTTP server（当它作为平台时）由模拟器的主引导启动，而非引擎。引擎只驱动客户端侧行为（Register、推送资源）。

### Decision: ResourceFactory 默认为 Fake

如果场景省略 `factory:` 键，则使用 `FakeFactory`。这让场景文件保持精简。