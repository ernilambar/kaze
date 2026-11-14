# Settings

Kova reads user settings from `settings.json` in the platform config
directory. The file is created with defaults the first time settings are
loaded (at app startup), so no manual setup is required.

| Platform | Path |
| --- | --- |
| macOS | `~/Library/Application Support/Kova/settings.json` |
| Windows | `%AppData%\Kova\settings.json` |
| Linux | `~/.config/Kova/settings.json` (`$XDG_CONFIG_HOME` is honored) |

## Example

```json
{
  "editor": {
    "font_family": "Menlo",
    "font_size": 13,
    "font_weight": 400
  }
}
```

## Editor settings

| Key | Type | Default | Description |
| --- | --- | --- | --- |
| `editor.font_family` | string | `Menlo` | CSS `font-family` for the editor. Use a monospace font. |
| `editor.font_size` | number | `13` | Font size in pixels. Fractional values are allowed. |
| `editor.font_weight` | number | `400` | CSS font weight; typical values are `100`–`900` (`400` normal, `700` bold). |

## Behavior

- Missing keys keep their defaults, so a partial file is valid.
- `font_family` is trimmed; an empty value falls back to `Menlo`.
- Non-positive `font_size` or `font_weight` values fall back to their
  defaults.
- Unknown keys are ignored.
- Malformed JSON is reported in the developer console; Kova runs with defaults
  and leaves the file untouched so it can be fixed.
- Values are read once at startup. Restart Kova after editing the file.

There is no settings UI yet. Press <kbd>Cmd</kbd>+<kbd>,</kbd>
(<kbd>Ctrl</kbd>+<kbd>,</kbd> on Windows and Linux) to open the file, or edit
it directly.

The shortcut launches `$VISUAL`, falling back to `$EDITOR`, and then to the
platform's default file handler. The value may include arguments, e.g.
`EDITOR="code -w"`, and paths containing spaces must be quoted. The editor is
started detached, so use a graphical editor; terminal editors such as `vim`
have no terminal to attach to when launched from Kova.
