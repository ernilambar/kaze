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

To test the real bundle (Dock icon, app menu, LaunchServices registration),
install it to `/Applications`:

```sh
./scripts/dev.sh             # debug build
./scripts/dev.sh --release   # verified release build via scripts/bundle.sh
```

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

## Versioning

The application version is defined once, in `wails.json`:

```json
"info": {
  "productVersion": "0.1.0"
}
```

That value populates the macOS bundle version (`CFBundleShortVersionString`,
`CFBundleVersion`), the Windows executable version, and the NSIS installer
metadata. To release a new version, bump `info.productVersion` there and tag
the commit (for example `v0.2.0`).

The macOS bundle identifier is fixed as `net.nilambar.kova` in
`build/darwin/Info.plist` and `build/darwin/Info.dev.plist` (Wails v2 has no
`wails.json` field for it). Do not change it casually: macOS keys permissions,
preferences, and update identity to it.

`frontend/package.json` is a private, unpublished package; its `version` field
is not the application version. If the app later displays its version at
runtime, inject it with `wails build -ldflags "-X main.version=$(...)"` derived
from `wails.json` rather than hardcoding a second copy.

## Releasing

Releases are macOS arm64 only; Intel Macs are not supported.

1. Bump `info.productVersion` in `wails.json`.
2. Commit the bump and merge it to `main`.
3. Run `./scripts/release.sh vX.Y.Z` with the matching tag.

`release.sh` requires a clean tree, verifies the tag against `wails.json`, runs
`scripts/bundle.sh` (quality gate plus `wails build -platform darwin/arm64`),
then tags and pushes. The tag triggers `.github/workflows/release.yml`, which
rebuilds on an Apple silicon runner and attaches `Kova-macos-arm64.zip` and
`SHA256SUMS` to the GitHub release.

`bundle.sh` fails on non-macOS hosts and on Intel Macs, and verifies the built
binary is `arm64`. The app is signed ad hoc; if Developer ID signing and
notarization are added later, update the release workflow accordingly.

## Quality gate

Run all of the following from the repository root and confirm each exits 0:

```sh
(cd frontend && pnpm exec tsc --noEmit)
(cd frontend && pnpm test)
(cd frontend && pnpm run build)
go vet ./...
go test -race ./...
test -z "$(gofmt -l .)"
go build ./...
```

If your change affects app startup, Wails configuration, or platform
packaging, also run `wails build` and launch the resulting app from
`build/bin`.

Tests are colocated with the code they cover: Go tests use `testing`
(`app_test.go`) and frontend tests use Vitest (`src/*.test.ts`). Keep
TypeScript tests focused on pure helpers; Monaco, native dialogs, and generated
bindings are covered by manual smoke testing.

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
