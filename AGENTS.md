# AGENTS.md

## Overview

Kova is a desktop text editor built with Wails v2 (Go backend) and Monaco Editor
(TypeScript frontend bundled by Vite). The Go binary embeds `frontend/dist` and
serves it as the app's assets; exported `App` methods are exposed to the UI via
generated Wails bindings.

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
go test ./...                          # Go tests (none yet)
go vet ./...                           # Go lint
gofmt -w .                             # Go format (write)
cd frontend && pnpm run build           # tsc + vite build to frontend/dist
cd frontend && pnpm exec tsc --noEmit        # TypeScript typecheck
```

There is no frontend linter/formatter configured (no ESLint/Prettier); do not
introduce one unprompted.

## Conventions

- **Bindings are generated.** `frontend/wailsjs/` is produced by Wails; never
  hand-edit it. Add frontend-callable behavior as exported methods on `App` in
  `app.go`, then let `wails dev`/`wails build` regenerate bindings.
- **Go owns native behavior.** File I/O, dialogs, and OS integration belong in
  Go; the TypeScript layer handles UI only.
- **Monaco stays excluded from Vite pre-bundling.** `optimizeDeps.exclude:
  ['monaco-editor']` exists because pre-bundling breaks worker URL resolution;
  keep it unless you also rework worker loading.
- **Embedded assets are build output.** `main.go` uses
  `//go:embed all:frontend/dist`, so `frontend/dist` must exist before building
  Go; run `pnpm run build` (or `wails build`) first.
- **Style:** idiomatic Go via `gofmt`; TypeScript is strict (`noUnusedLocals`,
  `noUnusedParameters`, `noImplicitReturns`) and uses 4-space indentation,
  single quotes, and named exports (`export function createEditor`).
- **Use pnpm for the frontend.** `frontend/pnpm-lock.yaml` is the lockfile;
  never add `package-lock.json`. `frontend/pnpm-workspace.yaml` allows the
  `esbuild` install script because pnpm blocks dependency build scripts by
  default.
- **Scope discipline:** no new dependencies, test frameworks, or config files
  without explicit approval.

## Quality gate

Run this exact sequence from the repository root and confirm every step exits 0
before declaring a task complete:

```sh
(cd frontend && pnpm exec tsc --noEmit)
(cd frontend && pnpm run build)
go vet ./...
go test ./...
test -z "$(gofmt -l .)"
go build ./...
```

`gofmt -l .` must print nothing. For changes that touch native app behavior or
Wails config, also run `wails build` and confirm it succeeds.
