# AGENTS.md

## Overview

Kaze: a desktop text editor. Wails v3 (Go) + Monaco (Svelte 5, TypeScript, Vite).
The binary embeds `frontend/dist`; exported `App` methods become UI bindings.

## Setup

```sh
go install github.com/wailsapp/wails/v3/cmd/wails3@latest
go mod download
cd frontend && pnpm install
```

Go 1.26+, Node 20+; `~/go/bin` on `PATH`. No `.env` needed.

## Commands

```sh
wails3 dev                          # hot reload
wails3 task darwin:package          # bin/kaze.app
wails3 generate bindings -ts -i     # regenerate frontend/bindings
go build ./... ; go vet ./... ; go test -race ./... ; gofmt -w .
(cd frontend && pnpm run check)     # svelte-check
(cd frontend && pnpm test)          # Vitest
(cd frontend && pnpm run build)     # svelte-check + vite -> frontend/dist
./scripts/dev.sh                    # debug build to /Applications
./scripts/bundle.sh                 # release gate + zip
./scripts/release.sh v0.1.0         # tag; CI builds
```

No frontend linter/formatter is configured; don't add one unprompted.

## Architecture

- **Bindings are generated.** Never hand-edit `frontend/bindings/`; add exported
  `App` methods and regenerate. Go owns file I/O, dialogs and OS integration;
  the frontend never imports Node APIs. `main.go` calls `app.attach(...)` before
  `Run`; tests never attach, so every use is nil-checked.
- **Launch.** `cli.go` parses argv into a `LaunchRequest` that survives the
  frontend not being ready. `+`, **Add Folder to Sidebar…**, `kaze <folder>` and
  Finder drops (`drop.go`) all funnel through `addFolderFromDialog`/`applyLaunch`;
  Option on drop opens a new window.
- **File tree** (`filetree.go`). Create/rename/move/duplicate refuse taken names;
  `MovePath` copies-then-deletes only on `EXDEV`; nothing deletes permanently
  (`TrashPath`, AppKit, main thread). Rename/Move take a full typed path and
  create missing folders. The row context menu (`ContextMenu.svelte`) is the
  tree's only file-manager UI; `NamePrompt.svelte` stays open on backend errors.
- **Rename/move keeps tabs open.** `retargetDocumentPaths` (`tabs.ts`) rewrites
  paths; the tab `id` survives, so Monaco keeps model, undo and dirty state.
- **Tabs are frontend state.** `App.svelte` holds them; `tabs.ts` has pure rules.
  Tabs match on symlink-resolved `realPath`; `path` keeps the opened spelling.
  `EditorPane` owns one model per id. New File creates on disk; New Buffer
  (<kbd>Cmd</kbd>+<kbd>N</kbd>) is untitled.
- **Sidebar** is multi-root; `ReadFolder(path, root)` lists one level, filtered by
  `files.exclude` relative to the root.
- **Excludes are globs** (`exclude.go`): bare name at any depth, leading `/`
  anchors, `**` spans directories, trailing `/` folders only. Search also honours
  `.gitignore` (`gitignore.go`); VCS directories are never searched.
- **Workspaces** (`workspacefile.go`): `.kaze-workspace` holds folders only,
  relative where inside its directory. VS Code `.code-workspace` is read
  (`folders` only); Save As always writes `.kaze-workspace`.
- **Session vs settings.** `session.json` (`session.go`, `workspace.ts`) holds
  machine-written state: tabs, roots, expanded folders, window frame, Open
  Recent, per-workspace parked tab sets (capped at 50). `settings.json` holds
  user config only. Untitled buffers are never persisted. Only the first window
  of a launch restores and writes the launch entry and window geometry.
- **Windows are independent editors in one process** (`windows.go`): each owns
  tabs, roots, search results and dirty flag; settings, theme and Open Recent
  are shared. Menu actions hit the focused window. The frontend `TitleBar.svelte`
  is the only title bar.
- **Window geometry** (`windowgeometry.go`): a saved frame is used only if ≥120×48
  points of its title bar are on a screen; else centred 1200×800. Windows are
  created hidden and placed on `WindowRuntimeReady`.
- **Saving** (`save.go`): `SaveDocument` returns `Conflict: true` without writing
  if mtime moved. Close/quit with dirty buffers emits `app:confirm-close`,
  answered by `ResolveClose`.
- **Encoding is preserved.** UTF-8 BOM stripped, LF/CRLF kept. Invalid UTF-8
  opens with U+FFFD; save refuses until `allow_invalid_utf8`. NUL byte = binary.
- **Watching polls** (`watch.go`, `configwatch.go`) every second. Clean buffers
  reload; dirty ones keep a notice.
- **Git is read-only** (`git.go`): status, `diff HEAD`, `show HEAD:<path>`. Never
  stages, commits or checks out. Diff tabs are read-only and not persisted.
- **Find in Files is Go-side** (`search.go`). Replace All uses JS-style `$1`
  templates expanded identically in Go and `search.ts`, edits open files through
  Monaco unsaved, skips files whose mtime moved, and reports unwritable files.
- **Settings** (`settings.json`). Keep `frontend/src/settings.ts` defaults in
  sync with Go. Writes go through `settingsMu` and `settingswrite.go`, which
  rewrites only changed keys so unknown keys and formatting survive.
  Settings/keybindings files open in Kaze tabs; `configwatch.go` live-reloads
  `settings.json` into every window, and a broken file keeps current values.
- **Themes** are Zed theme families (`themes.go`, `theme.ts`); user themes in
  `<config>/Kaze/themes` shadow built-ins. The first frame paints a palette
  cached in `localStorage` (`kaze.theme.startup`); keep the inline script in
  `index.html` in sync with `theme.ts`.
- **Keybindings are VS Code-shaped** (`keybindings.json`, JSONC). Defaults
  (`docs/keymaps.md`) live in three places: menu accelerators (`menu.go`),
  `DEFAULT_KEYBINDINGS` (`keybindings.ts`) and Monaco rules (`editor.ts`).
  `menu.go` drops a default accelerator claimed by a custom binding.
- **Menus** (`menu.go`) follow macOS layout and only emit `app:*` events; a
  command without its own event emits `app:run-command` with the palette id.
  Palette `detail` in `commands.ts` names the menu holding the command.
- **Preview is dependency-free** (`markdown.ts`); clicks never navigate the
  webview. SVG previews use an `<img>` data URL.
- **Keep Monaco out of Vite pre-bundling** (`optimizeDeps.exclude`); it breaks
  worker URLs.
- **`kaze` CLI** (`install.go`) writes `/usr/local/bin/kaze`; a single-instance
  lock forwards later invocations to the running app.

## Conventions

- `gofmt`; TS strict, 4 spaces, single quotes, named exports.
- Svelte 5 runes only (no `export let`/`$:`); scoped styles, shared tokens in
  `style.css`; `FolderTree` imports itself (no `<svelte:self>`).
- pnpm only; never `package-lock.json`.
- Tests are colocated. Vitest covers pure TS helpers, not Svelte/Monaco/bindings.
- Version lives in `build/config.yml` (`info.version`); bump it, then run
  `wails3 task common:update:build-assets`.
- Bundle ID `net.nilambar.kaze` — changing it resets macOS permissions/prefs.
- No new dependencies or config files without explicit approval.

## Quality gate

Run from the repository root; every step must exit 0:

```sh
export MACOSX_DEPLOYMENT_TARGET=13.0 CGO_CFLAGS="-mmacosx-version-min=13.0" CGO_LDFLAGS="-mmacosx-version-min=13.0"
(cd frontend && pnpm run check)
(cd frontend && pnpm test)
(cd frontend && pnpm run build)
go vet ./...
go test -race ./...
test -z "$(gofmt -l .)"
go build ./...
```

Native/Wails changes also need `wails3 task darwin:package` to succeed.
