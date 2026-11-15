# Command line tool

Kova can open files and folders straight from a terminal.

```sh
kova notes.txt      # open one file
kova .              # open a folder
kova one.txt two.txt  # two verbs, one window (see Limitations)
```

## Installing

Choose **Kova → Install Command Line Tool…** from the app menu. Kova writes a
launcher to `/usr/local/bin/kova`; because that directory is protected, macOS
asks for your administrator password. Cancelling the prompt leaves everything
unchanged.

The launcher is a small shell script that runs the app binary inside the
bundle. It records that path, so **re-run the menu item after moving or
renaming `Kova.app`** — otherwise the installed `kova` points at an
executable that no longer exists. Re-running the item is also how you
uninstall and reinstall; to remove the tool, delete `/usr/local/bin/kova`.

Opening a new terminal afterwards is not required: `/usr/local/bin` is already
on the default `PATH`. A shell started before installing may offer a stale
command cache; if `kova` reports `command not found`, run `hash -r`.

## What each argument does

| Argument | Result |
| --- | --- |
| A file | Opened in the editor, replacing the welcome buffer. |
| A folder | Remembered as the working folder, shown in the title bar. |
| Several arguments | The first readable file is opened. |
| A missing path | Reported in the status bar, and the rest still open. |
| Flags (`-x`, `--anything`) | Ignored, so they cannot be mistaken for files. |
| `--` | Everything after it is treated as a path, even if it starts with `-`. |

Relative paths are resolved by the installed launcher against the directory you
ran it from, so `kova notes.txt` works from any shell. If Kova is already
running, the arguments are forwarded to that window and it comes to the front;
a second window is never created.

## Limitations

- **One file at a time.** Opening two files keeps the first; multiple tabs
  arrive in a later phase.
- **Folders are remembered, not browsed.** There is no file tree yet, so `kova .`
  records the folder and shows it in the title bar. The status bar says so
  explicitly rather than implying the folder was opened.
- **Text files only.** Anything over 10 MB, or content that is not valid UTF-8,
  is refused with a message instead of being rendered as garbage.

## How forwarding works

Kova runs a single instance. A second `kova` invocation detects the running
copy, hands it its arguments, and exits, so the shell prompt returns
immediately. The running window is unminimised and raised.

Arguments are passed as absolute paths by the launcher. Wails reports the
second instance's working directory as the bundle's executable directory, so
resolving relative paths on the receiving side would point at the wrong
place — the launcher does it while the caller's directory is still known.
