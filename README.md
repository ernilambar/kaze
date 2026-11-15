# Kova

A minimal text editor for macOS.

## Features

- **Tabs** — one tab per open file, each with its own buffer and undo history;
  a dot marks unsaved edits, and closing the last tab returns to the welcome
  buffer.
- **File handling** — New, Open, Save, Save As and Save All. Closing a dirty
  tab, window or quit prompts to save, and saving over a file another program
  changed warns first.
- **Folder sidebar** — several folders as roots, children read only on
  expansion, [File Icons](https://github.com/file-icons/icons) glyphs matched
  by name, and the list persisted in `settings.json`.
- **Quick Open** (<kbd>Cmd</kbd>+<kbd>P</kbd>) — fuzzy-find a file by name or
  path across the sidebar folders, skipping version-control metadata and
  `node_modules`.
- **Command palette** (<kbd>Cmd</kbd>+<kbd>Shift</kbd>+<kbd>P</kbd>) — run any
  menu action by name.
- **Image viewer** — PNG, JPEG, GIF, WebP, BMP, TIFF and ICO open in a viewer
  with Cmd-scroll zoom and fit/actual-size toggle (up to 32 MB).
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
