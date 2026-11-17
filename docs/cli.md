# Command line tool

Kaze can open files and folders straight from a terminal.

```sh
kaze notes.txt           # open one file
kaze .                   # add a folder to the sidebar
kaze one.txt two.txt     # open both files, each in a tab
```

## Installing

Choose **Kaze → Install Command Line Tool…** from the app menu. Kaze writes a
launcher to `/usr/local/bin/kaze`; because that directory is protected, macOS
asks for your administrator password. Cancelling the prompt leaves everything
unchanged.

The launcher is a small shell script that runs the app binary inside the
bundle. It records that path, so **re-run the menu item after moving or
renaming `Kaze.app`** — otherwise the installed `kaze` points at an
executable that no longer exists. Re-running the item is also how you
uninstall and reinstall; to remove the tool, delete `/usr/local/bin/kaze`.

Opening a new terminal afterwards is not required: `/usr/local/bin` is already
on the default `PATH`. A shell started before installing may offer a stale
command cache; if `kaze` reports `command not found`, run `hash -r`.

## What each argument does

| Argument | Result |
| --- | --- |
| A file | Opened in a new tab, or focused if it is already open. |
| A folder | Added to the sidebar as a root, and saved for next time. |
| Several arguments | Every folder becomes a root; every readable file opens in its own tab. |
| A missing path | Reported in the status bar, and the rest still open. |
| `-h`, `-help`, `--help` | Print usage and exit without opening a window. |
| `-v`, `-V`, `-version`, `--version` | Print the version and exit. |
| `-n`, `--new-window` | Open a new window for the other arguments instead of the focused one, even with no paths. |
| Other flags (`-x`, `--anything`) | Ignored, so they cannot be mistaken for files. |
| `--` | Everything after it is treated as a path, even if it starts with `-`. |

Relative paths are resolved by the installed launcher against the directory you
ran it from, so `kaze notes.txt` works from any shell. If Kaze is already
running, the arguments are forwarded to the focused window and it comes to the
front; a second process is never started. `--new-window` changes that: the
arguments open in a window of their own, leaving the focused one as it was. A
cold launch has no focused window to reuse, so its startup window is already
the new window and the flag makes no difference there.

## Help and version

`kaze --help` prints the argument rules and `kaze --version` prints the version
taken from `info.version` in `build/config.yml`. Both are answered by the
invoked process, so they work even while Kaze is running and never raise the
window. Help wins when both flags are given. Nothing after `--` is read as a
flag, so `kaze -- --help` looks for a file named `--help`.

## Limitations

- **Edits are not saved to disk.** Tabs keep their buffers for the session;
  closing a tab or quitting the app discards changes made in it.
- **One root per folder.** Naming a folder that is already in the sidebar does
  not add a second copy of it.
- **Text files and images.** Text over 10 MB, or content that is not valid
  UTF-8, is refused with a message instead of being rendered as garbage.
  Supported images open in the viewer; images over 32 MB are refused.

## How forwarding works

Kaze runs a single instance. A second `kaze` invocation detects the running
copy, hands it its arguments, and exits, so the shell prompt returns
immediately. The running window is unminimised and raised.

Arguments are passed as absolute paths by the launcher. Wails reports the
second instance's working directory as the bundle's executable directory, so
resolving relative paths on the receiving side would point at the wrong
place — the launcher does it while the caller's directory is still known.
