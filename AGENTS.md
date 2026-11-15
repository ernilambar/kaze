# AGENTS.md

## Overview

Kova is a desktop text editor built with Wails v3 (Go backend) and Monaco Editor
(Svelte 5 + TypeScript frontend bundled by Vite). The Go binary embeds
`frontend/dist` and serves it as the app's assets; exported `App` methods are
exposed to the UI via generated Wails bindings.

## Setup

```sh
go install github.com/wailsapp/wails/v3/cmd/wails3@latest
go mod download
cd frontend && pnpm install
```

Requires Go 1.26+ and Node.js 20+. The Wails CLI is installed to `~/go/bin`;
ensure that directory is on `PATH`. No `.env` or other local config is needed.

## Commands

```sh
wails3 dev                             # dev mode with hot reload
wails3 task darwin:package             # packaged app in bin/kova.app
wails3 generate bindings -ts -i        # regenerate frontend/bindings
wails3 task --list                     # every build task
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

- **Bindings are generated.** `frontend/bindings/` is produced by `wails3
  generate bindings -ts -i`; never hand-edit it. Add frontend-callable behavior
  as exported methods on `App` in `app.go`, then let the build (or that command)
  regenerate bindings.
- **Go owns native behavior.** File I/O, dialogs, and OS integration belong in
  Go; the Svelte/TypeScript layer handles UI only.
- **The runtime handle is injected, not global.** `main.go` creates the
  `application.App`, registers `App` as its service, and calls `app.attach(...)`
  before `Run`. Menu callbacks and dialogs go through that handle; unit tests
  never attach one, so every use is nil-checked.
- **Filesystem access is a binding, not a web API.** Documents are read by
  `App.LoadDocument` in `document.go`, folder levels by `App.ReadFolder` in
  `folder.go`, and the CLI paths by `App.GetLaunchRequest` in `app.go`; the
  frontend never imports Node APIs. `cli.go` parses argv into a `LaunchRequest`
  that survives the frontend not being ready yet.
- **The sidebar is a folder tree with several roots.** `App.ReadFolder` lists
  one level at a time and the recursive `FolderTree.svelte` reads children on
  first expansion, so adding a large folder costs one directory read. Roots are
  saved through `App.AddSidebarFolder`/`App.RemoveSidebarFolder` into
  `settings.json`. `files.exclude` hides entries from the same read, and
  **View → Toggle Sidebar**
  (<kbd>Cmd</kbd>+<kbd>B</kbd>) saves `sidebar.visible`. A new window starts
  with the saved roots; **File → Open Folder in New Window…** saves the chosen
  folder as the window's only root and makes it the saved root set.
- **Workspaces are committable files.** `workspacefile.go` reads and writes
  `.kova-workspace` JSON files whose `folders` entries are relative to the file
  when they sit inside its directory, so the file can live in the repository.
  Opening one resolves and validates every folder before `SetSidebarFolders`
  replaces `sidebar.folders` in one write; the workspace name comes from the
  file name and is session state. Open, Save As and Close are in the File menu,
  the command palette and `workspacecommands.ts`.
- **Excludes are configurable globs.** `exclude.go` matches `files.exclude`
  and `search.exclude` for the tree, Quick Open and Find in Files: a bare name
  matches at any depth, a leading slash anchors it, `**` spans directories and
  a trailing slash means folders only. `gitignore.go` adds the `.gitignore`
  rules a search honors; version control directories are never searched even
  when the user removes them from the list.
- **Every folder entry point shares one code path.** The `+` button,
  **File → Add Folder to Sidebar…**, and `kova <folder>` all funnel through
  `addFolderFromDialog`/`applyLaunch` in `App.svelte`, so they share one busy
  flag and one error notice.
- **Open files are tabs held in frontend state.** `App.svelte` keeps the tab
  list and the active tab id; `tabs.ts` has the pure open/close rules (one tab
  per path, activate the right neighbour when the active tab closes).
  `OpenDocument` in `document.ts` carries a stable `id` (the path for a file,
  `untitled:<name>` for a **New File** buffer), the on-disk `content`, the
  file's `modified` mtime and a `dirty` flag, so **Save As** moves a buffer to
  its new path without the editor losing its Monaco model. `EditorPane` owns one
  model per open id and reports `onDirtyChange` by comparing the buffer's text
  with the text last loaded or saved; models are disposed when their tab closes.
  **File → Close Tab** (<kbd>Cmd</kbd>+<kbd>W</kbd>) emits `kova:close-tab`.
- **Tab navigation and the session are frontend state.** `tabs.ts` steps and
  jumps between tabs, and `App.svelte` keeps the Reopen Closed Tab stack.
  `workspace.ts` persists `session.open_files`/`session.active` whenever the
  file-backed tabs change, so a relaunch restores them; untitled buffers are
  deliberately left out because there is no text on disk to restore. Only the
  first window of a launch restores the session and writes it back: the launch
  request's `restore_session` flag marks that window, so a new window never
  replaces the saved session with its own empty or moved tab list.
- **Windows are independent editors in one process.** `windows.go` keeps one
  `windowSession` per Wails window (`main`, `window-2`, …); each owns its tabs,
  sidebar roots, search results and dirty flag, while settings, recent files and
  the theme stay shared. Native menu clicks act on the focused window, and a
  targeted event carries the sender's window name, which `onWindowEvent` in
  `windowstate.ts` checks, so a command cannot run in another window. **New
  Window** (<kbd>Cmd</kbd>+<kbd>Shift</kbd>+<kbd>N</kbd>), **Close Window**
  (<kbd>Cmd</kbd>+<kbd>Shift</kbd>+<kbd>W</kbd>), **Move Tab to New Window**,
  **Open Folder in New Window…** and **Bring All to Front** live in the File and
  Window menus and the command palette. Closing a window asks that window's
  dirty buffers first; closing the last one quits the app, while Cmd+Q asks
  every dirty window one at a time.
- **Saving writes what the frontend holds.** `SaveDocument` in `save.go` writes
  the buffer text the frontend hands it, and returns `Conflict: true` without
  writing when the file's mtime moved since `LoadDocument` reported it;
  `SaveFileDialog` and `OpenFileDialog` wrap the native panels. Each frontend
  pushes its own window's `SetUnsavedChanges`, so `ShouldQuit` and the
  `WindowClosing` hook can cancel a quit or window close and emit
  `kova:confirm-close` with a `close` or `quit` reason. The Save/Don't Save
  prompt answers with `ResolveClose(window, quit)`; a quit asks every dirty
  window in turn.
- **File watching polls rather than subscribing.** `watch.go` stats open
  buffers and expanded folders once a second and emits `kova:watched-changed`
  with the changed paths. The watcher records which windows asked for each path,
  so a change reaches only those windows and closing one drops its
  registrations. `App.svelte` reloads clean buffers, keeps dirty ones
  (with a notice), and refreshes the tree; `SaveDocument` rebaselines the path
  it wrote so Kova's own save is not echoed back. **View → Refresh** forces the
  same reread on demand.
- **Git is read-only.** `git.go` runs `git --no-optional-locks status
  --porcelain=v2`, `git diff HEAD` and `git show HEAD:<path>`; it never stages,
  commits or checks out. `GetGitStatus` supplies the explorer's status marks,
  the Changes view and the status bar's branch/ahead-behind label;
  `GetGitBaseline` gives the editor the committed text it diffs the buffer
  against for changed-line gutter bars; `GetGitDiff` fills read-only diff tabs
  (the frontend's `diff` document kind) opened from the Changes view. The
  Explorer/Changes switch lives in the status bar, and `Diff` tabs are excluded
  from the saved session.
- **Printing is native.** `PrintDocument` in `print.go` chooses text or image
  and calls `print_darwin.go`, whose cgo AppKit code (`print_darwin.m`) runs the
  standard print panel as a sheet on the main thread. `print_unsupported.go`
  covers other platforms and `CGO_ENABLED=0` builds.
- **`kova` is a shell launcher, not a second binary.** `install.go` writes a
  script to `/usr/local/bin/kova` that execs the bundle's executable, escalating
  through `osascript … with administrator privileges` when the directory is not
  writable. A single-instance lock forwards later invocations to the running
  window.
- **User settings live in `settings.json`.** `settings.go` loads and validates
  it from the platform config directory (`Kova/settings.json`), creating
  defaults on first run; `App.GetSettings` exposes it to the frontend, which
  applies the editor font values at startup and reads `sidebar.folders` for the
  tree roots. Keep `frontend/src/settings.ts`
  defaults in sync with `defaultSettings()` in Go (the TS copy is only the
  fallback when the backend call fails). `EditorSettings` also carries
  indentation, cursor and **View toggle** options; since some default to
  `true`, `loadSettings` decodes over `defaultSettings()` so an absent key
  keeps its default. `ThemeSettings` chooses `system`/`light`/`dark` and one
  variant name per appearance; `UISettings` holds the UI font family and size,
  tab icons, icon style and sidebar side. `App.UpdateEditorSettings`,
  `UpdateThemeSettings` and `UpdateUISettings` are the write paths; the exclude
  lists, recent files and session are written by `workspace.go`
  (`AddRecentFile`, `ClearRecentFiles`, `SetSession`, `SetSidebarVisible`)
  behind the same `settingsMu`.
- **Appearance is a Zed theme family.** `themes.go` embeds
  `themes/kova.json` (Kova Dark and Kova Light) and reads user files from
  `<config>/Kova/themes`, seeding `kova.json` when the folder has no JSON yet.
  Each file is a Zed family (`author`, `name`, `themes[]` with `appearance` and
  `style`). Kova consumes a documented subset: UI keys become CSS variables
  (`--bg`, `--fg`, `--border`, `--git-*`, …), `editor.*` keys become Monaco
  color ids, and `syntax` captures map to Monaco token rules in
  `frontend/src/theme.ts`. Missing keys fall back to the matching built-in
  palette; a malformed file is reported without hiding the rest. A user variant
  with the same name shadows the built-in one. `configwatch.go` polls the
  folder and keybindings file and emits `kova:config-changed`, so edits apply
  without a restart; `OpenThemesFolder` reveals the directory.
- **Custom keybindings are VS Code-shaped.** `keybindings.json` sits beside
  `settings.json`; `keybindings.go` accepts JSONC comments, validates keys
  (a modifier or a non-printing key), drops bad entries with a warning, and
  returns the accepted list to the frontend. `menu.go` registers every command
  by id and omits a default accelerator when that command, or its key, is
  claimed by a custom binding. `frontend/src/keybindings.ts` installs one
  capture-phase listener that runs the command, supports chord sequences and
  `when` contexts (`editorFocus`, `sidebarVisible`, `hasActiveTab`,
  `hasDirtyTab`, `!`, `&&`, `||`). `OpenKeybindings` creates the commented
  template.
- **The settings overlay is frontend state.** `SettingsOverlay.svelte` edits
  the appearance and interface sections and links to Settings JSON, Default
  Settings, Themes Folder and Keybindings. The native Settings… item emits
  `kova:settings-overlay`; `App.OpenDefaultSettings` writes
  `default-settings.json` beside the real file. The View menu's Appearance
  radio emits `kova:appearance`, and `SetAppearanceMode` moves the checkmark
  when the mode came from the overlay or a keybinding.
- **Window commands are native.** `window.go` exposes `MinimiseWindow`,
  `ZoomWindow` and `ToggleFullScreen` for one window, which the Window menu calls
  for the focused window and the command palette reaches through the bindings.
  `windows.go` adds `NewWindow`, `CloseWindow`, `MoveTabToNewWindow`,
  `OpenFolderInNewWindow` and `BringAllToFront`.
- **Editor commands forward from Go to the frontend.** `main.go` builds the
  Edit menu (case transforms, trim trailing whitespace) and the View menu
  (zoom, preview, appearance, checkable toggles) and emits `kova:transform-case`,
  `kova:trim-whitespace`, `kova:zoom`, `kova:toggle-preview` and
  `kova:toggle-view`; `editorcommands.ts` listens. A menu checkbox flips
  natively, so `App.SetViewOption` moves the same checkmark when the toggle
  came from the command palette. Monaco options are built in `editor.ts`
  (indentation lives on the model, so every buffer is updated too).
- **Find in Files is Go-side.** `SearchProject`/`ReplaceInProject` in
  `search.go` walk the sidebar roots with the excludes and `.gitignore`, then
  return flat matches (capped) that the results tab groups; `SearchPanel.svelte`
  is the form and `search.ts` has the grouping and summary helpers. A literal
  query is quoted before it becomes a regexp, and Replace All uses literal
  replacement unless regex mode is on.
- **Preview is dependency-free.** `markdown.ts` renders a Markdown subset from
  the buffer, escaping text and limiting link schemes before `{@html}`;
  SVG previews go through an `<img>` data URL so scripts never run. Transforms
  and whitespace helpers live in `case.ts` and `text.ts`, all covered by
  Vitest.
- **Monaco stays excluded from Vite pre-bundling.** `optimizeDeps.exclude:
  ['monaco-editor']` exists because pre-bundling breaks worker URL resolution;
  keep it unless you also rework worker loading.
- **Embedded assets are build output.** `main.go` uses
  `//go:embed all:frontend/dist`, so `frontend/dist` must exist before building
  Go; run `pnpm run build` (or `wails3 task darwin:package`) first.
- **Style:** idiomatic Go via `gofmt`; TypeScript is strict (`noUnusedLocals`,
  `noUnusedParameters`, `noImplicitReturns`) and uses 4-space indentation,
  single quotes, and named exports (`export function createEditor`).
- **Svelte 5 with runes.** Components live in `frontend/src/*.svelte`
  (`App.svelte` plus `TitleBar`, `EditorPane`, `StatusBar`, `Sidebar`, and the
  recursive `FolderTree`), mounted from `main.ts` with `mount()`;
  `svelte.config.js` pins `runes: true`. Use `$state`, `$props`, and
  `$derived`/`$effect` rather than legacy `export let`/`$:` syntax, and keep
  component styles scoped in the component's `<style>` block while shared
  design tokens stay in `style.css`. `FolderTree` imports itself for recursion;
  `<svelte:self>` is deprecated.
- **Use pnpm for the frontend.** `frontend/pnpm-lock.yaml` is the lockfile;
  never add `package-lock.json`. `frontend/pnpm-workspace.yaml` allows the
  `esbuild` install script because pnpm blocks dependency build scripts by
  default.
- **Tests are colocated.** Go tests sit beside the code (`app_test.go`);
  frontend tests use Vitest (`*.test.ts`) and cover pure helpers only, not
  Svelte components, Monaco, or generated bindings. Test logic in Go; test
  presentation helpers in TypeScript.
- **App version lives in `build/config.yml`.** `info.version` is the single
  source of truth for packaged metadata (macOS bundle, Windows exe, installer);
  bump it there, then run `wails3 task common:update:build-assets` to refresh
  the plists. `frontend/package.json`'s `version` is not the app version.
  `about.go` embeds the same file so the app menu and `kova --version` read the
  same values, and `scripts/config.sh` reads it from shell.
- **macOS bundle ID is `net.nilambar.kova`.** It lives in `build/config.yml` as
  `info.productIdentifier`, and `wails3 update build-assets` writes it into
  `build/darwin/Info.plist` and `Info.dev.plist`. Changing it later resets
  macOS permissions and app preferences.
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
Wails config, also run `wails3 task darwin:package` and confirm it succeeds.
