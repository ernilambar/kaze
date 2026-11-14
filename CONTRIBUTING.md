# Contributing to Kova

Thanks for your interest in improving Kova. This guide covers coding
conventions and the checks a change must pass before review.

## Setup and development

Prerequisites, local setup, `wails dev` hot reload, bundle testing, and builds
are covered in [docs/development.md](docs/development.md). Read it first; the
rest of this guide covers conventions and review.

## Project layout

| Path | Purpose |
| --- | --- |
| `main.go` | Wails bootstrap; embeds `frontend/dist` |
| `app.go` | `App` struct; exported methods become frontend bindings |
| `settings.go` | Loads and validates `settings.json`; defaults on first run |
| `frontend/src/` | TypeScript UI (`main.ts`, `editor.ts`, `style.css`) |
| `frontend/wailsjs/` | Generated bindings; never edit by hand |
| `docs/` | User documentation (`settings.md`) and developer documentation (`development.md`) |
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

## Releasing and quality gate

The release steps (`scripts/bundle.sh`, `scripts/release.sh`) and the exact
quality-gate commands are in [docs/development.md](docs/development.md).
Releases are macOS arm64 only; Intel Macs are not supported.

## Pull requests

- Keep changes focused: one logical change per pull request.
- Describe what changed and why, and note any manual verification performed.
- Update documentation (`README.md`, `AGENTS.md`, this file) when behavior or
  tooling changes.
- Ensure the quality gate in [docs/development.md](docs/development.md) passes before requesting review.

## Reporting issues

Include the operating system, Go/Node/pnpm/Wails versions, reproduction
steps, and any terminal or developer-console output. Screenshots help for UI
issues.
