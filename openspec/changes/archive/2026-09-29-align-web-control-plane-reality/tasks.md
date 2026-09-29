# Tasks

## 1. Filesystem Corrections

- [x] 1.1 Delete `web/` directory entirely (`rm -rf web/`) and verify `ls web/` returns "No such file or directory"
- [x] 1.2 Append `/internal/ui/dist/` to `.gitignore` with explanatory comment and verify `internal/ui/dist/` is ignored (`git check-ignore -v internal/ui/dist/index.html` returns a path)
- [x] 1.3 Confirm `internal/ui/dist/index.html` still exists locally (served by embed after Change 2 routing update)

## 2. Spec Alignment

- [x] 2.1 Update `openspec/specs/web-bff/spec.md` architecture decision to accurately describe the CDN SPA state and forward-reference Change 3 Vite migration
- [x] 2.2 Run `openspec validate align-web-control-plane-reality` and confirm zero errors

## 3. Verification

- [x] 3.1 `go build ./...` succeeds with zero errors
- [x] 3.2 `go test -race ./internal/ui/...` passes (existing tests unaffected by this change)
- [x] 3.3 `git status` shows only intended files modified/deleted (`web/` gone, `.gitignore` updated, spec delta)
