# Command line tool

Kaze opens files and folders from a terminal.

```sh
kaze notes.txt       # open or focus a file
kaze .               # add a folder to the sidebar
kaze a.txt b.txt     # open both
```

## Install

Choose **Kaze → Install Command Line Tool…**. Kaze writes
`/usr/local/bin/kaze`, asking for an administrator password when needed;
cancelling changes nothing.

The launcher contains the bundle's path, so re-run the menu item after moving
or renaming `Kaze.app`. Delete `/usr/local/bin/kaze` to uninstall.

## Arguments

| Argument | Result |
| --- | --- |
| A file | Opens in a tab, or focuses the existing tab. |
| A folder | Becomes a sidebar root and is saved for next launch. |
| Several paths | Every folder becomes a root; every readable file opens. |
| A missing path | Reported in the status bar; the rest still open. |
| `-h`, `-help`, `--help` | Prints usage and exits. |
| `-v`, `-V`, `-version`, `--version` | Prints the version and exits. |
| `-n`, `--new-window` | Opens the arguments in a new window, even with no paths. |
| `--` | Treats everything after it as a path. |

Other flags are ignored, so they cannot be mistaken for files. Relative paths
are made absolute by the launcher, from the directory the command ran in.

## Behaviour

Kaze runs one process. A second invocation hands its arguments to the focused
window and exits; `--new-window` gives the arguments a window of their own. A
cold launch already creates a window, so the flag makes no difference there.

Help and version are answered by the invoked process, so they work while Kaze
is running and never raise the window. Help wins when both are requested.

## Limits

- Opening uses the editor's normal rules: text over 10 MB, binary content, and
  images over 32 MB are refused with a message.
- Invalid UTF-8 opens with replacement characters; saving it asks for
  confirmation.
- A folder already in the sidebar is not added twice.
