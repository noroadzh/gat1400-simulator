## Tasks

### Task: Define Client struct and PostJSON

- [x] `client.go` — `Client` struct, `NewClient(...)`, `PostJSON(ctx, path, body) (*http.Response, error)`
- [x] Set `Content-Type: application/VIID+JSON` on all POST/PUT
- [x] Attach `User-Identify` header

### Task: Implement parseChallenge

- [x] `digest.go::parseChallenge(header string) (challenge, error)`
- [x] Handle quoted values, comma-separated directives
- [x] Return typed struct: Realm, Nonce, Qop, Opaque, Algorithm

### Task: Implement buildAuthorization

- [x] `digest.go::buildAuthorization(method, uri, username, password, realm, nonce, qop, nc, cnonce) string`
- [x] Compute HA1, HA2, response per RFC 2617 §3.2.2
- [x] Return `Authorization: Digest ...` header value

### Task: Implement digestPassword

- [x] `digest.go::digestPassword(username, realm, password) string`
- [x] `MD5(username:realm:password)` as lowercase hex

### Task: Implement NonceStore

- [x] `nonce_store.go` — wraps `storage.NewNonceStore`
- [x] `NonceStore.Issue(nonce, realm)` — record nonce
- [x] `NonceStore.Issue(nonce, realm)` returns error if nonce already consumed
- [x] `NonceStore.Close()`

### Task: Wire PostJSON with 401 retry

- [x] In `Client.PostJSON`: first request without auth
- [x] If 401: parse WWW-Authenticate, build Authorization, retry
- [x] Any other status code: return as-is

### Task: Unit tests

- [x] `digest_test.go` — parseChallenge coverage, buildAuthorization field presence
- [x] Verify HA1/HA2 values against RFC 2617 test vectors

### Verification

- `go test ./internal/adapter/wire/...` → exit 0
- `go test -race ./internal/adapter/wire/...` → exit 0
- Integration: start httpapi server, use wire client to Register → 200 OK