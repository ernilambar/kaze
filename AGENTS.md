# AGENTS.md

## Overview

Kova is a desktop text editor built with Wails v2 (Go backend) and Monaco Editor
(Svelte 5 + TypeScript frontend bundled by Vite). The Go binary embeds
`frontend/dist` and serves it as the app's assets; exported `App` methods are
exposed to the UI via generated Wails bindings.

## Setup

```sh
go install github.com/wailsapp/wails/v2/cmd/wails@latest
go mod download
cd frontend && pnpm install
```

Requires Go 1.26+ and Node.js 20+. The Wails CLI is installed to `~/go/bin`;
ensure that directory is on `PATH`. No `.env` or other local config is needed.

## Commands

```sh
wails dev                              # dev mode with hot reload
wails build                            # packaged app in build/bin
go build ./...                         # compile Go only
go test -race ./...                    # Go tests
go vet ./...                           # Go lint
gofmt -w .                             # Go format (write)
cd frontend && pnpm run build           # svelte-check + vite build to frontend/dist
cd frontend && pnpm run check           # svelte-check (Svelte + TypeScript)
cd frontend && pnpm test               # Vitest frontend tests
./scripts/dev.sh                       # debug build installed to /Applications
./scripts/bundle.sh                    # release gate + dist/*-macos-arm64.zip
./scripts/release.sh v0.1.0            # tag/push; CI builds the release
```

There is no frontend linter/formatter configured (no ESLint/Prettier); do not
introduce one unprompted.

## Conventions

- **Bindings are generated.** `frontend/wailsjs/` is produced by Wails; never
  hand-edit it. Add frontend-callable behavior as exported methods on `App` in
  `app.go`, then let `wails dev`/`wails build` regenerate bindings.
- **Go owns native behavior.** File I/O, dialogs, and OS integration belong in
  Go; the Svelte/TypeScript layer handles UI only.
- **User settings live in `settings.json`.** `settings.go` loads and validates
  it from the platform config directory (`Kova/settings.json`), creating
  defaults on first run; `App.GetSettings` exposes it to the frontend, which
  applies the editor font values at startup. Keep `frontend/src/settings.ts`
  defaults in sync with `defaultSettings()` in Go (the TS copy is only the
  fallback when the backend call fails).
- **Monaco stays excluded from Vite pre-bundling.** `optimizeDeps.exclude:
  ['monaco-editor']` exists because pre-bundling breaks worker URL resolution;
  keep it unless you also rework worker loading.
- **Embedded assets are build output.** `main.go` uses
  `//go:embed all:frontend/dist`, so `frontend/dist` must exist before building
  Go; run `pnpm run build` (or `wails build`) first.
- **Style:** idiomatic Go via `gofmt`; TypeScript is strict (`noUnusedLocals`,
  `noUnusedParameters`, `noImplicitReturns`) and uses 4-space indentation,
  single quotes, and named exports (`export function createEditor`).
- **Svelte 5 with runes.** Components live in `frontend/src/*.svelte`
  (`App.svelte` plus `TitleBar`, `EditorPane`, `StatusBar`), mounted from
  `main.ts` with `mount()`; `svelte.config.js` pins `runes: true`. Use `$state`,
  `$props`, and `$derived`/`$effect` rather than legacy `export let`/`$:` syntax,
  and keep component styles scoped in the component's `<style>` block while
  shared design tokens stay in `style.css`.
- **Use pnpm for the frontend.** `frontend/pnpm-lock.yaml` is the lockfile;
  never add `package-lock.json`. `frontend/pnpm-workspace.yaml` allows the
  `esbuild` install script because pnpm blocks dependency build scripts by
  default.
- **Tests are colocated.** Go tests sit beside the code (`app_test.go`);
  frontend tests use Vitest (`*.test.ts`) and cover pure helpers only, not
  Svelte components, Monaco, or generated bindings. Test logic in Go; test
  presentation helpers in TypeScript.
- **App version lives in `wails.json`.** `info.productVersion` is the single
  source of truth for packaged metadata (macOS bundle, Windows exe, NSIS
  installer); bump it there. `frontend/package.json`'s `version` is not the app
  version. A runtime version must be injected via `-ldflags
  "-X main.version=..."` derived from `wails.json`.
- **macOS bundle ID is `net.nilambar.kova`.** Wails v2 has no config field for
  it; it lives in `build/darwin/Info.plist` and `build/darwin/Info.dev.plist`.
  Changing it later resets macOS permissions and app preferences.
- **Scope discipline:** no new dependencies or config files without explicit
  approval.

## Quality gate

Run this exact sequence from the repository root and confirm every step exits 0
before declaring a task complete:

```sh
(cd frontend && pnpm run check)
(cd frontend && pnpm test)
(cd frontend && pnpm run build)
go vet ./...
go test -race ./...
test -z "$(gofmt -l .)"
go build ./...
```

`gofmt -l .` must print nothing. For changes that touch native app behavior or
Wails config, also run `wails build` and confirm it succeeds.
