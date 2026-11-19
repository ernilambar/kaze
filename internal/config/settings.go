package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"kaze/internal/pathutil"
)

const (
	DefaultFontFamily  = "Menlo"
	DefaultFontSize    = 13.0
	defaultFontWeight  = 400.0
	DefaultTabSize     = 4
	DefaultCursorWidth = 1

	defaultThemeMode    = "dark"
	DefaultThemeLight   = "Kaze Light"
	DefaultThemeDark    = "Kaze Dark"
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
	Theme   ThemeSettings   `json:"theme"`
	UI      UISettings      `json:"ui"`
}

// EditorSettings tunes the Monaco editor instance. Booleans that default to
// true rely on Default prepopulating the struct before the file is
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

	// These view options are edited in the settings overlay and applied to a
	// live editor. Rulers is the list of vertical guide columns; an empty list
	// draws none.
	Minimap           bool  `json:"minimap"`
	BracketPairColors bool  `json:"bracket_pair_colors"`
	Rulers            []int `json:"rulers"`

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

// SidebarSettings is the sidebar's saved appearance. The roots themselves are
// session state, because a folder list is what the user last had open, not a
// preference to edit by hand.
type SidebarSettings struct {
	// Visible is whether the sidebar is shown. It survives a relaunch, so
	// Cmd+B keeps the layout the user left behind.
	Visible bool `json:"visible"`
	// AutoReveal makes a tab change expand the tree to the active file.
	AutoReveal bool `json:"auto_reveal"`
}

// FilesSettings configures the explorer's hidden entries and what a save does
// to the text it writes.
type FilesSettings struct {
	// Exclude holds glob patterns matched against the path relative to a
	// sidebar root. A pattern with no slash matches a file or folder name at
	// any depth, so the default list hides every node_modules folder.
	Exclude []string `json:"exclude"`
	// TrimTrailingWhitespace removes spaces and tabs at line ends on save. It
	// edits the buffer too, so the saved text and the tab still agree.
	TrimTrailingWhitespace bool `json:"trim_trailing_whitespace"`
	// ConfirmDelete asks before moving a tree entry to the Trash. It defaults
	// to true because the action discards the file's location; a buffer with
	// unsaved changes always asks regardless.
	ConfirmDelete bool `json:"confirm_delete"`
}

// SearchSettings configures what Find in Files skips. The files list applies
// too, so a folder hidden from the explorer is never searched.
type SearchSettings struct {
	Exclude []string `json:"exclude"`
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

// DefaultExcludePatterns is what a first run hides. Version control metadata
// and installed packages hold thousands of files nobody opens by hand.
func DefaultExcludePatterns() []string {
	return []string{".git", ".hg", ".svn", "node_modules", ".DS_Store"}
}

func Default() Settings {
	return Settings{
		Editor: EditorSettings{
			FontFamily: DefaultFontFamily,
			FontSize:   DefaultFontSize,
			FontWeight: defaultFontWeight,

			TabSize:                DefaultTabSize,
			HardTabs:               false,
			Ligatures:              false,
			CursorWidth:            DefaultCursorWidth,
			CursorBlink:            true,
			CursorSurroundingLines: 0,
			ScrollBeyondLastLine:   true,
			EnsureFinalNewline:     false,

			Minimap:           false,
			BracketPairColors: false,
			Rulers:            []int{},

			WordWrap:            true,
			InvisibleCharacters: false,
			LineNumbers:         true,
			IndentGuides:        true,
			CodeFolding:         true,
			AutoCloseBrackets:   true,
			SmartIndent:         true,
		},
		Sidebar: SidebarSettings{Visible: true, AutoReveal: true},
		Files: FilesSettings{
			Exclude:                DefaultExcludePatterns(),
			TrimTrailingWhitespace: false,
			ConfirmDelete:          true,
		},
		Search: SearchSettings{Exclude: DefaultExcludePatterns()},
		Theme: ThemeSettings{
			Mode:  defaultThemeMode,
			Light: DefaultThemeLight,
			Dark:  DefaultThemeDark,
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

// Normalize replaces absent or unusable values with their defaults. The
// settings file is user-editable, so a missing key must not produce a broken
// editor.
func (s *Settings) Normalize() {
	s.Editor.FontFamily = strings.TrimSpace(s.Editor.FontFamily)
	if s.Editor.FontFamily == "" {
		s.Editor.FontFamily = DefaultFontFamily
	}
	if s.Editor.FontSize <= 0 {
		s.Editor.FontSize = DefaultFontSize
	}
	if s.Editor.FontWeight <= 0 {
		s.Editor.FontWeight = defaultFontWeight
	}
	if s.Editor.TabSize <= 0 {
		s.Editor.TabSize = DefaultTabSize
	}
	if s.Editor.CursorWidth <= 0 {
		s.Editor.CursorWidth = DefaultCursorWidth
	}
	if s.Editor.CursorSurroundingLines < 0 {
		s.Editor.CursorSurroundingLines = 0
	}
	s.Editor.Rulers = normalizeRulers(s.Editor.Rulers)

	s.Files.Exclude = normalizePatternList(s.Files.Exclude)
	s.Search.Exclude = normalizePatternList(s.Search.Exclude)

	s.Theme.Mode = NormalizeThemeMode(s.Theme.Mode)
	s.Theme.Light = strings.TrimSpace(s.Theme.Light)
	if s.Theme.Light == "" {
		s.Theme.Light = DefaultThemeLight
	}
	s.Theme.Dark = strings.TrimSpace(s.Theme.Dark)
	if s.Theme.Dark == "" {
		s.Theme.Dark = DefaultThemeDark
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

// NormalizeThemeMode keeps the three supported modes and treats a typo as the
// default instead of leaving an unusable value in the file.
func NormalizeThemeMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "light", "dark", "system":
		return strings.ToLower(strings.TrimSpace(mode))
	default:
		return defaultThemeMode
	}
}

// normalizeRulers keeps the positive columns, drops repeats and sorts them, so
// the list the settings overlay shows is the one Monaco draws.
func normalizeRulers(rulers []int) []int {
	normalized := make([]int, 0, len(rulers))
	seen := make(map[int]bool, len(rulers))
	for _, column := range rulers {
		if column <= 0 || seen[column] {
			continue
		}
		seen[column] = true
		normalized = append(normalized, column)
	}
	sort.Ints(normalized)
	return normalized
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

// NormalizeFileList keeps the first occurrence of each path and drops blanks,
// preserving order. Recent files and restored tabs are ordered, so unlike the
// sidebar roots they cannot be sorted.
func NormalizeFileList(files []string) []string {
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

// NormalizeFolderList trims paths, drops blanks and removes repeats while
// keeping the user's order. The settings file may be edited by hand, so the
// list is repaired rather than rejected.
func NormalizeFolderList(folders []string) []string {
	normalized := make([]string, 0, len(folders))
	for _, folder := range folders {
		folder = strings.TrimSpace(folder)
		if folder == "" {
			continue
		}
		normalized = append(normalized, folder)
	}
	return pathutil.UniquePaths(normalized)
}

// FilePath returns the per-user settings location:
// ~/Library/Application Support/<product name>/settings.json.
func FilePath(productName string) (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config directory: %w", err)
	}
	return filepath.Join(dir, productName, "settings.json"), nil
}

// Load reads path and normalizes the result. When the file does not
// exist the defaults are written there so the user has something to edit. An
// empty file reads as the defaults without a write: an editor saving in place
// truncates first, and rewriting it then would race that save.
func Load(path string) (Settings, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		settings := Default()
		if err := Write(path, settings); err != nil {
			return settings, err
		}
		return settings, nil
	}
	if err == nil && len(bytes.TrimSpace(data)) == 0 {
		return Default(), nil
	}
	if err != nil {
		return Default(), fmt.Errorf("read settings file: %w", err)
	}

	// Decode over the defaults so a key the user removed keeps its default
	// instead of decoding as the zero value; an explicit false still applies.
	settings := Default()
	if err := json.Unmarshal(data, &settings); err != nil {
		return Default(), fmt.Errorf("parse %s: %w", path, err)
	}
	settings.Normalize()
	return settings, nil
}

// EnsureFile creates path with defaults when it is missing. Unlike
// Load it does not fail on malformed content, so the user can still
// open and repair a broken file.
func EnsureFile(path string) error {
	_, err := os.Stat(path)
	if err == nil {
		return nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat settings file: %w", err)
	}
	return Write(path, Default())
}

func Write(path string, settings Settings) error {
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("encode settings: %w", err)
	}
	return writeSettingsFile(path, append(data, '\n'))
}

// writeSettingsFile replaces path with data through a temporary file in the
// same directory, so a concurrent read never sees a half-written settings
// file and a failed write leaves the old one intact.
func writeSettingsFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create settings directory: %w", err)
	}

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
