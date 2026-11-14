# Contributing to Kova

Thanks for your interest in improving Kova. This guide covers local setup,
coding conventions, and the checks a change must pass before review.

## Prerequisites

- Go 1.26 or newer
- Node.js 20 or newer
- pnpm (the frontend package manager)
- Wails CLI v2:

```sh
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

Make sure `$(go env GOPATH)/bin` is on your `PATH` so `wails` is found.

## Setup

```sh
go mod download
cd frontend && pnpm install
```

This installs Go modules and the frontend dependencies. pnpm stores packages
in a global content-addressable store and links them into `node_modules`, so
several projects share one copy.

## Development

Run the app with hot reload:

```sh
wails dev
```

This starts the Vite dev server and opens the native window. Frontend-only
iteration can also use `cd frontend && pnpm dev`, but the full app requires
`wails dev`.

## Project layout

| Path | Purpose |
| --- | --- |
| `main.go` | Wails bootstrap; embeds `frontend/dist` |
| `app.go` | `App` struct; exported methods become frontend bindings |
| `frontend/src/` | TypeScript UI (`main.ts`, `editor.ts`, `style.css`) |
| `frontend/wailsjs/` | Generated bindings; never edit by hand |
| `build/` | Platform packaging assets |
| `wails.json` | Wails project config and frontend hooks |

## Coding guidelines

- Go code must be `gofmt`-clean and idiomatic. Keep platform/native behavior
  (file I/O, dialogs, OS integration) in Go.
- Export a method on `App` when the frontend needs to call it; Wails generates
  the TypeScript bindings during `wails dev`/`wails build`. Do not hand-edit
  `frontend/wailsjs/`.
- TypeScript is strict (`noUnusedLocals`, `noUnusedParameters`,
  `noImplicitReturns`). Use named exports and keep UI concerns in the
  frontend.
- Monaco Editor must stay excluded from Vite dependency pre-bundling
  (`optimizeDeps.exclude` in `vite.config.ts`) unless worker loading is
  reworked; pre-bundling breaks worker URLs.
- Do not commit build output (`build/bin`, `frontend/dist`, `node_modules`).
- Use pnpm for the frontend; do not add `package-lock.json` or run
  `npm install`.

## Quality gate

Run all of the following from the repository root and confirm each exits 0:

```sh
(cd frontend && pnpm exec tsc --noEmit)
(cd frontend && pnpm run build)
go vet ./...
go test ./...
test -z "$(gofmt -l .)"
go build ./...
```

If your change affects app startup, Wails configuration, or platform
packaging, also run `wails build` and launch the resulting app from
`build/bin`.

There is currently no Go test suite (`go test ./...` reports "no test
files"). New tests are welcome; put them next to the code they cover.

## Pull requests

- Keep changes focused: one logical change per pull request.
- Describe what changed and why, and note any manual verification performed.
- Update documentation (`README.md`, `AGENTS.md`, this file) when behavior or
  tooling changes.
- Ensure the quality gate above passes before requesting review.

## Reporting issues

Include the operating system, Go/Node/pnpm/Wails versions, reproduction
steps, and any terminal or developer-console output. Screenshots help for UI
issues.
