# Contributing to Kaze

Thanks for your interest in improving Kaze. This guide covers coding
conventions and the checks a change must pass before review.

## Setup and development

Prerequisites, local setup, `wails3 dev` hot reload, bundle testing, and builds
are covered in [docs/development.md](docs/development.md). Read it first; the
rest of this guide covers conventions and review.

## Project layout

| Path | Purpose |
| --- | --- |
| `main.go` | Entry point; embeds `frontend/dist` and `build/config.yml`, calls `app.Run` |
| `internal/app/` | `App` struct, windows, menus, native glue; exported methods become frontend bindings |
| `internal/<pkg>/` | App-free logic: settings, git, search, documents, file tree, themes |
| `frontend/src/` | Svelte UI (`App.svelte` and components, `main.ts`, `style.css`) |
| `frontend/bindings/` | Generated bindings; never edit by hand |
| `docs/` | User documentation (`settings.md`) and developer documentation (`development.md`) |
| `build/` | macOS packaging assets (`config.yml` holds product metadata) |
| `Taskfile.yml` | Build, package and dev tasks (Wails v3) |

## Coding guidelines

- Go code must be `gofmt`-clean and idiomatic. Keep platform/native behavior
  (file I/O, dialogs, OS integration) in Go.
- Export a method on `App` when the frontend needs to call it; Wails generates
  the TypeScript bindings during `wails3 dev`, `wails3 task darwin:package`, or
  `wails3 generate bindings -ts -i`. Do not hand-edit `frontend/bindings/`.
- The UI is built with Svelte 5 and TypeScript, using runes (`$state`,
  `$props`) rather than legacy reactive syntax. TypeScript is strict
  (`noUnusedLocals`, `noUnusedParameters`, `noImplicitReturns`);
  `pnpm run check` (svelte-check) type-checks both `.ts` and `.svelte` files.
  Use named exports in `.ts` modules and keep UI concerns in the frontend.
- Monaco Editor must stay excluded from Vite dependency pre-bundling
  (`optimizeDeps.exclude` in `vite.config.ts`) unless worker loading is
  reworked; pre-bundling breaks worker URLs.
- Do not commit build output (`bin`, `dist`, `frontend/dist`, `node_modules`).
- Use pnpm for the frontend; do not add `package-lock.json` or run
  `npm install`.

## Versioning

The application version is defined once, in `build/config.yml`:

```yaml
info:
  version: "0.1.0"
```

That value populates the macOS bundle version (`CFBundleShortVersionString` and
`CFBundleVersion`).
To release a new version, bump `info.version` there, run `wails3 task
common:update:build-assets`, and tag the commit (for example `v0.2.0`).
`about.go` embeds the same file, so `kaze --version`, the app menu and the
bundle never disagree.

The macOS bundle identifier is `info.productIdentifier` (`net.nilambar.kaze`),
and `wails3 update build-assets` writes it into `build/darwin/Info.plist` and
`build/darwin/Info.dev.plist`. Do not change it casually: macOS keys
permissions, preferences, and update identity to it.

`frontend/package.json` is a private, unpublished package; its `version` field
is not the application version.

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

Include the macOS version, Go/Node/pnpm/Wails versions, reproduction steps, and
any terminal or developer-console output. Screenshots help for UI
issues.
