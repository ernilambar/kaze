# Keyboard shortcuts

macOS defaults. See [Custom keybindings](#custom-keybindings) to change them.

`⌘` Command · `⇧` Shift · `⌥` Option · `⌃` Control · `↩` Return

## Files & tabs

| Shortcut | Action |
| --- | --- |
| `⌘N` | New Buffer (untitled) |
| `⌃⌥⌘N` | New File… (named, on disk) |
| `⌘O` | Open… |
| `⌘P` | Quick Open… |
| `⌃R` | Open Recent… |
| `⌃⌘O` | Add Folder to Sidebar… |
| `⌘K F` | Close All Folders |
| `⌘S` | Save |
| `⌘⇧S` | Save As… |
| `⌥⌘S` | Save All |
| `⌘W` | Close Tab |
| `⌘K ⌘W` | Close All Tabs |
| `⌘⇧T` | Reopen Closed Tab |
| `⌃⇥` / `⌃⇧⇥` | Next / Previous Tab |
| `⌘⇧]` / `⌘⇧[` | Next / Previous Tab (alternate) |
| `⌘1` … `⌘8` | Jump to Tab |
| `⌘9` | Jump to Last Tab |
| `⌥⌘R` | Reveal in Finder |
| `⌥⌘C` | Copy File Path |
| `⌥⌘⇧C` | Copy Relative Path |

## Editing

| Shortcut | Action |
| --- | --- |
| `⌘F` | Find… |
| `⌥⌘F` | Replace… |
| `⌘G` / `⌘⇧G` | Find Next / Previous |
| `F3` / `⇧F3` | Find Next / Previous (alternate) |
| `⌘⇧F` | Find in Files… |
| `F4` / `⇧F4` | Next / Previous Result |
| `⌥⌘G` / `⌥⌘⇧G` | Next / Previous Result (alternate) |
| `⌃G` | Go to Line… |
| `⌘/` | Toggle Line Comment |
| `⌘⇧K` | Delete Line |
| `⌘L` | Select Line |
| `⌃J` / `⌘J` | Join Lines |
| `⌥↑` / `⌥↓` | Move Line Up / Down |
| `⇧⌥↑` / `⇧⌥↓` | Copy Line Up / Down |
| `⌘↩` / `⌘⇧↩` | Insert Line Below / Above |

## Multi-cursor

| Shortcut | Action |
| --- | --- |
| `⌥⌘↑` / `⌥⌘↓` | Add Cursor Above / Below |
| `⌃⌥↑` / `⌃⌥↓` | Add Cursor Above / Below (alternate) |
| `⌘D` | Add Next Occurrence |
| `⌃⌘G` | Select All Occurrences |
| `⌘⇧L` / `⇧⌥I` | Split Selection into Lines |
| `⌥` + click | Add a cursor |
| `⇧⌥` + drag | Column selection |

## View & window

| Shortcut | Action |
| --- | --- |
| `⌘⇧P` / `F1` | Command Palette |
| `⌘B` | Toggle Sidebar |
| `⌘⇧E` | Show Explorer |
| `⌃⇧G` | Show Changes |
| `⌘R` | Refresh |
| `⌥⌘Z` | Toggle Word Wrap |
| `⌥⌘I` | Toggle Invisible Characters |
| `⌘⇧V` | Toggle Preview (Markdown, SVG) |
| `⌘=` / `⌘-` / `⌘0` | Zoom In / Out / Actual Size |
| `⌘,` | Settings… |
| `⌘⇧N` | New Window |
| `⌘⇧W` | Close Window |
| `⌘M` | Minimize |
| `⌃⌘F` | Toggle Full Screen |
| `⌘H` / `⌥⌘H` | Hide / Hide Others |
| `⌘Q` | Quit |

Standard macOS editing keys work too: undo/redo, cut/copy/paste, select all,
word and line navigation, and indent.

On Apple keyboards `F3` and `F4` require `fn` unless function keys are set to
standard in System Settings; use `⌘G` and `⌥⌘G` instead. Inside the Find
widget, `⌥⌘C` and `⌥⌘R` toggle Match Case and Regex; elsewhere they copy the
file path and reveal the file.

## Custom keybindings

**Kaze → Open Keybindings…** opens `keybindings.json` beside `settings.json`.
It uses VS Code's JSONC format:

```jsonc
[
  { "key": "cmd+k cmd+s", "command": "save-all" },
  { "key": "cmd+shift+o", "command": "add-folder" },
  { "key": "cmd+shift+]", "command": "-next-tab" }
]
```

- `key` is one chord or a space-separated sequence. A binding needs a modifier
  unless the key does not type (`f1`–`f24`, arrows, `enter`, …).
- `command` is a command palette id. A leading `-` removes that command's
  default keys.
- `when` is optional: `editorFocus`, `sidebarVisible`, `hasActiveTab`,
  `hasDirtyTab`, combined with `!`, `&&`, and `||`.

A binding that takes a default key replaces it. Changes apply without a
restart.
