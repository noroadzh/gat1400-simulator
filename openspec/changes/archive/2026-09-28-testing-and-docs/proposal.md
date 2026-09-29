# Proposal: Testing Matrix and Documentation

## Status
In Progress — in the process of being completed.

## Motivation

The codebase has 100% unit test coverage on internal packages but is missing:
- **Contract tests**: Golden sample recordings that verify protocol correctness against real wire dumps
- **End-to-end tests**: Real socket dual-process tests that run the full stack (device client → server → DB)
- **Documentation**: ARCHITECTURE.md, PROTOCOL.md, USER_GUIDE.md, OPERATIONS.md, TESTING.md, CHANGELOG.md

## Goals

- `test/contract/golden/` — JSON files with request/response pairs for key protocol sequences
- `test/e2e/` — Go e2e tests that start a real protocol server and exercise it with the real client
- All 6 documentation files in `docs/`

## Non-Goals

- No performance benchmarks (deferred)
- No load/stress testing (deferred to OPERATIONS)
- No documentation for internal implementation details

## Open Questions

None.