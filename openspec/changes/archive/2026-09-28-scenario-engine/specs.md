## ADDED Requirements

### Requirement: Scenario MUST be loadable from YAML

The `LoadAll(dir)` method MUST read all `*.yaml` files in the directory and parse them into `Scenario` values. Files with malformed YAML MUST cause an error.

#### Scenario: Valid YAML file
WHEN `LoadAll(dir)` is called on a directory containing one valid YAML file
THEN it MUST return a slice with one element.

#### Scenario: Malformed YAML
WHEN `LoadAll(dir)` is called on a directory containing a malformed YAML file
THEN it MUST return an error.

### Requirement: Scenario MUST support four resource Kinds

Each Scenario MUST be able to declare resources of Kinds: Person, Face, Vehicle, Plate, NonMotorVehicle, Image, Object.

### Requirement: Each Node MUST produce exactly one runnable

For each `NodeSpec` in the YAML, the engine MUST produce exactly one `Runnable`. The Runnable's `Start` MUST spawn at most one goroutine per resource declaration.

### Requirement: ScenarioEngine.Start MUST respect autoStart

When `ScheduleSpec.AutoStart=true`, calling `engine.AutoStart(ctx)` MUST start that scenario. When false, the scenario MUST NOT be started automatically.

### Requirement: ResourceFactory MUST produce deterministic IDs

For a given `seed` and `seq`, the factory MUST produce the same `Resource.ID` on every invocation. This is critical for golden sample tests.

### Requirement: FaultInjector MUST be probabilistic and reproducible

For a given seed, the same target MUST produce the same fault decisions across runs (using `math/rand` with explicit seed per target).

### Requirement: Engine MUST support engine-less operation

When constructed with a nil `Runnable` factory (e.g., for testing), `Start` MUST mark the scenario as running but MUST NOT spawn any goroutines. `IsRunning(id)` MUST still return true.

### Requirement: Engine MUST support Start/Stop on unregistered scenario

`Start(unknownID)` MUST return `ErrScenarioNotFound`.

---

## ADDED Architecture Decisions

### Decision: YAML is the source of truth for scenarios

Scenarios are defined in YAML files, not Go code. This allows operations to swap scenarios without recompiling.

### Decision: Engine does not own HTTP servers

Each Node's HTTP server (when it acts as a platform) is started by the simulator's main bootstrap, not by the engine. The engine only drives client-side behavior (Register, push resources).

### Decision: ResourceFactory defaults to Fake

If a scenario omits the `factory:` key, `FakeFactory` is used. This keeps scenarios minimal.