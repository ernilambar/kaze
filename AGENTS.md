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

## Conventions

- **Architecture.** Bindings are generated — never hand-edit
  `frontend/bindings/`; add exported `App` methods and regenerate. Go owns file
  I/O, dialogs and OS integration; the frontend never imports Node APIs.
  `main.go` builds `application.App`, registers `App`, calls `app.attach(...)`
  before `Run`; tests never attach, so every use is nil-checked.
- **Filesystem is a binding.** `LoadDocument` (`document.go`), `ReadFolder`
  (`folder.go`), `GetLaunchRequest` (`app.go`); `cli.go` parses argv into a
  `LaunchRequest` that survives the frontend not being ready.
- **The tree is a file manager** (`filetree.go`): `CreateFile`, `CreateFolder`,
  `RenamePath`, `DuplicatePath`, `DuplicatePathSuggestion`, `CopyPath`,
  `MovePath`, `TrashPath`. Names are one segment, destinations real folders; New
  File uses `O_EXCL`, rename/move/duplicate refuse taken names, `CopyPath` picks
  the first free `name copy.ext`, and `MovePath` falls back to copy-then-delete
  only on `EXDEV`. `RenamePath` takes a typed path (a bare name renames in
  place), creates missing folders and reads a relative path from the entry's own
  folder; Move… uses it too, with the folder part selected. `DuplicatePath`
  takes the same typed target and refuses a folder copied into itself;
  `DuplicatePathSuggestion` supplies the dialog's prefilled `name copy.ext`.
  Nothing deletes permanently: `TrashPath` uses AppKit on the main thread
  (`trash_darwin.go`); `trash_other.go` fails loudly. Results carry `path` and
  symlink-resolved `real_path`; tabs match on the latter.
- **Rename/move keeps tabs open.** `retargetDocumentPaths` (`tabs.ts`) rewrites
  `path`/`realPath` for the tab, its diff tab and tabs under a renamed folder;
  the `id` survives, so Monaco keeps model, undo and dirty state. `applyTreeMove`
  rereads git baseline/diff. Cutting is cleared if it held the moved entry.
- **Row menu is the tree's only file-manager UI.** `FolderTree.svelte` reports
  `isDir` via `onContextMenu`; `ContextMenu.svelte` offers tree actions (open,
  new file/folder, cut/copy/paste, copy name/path/relative path, duplicate…,
  rename, move, trash) and tab actions (close variants, move to new window,
  reveal in Finder). `tabsToClose` picks the group; `closeTabsInOrder` goes
  through the dirty prompt. Folder rows and empty space are drop targets; drag
  state lives in `filetree.ts`; a folder can't be dropped into its own subtree.
  Move… and Rename… edit the entry's full path and create folders that do not
  exist yet; Move selects the folder part, Rename the file stem.
  `NamePrompt.svelte` stays open with the backend error so a taken name is
  corrected in place.
- **New File creates on disk; New Buffer is untitled.**
  <kbd>Cmd</kbd>+<kbd>N</kbd> asks for a name in the selected folder, creates and
  opens it; **File → New Folder…** does the same for a directory;
  <kbd>Cmd</kbd>+<kbd>Alt</kbd>+<kbd>N</kbd> opens an untitled buffer. Menus only
  emit `app:new-file`/`new-folder`/`new-buffer`.
- **Sidebar is a multi-root folder tree.** `ReadFolder` lists one level;
  `FolderTree.svelte` loads children on first expansion. Roots persist via
  `AddSidebarFolder`/`RemoveSidebarFolder`; `files.exclude` filters the same
  read. **View → Toggle Sidebar** (<kbd>Cmd</kbd>+<kbd>B</kbd>) saves
  `sidebar.visible`. New windows start from the saved roots; **Open Folder in
  New Window…** makes the chosen folder the window's only and saved root.
- **Workspaces are committable.** `workspacefile.go` reads/writes
  `.kaze-workspace`; folders inside its directory are stored relative. Open
  validates all folders before `SetSidebarFolders` replaces the list in one
  write. Also accepts VS Code `.code-workspace` (JSONC): only `folders`
  (`path`/`file://` `uri`) is read. Save As always writes `.kaze-workspace`.
- **Excludes are globs.** `exclude.go` matches `files.exclude`/`search.exclude`:
  bare name at any depth, leading `/` anchors, `**` spans directories, trailing
  `/` folders only. `gitignore.go` adds `.gitignore` rules for search; VCS
  directories are never searched.
- **Folder entry points share one path.** `+`, **Add Folder to Sidebar…** and
  `kaze <folder>` funnel through `addFolderFromDialog`/`applyLaunch`. Finder
  drops use `EnableFileDrop` + `WindowFilesDropped`, `data-file-drop-target`,
  `handleFileDrop` and `queueLaunchTo`; `drop.go` splits files/folders; Option
  (`modifier_darwin.go`) opens a new window with the folder as only root.
- **Tabs are frontend state.** `App.svelte` holds the list and active id;
  `tabs.ts` has pure open/close rules. A tab has a stable `id`, on-disk
  `content`, mtime and dirty flag; Save As moves a buffer without losing its
  Monaco model. Tabs match on `realPath` (`LoadDocument`/`SaveDocument` return
  `real_path` via `EvalSymlinks`); `path` keeps the opened spelling. `EditorPane`
  owns one model per id and disposes it on close. Dragging reorders through
  `moveDocumentTab`. **File → Close Tab** (<kbd>Cmd</kbd>+<kbd>W</kbd>) emits
  `app:close-tab`.
- **Session is frontend state.** `workspace.ts` persists
  `session.open_files`/`session.active` for file-backed tabs only; untitled
  buffers are excluded. Only the window whose launch request has
  `restore_session` restores and writes it.
- **Windows are independent editors in one process.** `windows.go` keeps one
  `windowSession` per window; each owns tabs, roots, search results and dirty
  flag, while settings, recent files and theme stay shared. Menu actions hit the
  focused window; targeted events carry the sender name checked by
  `windowstate.ts`. New/Close Window, Move Tab to New Window, Open Folder in New
  Window and Bring All to Front are in menus and palette. Closing asks that
  window's dirty buffers; the last window quits; Cmd+Q asks each dirty window.
- **Title bar is the frontend's.** `MacTitleBarHidden` extends content under the
  traffic lights; `TitleBar.svelte` (32px, 2px top padding, draggable) is the only
  title bar on macOS. Other platforms keep the native bar above it.
- **Window geometry is saved and validated.** `windowgeometry.go`: the first
  window restores `settings.json`'s `window` frame; new windows cascade 24px
  from the oldest. A saved frame is used only while ≥120×48 points of its title
  bar land on a screen's `WorkArea`; otherwise the window is centred at 1200×800
  capped to the primary work area. Windows are created hidden and placed on
  `WindowRuntimeReady` (screens exist only after `Run`), then zoom is applied.
  Maximised/fullscreen saves that state plus the normal frame.
  `saveWindowGeometry` writes on the first window's close or quit; later windows
  never overwrite it.
- **Saving writes what the frontend holds.** `SaveDocument` (`save.go`) returns
  `Conflict: true` without writing when mtime moved since load; dialogs are
  native. Each window pushes `SetUnsavedChanges`; `ShouldQuit`/`WindowClosing`
  cancel and emit `app:confirm-close` (`close`/`quit`), answered with
  `ResolveClose(window, quit)`.
- **Encoding is preserved.** `LoadDocument` strips a UTF-8 BOM and reports
  `lf`/`crlf`; the model's EOL is set and restored on save. Invalid UTF-8 opens
  with U+FFFD and `invalid_utf8`; save refuses until `allow_invalid_utf8`. A NUL
  byte still means binary.
- **Watching polls.** `watch.go` stats open buffers and expanded folders every
  second and emits `app:watched-changed` only to registered windows. Clean
  buffers reload; dirty ones stay with a notice; the tree refreshes.
  `SaveDocument` rebaselines its own write. **View → Refresh** forces a reread.
- **Git is read-only.** `git.go` runs `status --porcelain=v2`, `diff HEAD` and
  `show HEAD:<path>`; never stages/commits/checks out. Status feeds marks, the
  Changes view and status bar; baseline feeds gutter diff; diff tabs are
  read-only and excluded from the session.
- **Printing is native.** `PrintDocument` → AppKit print panel as a sheet
  (`print_darwin.go`); `print_unsupported.go` covers other builds.
- **`kaze` is a shell launcher.** `install.go` writes `/usr/local/bin/kaze`
  execing the bundle (escalating via `osascript` when needed); a single-instance
  lock forwards later invocations to the running window.
- **Settings live in `settings.json`** (`Kaze/settings.json`), created with
  defaults. Keep `frontend/src/settings.ts` defaults in sync with Go (TS is only
  the fallback); `loadSettings` decodes over defaults so absent keys keep theirs.
  Sections: editor, sidebar, files, search, recent, session, theme, ui, window.
  Write paths use `settingsMu` and `settingswrite.go`, which rewrites only
  changed keys so unknown keys and formatting survive. The overlay edits
  configurable sections; recent files, session and window frame are state.
- **Appearance is a Zed theme family.** `themes.go` embeds `themes/kaze.json` and
  reads `<config>/Kaze/themes`. UI keys become CSS variables, `editor.*` Monaco
  colors, `syntax` token rules (`theme.ts`). Missing keys fall back; malformed
  files are reported without hiding the rest; a user variant shadows a built-in.
  `configwatch.go` polls the folder and keybindings, emitting
  `app:config-changed`.
- **First frame uses a cached palette.** WKWebView paints white before async
  settings/themes arrive, so `applyThemePalette` caches surface colors in
  `localStorage` (`kaze.theme.startup`); the inline script in `index.html` and
  `main.ts` paint it, and `main.ts` selects a throwaway Monaco theme. Keep the
  inline copy's key/shape/hex test in sync with `theme.ts`.
- **Keybindings are VS Code-shaped.** `keybindings.json` beside settings;
  `keybindings.go` accepts JSONC, validates keys and warns on bad entries.
  `menu.go` omits a default accelerator claimed by a custom binding.
  `keybindings.ts` installs one capture-phase listener with chords and `when`
  contexts (`editorFocus`, `sidebarVisible`, `hasActiveTab`, `hasDirtyTab`, `!`,
  `&&`, `||`).
- **Settings overlay is frontend state.** One section per settings block; edits
  save as they commit (editor saves chained). Native Settings… emits
  `app:settings-overlay`; `OpenDefaultSettings` writes `default-settings.json`;
  Appearance radio emits `app:appearance` and `SetAppearanceMode` syncs the
  checkmark.
- **Window commands are native.** `window.go`: `MinimiseWindow`, `ZoomWindow`,
  `ToggleFullScreen`; `windows.go`: `NewWindow`, `CloseWindow`,
  `MoveTabToNewWindow`, `OpenFolderInNewWindow`, `BringAllToFront`.
- **Editor commands forward Go → frontend.** `menu.go` builds the Edit, Selection
  and View menus and emits `app:find`, `app:replace`, `app:go-to-line`,
  `app:find-next-result`, `app:find-previous-result`, `app:transform-case`,
  `app:trim-whitespace`, `app:selection-command`, `app:zoom`,
  `app:toggle-preview`, `app:toggle-view`. `editorcommands.ts` listens;
  `EditorPane.runEditorAction`/`runSelectionCommand` map ids to Monaco actions;
  `App.SetViewOption` moves a checkmark for palette toggles. Monaco options live
  in `editor.ts`; indentation is on the model. F4/⇧F4 step Find in Files results.
- **Language and status bar are frontend state.** `editor.ts` resolves language:
  per-tab override, else path, else `#!`; Auto Detect clears the override.
  `StatusBar.svelte` shows cursor/selection, line ending (click toggles LF/CRLF
  and dirties), language (click opens the palette) and invalid-UTF-8 warnings.
- **Find in Files is Go-side.** `SearchProject`/`ReplaceInProject` (`search.go`)
  walk roots with excludes + `.gitignore`, returning capped flat matches;
  `SearchPanel.svelte` is the form and `search.ts` groups/summarizes. Literal
  queries are quoted; Replace All uses literal replacement unless regex mode,
  confirms counts, edits open files through Monaco unsaved, skips files whose
  mtime moved, and rebaselines the watcher for written files.
- **Preview is dependency-free.** `markdown.ts` renders a Markdown subset with
  escaping and limited link schemes; SVG previews use an `<img>` data URL.
  `case.ts`/`text.ts` are covered by Vitest.
- **Keep Monaco out of Vite pre-bundling.** `optimizeDeps.exclude:
  ['monaco-editor']`; pre-bundling breaks worker URLs.
- **Assets are build output.** `main.go` embeds `all:frontend/dist`; build the
  frontend first (`pnpm run build` or the package task).
- **Frontend conventions.** `gofmt`; TS strict (`noUnusedLocals`,
  `noUnusedParameters`, `noImplicitReturns`), 4 spaces, single quotes, named
  exports. Svelte 5 runes (`$state`/`$props`/`$derived`/`$effect`, not
  `export let`/`$:`); component styles scoped, shared tokens in `style.css`;
  `FolderTree` imports itself for recursion (`<svelte:self>` is deprecated). Use
  pnpm only (`pnpm-lock.yaml`; never `package-lock.json`);
  `pnpm-workspace.yaml` allows the `esbuild` build script.
- **Tests are colocated.** Go tests beside code; Vitest covers pure TS helpers,
  not Svelte/Monaco/bindings. Test logic in Go, presentation helpers in TS.
- **Version lives in `build/config.yml`** (`info.version`). Bump it, then run
  `wails3 task common:update:build-assets`. `about.go` embeds the same file;
  `scripts/config.sh` reads it.
- **Bundle ID is `net.nilambar.kaze`** (`info.productIdentifier`, written into
  the plists by build-assets). Changing it resets macOS permissions/prefs.
- **Scope discipline:** no new dependencies or config files without explicit
  approval.

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

`gofmt -l .` must print nothing. Native/Wails changes also need
`wails3 task darwin:package` to succeed.
