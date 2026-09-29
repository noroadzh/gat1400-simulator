## Tasks

### Task: Define Scenario domain type

- [x] `internal/domain/scenario/scenario.go` — `Scenario`, `ScheduleSpec`, `NodeSpec`, `ResourceSpec`, `SubscribeSpec`, `FaultSpec`

### Task: YAML loader

- [x] `internal/adapter/scenario/loader.go` — `LoadAll(dir) ([]Scenario, error)`
- [x] Use `gopkg.in/yaml.v3` for parsing
- [x] Return errors with file path context

### Task: Resource Factory

- [x] `internal/adapter/scenario/factory.go` — `FakeFactory`, `StaticFactory`
- [x] `Make(kind, attrs)` returns Resource
- [x] Use seeded `math/rand` for determinism

### Task: Fault Injector

- [x] `internal/adapter/scenario/faults.go` — `ProbabilityFaultInjector`
- [x] `ShouldFault(target)` returns FaultDecision

### Task: ScenarioEngine

- [x] `internal/adapter/scenario/engine.go` — `ScenarioEngine` struct
- [x] `Start(ctx, s)`, `Stop(id)`, `AutoStart(ctx)`
- [x] `IsRunning(id)`, `Running() []string`

### Task: Wire up Runnable

- [x] `internal/adapter/scenario/runnable.go` — wraps `adapter/wire.Client` and a ticker loop
- [x] Each ResourceSpec produces a goroutine that POSTs on tick

### Task: Sample scenarios

- [x] `configs/scenarios/single-device.yaml` — minimal 1 device, 1 platform
- [x] `configs/scenarios/small-town.yaml` — 3 devices, 1 platform with cascade
- [x] `configs/scenarios/load-test.yaml` — 10 devices, high resource rate

### Task: Unit tests

- [x] `loader_test.go` — valid YAML, malformed YAML, missing files
- [x] `factory_test.go` — determinism, kind coverage
- [x] `faults_test.go` — probability correctness, reproducibility
- [x] `engine_test.go` — start/stop, autoStart, unknown id

### Verification

- `go test ./internal/adapter/scenario/...` → exit 0
- Load `configs/scenarios/*.yaml` → all parse without error
- `engine.AutoStart` starts exactly the scenarios with `autoStart: true`