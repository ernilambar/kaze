package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestLoadSettingsCreatesDefaultFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app", "settings.json")

	settings, err := loadSettings(path)
	if err != nil {
		t.Fatalf("loadSettings() error = %v", err)
	}

	want := defaultSettings()
	if !reflect.DeepEqual(settings, want) {
		t.Fatalf("loadSettings() = %+v, want %+v", settings, want)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("settings file was not created: %v", err)
	}

	var written map[string]map[string]any
	if err := json.Unmarshal(data, &written); err != nil {
		t.Fatalf("settings file is not valid JSON: %v", err)
	}
	editor := written["editor"]
	if editor["font_family"] != "Menlo" {
		t.Errorf("font_family = %v, want Menlo", editor["font_family"])
	}
	if editor["font_size"] != 13.0 {
		t.Errorf("font_size = %v, want 13", editor["font_size"])
	}
	if editor["font_weight"] != 400.0 {
		t.Errorf("font_weight = %v, want 400", editor["font_weight"])
	}
}

func TestLoadSettingsReadsUserValues(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	content := `{
		"editor": {
			"font_family": "JetBrains Mono",
			"font_size": 16.5,
			"font_weight": 300
		}
	}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	settings, err := loadSettings(path)
	if err != nil {
		t.Fatalf("loadSettings() error = %v", err)
	}

	want := defaultSettings()
	want.Editor.FontFamily = "JetBrains Mono"
	want.Editor.FontSize = 16.5
	want.Editor.FontWeight = 300
	if !reflect.DeepEqual(settings, want) {
		t.Fatalf("loadSettings() = %+v, want %+v", settings, want)
	}
}

func TestLoadSettingsFillsMissingValues(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte(`{"editor":{"font_size":18}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	settings, err := loadSettings(path)
	if err != nil {
		t.Fatalf("loadSettings() error = %v", err)
	}

	if settings.Editor.FontFamily != defaultFontFamily {
		t.Errorf("font_family = %q, want %q", settings.Editor.FontFamily, defaultFontFamily)
	}
	if settings.Editor.FontSize != 18 {
		t.Errorf("font_size = %v, want 18", settings.Editor.FontSize)
	}
	if settings.Editor.FontWeight != defaultFontWeight {
		t.Errorf("font_weight = %v, want %v", settings.Editor.FontWeight, defaultFontWeight)
	}
}

func TestLoadSettingsRejectsInvalidValues(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	content := `{"editor":{"font_family":"  ","font_size":-4,"font_weight":0}}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	settings, err := loadSettings(path)
	if err != nil {
		t.Fatalf("loadSettings() error = %v", err)
	}

	if !reflect.DeepEqual(settings, defaultSettings()) {
		t.Fatalf("loadSettings() = %+v, want defaults %+v", settings, defaultSettings())
	}
}

func TestLoadSettingsMalformedFileKeepsContent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	content := `{"editor":`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	settings, err := loadSettings(path)
	if err == nil {
		t.Fatal("loadSettings() error = nil, want parse error")
	}
	if !reflect.DeepEqual(settings, defaultSettings()) {
		t.Fatalf("loadSettings() = %+v, want defaults %+v", settings, defaultSettings())
	}

	data, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(data) != content {
		t.Fatalf("malformed file was modified: %q", data)
	}
}

func TestLoadSettingsReadsEmptyFileAsDefaultsWithoutWriting(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte(" \n"), 0o644); err != nil {
		t.Fatal(err)
	}

	settings, err := loadSettings(path)
	if err != nil {
		t.Fatalf("loadSettings() error = %v", err)
	}
	if settings.Editor.FontFamily != defaultSettings().Editor.FontFamily {
		t.Fatalf("font family = %q, want the default", settings.Editor.FontFamily)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != " \n" {
		t.Fatalf("empty settings file was rewritten: %q", data)
	}
}

func TestSettingsFilePathUsesConfigDirectory(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HOME", root)

	path, err := settingsFilePath()
	if err != nil {
		t.Fatalf("settingsFilePath() error = %v", err)
	}
	if !strings.HasSuffix(path, filepath.Join(appMeta.ProductName, "settings.json")) {
		t.Fatalf("settingsFilePath() = %q, want path under %s", path, appMeta.ProductName)
	}
}

func TestLoadSettingsDefaultsToAutoReveal(t *testing.T) {
	settings, err := loadSettings(filepath.Join(t.TempDir(), "settings.json"))
	if err != nil {
		t.Fatalf("loadSettings() error = %v", err)
	}
	if !settings.Sidebar.AutoReveal {
		t.Fatal("Sidebar.AutoReveal = false, want true by default")
	}
}

func TestLoadSettingsReadsAutoReveal(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	content := `{"sidebar":{"auto_reveal":false}}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	settings, err := loadSettings(path)
	if err != nil {
		t.Fatalf("loadSettings() error = %v", err)
	}
	if settings.Sidebar.AutoReveal {
		t.Fatal("Sidebar.AutoReveal = true, want the saved false")
	}
}

func TestNormalizeFolderListRepairsUserContent(t *testing.T) {
	got := normalizeFolderList([]string{" /one ", "", "/two", "/one", "   "})

	want := []string{"/one", "/two"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeFolderList() = %q, want %q", got, want)
	}
}

func TestNormalizeFolderListReturnsEmptySliceForNil(t *testing.T) {
	got := normalizeFolderList(nil)

	if got == nil {
		t.Fatal("normalizeFolderList(nil) = nil, want an empty slice")
	}
	if len(got) != 0 {
		t.Fatalf("normalizeFolderList(nil) = %q, want no folders", got)
	}
}

func TestLoadSettingsKeepsViewOptionDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte(`{"editor":{"font_size":18}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	settings, err := loadSettings(path)
	if err != nil {
		t.Fatalf("loadSettings() error = %v", err)
	}

	// Absent keys keep the defaults the editor was designed around, so an old
	// settings file does not silently turn off word wrap or line numbers.
	defaults := defaultSettings().Editor
	if !settings.Editor.WordWrap || !settings.Editor.LineNumbers ||
		!settings.Editor.IndentGuides || !settings.Editor.CodeFolding ||
		!settings.Editor.AutoCloseBrackets || !settings.Editor.SmartIndent {
		t.Errorf("missing view options did not keep their defaults: %+v", settings.Editor)
	}
	if settings.Editor.CursorBlink != defaults.CursorBlink {
		t.Errorf("cursor_blink = %v, want %v", settings.Editor.CursorBlink, defaults.CursorBlink)
	}
}

func TestLoadSettingsReadsEditorOptions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	content := `{"editor":{
		"tab_size":2,
		"hard_tabs":true,
		"ligatures":true,
		"cursor_width":3,
		"cursor_blink":false,
		"cursor_surrounding_lines":2,
		"scroll_beyond_last_line":true,
		"ensure_final_newline":true,
		"word_wrap":false,
		"invisible_characters":true,
		"line_numbers":false,
		"indent_guides":false,
		"code_folding":false,
		"auto_close_brackets":false,
		"smart_indent":false
	}}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	settings, err := loadSettings(path)
	if err != nil {
		t.Fatalf("loadSettings() error = %v", err)
	}

	editor := settings.Editor
	if editor.TabSize != 2 || !editor.HardTabs || !editor.Ligatures {
		t.Errorf("indentation settings = %+v", editor)
	}
	if editor.CursorWidth != 3 || editor.CursorBlink || editor.CursorSurroundingLines != 2 {
		t.Errorf("cursor settings = %+v", editor)
	}
	if !editor.ScrollBeyondLastLine || !editor.EnsureFinalNewline {
		t.Errorf("save/scroll settings = %+v", editor)
	}
	if editor.WordWrap || !editor.InvisibleCharacters || editor.LineNumbers ||
		editor.IndentGuides || editor.CodeFolding || editor.AutoCloseBrackets || editor.SmartIndent {
		t.Errorf("view settings = %+v", editor)
	}
}

func TestNormalizeRulers(t *testing.T) {
	settings := defaultSettings()
	settings.Editor.Rulers = []int{120, 0, 80, 120, -4, 100}
	settings.normalize()

	if !reflect.DeepEqual(settings.Editor.Rulers, []int{80, 100, 120}) {
		t.Errorf("rulers = %v, want [80 100 120]", settings.Editor.Rulers)
	}
}

func TestUpdateEditorSettingsPersistsAndRepairs(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	app := NewApp()

	editor := defaultSettings().Editor
	editor.TabSize = 0 // repaired to the default
	editor.CursorWidth = -2
	editor.CursorSurroundingLines = -1
	editor.HardTabs = true
	editor.WordWrap = false

	updated, err := app.UpdateEditorSettings(editor)
	if err != nil {
		t.Fatalf("UpdateEditorSettings() error = %v", err)
	}
	if updated.TabSize != defaultTabSize {
		t.Errorf("tab_size = %d, want %d", updated.TabSize, defaultTabSize)
	}
	if updated.CursorWidth != defaultCursorWidth {
		t.Errorf("cursor_width = %d, want %d", updated.CursorWidth, defaultCursorWidth)
	}
	if updated.CursorSurroundingLines != 0 {
		t.Errorf("cursor_surrounding_lines = %d, want 0", updated.CursorSurroundingLines)
	}
	if !updated.HardTabs || updated.WordWrap {
		t.Errorf("explicit values were not kept: %+v", updated)
	}

	settings, err := app.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings() error = %v", err)
	}
	if !reflect.DeepEqual(settings.Editor, updated) {
		t.Fatalf("settings.Editor = %+v, want %+v", settings.Editor, updated)
	}
}

func TestUpdateExcludeSettingsPersistAndRepair(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	app := NewApp()

	files, err := app.UpdateFilesSettings(FilesSettings{
		Exclude: []string{"  node_modules  ", "", "*.log", "*.log"},
	})
	if err != nil {
		t.Fatalf("UpdateFilesSettings() error = %v", err)
	}
	if !reflect.DeepEqual(files.Exclude, []string{"node_modules", "*.log"}) {
		t.Fatalf("files exclude = %q, want trimmed unique patterns", files.Exclude)
	}

	search, err := app.UpdateSearchSettings(SearchSettings{Exclude: []string{" vendor ", "vendor"}})
	if err != nil {
		t.Fatalf("UpdateSearchSettings() error = %v", err)
	}
	if !reflect.DeepEqual(search.Exclude, []string{"vendor"}) {
		t.Fatalf("search exclude = %q, want trimmed unique patterns", search.Exclude)
	}

	settings, err := app.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings() error = %v", err)
	}
	if !reflect.DeepEqual(settings.Files.Exclude, files.Exclude) {
		t.Fatalf("settings.Files.Exclude = %q, want %q", settings.Files.Exclude, files.Exclude)
	}
	if !reflect.DeepEqual(settings.Search.Exclude, search.Exclude) {
		t.Fatalf("settings.Search.Exclude = %q, want %q", settings.Search.Exclude, search.Exclude)
	}
}

func TestNormalizeThemeAndUI(t *testing.T) {
	settings := defaultSettings()
	settings.Theme = ThemeSettings{Mode: "LIGHT", Light: "  ", Dark: "  Custom Dark  "}
	settings.UI = UISettings{
		FontFamily:      "  Fira Sans  ",
		FontSize:        0,
		TabIcons:        false,
		IconStyle:       "neon",
		SidebarPosition: "RIGHT",
	}
	settings.normalize()

	if settings.Theme.Mode != "light" {
		t.Errorf("theme mode = %q, want light", settings.Theme.Mode)
	}
	if settings.Theme.Light != defaultThemeLight {
		t.Errorf("light theme = %q, want %q", settings.Theme.Light, defaultThemeLight)
	}
	if settings.Theme.Dark != "Custom Dark" {
		t.Errorf("dark theme = %q, want Custom Dark", settings.Theme.Dark)
	}
	if settings.UI.FontFamily != "Fira Sans" {
		t.Errorf("ui font = %q, want Fira Sans", settings.UI.FontFamily)
	}
	if settings.UI.FontSize != defaultUIFontSize {
		t.Errorf("ui font size = %v, want %v", settings.UI.FontSize, defaultUIFontSize)
	}
	if settings.UI.TabIcons {
		t.Error("tab icons = true, want the explicit false")
	}
	if settings.UI.IconStyle != defaultUIIconStyle {
		t.Errorf("icon style = %q, want %q", settings.UI.IconStyle, defaultUIIconStyle)
	}
	if settings.UI.SidebarPosition != "right" {
		t.Errorf("sidebar position = %q, want right", settings.UI.SidebarPosition)
	}
}

func TestNormalizeThemeModeKeepsSystem(t *testing.T) {
	if got := normalizeThemeMode(" System "); got != "system" {
		t.Errorf("theme mode = %q, want system", got)
	}
	if got := normalizeThemeMode("neon"); got != defaultThemeMode {
		t.Errorf("theme mode = %q, want %q", got, defaultThemeMode)
	}
}

func TestUpdateSettingsKeepsUnknownKeysAndFormatting(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	path, err := settingsFilePath()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	// Unknown keys and hand-picked spacing are what an edited file looks like;
	// a view toggle must neither drop the first nor reformat the second.
	content := "{\n" +
		"  \"editor\": {\n" +
		"    \"font_size\": 12,\n" +
		"    \"cursor_shape\": \"block\"\n" +
		"  },\n" +
		"  \"sidebar\":   { \"folders\": [\"/tmp\"], \"visible\": true, \"pinned\": [\"notes\"] },\n" +
		"  \"custom\":{\"keep\":true}\n" +
		"}\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	editor := defaultSettings().Editor
	editor.FontSize = 15
	app := NewApp()
	if _, err := app.UpdateEditorSettings(editor); err != nil {
		t.Fatalf("UpdateEditorSettings() error = %v", err)
	}

	// The file has no ui section; the write must add it without disturbing the
	// sections the user kept.
	if _, err := app.UpdateUISettings(UISettings{
		FontFamily:      "Inter",
		FontSize:        14,
		TabIcons:        true,
		IconStyle:       "monochrome",
		SidebarPosition: "right",
	}); err != nil {
		t.Fatalf("UpdateUISettings() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	var document map[string]json.RawMessage
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatalf("settings file is not valid JSON: %v\n%s", err, data)
	}
	var stored EditorSettings
	if err := json.Unmarshal(document["editor"], &stored); err != nil {
		t.Fatal(err)
	}
	if stored.FontSize != 15 {
		t.Errorf("editor font_size = %v, want 15", stored.FontSize)
	}
	var storedUI UISettings
	if err := json.Unmarshal(document["ui"], &storedUI); err != nil {
		t.Fatalf("ui section was not added: %v", err)
	}
	if storedUI.FontFamily != "Inter" || storedUI.FontSize != 14 {
		t.Errorf("ui section = %+v, want the new values", storedUI)
	}
	if !bytes.Contains(document["editor"], []byte(`"cursor_shape": "block"`)) {
		t.Errorf("editor = %s, want the unknown key kept with its formatting", document["editor"])
	}
	if _, ok := document["custom"]; !ok {
		t.Error("the custom section was dropped")
	}
	if !bytes.Contains(data, []byte(`"sidebar":   { "folders": ["/tmp"], "visible": true, "pinned": ["notes"] }`)) {
		t.Errorf("an untouched section was reformatted:\n%s", data)
	}
}

func TestUpdateThemeSettingsPersistsNormalizedValues(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	app := NewApp()

	got, err := app.UpdateThemeSettings(ThemeSettings{Mode: "dark", Light: "", Dark: "  "})
	if err != nil {
		t.Fatalf("UpdateThemeSettings() error = %v", err)
	}
	if got.Mode != "dark" || got.Light != defaultThemeLight || got.Dark != defaultThemeDark {
		t.Fatalf("UpdateThemeSettings() = %+v, want normalized defaults", got)
	}

	path, err := settingsFilePath()
	if err != nil {
		t.Fatal(err)
	}
	settings, err := loadSettings(path)
	if err != nil {
		t.Fatal(err)
	}
	if settings.Theme != got {
		t.Fatalf("stored theme = %+v, want %+v", settings.Theme, got)
	}
}

func TestUpdateUISettingsPersistsNormalizedValues(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	app := NewApp()

	got, err := app.UpdateUISettings(UISettings{
		FontFamily:      "  IBM Plex Sans ",
		FontSize:        15,
		TabIcons:        false,
		IconStyle:       "monochrome",
		SidebarPosition: "right",
	})
	if err != nil {
		t.Fatalf("UpdateUISettings() error = %v", err)
	}
	if got.FontFamily != "IBM Plex Sans" || got.FontSize != 15 || got.TabIcons {
		t.Fatalf("UpdateUISettings() = %+v, want the stored values", got)
	}
	if got.IconStyle != "monochrome" || got.SidebarPosition != "right" {
		t.Fatalf("UpdateUISettings() = %+v, want the selected layout", got)
	}
}

func TestUpdateSettingsFillsEmptySection(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	path, err := settingsFilePath()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	content := "{\n  \"editor\": {\n    \"font_size\": 12\n  },\n  \"sidebar\": {},\n  \"custom\": { \"keep\": true }\n}\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := NewApp().SetSidebarVisible(false); err != nil {
		t.Fatalf("SetSidebarVisible() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]json.RawMessage
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatalf("settings file is not valid JSON: %v\n%s", err, data)
	}
	var sidebar SidebarSettings
	if err := json.Unmarshal(document["sidebar"], &sidebar); err != nil {
		t.Fatal(err)
	}
	if sidebar.Visible {
		t.Errorf("sidebar = %+v, want a hidden tree", sidebar)
	}
	if _, ok := document["custom"]; !ok {
		t.Error("the custom section was dropped")
	}
}

func TestSettingsFileCreatesDefaults(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	got, err := NewApp().SettingsFile()
	if err != nil {
		t.Fatalf("SettingsFile() error = %v", err)
	}

	want := filepath.Join(home, "Library", "Application Support", appMeta.ProductName, "settings.json")
	if got != want {
		t.Fatalf("SettingsFile() = %q, want %q", got, want)
	}
	if _, err := os.Stat(want); err != nil {
		t.Fatalf("settings file was not created: %v", err)
	}
}
