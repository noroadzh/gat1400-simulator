# Design: Scenario Engine

## Scenario YAML Format

```yaml
id: small-town-v1
name: Small town — 3 cameras, 1 platform
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

## Engine Structure

```go
type ScenarioEngine struct {
    log      *slog.Logger
    factory  ResourceFactory
    faults   FaultInjector
    nodes    []Node
    runnables map[string]Runnable
    running   map[id]bool
}

type Runnable interface {
    Start(ctx context.Context) error
    Stop() error
}
```

The engine does not own HTTP servers directly. Instead, each `Node` declaration produces a "runnable" that:
1. Spawns an HTTP client (`adapter/wire.Client`) configured with `Upstream`
2. Calls `Register` at startup
3. Schedules a `time.Ticker` for each `Resource` declaration
4. For each tick, generates a Resource via `ResourceFactory` and POSTs via the client

## Resource Factory

```go
type ResourceFactory interface {
    Make(kind Kind, attrs Metadata) (Resource, error)
}
```

`FakeFactory` produces random values:
- Person: random ID (UUID), random gender, random age (18-90), random ethnicity
- Face: random face ID, parent Person reference, fake JPEG bytes (or empty)
- Vehicle: random plate, random color, random brand
- Plate: random 6-char Chinese plate
- Image: random JPEG header bytes (or empty)

## Fault Injector

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

`ProbabilityFaultInjector` uses a seeded RNG per target so that the fault behavior is reproducible across runs (useful for golden sample tests).

## Lifecycle

1. `engine.LoadAll(scenariosDir)` → reads YAML files into memory
2. `engine.Scenarios()` → returns the loaded scenarios
3. `engine.Start(ctx, s)` → starts the default ConfigNode, then all enabled Client nodes
4. `engine.Stop(id)` → gracefully stops the ConfigNode and Clients

The engine DOES NOT implement graph loops. A loop in the scenario YAML is a user mistake.