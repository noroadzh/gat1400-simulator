# Design: align-web-control-plane-reality

## Overview

This change is a documentation and hygiene correction. No functional code is modified. The goal is to make OpenSpec artifacts reflect reality so future implementors are not misled.

## Current State vs. Specification

```
                        OpenSpec claim               Actual state
                        ─────────────────────────     ──────────────────────────────────────
web/package.json        exists (task [x])            does NOT exist
web/vite.config.ts       exists (task [x])            does NOT exist
web/src/                 has Vue components           empty directory
web/public/              has static assets            empty directory
web/dist/               pnpm build output            hand-written CDN SPA (gitignored)
internal/ui/dist/        referenced in spec           git-tracked CDN SPA (actual source)
```

## Decision: Do Not Modify Archived Change

The archived change `2026-09-28-web-control-plane` is a historical record. Modifying its `tasks.md` would rewrite history. Instead, this change's `proposal.md` documents the deviation explicitly.

## Change 1 Artifacts

| Artifact | Action |
|---|---|
| `proposal.md` | ✅ Written |
| `design.md` | ✅ Written (this file) |
| `specs/web-bff/spec.md` | Delta: update architecture decision text |
| `tasks.md` | Checklist for the two file-system actions |

## File Operations

### 1. Delete `web/` directory

```bash
rm -rf web/
```

Rationale: `web/` is entirely redundant. `web/dist/` was gitignored (`.gitignore` line 15), `web/src/` and `web/public/` were empty. There is nothing of value.

### 2. Update `.gitignore`

Append to `.gitignore`:

```
# Frontend SPA build output (Vite output goes here when npm build is introduced)
# Currently served from internal/ui/dist/ via embed.FS
/internal/ui/dist/
```

### 3. Spec delta: `specs/web-bff/spec.md`

Revise the architecture decision "frontend bundle is embedded at compile time":

**Before:**
> The frontend build artifact (`web/dist/`) is embedded via `embed.FS` and served directly from memory.

**After:**
> The frontend SPA is currently a hand-written CDN-based single-file application served from `internal/ui/dist/` via `embed.FS`. This directory will be replaced by the Vite build output in Change 3 (`vue-componentize-and-dockerize`).

## Build Verification

No build artifacts are changed. The following commands verify the change is safe:

```bash
# Confirm web/ is gone
ls web/ 2>&1  # → No such file or directory

# Confirm internal/ui/dist/ is still present (served by BFF)
ls internal/ui/dist/index.html  # → exists

# Go build still works
go build ./...

# Go tests still pass
go test -race ./...
```

## Open Questions

None. The scope is intentionally minimal.
