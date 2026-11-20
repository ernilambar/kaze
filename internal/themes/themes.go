package themes

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

const (
	seedFile     = "kaze.json"
	seedHashFile = ".kaze-seed"
	// legacySeedHash is kaze.json as it was written before the hash record
	// existed. An untouched copy of it is replaced by the current palette.
	legacySeedHash = "92ffb8bcfb74bb83894ccbf341d338046ef35d6aca009c188c1a77e81ae37065"
)

// builtinThemeJSON is the pair of palettes the app ships with. It is written to
// the user's themes folder on first use so there is a file to copy and edit.
//
//go:embed kaze.json
var builtinThemeJSON []byte

// themeFamily is the on-disk shape of a Zed theme family: one file carries the
// light and dark variants that belong together.
type themeFamily struct {
	Schema string         `json:"$schema"`
	Name   string         `json:"name"`
	Author string         `json:"author"`
	Themes []themeVariant `json:"themes"`
}

// themeVariant is one light or dark palette.
type themeVariant struct {
	Name       string     `json:"name"`
	Appearance string     `json:"appearance"`
	Style      themeStyle `json:"style"`
}

// themeStyle is the subset of Zed's style keys the app consumes. Unknown keys are
// ignored by encoding/json, so a full Zed theme parses without complaint.
type themeStyle struct {
	Background           string                    `json:"background"`
	SurfaceBackground    string                    `json:"surface.background"`
	PanelBackground      string                    `json:"panel.background"`
	ElementBackground    string                    `json:"element.background"`
	ElementHover         string                    `json:"element.hover"`
	ElementSelected      string                    `json:"element.selected"`
	Text                 string                    `json:"text"`
	TextAccent           string                    `json:"text.accent"`
	TextMuted            string                    `json:"text.muted"`
	Border               string                    `json:"border"`
	Error                string                    `json:"error"`
	Created              string                    `json:"created"`
	Modified             string                    `json:"modified"`
	Deleted              string                    `json:"deleted"`
	Conflict             string                    `json:"conflict"`
	SearchMatch          string                    `json:"search.match_background"`
	SearchActive         string                    `json:"search.active_match_background"`
	EditorBackground     string                    `json:"editor.background"`
	EditorForeground     string                    `json:"editor.foreground"`
	EditorGutter         string                    `json:"editor.gutter.background"`
	EditorLineNumber     string                    `json:"editor.line_number"`
	EditorActiveLineNum  string                    `json:"editor.active_line_number"`
	EditorActiveLine     string                    `json:"editor.active_line.background"`
	EditorIndentGuide    string                    `json:"editor.indent_guide"`
	EditorIndentGuideAct string                    `json:"editor.indent_guide_active"`
	Syntax               map[string]themeHighlight `json:"syntax"`
}

// themeHighlight is one syntax capture's color and font style.
type themeHighlight struct {
	Color      string `json:"color"`
	FontStyle  string `json:"font_style"`
	FontWeight int    `json:"font_weight"`
}

// ThemeSummary is one variant in the theme picker. Name is what settings.json
// stores; Family is what the file calls the group it belongs to.
type ThemeSummary struct {
	Family     string       `json:"family"`
	Name       string       `json:"name"`
	Appearance string       `json:"appearance"`
	Preview    ThemePreview `json:"preview"`
}

// ThemePreview is the handful of colors the picker paints for a variant. It is
// resolved through mergeStyle, so a partial user theme still previews with the
// built-in contrast instead of blank swatches.
type ThemePreview struct {
	Background string `json:"background"`
	Foreground string `json:"foreground"`
	Accent     string `json:"accent"`
	Keyword    string `json:"keyword"`
	String     string `json:"string"`
	Comment    string `json:"comment"`
}

// ThemeCatalog is the set of readable themes plus the files that failed to
// parse. One broken theme file must not hide the working ones.
type ThemeCatalog struct {
	Themes []ThemeSummary `json:"themes"`
	Errors []string       `json:"errors"`
}

// ThemePalette is the frontend-facing form of one theme. UI keys are CSS
// variable names without the leading `--`; Editor keys are Monaco color ids.
type ThemePalette struct {
	Family     string                    `json:"family"`
	Name       string                    `json:"name"`
	Appearance string                    `json:"appearance"`
	UI         map[string]string         `json:"ui"`
	Editor     map[string]string         `json:"editor"`
	Syntax     map[string]themeHighlight `json:"syntax"`
}

var (
	builtinOnce   sync.Once
	builtinFamily themeFamily
	builtinErr    error
)

func Builtin() (themeFamily, error) {
	builtinOnce.Do(func() {
		builtinFamily, builtinErr = parseThemeFamily(builtinThemeJSON)
	})
	return builtinFamily, builtinErr
}

func parseThemeFamily(data []byte) (themeFamily, error) {
	var family themeFamily
	if err := json.Unmarshal(data, &family); err != nil {
		return themeFamily{}, err
	}
	if len(family.Themes) == 0 {
		return themeFamily{}, errors.New("theme family has no themes")
	}
	return family, nil
}

// EnsureDir creates the themes folder and keeps the example copy of the
// built-in theme current. The copy exists so there is a file to edit, and it
// shadows the embedded variant of the same name; refreshing it while the user
// has not changed it lets a built-in palette update reach existing installs.
func EnsureDir(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create themes directory: %w", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read themes directory: %w", err)
	}
	hasTheme := false
	for _, entry := range entries {
		if !entry.IsDir() && strings.EqualFold(filepath.Ext(entry.Name()), ".json") {
			hasTheme = true
			break
		}
	}
	if !hasTheme {
		return writeSeed(dir)
	}
	return refreshSeed(dir)
}

// writeSeed writes the built-in theme and records its hash. A copy whose hash
// still matches is the one the app wrote, so a later launch may replace it.
func writeSeed(dir string) error {
	if err := os.WriteFile(filepath.Join(dir, seedFile), builtinThemeJSON, 0o644); err != nil {
		return fmt.Errorf("write built-in theme: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, seedHashFile), []byte(seedHash(builtinThemeJSON)), 0o644); err != nil {
		return fmt.Errorf("write built-in theme hash: %w", err)
	}
	return nil
}

// refreshSeed replaces kaze.json only when it is still the copy the app wrote.
// An edited file belongs to the user and is left alone.
func refreshSeed(dir string) error {
	data, err := os.ReadFile(filepath.Join(dir, seedFile))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read theme seed: %w", err)
	}
	recorded, err := os.ReadFile(filepath.Join(dir, seedHashFile))
	if err == nil {
		if strings.TrimSpace(string(recorded)) == seedHash(data) {
			return writeSeed(dir)
		}
		return nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("read theme seed hash: %w", err)
	}
	// Installs seeded before the hash file existed have no record. Recognise
	// the one palette that was shipped that way.
	if seedHash(data) == legacySeedHash {
		return writeSeed(dir)
	}
	return nil
}

func seedHash(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func themeFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".json") {
			continue
		}
		files = append(files, filepath.Join(dir, entry.Name()))
	}
	sort.Strings(files)
	return files, nil
}

// Entry is one variant plus the family name it came from.
type Entry struct {
	Family  string
	Variant themeVariant
}

// LoadEntries reads the built-in family first, then every user file in
// filename order. A later file overrides an earlier variant with the same name,
// so a user file with a built-in variant's name shadows the embedded one.
func LoadEntries(dir string) ([]Entry, []string) {
	entries := make([]Entry, 0)
	problems := make([]string, 0)

	family, err := Builtin()
	if err != nil {
		problems = append(problems, "built-in theme: "+err.Error())
	} else {
		for _, variant := range family.Themes {
			if strings.TrimSpace(variant.Name) == "" {
				continue
			}
			entries = append(entries, entryFor(family.Name, variant))
		}
	}

	files, err := themeFiles(dir)
	if err != nil {
		problems = append(problems, "themes folder: "+err.Error())
		return entries, problems
	}
	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			problems = append(problems, filepath.Base(path)+": "+err.Error())
			continue
		}
		family, err := parseThemeFamily(data)
		if err != nil {
			problems = append(problems, filepath.Base(path)+": "+err.Error())
			continue
		}
		name := strings.TrimSpace(family.Name)
		if name == "" {
			name = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		}
		for _, variant := range family.Themes {
			if strings.TrimSpace(variant.Name) == "" {
				continue
			}
			entries = append(entries, entryFor(name, variant))
		}
	}
	return entries, problems
}

func entryFor(family string, variant themeVariant) Entry {
	variant.Appearance = normalizeAppearance(variant.Appearance)
	return Entry{Family: family, Variant: variant}
}

func normalizeAppearance(appearance string) string {
	if strings.EqualFold(strings.TrimSpace(appearance), "light") {
		return "light"
	}
	return "dark"
}

// CatalogFromEntries deduplicates by variant name, keeping the last one, and
// sorts the result for a stable picker.
func CatalogFromEntries(entries []Entry) []ThemeSummary {
	byName := make(map[string]ThemeSummary, len(entries))
	for _, entry := range entries {
		style := mergeStyle(builtinStyleFor(entry.Variant.Appearance), entry.Variant.Style)
		foreground := firstColor(style.EditorForeground, style.Text)
		byName[entry.Variant.Name] = ThemeSummary{
			Family:     entry.Family,
			Name:       entry.Variant.Name,
			Appearance: entry.Variant.Appearance,
			Preview: ThemePreview{
				Background: firstColor(style.EditorBackground, style.Background),
				Foreground: foreground,
				Accent:     firstColor(style.TextAccent, foreground),
				Keyword:    syntaxPreviewColor(style, "keyword", foreground),
				String:     syntaxPreviewColor(style, "string", foreground),
				Comment:    syntaxPreviewColor(style, "comment", foreground),
			},
		}
	}
	themes := make([]ThemeSummary, 0, len(byName))
	for _, theme := range byName {
		themes = append(themes, theme)
	}
	sort.Slice(themes, func(i, j int) bool {
		if themes[i].Family != themes[j].Family {
			return themes[i].Family < themes[j].Family
		}
		return themes[i].Name < themes[j].Name
	})
	return themes
}

// syntaxPreviewColor picks one resolved syntax color for the picker, falling
// back to the editor foreground when the capture is absent.
func syntaxPreviewColor(style themeStyle, capture, fallback string) string {
	if highlight, ok := style.Syntax[capture]; ok {
		return firstColor(highlight.Color, fallback)
	}
	return fallback
}

func FindVariant(entries []Entry, name string) (Entry, bool) {
	var found Entry
	ok := false
	for _, entry := range entries {
		if entry.Variant.Name == name {
			found = entry
			ok = true
		}
	}
	return found, ok
}

// builtinStyleFor returns the palette the app falls back to for an appearance.
// A user theme only needs to state the colors it changes; the rest keep the
// built-in contrast instead of turning into transparent or empty CSS values.
func builtinStyleFor(appearance string) themeStyle {
	family, err := Builtin()
	if err != nil {
		return themeStyle{}
	}
	for _, variant := range family.Themes {
		if normalizeAppearance(variant.Appearance) == appearance {
			return variant.Style
		}
	}
	return themeStyle{}
}

func firstColor(value, fallback string) string {
	value = strings.TrimSpace(value)
	if isHexColor(value) {
		return value
	}
	return strings.TrimSpace(fallback)
}

func isHexColor(value string) bool {
	switch len(value) {
	case 4, 5, 7, 9:
	default:
		return false
	}
	if value[0] != '#' {
		return false
	}
	for _, r := range value[1:] {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			return false
		}
	}
	return true
}

func mergeStyle(base, over themeStyle) themeStyle {
	merged := base
	merged.Background = firstColor(over.Background, base.Background)
	merged.SurfaceBackground = firstColor(over.SurfaceBackground, base.SurfaceBackground)
	merged.PanelBackground = firstColor(over.PanelBackground, base.PanelBackground)
	merged.ElementBackground = firstColor(over.ElementBackground, base.ElementBackground)
	merged.ElementHover = firstColor(over.ElementHover, base.ElementHover)
	merged.ElementSelected = firstColor(over.ElementSelected, base.ElementSelected)
	merged.Text = firstColor(over.Text, base.Text)
	merged.TextAccent = firstColor(over.TextAccent, base.TextAccent)
	merged.TextMuted = firstColor(over.TextMuted, base.TextMuted)
	merged.Border = firstColor(over.Border, base.Border)
	merged.Error = firstColor(over.Error, base.Error)
	merged.Created = firstColor(over.Created, base.Created)
	merged.Modified = firstColor(over.Modified, base.Modified)
	merged.Deleted = firstColor(over.Deleted, base.Deleted)
	merged.Conflict = firstColor(over.Conflict, base.Conflict)
	merged.SearchMatch = firstColor(over.SearchMatch, base.SearchMatch)
	merged.SearchActive = firstColor(over.SearchActive, base.SearchActive)
	merged.EditorBackground = firstColor(over.EditorBackground, base.EditorBackground)
	merged.EditorForeground = firstColor(over.EditorForeground, base.EditorForeground)
	merged.EditorGutter = firstColor(over.EditorGutter, base.EditorGutter)
	merged.EditorLineNumber = firstColor(over.EditorLineNumber, base.EditorLineNumber)
	merged.EditorActiveLineNum = firstColor(over.EditorActiveLineNum, base.EditorActiveLineNum)
	merged.EditorActiveLine = firstColor(over.EditorActiveLine, base.EditorActiveLine)
	merged.EditorIndentGuide = firstColor(over.EditorIndentGuide, base.EditorIndentGuide)
	merged.EditorIndentGuideAct = firstColor(over.EditorIndentGuideAct, base.EditorIndentGuideAct)
	merged.Syntax = mergeSyntax(base.Syntax, over.Syntax)
	return merged
}

func mergeSyntax(base, over map[string]themeHighlight) map[string]themeHighlight {
	merged := make(map[string]themeHighlight, len(base)+len(over))
	for capture, highlight := range base {
		merged[capture] = highlight
	}
	for capture, highlight := range over {
		previous := merged[capture]
		if strings.TrimSpace(highlight.Color) != "" {
			previous.Color = highlight.Color
		}
		if strings.TrimSpace(highlight.FontStyle) != "" {
			previous.FontStyle = highlight.FontStyle
		}
		if highlight.FontWeight != 0 {
			previous.FontWeight = highlight.FontWeight
		}
		merged[capture] = previous
	}
	return merged
}

func PaletteFromEntry(entry Entry) ThemePalette {
	style := mergeStyle(builtinStyleFor(entry.Variant.Appearance), entry.Variant.Style)
	ui := map[string]string{
		"bg":             style.Background,
		"editor-bg":      style.EditorBackground,
		"bg-raised":      style.SurfaceBackground,
		"bg-sidebar":     style.PanelBackground,
		"bg-hover":       style.ElementHover,
		"bg-selected":    style.ElementSelected,
		"fg":             style.Text,
		"fg-muted":       style.TextMuted,
		"fg-danger":      style.Error,
		"link":           style.TextAccent,
		"border":         style.Border,
		"git-added":      style.Created,
		"git-modified":   style.Modified,
		"git-deleted":    style.Deleted,
		"git-untracked":  style.Created,
		"git-conflicted": style.Conflict,
	}
	editor := map[string]string{
		"editor.background":                   style.EditorBackground,
		"editor.foreground":                   style.EditorForeground,
		"editorGutter.background":             style.EditorGutter,
		"editorLineNumber.foreground":         style.EditorLineNumber,
		"editorLineNumber.activeForeground":   style.EditorActiveLineNum,
		"editor.lineHighlightBackground":      style.EditorActiveLine,
		"editorIndentGuide.background1":       style.EditorIndentGuide,
		"editorIndentGuide.activeBackground1": style.EditorIndentGuideAct,
		"editor.findMatchBackground":          style.SearchMatch,
		"editor.findMatchHighlightBackground": style.SearchMatch,
	}
	for key, value := range editor {
		if strings.TrimSpace(value) == "" {
			delete(editor, key)
		}
	}
	return ThemePalette{
		Family:     entry.Family,
		Name:       entry.Variant.Name,
		Appearance: entry.Variant.Appearance,
		UI:         ui,
		Editor:     editor,
		Syntax:     style.Syntax,
	}
}
