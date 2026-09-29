# Proposal: align-web-control-plane-reality

## Why

The web control plane implementation diverged from its OpenSpec specification. The archived change `2026-09-28-web-control-plane` claimed delivery of `web/package.json`, `web/vite.config.ts`, and `pnpm build` tooling—none of which exist. The frontend is currently a hand-written 380-line CDN-based single-file SPA, not a Vite-built Vue 3 application. This change formally acknowledges the gap and establishes the corrected state.

## What Changes

- **Delete** the `web/` directory (empty `src/` and `public/`, gitignored `dist/`, no npm project files)
- **Add** `/internal/ui/dist/` to `.gitignore` — it becomes the sole SPA artifact location, replacing the `web/dist/` reference in spec
- **Declare** in `specs/web-bff/spec.md` that the frontend is currently a CDN-based single-file SPA, and that Change 3 (`vue-componentize-and-dockerize`) will introduce the Vite build pipeline
- The archived change `2026-09-28-web-control-plane` is **not modified** (it is history); this proposal documents the deviation for future readers

## Capabilities

### Modified Capabilities

- **web-bff** (`openspec/specs/web-bff/spec.md`): Revise the "frontend bundle is embedded at compile time" architecture decision to accurately reflect the current state: "frontend is a hand-written CDN-based SPA served from `internal/ui/dist/`". Add a forward reference to the planned Vite migration in Change 3.

### New Capabilities

None.

## Impact

- **Code**: `web/` directory deleted; `.gitignore` updated; `openspec/specs/web-bff/spec.md` delta
- **Git history**: No destructive rewrites; `web/` was untracked in the latest commit
- **Downstream**: Change 2 (`spa-root-routing`) and Change 3 (`vue-componentize-and-dockerize`) build on the corrected state established here
