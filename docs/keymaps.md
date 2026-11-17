# Keyboard Shortcuts

Defaults on macOS. They follow VS Code; the differences are listed under
[Differences from VS Code](#differences-from-vs-code). Any of them can be
changed in `keybindings.json`; see [Custom keybindings](#custom-keybindings).

## Files & tabs

| Shortcut | Keys | Action |
| --- | --- | --- |
| `⌘N` | `Cmd + N` | New Buffer (untitled) |
| `⌃⌥⌘N` | `Ctrl + Option + Cmd + N` | New File… (named, on disk) |
| `⌘⇧N` | `Cmd + Shift + N` | New Window |
| `⌘O` | `Cmd + O` | Open… |
| `⌘P` | `Cmd + P` | Quick Open… |
| `⌃R` | `Ctrl + R` | Open Recent… |
| `⌃⌘O` | `Ctrl + Cmd + O` | Add Folder to Sidebar… |
| `⌘K F` | `Cmd + K`, then `F` | Close All Folders |
| `⌘S` | `Cmd + S` | Save |
| `⌘⇧S` | `Cmd + Shift + S` | Save As… |
| `⌥⌘S` | `Cmd + Option + S` | Save All |
| `⌘W` | `Cmd + W` | Close Tab |
| `⌘K ⌘W` | `Cmd + K`, then `Cmd + W` | Close All Tabs |
| `⌘⇧W` | `Cmd + Shift + W` | Close Window |
| `⌘⇧T` | `Cmd + Shift + T` | Reopen Closed Tab |
| `⌘⇧]` / `⌘⇧[` | `Cmd + Shift + ]` / `Cmd + Shift + [` | Next / Previous Tab |
| `⌃⇥` / `⌃⇧⇥` | `Ctrl + Tab` / `Ctrl + Shift + Tab` | Next / Previous Tab |
| `⌘1` … `⌘8` | `Cmd + 1` … `Cmd + 8` | Jump to Tab |
| `⌘9` | `Cmd + 9` | Jump to Last Tab |
| `⌥⌘R` | `Cmd + Option + R` | Reveal in Finder |
| `⌥⌘C` | `Cmd + Option + C` | Copy File Path |
| `⌥⌘⇧C` | `Cmd + Option + Shift + C` | Copy Relative Path |

## Editing

| Shortcut | Keys | Action |
| --- | --- | --- |
| `⌘/` | `Cmd + /` | Toggle Line Comment |
| `⌘⇧K` | `Cmd + Shift + K` | Delete Line |
| `⌘L` | `Cmd + L` | Select Line |
| `⌃J` / `⌘J` | `Ctrl + J` / `Cmd + J` | Join Lines |
| `⌥↑` / `⌥↓` | `Option + Up` / `Option + Down` | Move Line Up / Down |
| `⇧⌥↑` / `⇧⌥↓` | `Shift + Option + Up` / `Shift + Option + Down` | Copy Line Up / Down |
| `⌘↩` | `Cmd + Enter` | Insert Line Below |
| `⌘⇧↩` | `Cmd + Shift + Enter` | Insert Line Above |
| `⌃G` | `Ctrl + G` | Go to Line… |
| `⌘F` | `Cmd + F` | Find… |
| `⌥⌘F` | `Cmd + Option + F` | Find and Replace… |
| `⌘G` / `⌘⇧G` | `Cmd + G` / `Cmd + Shift + G` | Find Next / Previous |
| `F3` / `⇧F3` | `F3` / `Shift + F3` | Find Next / Previous (alternate) |
| `⌘⇧F` | `Cmd + Shift + F` | Find in Files… |
| `F4` / `⇧F4` | `F4` / `Shift + F4` | Next / Previous Result |
| `⌥⌘G` / `⌥⌘⇧G` | `Cmd + Option + G` / `Cmd + Option + Shift + G` | Next / Previous Result (alternate) |

The usual macOS shortcuts also work: undo and redo (`⌘Z` / `⌘⇧Z`), cut, copy
and paste (`⌘X` / `⌘C` / `⌘V`), select all (`⌘A`), word and line navigation,
and indent.

On Apple keyboards `F3` and `F4` work only with `fn` held, unless the function
keys are set to standard in System Settings. Use `⌘G` and `⌥⌘G` instead.

Inside the Find widget, `⌥⌘C` toggles Match Case and `⌥⌘R` toggles Regex.
Elsewhere in the editor, those keys copy the file path and reveal the file in
Finder.

## Multi-cursor

| Shortcut | Keys | Action |
| --- | --- | --- |
| `⌥⌘↑` / `⌥⌘↓` | `Cmd + Option + Up` / `Cmd + Option + Down` | Add Cursor Above / Below |
| `⌃⌥↑` / `⌃⌥↓` | `Ctrl + Option + Up` / `Ctrl + Option + Down` | Add Cursor Above / Below (alternate) |
| `⌥` + click | `Option` + click | Add a cursor at the click |
| `⇧⌥` + drag | `Shift + Option` + drag | Column (block) selection |
| `⌘⇧L` / `⇧⌥I` | `Cmd + Shift + L` / `Shift + Option + I` | Split into Lines |
| `⌘D` | `Cmd + D` | Add Next Occurrence |
| `⌃⌘G` | `Ctrl + Cmd + G` | Select All Occurrences |

## View & window

| Shortcut | Keys | Action |
| --- | --- | --- |
| `⌘B` | `Cmd + B` | Toggle Sidebar |
| `⌘⇧P` / `F1` | `Cmd + Shift + P` / `F1` | Command Palette |
| `⌘⇧E` | `Cmd + Shift + E` | Show Explorer |
| `⌃⇧G` | `Ctrl + Shift + G` | Show Changes |
| `⌘R` | `Cmd + R` | Refresh |
| `⌥⌘Z` | `Cmd + Option + Z` | Toggle Word Wrap |
| `⌥⌘I` | `Cmd + Option + I` | Toggle Invisible Characters |
| `⌘⇧V` | `Cmd + Shift + V` | Toggle Preview (Markdown, SVG) |
| `⌘=` / `⌘-` / `⌘0` | `Cmd + =` / `Cmd + -` / `Cmd + 0` | Zoom In / Zoom Out / Actual Size |
| `⌘,` | `Cmd + ,` | Settings… |
| `⌘M` | `Cmd + M` | Minimize |
| `⌃⌘F` | `Ctrl + Cmd + F` | Toggle Full Screen |
| `⌘H` / `⌥⌘H` | `Cmd + H` / `Cmd + Option + H` | Hide / Hide Others |
| `⌘Q` | `Cmd + Q` | Quit |

## Differences from VS Code

| Action | Kaze | VS Code | Why |
| --- | --- | --- | --- |
| Toggle Word Wrap | `⌥⌘Z` | `⌥Z` | Option plus a letter types a character (`Ω` on a US layout); `⌥Z` would block it |
| Jump to Tab | `⌘1` … `⌘9` | `⌃1` … `⌃9` | VS Code's `⌘1` … `⌘9` pick editor groups, which Kaze does not have |
| Join Lines | also `⌘J` | `⌃J` only | `⌘J` toggles VS Code's panel; Kaze has none |
| Copy Path, Reveal in Finder | `⌥⌘C`, `⌥⌘R` work in the editor | only outside the editor | In VS Code's editor those keys toggle find options even with Find closed |
| Split into Lines | `⌘⇧L` (and `⇧⌥I`) | `⇧⌥I` | Sublime's key, kept by choice |
| Select All Occurrences | `⌃⌘G` | `⌘⇧L` | `⌘⇧L` is Split into Lines |
| Ctrl+Tab | next tab in order | most recently used | Kaze has no recent-tab picker |

Kaze also adds keys VS Code leaves unbound: `⌃⌘O`, `⌘R`, `⌥⌘G` / `⌥⌘⇧G`,
`⌃⌥↑` / `⌃⌥↓`, `⌥⌘I`, `⌥⌘Z`.

## Modifier keys

| Symbol | Key |
| --- | --- |
| `⌘` | Command |
| `⇧` | Shift |
| `⌥` | Option |
| `⌃` | Control |
| `↩` | Return (Enter) |

## Custom keybindings

**Kaze → Open Keybindings…** opens `keybindings.json`, next to
`settings.json`. It uses VS Code's format and may contain comments:

```jsonc
[
  { "key": "cmd+k cmd+s", "command": "save-all" },
  { "key": "cmd+shift+o", "command": "add-folder" },
  { "key": "cmd+shift+]", "command": "-next-tab" }
]
```

- `key` is one chord or a space-separated chord sequence; a binding needs a
  modifier unless the key does not type (`f1`–`f24`, arrows, `enter`, …).
- `command` is a command palette id, such as `save`, `next-tab`,
  `toggle-wordWrap` or `go-to-tab-9`. A leading `-` removes that command's
  default keys.
- `when` is optional: `editorFocus`, `sidebarVisible`, `hasActiveTab`,
  `hasDirtyTab`, combined with `!`, `&&` and `||`.

A binding that takes a default key replaces it. Changes apply without a
restart.
