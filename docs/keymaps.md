# Keyboard shortcuts

`⌘` Command · `⇧` Shift · `⌥` Option · `⌃` Control · `↩` Return

## Files & tabs

| Shortcut | Action |
| --- | --- |
| `⌘N` | New Buffer (untitled) |
| `⌃⌥⌘N` | New File… (on disk) |
| `⌘O` | Open… |
| `⌘P` | Quick Open… |
| `⌃R` | Open Recent… |
| `⌃⌘O` | Add Folder to Sidebar… |
| `⌘S` / `⌘⇧S` / `⌥⌘S` | Save / Save As… / Save All |
| `⌘W` / `⌘K ⌘W` | Close Tab / Close All Tabs |
| `⌘⇧T` | Reopen Closed Tab |
| `⌃⇥` / `⌃⇧⇥` | Next / Previous Tab |
| `⌘1` … `⌘9` | Jump to Tab (`⌘9` = last) |
| `⌥⌘R` | Reveal in Finder |
| `⌥⌘C` / `⌥⌘⇧C` | Copy Path / Relative Path |

## Editing

| Shortcut | Action |
| --- | --- |
| `⌘F` / `⌥⌘F` | Find… / Replace… |
| `⌘G` / `⌘⇧G` | Find Next / Previous |
| `⌘⇧F` | Find in Files… |
| `F4` / `⇧F4` | Next / Previous Result |
| `⌃G` | Go to Line… |
| `⌘/` | Toggle Line Comment |
| `⌘⇧K` | Delete Line |
| `⌥↑` / `⌥↓` | Move Line Up / Down |
| `⌥⌘↑` / `⌥⌘↓` | Add Cursor Above / Below |
| `⌘D` | Add Next Occurrence |
| `⌘⇧L` | Split Selection into Lines |

## View & window

| Shortcut | Action |
| --- | --- |
| `⌘⇧P` | Command Palette |
| `⌘B` | Toggle Sidebar |
| `⌘⇧V` | Toggle Preview |
| `⌥⌘Z` | Toggle Word Wrap |
| `⌘=` / `⌘-` / `⌘0` | Zoom In / Out / Reset |
| `⌘,` | Settings… |
| `⌘⇧N` / `⌘⇧W` | New / Close Window |

Menus show most other shortcuts. Standard macOS editing keys work too.

## Custom keybindings

**Kaze → Open Keybindings…** opens `keybindings.json` (VS Code JSONC format):

```jsonc
[
  { "key": "cmd+k cmd+s", "command": "save-all" },
  { "key": "cmd+shift+]", "command": "-next-tab" }
]
```

- `key`: one chord or a space-separated sequence; needs a modifier unless the
  key doesn't type (`f1`, arrows, `enter`, …).
- `command`: a command palette id; a leading `-` removes its default keys.
- `when` (optional): `editorFocus`, `sidebarVisible`, `hasActiveTab`,
  `hasDirtyTab`, with `!`, `&&`, `||`.

Changes apply without a restart.
