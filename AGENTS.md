# AGENTS.md

## Overview

Kaze is a desktop text editor built with Wails v3 (Go backend) and Monaco Editor
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
wails3 task darwin:package             # packaged app in bin/kaze.app
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
- **The tree is a file manager, not just a viewer.** `filetree.go` adds
  `App.CreateFile`, `CreateFolder`, `RenamePath`, `DuplicatePath`, `CopyPath`,
  `MovePath` and `TrashPath`. Every one validates what it is handed: a name is
  one path segment, so a caller cannot walk up the tree, and a destination is a
  real folder, so a copy never lands somewhere the user cannot see. New File
  opens with `O_EXCL` and a rename or move refuses a taken name, so nothing is
  overwritten by accident. Duplicate and Copy land on the first free
  `name copy.ext` (keeping its own name when a copy lands in another folder and
  that name is free); `MovePath` renames and falls back to copy-then-delete only
  across volumes (`EXDEV`). **Nothing deletes for good:**
  `TrashPath` goes through the platform's Trash (`trash_darwin.go`, cgo AppKit
  `trashItemAtURL:`, called on the main thread) and `trash_other.go` fails loudly
  elsewhere. Each command returns a `FileTreeResult` with both the written path
  and its symlink-resolved `real_path`, because tabs match on the latter.
- **A rename or move keeps the tabs open.** `retargetDocumentPaths` in `tabs.ts`
  rewrites a tab's `path`/`realPath` and the diff tab that watches the same file,
  retargeting every tab under a renamed folder in one pass; the tab keeps its
  `id`, so Monaco keeps the model, the text, the undo history and the dirty flag.
  `applyTreeMove` in `App.svelte` applies it and then rereads the file's git
  baseline and any diff of it, because HEAD still knows the old name. Cutting is
  cleared when the moved entry was the thing the clipboard held.
- **The row menu is the tree's only file manager UI.** `FolderTree.svelte`
  reports `isDir` through `onContextMenu`, so `ContextMenu.svelte`
  (which now draws separators and disabled rows) offers New File, New Folder,
  Cut/Copy/Paste, Duplicate, Rename and Move to Trash for a tree row. A tab's
  menu offers Close Tab, Move to New Window, Close Other Tabs, Close Tabs to
  Right, Close Tabs with Deleted Files, Close All and, for a file-backed tab,
  Reveal in Finder: `tabsToClose` in `tabs.ts` picks the group from the
  right-clicked tab, and `closeTabsInOrder` in `App.svelte` closes it through
  the normal dirty-buffer prompt. A folder row, and the empty space under its
  children, is the drop target: drag state lives in `filetree.ts` (a drag spans
  components), a drop reads either the tree payload or the local path of the
  file dragged in from Finder, and a folder cannot be dropped into its own
  subtree. Names are asked for in `NamePrompt.svelte`, which stays open with the
  backend's refusal in it so a taken name is corrected in place.
- **File → New File creates on disk; New Buffer is the untitled one.**
  <kbd>Cmd</kbd>+<kbd>N</kbd> asks the tree for a name in the selected folder
  (the selected file's folder when a file is selected), creates the file and
  opens it; **File → New Folder…** does the same for a directory and
  <kbd>Cmd</kbd>+<kbd>Alt</kbd>+<kbd>N</kbd> opens an untitled buffer. The menu
  only emits `app:new-file`, `app:new-folder` and `app:new-buffer`, because the
  sidebar roots are frontend state.
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
  `.kaze-workspace` JSON files whose `folders` entries are relative to the file
  when they sit inside its directory, so the file can live in the repository.
  Opening one resolves and validates every folder before `SetSidebarFolders`
  replaces `sidebar.folders` in one write; the workspace name comes from the
  file name and is session state. Open, Save As and Close are in the File menu,
  the command palette and `workspacecommands.ts`. Open also accepts a VS Code
  `.code-workspace` (JSONC): only `folders` (`path`, or a `file://` `uri`) is
  read, everything else is ignored, and Save As always writes `.kaze-workspace`.
- **Excludes are configurable globs.** `exclude.go` matches `files.exclude`
  and `search.exclude` for the tree, Quick Open and Find in Files: a bare name
  matches at any depth, a leading slash anchors it, `**` spans directories and
  a trailing slash means folders only. `gitignore.go` adds the `.gitignore`
  rules a search honors; version control directories are never searched even
  when the user removes them from the list.
- **Every folder entry point shares one code path.** The `+` button,
  **File → Add Folder to Sidebar…**, and `kaze <folder>` all funnel through
  `addFolderFromDialog`/`applyLaunch` in `App.svelte`, so they share one busy
  flag and one error notice. Finder drops do too: `openWindow` enables
  `EnableFileDrop` and listens for `WindowFilesDropped`, the frontend marks the
  window root with `data-file-drop-target`, and `handleFileDrop` turns the
  paths into a `LaunchRequest` for that window through `queueLaunchTo`.
  `drop.go` classifies files as tabs and folders as roots; Option, read from
  AppKit in `modifier_darwin.go`, opens the drop in a new window and makes a
  dropped folder its only root.
- **Open files are tabs held in frontend state.** `App.svelte` keeps the tab
  list and the active tab id; `tabs.ts` has the pure open/close rules (one tab
  per file, activate the right neighbour when the active tab closes).
  `OpenDocument` in `document.ts` carries a stable `id` (the path for a file,
  `untitled:<name>` for a **New File** buffer), the on-disk `content`, the
  file's `modified` mtime and a `dirty` flag, so **Save As** moves a buffer to
  its new path without the editor losing its Monaco model. Tabs are matched on
  `realPath`: `LoadDocument` and `SaveDocument` report `real_path`, the path
  with `filepath.EvalSymlinks` applied, so a file opened through a symlink or a
  symlinked folder and the same file opened directly share one tab, while `path`
  keeps the spelling that was opened for display, reveal and relative paths.
  `EditorPane` owns one model per open id and reports `onDirtyChange` by
  comparing the buffer's text with the text last loaded or saved; models are
  disposed when their tab closes.
  Dragging a tab onto another reorders the list through `moveDocumentTab`, whose
  result is also the order the session saves.
  **File → Close Tab** (<kbd>Cmd</kbd>+<kbd>W</kbd>) emits `app:close-tab`.
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
- **The title bar is the frontend's.** `openWindow` in `windows.go` passes
  `MacTitleBarHidden`, so macOS hides its native title bar and extends the
  content under the traffic lights; `TitleBar.svelte` is then the window's only
  title bar, 32px tall on macOS with 2px top padding (so the text's visual
  centre meets the lights' centre, measured from a screenshot), padded on macOS to leave room for the buttons and marked
  `--wails-draggable: drag` so the strip moves and double-click-zooms the
  window. Other platforms keep their native title bar above the same strip.
- **Saving writes what the frontend holds.** `SaveDocument` in `save.go` writes
  the buffer text the frontend hands it, and returns `Conflict: true` without
  writing when the file's mtime moved since `LoadDocument` reported it;
  `SaveFileDialog` and `OpenFileDialog` wrap the native panels. Each frontend
  pushes its own window's `SetUnsavedChanges`, so `ShouldQuit` and the
  `WindowClosing` hook can cancel a quit or window close and emit
  `app:confirm-close` with a `close` or `quit` reason. The Save/Don't Save
  prompt answers with `ResolveClose(window, quit)`; a quit asks every dirty
  window in turn.
- **Text files keep their encoding.** `LoadDocument` strips a UTF-8 BOM and
  reports it with the file's `lf`/`crlf` line ending; the editor sets the Monaco
  model's EOL and `SaveDocument` restores both. A file that is not valid UTF-8
  opens with U+FFFD replacement characters and `invalid_utf8` set, and a save
  returns `InvalidUTF8: true` without writing until the frontend confirms with
  `allow_invalid_utf8`; a NUL byte still marks a real binary and is rejected.
- **File watching polls rather than subscribing.** `watch.go` stats open
  buffers and expanded folders once a second and emits `app:watched-changed`
  with the changed paths. The watcher records which windows asked for each path,
  so a change reaches only those windows and closing one drops its
  registrations. `App.svelte` reloads clean buffers, keeps dirty ones
  (with a notice), and refreshes the tree; `SaveDocument` rebaselines the path
  it wrote so Kaze's own save is not echoed back. **View → Refresh** forces the
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
- **`kaze` is a shell launcher, not a second binary.** `install.go` writes a
  script to `/usr/local/bin/kaze` that execs the bundle's executable, escalating
  through `osascript … with administrator privileges` when the directory is not
  writable. A single-instance lock forwards later invocations to the running
  window.
- **User settings live in `settings.json`.** `settings.go` loads and validates
  it from the platform config directory (`Kaze/settings.json`), creating
  defaults on first run; `App.GetSettings` exposes it to the frontend, which
  applies the editor font values at startup and reads `sidebar.folders` for the
  tree roots. Keep `frontend/src/settings.ts`
  defaults in sync with `defaultSettings()` in Go (the TS copy is only the
  fallback when the backend call fails). `EditorSettings` also carries
  indentation, cursor and **View toggle** options; since some default to
  `true`, `loadSettings` decodes over `defaultSettings()` so an absent key
  keeps its default. `ThemeSettings` chooses `system`/`light`/`dark` (default
  `dark`) and one variant name per appearance; `UISettings` holds the UI font
  family and size, tab icons, icon style and sidebar side. `App.UpdateEditorSettings`,
  `UpdateThemeSettings`, `UpdateUISettings`, `UpdateFilesSettings` and
  `UpdateSearchSettings` are the write paths; the recent files and session are
  written by `workspace.go` (`AddRecentFile`, `ClearRecentFiles`, `SetSession`,
  `SetSidebarVisible`) behind the same `settingsMu`. Those paths are surgical:
  `settingswrite.go` re-reads the document and rewrites only the keys of the
  section the command owns, so unknown keys and a hand-edited file's formatting
  survive a toggle. Every configurable section
  is editable from the settings overlay; recent files and the session are
  state, managed by Open Recent and the tab bar.
- **Appearance is a Zed theme family.** `themes.go` embeds
  `themes/kaze.json` (Kaze Dark and Kaze Light) and reads user files from
  `<config>/Kaze/themes`, seeding `kaze.json` when the folder has no JSON yet.
  Each file is a Zed family (`author`, `name`, `themes[]` with `appearance` and
  `style`). Kaze consumes a documented subset: UI keys become CSS variables
  (`--bg`, `--fg`, `--border`, `--git-*`, …), `editor.*` keys become Monaco
  color ids, and `syntax` captures map to Monaco token rules in
  `frontend/src/theme.ts`. Missing keys fall back to the matching built-in
  palette; a malformed file is reported without hiding the rest. A user variant
  with the same name shadows the built-in one. `configwatch.go` polls the
  folder and keybindings file and emits `app:config-changed`, so edits apply
  without a restart; `OpenThemesFolder` reveals the directory.
- **The first frame is painted from a cached palette.** Settings and theme files
  are read asynchronously after the window is already visible, and WKWebView
  paints white before any CSS arrives, so `applyThemePalette` caches the surface
  colours in `localStorage` (`kaze.theme.startup`, written by
  `cacheStartupTheme`). An inline script in `frontend/index.html` and
  `main.ts` paint that cache before the module graph loads, and `main.ts` also
  selects a throwaway Monaco theme from it — otherwise Monaco's light `vs`
  default flashes. Keep the inline copy's key, shape and hex test in sync with
  `theme.ts`; a missing or malformed entry just falls back to the built-in dark
  palette, matching the window's `BackgroundColour`.
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
- **The settings overlay is frontend state.** `SettingsOverlay.svelte` has a
  section per settings block: editor (typography, indentation, cursor, view
  toggles, final newline), appearance, interface, files (the explorer and
  search excludes as one-glob-per-line text), sidebar (visibility and roots)
  and advanced links to Settings JSON, Default Settings, Themes Folder and
  Keybindings. Edits save as they commit; editor saves are chained so the newest
  change wins, and the View menu checkmarks follow. The native Settings… item
  emits `app:settings-overlay`; `App.OpenDefaultSettings` writes
  `default-settings.json` beside the real file. The View menu's Appearance
  radio emits `app:appearance`, and `SetAppearanceMode` moves the checkmark
  when the mode came from the overlay or a keybinding.
- **Window commands are native.** `window.go` exposes `MinimiseWindow`,
  `ZoomWindow` and `ToggleFullScreen` for one window, which the Window menu calls
  for the focused window and the command palette reaches through the bindings.
  `windows.go` adds `NewWindow`, `CloseWindow`, `MoveTabToNewWindow`,
  `OpenFolderInNewWindow` and `BringAllToFront`.
- **Editor commands forward from Go to the frontend.** `menu.go` builds the
  Edit menu (a Find submenu for Monaco's Find/Replace widgets and Go to Line,
  case transforms, trim trailing whitespace), the Selection menu (Monaco's
  multi-cursor, line and comment actions) and the View menu (zoom, preview,
  appearance, checkable toggles) and emits `app:find`, `app:replace`,
  `app:go-to-line`, `app:find-next-result`, `app:find-previous-result`,
  `app:transform-case`, `app:trim-whitespace`, `app:selection-command`,
  `app:zoom`, `app:toggle-preview` and `app:toggle-view`; `editorcommands.ts`
  listens, `EditorPane.runEditorAction` runs a Monaco action id and
  `runSelectionCommand` maps a Selection id to its Monaco action. A menu
  checkbox flips natively, so `App.SetViewOption` moves the same checkmark
  when the toggle came from the command palette. Monaco options are built in
  `editor.ts` (indentation lives on the model, so every buffer is updated too).
  F4 and ⇧F4 step through the Find in Files results the window holds, opening
  the file and revealing the hit.
- **Language and the status bar are frontend state.** `editor.ts` resolves a
  buffer's language: an explicit **Change Language Mode** override stored on the
  tab wins, otherwise the path decides, and a `#!` first line is used only when
  the path names no language. The palette's Auto Detect row clears the override.
  `StatusBar.svelte` prints the cursor and selection count, the tab's line
  ending (clicking it switches the Monaco model between LF and CRLF, which
  marks the buffer dirty), the language (clicking it opens the mode palette)
  and an invalid-UTF-8 warning for a file read with replacement characters.
- **Find in Files is Go-side.** `SearchProject`/`ReplaceInProject` in
  `search.go` walk the sidebar roots with the excludes and `.gitignore`, then
  return flat matches (capped) that the results tab groups; `SearchPanel.svelte`
  is the form and `search.ts` has the grouping and summary helpers. A literal
  query is quoted before it becomes a regexp, and Replace All uses literal
  replacement unless regex mode is on. **Replace All is safe:** it confirms the
  match and file counts first, edits files open in a tab through Monaco (left
  unsaved and undoable, and not written on disk), skips files whose mtime moved
  since the search and names them in the result, and rebaselines the watcher for
  the files it does write.
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
  `about.go` embeds the same file so the app menu and `kaze --version` read the
  same values, and `scripts/config.sh` reads it from shell.
- **macOS bundle ID is `net.nilambar.kaze`.** It lives in `build/config.yml` as
  `info.productIdentifier`, and `wails3 update build-assets` writes it into
  `build/darwin/Info.plist` and `Info.dev.plist`. Changing it later resets
  macOS permissions and app preferences.
- **Scope discipline:** no new dependencies or config files without explicit
  approval.

## Quality gate

Run this exact sequence from the repository root and confirm every step exits 0
before declaring a task complete:

```sh
# Keep cgo objects at the deployment target declared in build/darwin/Info.plist.
export MACOSX_DEPLOYMENT_TARGET=13.0 CGO_CFLAGS="-mmacosx-version-min=13.0" CGO_LDFLAGS="-mmacosx-version-min=13.0"
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
