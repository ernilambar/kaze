# Kaze

A lightweight macOS editor for code and plain text.

## Features

- **Tabs and multiple windows** — each window has its own tabs and sidebar.
- **Multi-root folder sidebar** — with file operations, drag to move and Trash.
- **Workspaces** — committable `.kaze-workspace` files; opens `.code-workspace`.
- **Quick Open** (<kbd>Cmd</kbd>+<kbd>P</kbd>) and **command palette**
  (<kbd>Cmd</kbd>+<kbd>Shift</kbd>+<kbd>P</kbd>).
- **Find in Files** — regex, globs, `.gitignore`-aware, Replace All.
- **Monaco editor** — syntax highlighting, multi-cursor, case transforms.
- **Markdown and SVG preview**, plus an image viewer.
- **Live reload** — external changes reload clean tabs and refresh the tree.
- **Git status** — file and folder marks in the tree, plus gutter diff (read-only).
- **Zed themes** and VS Code-style keybindings.
- **`kaze` command line tool**.

## Install

The recommended way is the Homebrew tap:

```sh
brew install --cask ernilambar/tap/kaze
```

Apple silicon and macOS 15 or later only. The cask installs `Kaze.app` into
`/Applications` and clears the quarantine flag, so the first launch works
immediately. Upgrade with `brew upgrade --cask kaze`.

### Manual download

Download `Kaze-macos-arm64.zip` from the latest release, unzip it and move
`Kaze.app` to `/Applications`.

The app is not notarized, so macOS blocks the first launch. Clear the
quarantine flag once:

```sh
xattr -dr com.apple.quarantine /Applications/Kaze.app
```

Or open it, then choose **Open Anyway** in **System Settings → Privacy &
Security**.

## Command line

Install the tool from **Kaze → Install CLI…**, then:

```sh
kaze notes.txt  # open a file
kaze .          # add a folder to the sidebar
kaze --help     # usage without opening a window
```

Details: [docs/cli.md](docs/cli.md).

## Docs

- [Keyboard shortcuts](docs/keymaps.md) — defaults and `keybindings.json`
- [Settings](docs/settings.md) — `~/Library/Application Support/Kaze/settings.json`
- [Development](docs/development.md) — build, hot reload, release

## License

[MIT](LICENSE) © [Nilambar Sharma](https://www.nilambar.net)
