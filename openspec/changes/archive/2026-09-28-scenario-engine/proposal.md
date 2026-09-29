# Proposal: Scenario Engine

## Status
Archived — implemented as the YAML-driven orchestration engine.

## Motivation

Manually configuring 50+ nodes and resource sequences in code is not maintainable. We need:
- A YAML-based scenario format that captures node topology, resource sequences, subscriptions, and fault injection
- An engine that loads a scenario file and brings up all nodes
- A resource factory that produces realistic Person/Face/Vehicle objects (random attributes, optional pre-canned images)
- A subscription matrix that links subscribes between platforms (cascade)
- A fault injection framework for testing edge cases (delays, packet loss, malformed bodies)

## Goals

- YAML scenario format (`*.yaml` files in `configs/scenarios/`)
- `ScenarioEngine.Start(ctx, s)` and `ScenarioEngine.Stop(id)` methods
- ResourceFactory interface with `Fake` (random) and `Static` (from file) implementations
- FaultInjector with `Delay`, `Drop`, `Reorder`, `Malformed` types
- Auto-start support (`ScheduleSpec.AutoStart: true`)

## Non-Goals

- No scenario editing UI (handled by web-control-plane)
- No live migration of running scenarios
- No distributed coordination across processes

## Open Questions

None.