# Changelog

## 0.1.6 - 2026-11-25
- Added: expanded folders are restored with the tree

## 0.1.5 - 2026-11-25
- Added: Homebrew tap installation instructions
- Fixed: gutter baselines refresh after commits made outside Kaze

## 0.1.4 - 2026-11-23
- Added: syntax highlighting for Go module files
- Changed: Kaze now requires macOS 15 or later

## 0.1.3 - 2026-11-22
- Added: macOS document types so Finder can open files with Kaze
- Added: Switch Workspace command in the command palette
- Added: resizable sidebar
- Added: syntax highlighting for Svelte, TOML, dotenv, ignore, log, plist, patch, and diff files
- Added: git change marks on folders, dimmed ignored entries, and nested repository discovery
- Changed: diff tabs color additions and deletions
- Fixed: quoted `$(...)` broke shell highlighting for the rest of the file
- Fixed: spurious TypeScript diagnostics for files outside a project
- Fixed: git gutter marks for ignored files
- Fixed: deleted files left out of tree decorations

## 0.1.2 - 2026-11-18
- Added: settings files open in Kaze tabs and reload when changed on disk
- Fixed: sidebar roots can no longer be moved or trashed from the tree

## 0.1.1 - 2026-11-17
- Added: per-window sidebar roots and `.gitignore`-aware Quick Open
- Added: per-file indentation detection with a status-bar picker
- Added: Monaco view options and line commands

## 0.1.0 - 2026-11-17
- Initial release
