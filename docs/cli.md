# Command line tool

```sh
kaze notes.txt       # open or focus a file
kaze .               # add a folder to the sidebar
kaze a.txt b.txt     # open both
```

## Install

**Kaze → Install CLI…** writes `/usr/local/bin/kaze` (asks for an
admin password if needed). Re-run it after moving `Kaze.app`; delete the file
to uninstall.

## Arguments

| Argument | Result |
| --- | --- |
| A file | Opens in a tab, or focuses the existing one. |
| A folder | Becomes a saved sidebar root. |
| A missing path | Reported in the status bar; the rest still open. |
| `-h`, `--help` | Prints usage and exits. |
| `-v`, `--version` | Prints the version and exits. |
| `-n`, `--new-window` | Opens the arguments in a new window. |
| `--` | Treats everything after it as a path. |

Other flags are ignored. Relative paths resolve from the current directory.

## Behaviour

- Kaze runs one process; later invocations hand their arguments to the focused
  window.
- Help and version never raise the window.
- Files over 10 MB, binary files and images over 32 MB are refused.
