# Proposal: HTTP Client with Digest Auth

## Status
Archived — implemented as the protocol client adapter.

## Motivation

The simulator must act as a GAT 1400 **device** (UAC), pushing resources to a platform (UAS) via HTTP. The platform protects `/VIID/System/Register` and `/VIID/System/UnRegister` with HTTP Digest auth. The client must:

1. Send a request → receive `401 Unauthorized` with `WWW-Authenticate: Digest realm="...", nonce="...", qop="auth", opaque="..."`
2. Compute the digest response
3. Retry the request with `Authorization: Digest ...`
4. Persist the nonce for reuse within the validity window

## Goals

- Fully transparent auth: caller sees only the successful response
- Persistent nonce store (SQLite) survives client restarts
- `User-Identify` header on every request
- `Content-Type: application/VIID+JSON` on every request

## Non-Goals

- No TLS (handled at deployment level — nginx sidecar)
- No connection pooling beyond stdlib HTTP transport
- No multi-part upload

## Open Questions

None.