# Kova

A minimal text editor for macOS.

## Files

**File → New File** (<kbd>Cmd</kbd>+<kbd>N</kbd>) opens an empty buffer, and its
first **Save** asks where to put it. **Open…** (<kbd>Cmd</kbd>+<kbd>O</kbd>) picks
a file from disk; **Save** (<kbd>Cmd</kbd>+<kbd>S</kbd>), **Save As…**
(<kbd>Cmd</kbd>+<kbd>Shift</kbd>+<kbd>S</kbd>) and **Save All**
(<kbd>Cmd</kbd>+<kbd>Alt</kbd>+<kbd>S</kbd>) write buffers back.

A buffer with edits that are not on disk shows a dot on its tab. Closing that
tab, closing the window or quitting asks whether to save first. Saving a file
another program changed in the meantime warns before overwriting it, so the
other program's work is never lost by accident.

**Print…** opens the macOS print panel for the active buffer, laid out in the
editor font; images print scaled to one page.

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

## Tabs

Every opened file gets a tab above the editor. Click a tab to switch to it, or
close it with its `×` or **File → Close Tab** (<kbd>Cmd</kbd>+<kbd>W</kbd>).
Each tab keeps its own buffer and undo history, so switching away and back does
not lose edits, and a dot marks a buffer with unsaved changes. Closing the last
tab returns to the welcome buffer.

## Images

PNG, JPEG, GIF, WebP, BMP, TIFF and ICO files open in a viewer instead of the
editor. A new image is fitted to the pane without being enlarged past its
natural size. Hold <kbd>Cmd</kbd> while scrolling to zoom around the pointer,
and double-click to toggle between fit and actual size; plain scrolling pans a
larger image. Images up to 32 MB are loaded.

## Quick Open

Press **File → Quick Open…** (<kbd>Cmd</kbd>+<kbd>P</kbd>) to fuzzy-find a file
by name or path across the sidebar folders. Type to narrow the list, move with
the arrow keys, and press <kbd>Enter</kbd> to open the selected file;
<kbd>Esc</kbd> dismisses the palette. Version-control metadata and
`node_modules` are skipped so generated trees do not bury your files.

## Command palette

Press **File → Command Palette…** (<kbd>Cmd</kbd>+<kbd>Shift</kbd>+<kbd>P</kbd>)
to run a command by name. It lists the same actions as the menus—new file,
open, save, save as, save all, print, Quick Open, adding a folder, closing a
tab, Settings, and installing the command line tool—and ranks matches exactly
like Quick Open. Type to narrow the list, use the arrow keys to move, and press
<kbd>Enter</kbd> to run the selected command; <kbd>Esc</kbd> dismisses the
palette.

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
