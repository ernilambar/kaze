# Settings

Kaze reads `~/Library/Application Support/Kaze/settings.json`, creating it with
defaults on first run. Missing keys keep their defaults; unknown keys are
ignored. Edit it from **Settings…** (`⌘,`); hand edits need a restart.

## Editor

| Key | Default | Description |
| --- | --- | --- |
| `editor.font_family` | `Menlo` | Editor font. |
| `editor.font_size` | `13` | Font size in pixels. |
| `editor.font_weight` | `400` | CSS font weight. |
| `editor.tab_size` | `4` | Fallback indent width; each file's own indent is detected. |
| `editor.hard_tabs` | `false` | Fallback indent style; each file's own indent is detected. |
| `editor.ligatures` | `false` | Font ligatures. |
| `editor.cursor_width` | `1` | Cursor width in pixels. |
| `editor.cursor_blink` | `true` | Blink the cursor. |
| `editor.cursor_surrounding_lines` | `0` | Lines kept visible around the cursor. |
| `editor.scroll_beyond_last_line` | `true` | Scroll past the last line. |
| `editor.ensure_final_newline` | `false` | Add a trailing newline on save. |
| `editor.minimap` | `false` | Show the minimap beside the scrollbar. |
| `editor.bracket_pair_colors` | `false` | Color and highlight matching bracket pairs. |
| `editor.rulers` | `[]` | Columns for vertical guide lines. |
| `editor.word_wrap` | `true` | Wrap long lines. |
| `editor.invisible_characters` | `false` | Draw whitespace. |
| `editor.line_numbers` | `true` | Line-number gutter. |
| `editor.indent_guides` | `true` | Indent guides. |
| `editor.code_folding` | `true` | Folding controls. |
| `editor.auto_close_brackets` | `true` | Close brackets and quotes. |
| `editor.smart_indent` | `true` | Indent from context. |

## Sidebar

| Key | Default | Description |
| --- | --- | --- |
| `sidebar.visible` | `true` | Show the sidebar (`⌘B`). |
| `sidebar.auto_reveal` | `true` | Reveal the active file in the tree. |

## Excludes

| Key | Default | Description |
| --- | --- | --- |
| `files.exclude` | `[".git", ".hg", ".svn", "node_modules", ".DS_Store"]` | Hidden from the tree and Quick Open. |
| `files.trim_trailing_whitespace` | `false` | Remove spaces and tabs at line ends on save. |
| `search.exclude` | same list | Also skipped by Find in Files. |

Globs, relative to a sidebar root:

- `node_modules`, `*.log` — any depth.
- `/dist` — anchored to the root; `**` spans directories.
- `dist/` — folders only.

Find in Files also honours `.gitignore`.

## Appearance

| Key | Default | Description |
| --- | --- | --- |
| `theme.mode` | `dark` | `system`, `light` or `dark`. |
| `theme.light` | `Kaze Light` | Light theme variant. |
| `theme.dark` | `Kaze Dark` | Dark theme variant. |

Themes are Zed theme files in `themes/` beside `settings.json`; names are their
`name` fields.

## Interface

| Key | Default | Description |
| --- | --- | --- |
| `ui.font_family` | `""` | Chrome font; empty = system font. |
| `ui.font_size` | `13` | Chrome text size (9–48). |
| `ui.tab_icons` | `true` | File icons in tabs. |
| `ui.icon_style` | `colorful` | `colorful` or `monochrome`. |
| `ui.sidebar_position` | `left` | `left` or `right`. |

## Session

Open folders, tabs, window frame and Open Recent are app state, kept in
`session.json` beside `settings.json` — not settings. Untitled buffers are not
restored. A corrupt file is moved to `session.json.bak`.
