# Settings

Kaze reads `~/Library/Application Support/Kaze/settings.json`, creating it with
defaults on first run. A partial file is valid: missing keys keep their
defaults, and unknown keys are ignored.

## Editor

| Key | Type | Default | Description |
| --- | --- | --- | --- |
| `editor.font_family` | string | `Menlo` | Editor font; empty falls back to `Menlo`. |
| `editor.font_size` | number | `13` | Font size in pixels. |
| `editor.font_weight` | number | `400` | CSS font weight. |
| `editor.tab_size` | number | `4` | Columns per tab and indent level. |
| `editor.hard_tabs` | boolean | `false` | Insert tab characters instead of spaces. |
| `editor.ligatures` | boolean | `false` | Render font ligatures. |
| `editor.cursor_width` | number | `1` | Line cursor width in pixels. |
| `editor.cursor_blink` | boolean | `true` | Blink the cursor. |
| `editor.cursor_surrounding_lines` | number | `0` | Lines kept visible above and below the cursor. |
| `editor.scroll_beyond_last_line` | boolean | `true` | Allow scrolling past the last line. |
| `editor.ensure_final_newline` | boolean | `false` | Add a trailing newline on save. |
| `editor.word_wrap` | boolean | `true` | Wrap long lines. |
| `editor.invisible_characters` | boolean | `false` | Draw spaces, tabs, and line ends. |
| `editor.line_numbers` | boolean | `true` | Show the line-number gutter. |
| `editor.indent_guides` | boolean | `true` | Draw indent guides. |
| `editor.code_folding` | boolean | `true` | Show folding controls. |
| `editor.auto_close_brackets` | boolean | `true` | Close brackets and quotes while typing. |
| `editor.smart_indent` | boolean | `true` | Follow the previous line's indentation and brackets. |

The View menu's checkboxes write these last seven keys.

## Sidebar

| Key | Type | Default | Description |
| --- | --- | --- | --- |
| `sidebar.folders` | string[] | `[]` | Absolute root folders, in display order. |
| `sidebar.visible` | boolean | `true` | Show the sidebar; **Toggle Sidebar** (`⌘B`) writes it. |
| `sidebar.auto_reveal` | boolean | `true` | Expand the tree to the active file on tab change. |

Adding a folder with the sidebar's `+`, **File → Add Folder to Sidebar…**
(`⌃⌘O`), or `kaze .` writes this list. Hand edits are normalized: entries are
trimmed, blanks dropped, and duplicates removed.

## Excludes

| Key | Type | Default | Description |
| --- | --- | --- | --- |
| `files.exclude` | string[] | `[".git", ".hg", ".svn", "node_modules", ".DS_Store"]` | Hidden from the tree and Quick Open. |
| `search.exclude` | string[] | same list | Extra paths skipped by Find in Files; `files.exclude` applies too. |

Patterns are globs matched against a path relative to a sidebar root:

- A bare name (`node_modules`, `*.log`) matches at any depth.
- A leading slash (`/dist`) anchors it to the root.
- A slash elsewhere (`build/output`, `src/**`) makes it root-relative; `**`
  spans directories.
- A trailing slash (`dist/`) matches folders only.

Find in Files also honours every `.gitignore` it walks past and never descends
into `.git`, `.hg`, or `.svn`. The panel's include/exclude fields narrow one
search. Entries are trimmed, blanks dropped, and duplicates removed.

## Appearance

| Key | Type | Default | Description |
| --- | --- | --- | --- |
| `theme.mode` | string | `dark` | `system` follows macOS; otherwise `light` or `dark`. |
| `theme.light` | string | `Kaze Light` | Theme variant used for light appearance. |
| `theme.dark` | string | `Kaze Dark` | Theme variant used for dark appearance. |

Theme names are the `name` fields inside the Zed theme files in the `themes/`
folder beside `settings.json`. Kaze writes the built-in `kaze.json` there on
first use. A missing or unknown name falls back to the first theme for that
appearance.

## Interface

| Key | Type | Default | Description |
| --- | --- | --- | --- |
| `ui.font_family` | string | `""` | Chrome font; empty uses the system UI font. |
| `ui.font_size` | number | `13` | Chrome text size in pixels, clamped to 9–48. |
| `ui.tab_icons` | boolean | `true` | Show file icons in the tab bar. |
| `ui.icon_style` | string | `colorful` | `colorful` or `monochrome`. |
| `ui.sidebar_position` | string | `left` | `left` or `right`. |

## App-written state

`settings.json` also holds `window`, the last window frame. It is written when
the first window closes or the app quits, not by the settings panel; a frame
that is too small or would put the title bar off-screen is ignored.

Tabs, workspaces, and the Open Recent list live in `session.json` beside the
settings file:

| Key | Description |
| --- | --- |
| `recent.files` | Open Recent history, newest first, capped at 20. |
| `window` | Launch window's workspace, folders, open files, active tab, expanded folders. |
| `workspaces` | One parked tab set per workspace file, keyed by canonical path, capped at 50. |

Only file-backed tabs are recorded; untitled buffers are not restored. Missing
files are skipped. A corrupt `session.json` is moved to `session.json.bak` when
it is read. On first run after upgrading, a `recent` section left in
`settings.json` is moved here and removed from the settings file.

## Behaviour

- `font_family` is trimmed. Non-positive editor sizes fall back to their
  defaults, as do invalid theme/interface values.
- Malformed JSON is reported in the developer console; Kaze runs with defaults
  and leaves the file untouched so it can be repaired.
- Settings are read at startup. Excludes are read again for each folder level
  and each search. Changes made in the settings panel apply immediately; after
  editing the file by hand, restart Kaze.

The settings panel (**Settings…**, `⌘,`) covers Editor, Appearance, Interface,
Files, Sidebar, and Advanced. Advanced opens the settings file, the generated
`default-settings.json`, the themes folder, and `keybindings.json`. The
settings file opens with `$VISUAL`, then `$EDITOR`, then the default file
handler; the editor is launched detached, so use a graphical one.
