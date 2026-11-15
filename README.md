# Kova

A minimal text editor for macOS.

## Sidebar

Add folders with the `+` button in the sidebar header or
**File → Add Folder to Sidebar…** (<kbd>Cmd</kbd>+<kbd>Shift</kbd>+<kbd>O</kbd>).
Several folders can be open at once, each as its own root; children are read
only when a folder is expanded. Click a file to open it in the editor, and use
the `×` on a root to remove it. Files and folders carry
[File Icons](https://github.com/file-icons/icons) glyphs matched by name, so
`.ts`, `.rs`, `Dockerfile` and friends are recognisable at a glance. The list is
saved in `settings.json`, so it comes back on the next launch.

`kova .` adds the current directory to the sidebar in the same way. See
[docs/cli.md](docs/cli.md) for the full argument rules and current
limitations, and [docs/settings.md](docs/settings.md) for the saved format.

## Command line

Install the tool from **Kova → Install Command Line Tool…**, then:

```sh
kova notes.txt  # open a file
kova .          # add a folder to the sidebar
kova --help     # usage without opening a window
```

## Settings

Kova reads editor settings from `settings.json` in the platform config
directory and creates it with defaults on first launch. See
[docs/settings.md](docs/settings.md) for the file locations, supported fields,
and validation rules.

## Development

Building from source, running with hot reload, and releasing: see
[docs/development.md](docs/development.md).

## License

[MIT](LICENSE) © [Nilambar Sharma](https://www.nilambar.net)
