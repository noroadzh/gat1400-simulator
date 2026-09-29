# Proposal: Web Control Plane

## Status
Archived — implemented as the BFF + Vue3 frontend.

## Motivation

Operators need a browser-based dashboard to:
- Inspect node status (online/stopped/error) in real-time
- Add/remove nodes without editing YAML
- Start/stop scenarios, see their progress
- Browse resource collections (Person, Face, Vehicle, etc.)
- Configure subscriptions and dispositions
- Review captured HTTP traffic (request/response inspector)

## Goals

- Echo-based BFF with REST API under `/api/control/...`
- WebSocket Hub for real-time events (node status, capture arrival)
- Vue3 + ElementPlus frontend, embedded via `embed.FS`
- 6 pages: Dashboard, Nodes, Scenarios, Resources, Subscriptions, Captures
- No external CDN dependency at runtime (everything bundled)

## Non-Goals

- No multi-user / RBAC
- No historical analytics (only live state)
- No scenario editing (operators use YAML + BFF triggers)

## Open Questions

None.