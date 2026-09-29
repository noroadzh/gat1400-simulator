# Proposal: Domain Models

## Status
Archived — all domain types defined and tested.

## Motivation

The GAT 1400 protocol revolves around a small set of persistent entities — Nodes, Resources, Subscriptions, Dispositions — and a uniform response envelope. Before building any HTTP layer or scenario engine, we need stable, well-tested domain types that encode the protocol's invariants:
- DeviceID is a 20-digit string in layout `8+2+2+2+6`
- Resource kinds (Person, Face, Vehicle, Plate, Image, Object, NonMotorVehicle) are an exhaustive enum
- HTTP Status codes (0/1/2/3/4) map to protocol-level codes (OK/Invalid/NotFound/Unauthorized/ServerError)

## Goals

- Domain models in `internal/domain/` with zero external dependencies
- ID generator for DeviceID, UUID, Nonce
- Validation functions (`Sanity()`) on each entity that return sentinel errors
- Comprehensive unit tests covering all valid + invalid combinations

## Non-Goals

- No persistence (handled by adapter-storage in a later change)
- No HTTP DTOs (handled by adapter-httpapi)
- No UI representation (handled by ui server bindings)

## Open Questions

None.