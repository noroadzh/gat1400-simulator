# Proposal: GAT 1400 HTTP API Adapter

## Status
Archived — implemented as the protocol REST router.

## Motivation

GA/T 1400.4 specifies a REST API with four route families:
- `/VIID/System/...` — Register, UnRegister, Keepalive, Time
- `/VIID/<Resource>...` — Collection (Person, Face, Vehicle, etc.)
- `/VIID/Subscribes`, `/VIID/SubscribeNotifications`, `/VIID/Dispositions` — Cascade
- `/VIID/APEs`, `/VIID/APSs`, `/VIID/Tollgates`, `/VIID/Lanes` — Catalog

We need a server that:
- Speaks `application/VIID+JSON` content type
- Authenticates via HTTP Digest (RFC 2617, qop=auth)
- Accepts `User-Identify` header from the client
- Returns a uniform `ResponseStatus` envelope

## Goals

- Echo-based server with one route per protocol verb
- Custom binder to handle both `application/VIID+JSON` and `application/json`
- Digest auth middleware with persistent nonce store (SQLite)
- User-Identify middleware that calls `NodeService.MarkSeen`
- Capture middleware that records every inbound/outbound request

## Non-Goals

- Client-side retry handled in `adapter-wire`
- Authentication secrets generation handled by `app/config`
- TLS termination (deferred to OPERATIONS doc — typically deployed behind nginx)

## Open Questions

None.