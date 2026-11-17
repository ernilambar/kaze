# Settings

Kaze reads user settings from `settings.json` in the platform config
directory. The file is created with defaults the first time settings are
loaded (at app startup), so no manual setup is required.

| Platform | Path |
| --- | --- |
| macOS | `~/Library/Application Support/Kaze/settings.json` |
| Windows | `%AppData%\Kaze\settings.json` |
| Linux | `~/.config/Kaze/settings.json` (`$XDG_CONFIG_HOME` is honored) |

## Example

```json
{
  "editor": {
    "font_family": "Menlo",
    "font_size": 13,
    "font_weight": 400,
    "tab_size": 4,
    "hard_tabs": false,
    "ligatures": false,
    "cursor_width": 1,
    "cursor_blink": true,
    "cursor_surrounding_lines": 0,
    "scroll_beyond_last_line": true,
    "ensure_final_newline": false,
    "word_wrap": true,
    "invisible_characters": false,
    "line_numbers": true,
    "indent_guides": true,
    "code_folding": true,
    "auto_close_brackets": true,
    "smart_indent": true
  },
  "sidebar": {
    "folders": [
      "/Users/me/projects/kaze"
    ],
    "visible": true,
    "auto_reveal": true
  },
  "files": {
    "exclude": [".git", ".hg", ".svn", "node_modules", ".DS_Store"]
  },
  "search": {
    "exclude": [".git", ".hg", ".svn", "node_modules", ".DS_Store"]
  }
}
```

## Editor settings

| Key | Type | Default | Description |
| --- | --- | --- | --- |
| `editor.font_family` | string | `Menlo` | CSS `font-family` for the editor. Use a monospace font. |
| `editor.font_size` | number | `13` | Font size in pixels. Fractional values are allowed. |
| `editor.font_weight` | number | `400` | CSS font weight; typical values are `100`–`900` (`400` normal, `700` bold). |
| `editor.tab_size` | number | `4` | Columns a tab occupies and an indent level adds. |
| `editor.hard_tabs` | boolean | `false` | Pressing Tab inserts a tab character instead of spaces. |
| `editor.ligatures` | boolean | `false` | Render font ligatures such as `->` and `=>`. |
| `editor.cursor_width` | number | `1` | Width in pixels of the line cursor. |
| `editor.cursor_blink` | boolean | `true` | Blink the cursor; `false` keeps it solid. |
| `editor.cursor_surrounding_lines` | number | `0` | Lines always kept visible above and below the cursor. |
| `editor.scroll_beyond_last_line` | boolean | `true` | Allow scrolling past the last line. |
| `editor.ensure_final_newline` | boolean | `false` | Append a newline on save when the buffer has none. |
| `editor.word_wrap` | boolean | `true` | Wrap long lines instead of scrolling horizontally. |
| `editor.invisible_characters` | boolean | `false` | Draw spaces, tabs and line ends. |
| `editor.line_numbers` | boolean | `true` | Show the line number gutter. |
| `editor.indent_guides` | boolean | `true` | Draw vertical guides at indent levels. |
| `editor.code_folding` | boolean | `true` | Show folding controls in the gutter. |
| `editor.auto_close_brackets` | boolean | `true` | Close brackets and quotes as you type. |
| `editor.smart_indent` | boolean | `true` | Keep the previous line's indentation and follow brackets. |

## Sidebar settings

| Key | Type | Default | Description |
| --- | --- | --- | --- |
| `sidebar.folders` | string[] | `[]` | Absolute paths of the folders shown in the sidebar, in display order. |
| `sidebar.visible` | boolean | `true` | Show the folder tree. **View → Toggle Sidebar** (<kbd>Cmd</kbd>+<kbd>B</kbd>) writes it. |
| `sidebar.auto_reveal` | boolean | `true` | Expand the tree to the active file whenever the active tab changes. |

Kaze writes `sidebar.folders` itself when you add a folder from the sidebar's
`+` button, the **File → Add Folder to Sidebar…** menu item
(<kbd>Cmd</kbd>+<kbd>Shift</kbd>+<kbd>O</kbd>), or `kaze .`. Editing the list by
hand is supported: entries are trimmed, blank entries are dropped, and
repeated paths are collapsed to the first occurrence. A folder that has since
been moved or deleted still loads, and its row shows an error when expanded.

## File and search excludes

| Key | Type | Default | Description |
| --- | --- | --- | --- |
| `files.exclude` | string[] | `[".git", ".hg", ".svn", "node_modules", ".DS_Store"]` | Paths hidden from the folder tree and Quick Open. |
| `search.exclude` | string[] | same as `files.exclude` | Extra paths skipped by Find in Files. `files.exclude` applies there too. |

Patterns are globs matched against the path relative to a sidebar root:

- a bare name (`node_modules`, `*.log`) matches at any depth;
- a leading slash (`/dist`) anchors the pattern to the root;
- a pattern containing a slash (`build/output`, `src/**`) is root-relative, and
  `**` matches any number of directories;
- a trailing slash (`dist/`) applies to folders only.

Find in Files also honours every `.gitignore` file it walks past, and never
descends into `.git`, `.hg` or `.svn` even when those are removed from the
list. The panel's include and exclude fields narrow one search: exclude
patterns there are merged with both lists. Entries are trimmed, blanks are
dropped and duplicates are removed.

## Recent files and session

`recent.files` is the only list left in `settings.json`:

| Key | Type | Description |
| --- | --- | --- |
| `recent.files` | string[] | Files shown by **File → Open Recent…** (<kbd>Ctrl</kbd>+<kbd>R</kbd>), newest first, capped at 20. |

Tabs and workspace sessions are app-written state in `session.json`, beside
`settings.json`, not editable settings:

- `window` holds the launch window's file-backed tabs, active tab, expanded
  folders and workspace path.
- `workspaces` parks one tab set per workspace file, keyed by canonical path.
  Switching to a workspace saves the current set, closes the open tabs after
  the save prompt, and restores the parked set. Closing a workspace keeps its
  parked set.
- Only tabs backed by a file are recorded: an untitled buffer exists only in
  memory and is not restored empty. Missing files are skipped on restore.
- A workspace file that moved or was renamed is matched by a unique set of
  canonical folders when the old file no longer exists; ambiguous matches and
  existing copies are ignored. The map is capped at 50 workspaces.
- A corrupt `session.json` is moved to `session.json.bak` before the next write.

The **View** menu's checkable toggles (`word_wrap` through `smart_indent`) are
written back here whenever you flip them, from the menu or the command palette,
so the editor comes back the way you left it. The values are stored as the
booleans above, not as a separate UI state.

## Behavior

- Missing keys keep their defaults, so a partial file is valid.
- `font_family` is trimmed; an empty value falls back to `Menlo`.
- Non-positive `font_size` or `font_weight` values fall back to their
  defaults; the same goes for `tab_size` and `cursor_width`, and a negative
  `cursor_surrounding_lines` becomes `0`.
- Unknown keys are ignored.
- Malformed JSON is reported in the developer console; Kaze runs with defaults
  and leaves the file untouched so it can be fixed.
- Editor values, `sidebar.visible`, `sidebar.auto_reveal` and the recent list
  are read at startup; excludes are read when a folder level is opened and when
  a search runs. The settings panel applies changes immediately; after editing
  the file by hand, restart Kaze to reread the editor values or the sidebar.

The settings panel (**Settings…**, <kbd>Cmd</kbd>+<kbd>,</kbd>) edits every
configurable value in the file: editor typography, indentation, cursor, view
toggles and final-newline handling; appearance; interface; explorer and search
excludes; and the sidebar's visibility, reveal behavior and folders. Values are
saved to `settings.json` as they change, and the View menu checkmarks follow the
panel's toggles. The panel's **Advanced** section links to the JSON file, the default
settings, the themes folder and the keybindings.

`recent.files` is managed by the app: the Open Recent menu is its UI, so the
panel does not list it as an editable setting. Tabs and workspace sessions live
in `session.json` instead.

The shortcut launches `$VISUAL`, falling back to `$EDITOR`, and then to the
platform's default file handler. The value may include arguments, e.g.
`EDITOR="code -w"`, and paths containing spaces must be quoted. The editor is
started detached, so use a graphical editor; terminal editors such as `vim`
have no terminal to attach to when launched from Kaze.
