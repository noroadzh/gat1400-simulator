# Design: HTTP Client with Digest Auth

## Client Structure

```
internal/adapter/wire/
├── client.go    # Client struct, http.Client wrapping, PostJSON
├── digest.go    # parseChallenge, buildAuthorization, digestPassword
└── nonce_store.go # NonceStore wrapper over storage.NonceStore
```

```go
type Client struct {
    BaseURL   string        // e.g. "http://localhost:19001"
    NodeID    string        // User-Identify value
    User      string        // Digest username
    Password  string        // Digest password
    HTTP      *http.Client  // stdlib, configurable timeout
    nonces    *NonceStore   // SQLite-backed nonce persistence
    idGen     ids.Generator
    log       *slog.Logger
}
```

## Auth Flow

```
Client.PostJSON(ctx, "/VIID/System/Register", body)

  Step 1: POST /VIID/System/Register (no Authorization)
  ← 401 WWW-Authenticate: Digest realm="viid", nonce="abc", qop="auth", opaque=""

  Step 2: Parse challenge → NonceStore.Issue(nonce, serverRealm)
  Step 3: Build Authorization header (RFC 2617 §3.2.2)
  Step 4: POST /VIID/System/Register (with Authorization)
  ← 200 OK + ResponseStatus.StatusCode=0
```

## Digest Computation

```
HA1  = MD5(username ":" realm ":" password)
HA2  = MD5(method ":" requestURI)
resp = MD5(HA1 ":" nonce ":" nc ":" cnonce ":" qop ":" HA2)
```

Parameters:
- `username`, `password`, `realm` — from challenge / config
- `nonce` — from server `WWW-Authenticate`
- `nc` — nonce counter, 8 hex digits, increments per request (e.g. `00000001`)
- `cnonce` — client nonce, 16 hex digits, generated per request via `ids.Nonce()`
- `qop` — `"auth"` (auth-int not supported)
- `algorithm` — `"MD5"` (MD5-sess not supported)

## Nonce Persistence

NonceStore wraps `storage.NonceStore` (SQLite). On a server restart, previously issued nonces may still be valid. The client stores nonces it has used to avoid replay. The SQLite file lives at `data/nonces.db`.

## PostJSON Signature

```go
func (c *Client) PostJSON(ctx context.Context, path string, body any) (*http.Response, error)
```

The body is JSON-encoded. The Content-Type is always `application/VIID+JSON`. The caller is responsible for closing the response body.