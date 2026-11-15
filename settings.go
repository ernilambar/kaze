package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	defaultFontFamily  = "Menlo"
	defaultFontSize    = 13.0
	defaultFontWeight  = 400.0
	defaultTabSize     = 4
	defaultCursorWidth = 1

	defaultThemeMode    = "system"
	defaultThemeLight   = "Kova Light"
	defaultThemeDark    = "Kova Dark"
	defaultUIFontSize   = 13.0
	defaultUIIconStyle  = "colorful"
	defaultSidebarPlace = "left"
)

// Settings is the contents of the user's settings.json file.
type Settings struct {
	Editor  EditorSettings  `json:"editor"`
	Sidebar SidebarSettings `json:"sidebar"`
	Files   FilesSettings   `json:"files"`
	Search  SearchSettings  `json:"search"`
	Recent  RecentSettings  `json:"recent"`
	Session SessionSettings `json:"session"`
	Theme   ThemeSettings   `json:"theme"`
	UI      UISettings      `json:"ui"`
}

// EditorSettings tunes the Monaco editor instance. Booleans that default to
// true rely on defaultSettings prepopulating the struct before the file is
// decoded; a missing key keeps its default while an explicit false stays false.
type EditorSettings struct {
	FontFamily string  `json:"font_family"`
	FontSize   float64 `json:"font_size"`
	FontWeight float64 `json:"font_weight"`

	TabSize     int  `json:"tab_size"`
	HardTabs    bool `json:"hard_tabs"`
	Ligatures   bool `json:"ligatures"`
	CursorWidth int  `json:"cursor_width"`
	CursorBlink bool `json:"cursor_blink"`
	// CursorSurroundingLines keeps this many lines visible above and below the
	// cursor while scrolling.
	CursorSurroundingLines int  `json:"cursor_surrounding_lines"`
	ScrollBeyondLastLine   bool `json:"scroll_beyond_last_line"`
	EnsureFinalNewline     bool `json:"ensure_final_newline"`

	// The view toggles are editor options too. Keeping them here means a
	// View menu choice survives a relaunch, and the menu starts checked.
	WordWrap            bool `json:"word_wrap"`
	InvisibleCharacters bool `json:"invisible_characters"`
	LineNumbers         bool `json:"line_numbers"`
	IndentGuides        bool `json:"indent_guides"`
	CodeFolding         bool `json:"code_folding"`
	AutoCloseBrackets   bool `json:"auto_close_brackets"`
	SmartIndent         bool `json:"smart_indent"`
}

// SidebarSettings is the folder tree's saved state. Opening a workspace
// replaces the roots with the file's folders; the list is still persisted so a
// relaunch without the workspace file keeps its tree.
type SidebarSettings struct {
	Folders []string `json:"folders"`
	// Visible is whether the sidebar is shown. It survives a relaunch, so
	// Cmd+B keeps the layout the user left behind.
	Visible bool `json:"visible"`
}

// FilesSettings configures which entries the explorer and Quick Open hide.
type FilesSettings struct {
	// Exclude holds glob patterns matched against the path relative to a
	// sidebar root. A pattern with no slash matches a file or folder name at
	// any depth, so the default list hides every node_modules folder.
	Exclude []string `json:"exclude"`
}

// SearchSettings configures what Find in Files skips. The files list applies
// too, so a folder hidden from the explorer is never searched.
type SearchSettings struct {
	Exclude []string `json:"exclude"`
}

// RecentSettings is the Open Recent list, newest first.
type RecentSettings struct {
	Files []string `json:"files"`
}

// SessionSettings is the set of tabs restored on the next launch. Only tabs
// backed by a file are recorded: an untitled buffer has no text on disk, and
// restoring it empty would look like the work was lost.
type SessionSettings struct {
	OpenFiles []string `json:"open_files"`
	Active    string   `json:"active"`
}

// ThemeSettings selects the palette. Mode is system, light or dark; the other
// two name the variant to use for each appearance. Names are variant names,
// matching the `name` field inside a Zed theme file.
type ThemeSettings struct {
	Mode  string `json:"mode"`
	Light string `json:"light"`
	Dark  string `json:"dark"`
}

// UISettings controls the chrome around the editor: typography, the explorer's
// file icons and which side the sidebar opens on.
type UISettings struct {
	FontFamily      string  `json:"font_family"`
	FontSize        float64 `json:"font_size"`
	TabIcons        bool    `json:"tab_icons"`
	IconStyle       string  `json:"icon_style"`
	SidebarPosition string  `json:"sidebar_position"`
}

// maxRecentFiles caps the Open Recent list. More than a screenful is never
// read, and the list is stored in settings.json.
const maxRecentFiles = 20

// maxSessionFiles caps restored tabs, so a settings file edited by hand cannot
// make startup open an unbounded number of files.
const maxSessionFiles = 40

// defaultExcludePatterns is what a first run hides. Version control metadata
// and installed packages hold thousands of files nobody opens by hand.
func defaultExcludePatterns() []string {
	return []string{".git", ".hg", ".svn", "node_modules", ".DS_Store"}
}

func defaultSettings() Settings {
	return Settings{
		Editor: EditorSettings{
			FontFamily: defaultFontFamily,
			FontSize:   defaultFontSize,
			FontWeight: defaultFontWeight,

			TabSize:                defaultTabSize,
			HardTabs:               false,
			Ligatures:              false,
			CursorWidth:            defaultCursorWidth,
			CursorBlink:            true,
			CursorSurroundingLines: 0,
			ScrollBeyondLastLine:   false,
			EnsureFinalNewline:     false,

			WordWrap:            true,
			InvisibleCharacters: false,
			LineNumbers:         true,
			IndentGuides:        true,
			CodeFolding:         true,
			AutoCloseBrackets:   true,
			SmartIndent:         true,
		},
		Sidebar: SidebarSettings{Folders: []string{}, Visible: true},
		Files:   FilesSettings{Exclude: defaultExcludePatterns()},
		Search:  SearchSettings{Exclude: defaultExcludePatterns()},
		Recent:  RecentSettings{Files: []string{}},
		Session: SessionSettings{OpenFiles: []string{}},
		Theme: ThemeSettings{
			Mode:  defaultThemeMode,
			Light: defaultThemeLight,
			Dark:  defaultThemeDark,
		},
		UI: UISettings{
			FontFamily:      "",
			FontSize:        defaultUIFontSize,
			TabIcons:        true,
			IconStyle:       defaultUIIconStyle,
			SidebarPosition: defaultSidebarPlace,
		},
	}
}

// normalize replaces absent or unusable values with their defaults. The
// settings file is user-editable, so a missing key must not produce a broken
// editor.
func (s *Settings) normalize() {
	s.Editor.FontFamily = strings.TrimSpace(s.Editor.FontFamily)
	if s.Editor.FontFamily == "" {
		s.Editor.FontFamily = defaultFontFamily
	}
	if s.Editor.FontSize <= 0 {
		s.Editor.FontSize = defaultFontSize
	}
	if s.Editor.FontWeight <= 0 {
		s.Editor.FontWeight = defaultFontWeight
	}
	if s.Editor.TabSize <= 0 {
		s.Editor.TabSize = defaultTabSize
	}
	if s.Editor.CursorWidth <= 0 {
		s.Editor.CursorWidth = defaultCursorWidth
	}
	if s.Editor.CursorSurroundingLines < 0 {
		s.Editor.CursorSurroundingLines = 0
	}

	s.Sidebar.Folders = normalizeFolderList(s.Sidebar.Folders)
	s.Files.Exclude = normalizePatternList(s.Files.Exclude)
	s.Search.Exclude = normalizePatternList(s.Search.Exclude)

	s.Recent.Files = normalizeFileList(s.Recent.Files)
	if len(s.Recent.Files) > maxRecentFiles {
		s.Recent.Files = s.Recent.Files[:maxRecentFiles]
	}

	s.Session.OpenFiles = normalizeFileList(s.Session.OpenFiles)
	if len(s.Session.OpenFiles) > maxSessionFiles {
		s.Session.OpenFiles = s.Session.OpenFiles[:maxSessionFiles]
	}
	s.Session.Active = strings.TrimSpace(s.Session.Active)

	s.Theme.Mode = normalizeThemeMode(s.Theme.Mode)
	s.Theme.Light = strings.TrimSpace(s.Theme.Light)
	if s.Theme.Light == "" {
		s.Theme.Light = defaultThemeLight
	}
	s.Theme.Dark = strings.TrimSpace(s.Theme.Dark)
	if s.Theme.Dark == "" {
		s.Theme.Dark = defaultThemeDark
	}

	s.UI.FontFamily = strings.TrimSpace(s.UI.FontFamily)
	if s.UI.FontSize <= 0 {
		s.UI.FontSize = defaultUIFontSize
	}
	if s.UI.FontSize < 9 {
		s.UI.FontSize = 9
	}
	if s.UI.FontSize > 48 {
		s.UI.FontSize = 48
	}
	if !strings.EqualFold(s.UI.IconStyle, "monochrome") {
		s.UI.IconStyle = defaultUIIconStyle
	} else {
		s.UI.IconStyle = "monochrome"
	}
	if !strings.EqualFold(s.UI.SidebarPosition, "right") {
		s.UI.SidebarPosition = defaultSidebarPlace
	} else {
		s.UI.SidebarPosition = "right"
	}
}

// normalizeThemeMode keeps the three supported modes and treats a typo as the
// system default instead of leaving an unusable value in the file.
func normalizeThemeMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "light", "dark":
		return strings.ToLower(strings.TrimSpace(mode))
	default:
		return defaultThemeMode
	}
}

// normalizePatternList trims the glob patterns and drops blanks. Duplicates are
// removed because the settings file is user-editable and repeating a pattern
// would otherwise make every walk test it twice.
func normalizePatternList(patterns []string) []string {
	normalized := make([]string, 0, len(patterns))
	seen := make(map[string]bool, len(patterns))
	for _, pattern := range patterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" || seen[pattern] {
			continue
		}
		seen[pattern] = true
		normalized = append(normalized, pattern)
	}
	return normalized
}

// normalizeFileList keeps the first occurrence of each path and drops blanks,
// preserving order. Recent files and restored tabs are ordered, so unlike the
// sidebar roots they cannot be sorted.
func normalizeFileList(files []string) []string {
	normalized := make([]string, 0, len(files))
	seen := make(map[string]bool, len(files))
	for _, file := range files {
		file = strings.TrimSpace(file)
		if file == "" || seen[file] {
			continue
		}
		seen[file] = true
		normalized = append(normalized, file)
	}
	return normalized
}

// normalizeFolderList trims paths, drops blanks and removes repeats while
// keeping the user's order. The settings file may be edited by hand, so the
// list is repaired rather than rejected.
func normalizeFolderList(folders []string) []string {
	normalized := make([]string, 0, len(folders))
	for _, folder := range folders {
		folder = strings.TrimSpace(folder)
		if folder == "" {
			continue
		}
		normalized = append(normalized, folder)
	}
	return uniquePaths(normalized)
}

// settingsFilePath returns the per-user settings location, following the
// platform config directory: ~/Library/Application Support/<product
// name>/settings.json on macOS.
func settingsFilePath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config directory: %w", err)
	}
	return filepath.Join(dir, appMeta.ProductName, "settings.json"), nil
}

// loadSettings reads path and normalizes the result. When the file does not
// exist (or is empty) the defaults are written there so the user has something
// to edit.
func loadSettings(path string) (Settings, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) || (err == nil && len(bytes.TrimSpace(data)) == 0) {
		settings := defaultSettings()
		if err := writeSettings(path, settings); err != nil {
			return settings, err
		}
		return settings, nil
	}
	if err != nil {
		return defaultSettings(), fmt.Errorf("read settings file: %w", err)
	}

	// Decode over the defaults so a key the user removed keeps its default
	// instead of decoding as the zero value; an explicit false still applies.
	settings := defaultSettings()
	if err := json.Unmarshal(data, &settings); err != nil {
		return defaultSettings(), fmt.Errorf("parse %s: %w", path, err)
	}
	settings.normalize()
	return settings, nil
}

// ensureSettingsFile creates path with defaults when it is missing. Unlike
// loadSettings it does not fail on malformed content, so the user can still
// open and repair a broken file.
func ensureSettingsFile(path string) error {
	_, err := os.Stat(path)
	if err == nil {
		return nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat settings file: %w", err)
	}
	return writeSettings(path, defaultSettings())
}

func writeSettings(path string, settings Settings) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create settings directory: %w", err)
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("encode settings: %w", err)
	}
	data = append(data, '\n')

	// Write through a temporary file and rename, so a concurrent read never sees
	// a half-written settings file.
	tmp, err := os.CreateTemp(dir, "settings-*.json")
	if err != nil {
		return fmt.Errorf("write settings file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write settings file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("write settings file: %w", err)
	}
	if err := os.Chmod(tmpName, 0o644); err != nil {
		return fmt.Errorf("write settings file: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("write settings file: %w", err)
	}
	return nil
}
