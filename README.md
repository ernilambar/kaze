# Kova

A minimal text editor for macOS.

## Features

- **Tabs** — one tab per open file, each with its own buffer and undo history;
  a dot marks unsaved edits, and closing the last tab returns to the welcome
  screen.
- **Welcome screen** — with no open tab, the editor shows the Kova icon over
  three actions and their shortcuts: New File, Open File and Add Folder.
- **Workspaces** — Open, Save As and Close; the workspace file keeps folders
  relative to itself, so it can be committed with the project.
- **Multiple windows** — New Window, Close Window, Move Tab to New Window,
  Open Folder in New Window and Bring All to Front, alongside Minimize, Zoom and
  Toggle Full Screen from the Window menu or the command palette. Each window
  has its own tabs and sidebar; settings and themes are shared.
- **File handling** — New, Open, Save, Save As and Save All. Closing a dirty
  tab, window or quit prompts to save, and saving over a file another program
  changed warns first.
- **Folder sidebar** — several folders as roots, children read only on
  expansion, [File Icons](https://github.com/file-icons/icons) glyphs matched
  by name, and the list persisted in `settings.json`. **Toggle Sidebar**
  (<kbd>Cmd</kbd>+<kbd>B</kbd>) and **Close All Folders** are in the View and
  File menus.
- **Quick Open** (<kbd>Cmd</kbd>+<kbd>P</kbd>) — fuzzy-find a file by name or
  path across the sidebar folders, skipping whatever `files.exclude` hides.
- **Find in Files** (<kbd>Cmd</kbd>+<kbd>Shift</kbd>+<kbd>F</kbd>) — search the
  sidebar folders with literal or regular-expression matching, case and
  include-glob options, results in a tab, and Replace All. Excludes and every
  `.gitignore` are honoured.
- **Navigation** — Open Recent (<kbd>Ctrl</kbd>+<kbd>R</kbd>) with Clear Recent,
  Reopen Closed Tab, next/previous tab, jump to a tab with
  <kbd>Cmd</kbd>+<kbd>1</kbd>–<kbd>9</kbd>, and Reveal in Finder or copy the
  absolute, relative or file name from the File menu.
- **Fresh state** — Kova watches the open buffers and expanded folders, so
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
- **View toggles and zoom** — word wrap, invisible characters, line numbers,
  indent guides, folding, auto-close brackets and smart indent from the View
  menu; zoom in, out and back to actual size.
- **Image viewer** — PNG, JPEG, GIF, WebP, BMP, TIFF and ICO open in a viewer
  with Cmd-scroll zoom, fit/actual-size toggle and the image's pixel size in
  the status bar (up to 32 MB).
- **Native printing** — the macOS print panel for the active buffer, laid out
  in the editor font; images print scaled to one page.
- **Command line tool** — open files or add folders from a shell.

## Command line

Install the tool from **Kova → Install Command Line Tool…**, then:

```sh
kova notes.txt  # open a file
kova .          # add a folder to the sidebar
kova --help     # usage without opening a window
```

Argument rules and current limitations: [docs/cli.md](docs/cli.md).

## Settings

Kova creates `settings.json` in the platform config directory on first launch
and reads editor settings from it. See [docs/settings.md](docs/settings.md) for
locations, supported fields and validation rules.

## Development

Building from source, running with hot reload, and releasing: see
[docs/development.md](docs/development.md).

## License

[MIT](LICENSE) © [Nilambar Sharma](https://www.nilambar.net)
