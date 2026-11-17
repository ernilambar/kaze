# Kaze

A minimal text editor for macOS.

## Features

- **Tabs** — one tab per open file, each with its own buffer and undo history;
  a dot marks unsaved edits, and closing the last tab returns to the welcome
  screen. Right-click a tab to close it, move it to a new window, close the
  other tabs, the tabs to its right, the tabs whose files were deleted, or all
  tabs.
- **Welcome screen** — with no open tab, the editor shows the Kaze icon over
  three actions and their shortcuts: New File, Open File and Add Folder.
- **Workspaces** — Open, Save As and Close; the workspace file keeps folders
  relative to itself, so it can be committed with the project. Each workspace
  has its own tab set parked in `session.json`, restored when you switch back.
  VS Code `.code-workspace` files open too (folders only).
- **Multiple windows** — New Window, Open Folder in New Window and Close Window
  from the File menu; Move Tab to New Window, Bring All to Front, Minimize, Zoom
  and Toggle Full Screen from the Window menu; all of them from the command
  palette. Each window
  has its own tabs and sidebar; settings and themes are shared.
- **File handling** — New File, New Folder, New Buffer, Open, Save, Save As and
  Save All. Closing a dirty tab, window or quit prompts to save, and saving over
  a file another program changed warns first. Drag files or folders in from
  Finder; files open as tabs, folders join the sidebar, and holding
  <kbd>Option</kbd> opens the drop in a new window.
- **Folder sidebar** — several folders as roots, children read only on
  expansion, [File Icons](https://github.com/file-icons/icons) glyphs matched
  by name, and the list persisted in `session.json`. **Toggle Sidebar**
  (<kbd>Cmd</kbd>+<kbd>B</kbd>) and **Close All Folders** are in the View and
  File menus.
- **File tree operations** — Open, New File, New Folder, Rename, Duplicate,
  Cut/Copy/Paste, Move, Copy Name/Path/Relative Path, Move to Trash (the macOS
  Trash, not an unlink) and drag to move, from each row's context menu or a drop
  on a folder row. A renamed or moved file keeps its tab open, with its edits
  and undo history.
- **Quick Open** (<kbd>Cmd</kbd>+<kbd>P</kbd>) — fuzzy-find a file by name or
  path across the sidebar folders, skipping whatever `files.exclude` hides.
- **Find in Files** (<kbd>Cmd</kbd>+<kbd>Shift</kbd>+<kbd>F</kbd>) — search the
  sidebar folders with literal or regular-expression matching, case options,
  include and exclude globs, results in a tab, and Replace All.
  <kbd>F4</kbd>/<kbd>Shift</kbd>+<kbd>F4</kbd> hop through the matches.
  Excludes from settings and every `.gitignore` are honoured.
- **Navigation** — Open Recent (<kbd>Ctrl</kbd>+<kbd>R</kbd>; Clear Recent is in
  the palette), Reopen Closed Tab, and from the Go menu Quick Open, Go to Line
  (<kbd>Ctrl</kbd>+<kbd>G</kbd>), next/previous tab and jump to a tab with
  <kbd>Cmd</kbd>+<kbd>1</kbd>–<kbd>9</kbd>; Find/Replace from the Edit menu, and Reveal in
  Finder or copy the absolute, relative or file name from the File menu.
- **Fresh state** — Kaze watches the open buffers and expanded folders, so
  external edits reload clean tabs and the tree follows new files; **View →
  Refresh** rereads on demand. Restored tabs reopen on the next launch.
- **Command palette** (<kbd>Cmd</kbd>+<kbd>Shift</kbd>+<kbd>P</kbd>) — run any
  menu action by name.
- **Syntax highlighting** — the language comes from the file's extension and
  the status bar names it.
- **Markdown and SVG preview** (<kbd>Ctrl</kbd>+<kbd>Cmd</kbd>+<kbd>V</kbd>) —
  render the buffer in place, unsaved edits included.
- **Text transforms** — upper, lower, title, camel, pascal, snake, kebab and
  constant case for the selection (or the whole buffer), plus trim trailing
  whitespace.
- **Selection menu** — multi-cursor (add cursor above/below, add the next
  occurrence, split the selection into lines), column selection, move/copy/join
  or delete lines, and toggle the line comment, from the menu or the command
  palette.
- **View toggles and zoom** — word wrap, invisible characters, line numbers,
  indent guides, folding, auto-close brackets and smart indent from the View
  menu; zoom in, out and back to actual size.
- **Image viewer** — PNG, JPEG, GIF, WebP, BMP, TIFF and ICO open in a viewer
  with Cmd-scroll zoom, fit/actual-size toggle and the image's pixel size in
  the status bar (up to 32 MB).
- **Native printing** — the macOS print panel for the active buffer, laid out
  in the editor font; images print scaled to one page.
- **Command line tool** — open files or add folders from a shell.

## Install

Download `Kaze-macos-arm64.zip` from the latest release, unzip it and move
`Kaze.app` to `/Applications`. Apple silicon and macOS 13 or later only.

The app is not notarized, so macOS blocks the first launch. Clear the
quarantine flag once:

```sh
xattr -dr com.apple.quarantine /Applications/Kaze.app
```

Or open it, then choose **Open Anyway** in **System Settings → Privacy &
Security**.

## Command line

Install the tool from **Kaze → Install Command Line Tool…**, then:

```sh
kaze notes.txt  # open a file
kaze .          # add a folder to the sidebar
kaze --help     # usage without opening a window
```

Argument rules and current limitations: [docs/cli.md](docs/cli.md).

## Keyboard shortcuts

Default shortcuts and `keybindings.json`: see [docs/keymaps.md](docs/keymaps.md).

## Settings

Kaze creates `settings.json` in `~/Library/Application Support/Kaze/` on first
launch and reads editor settings from it. See [docs/settings.md](docs/settings.md)
for the supported fields and validation rules.

## Development

Building from source, running with hot reload, and releasing: see
[docs/development.md](docs/development.md).

## License

[MIT](LICENSE) © [Nilambar Sharma](https://www.nilambar.net)
