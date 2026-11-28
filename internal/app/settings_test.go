package app

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"kaze/internal/config"
)

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

func TestUpdateEditorSettingsPersistsAndRepairs(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	app := NewApp()

	editor := config.Default().Editor
	editor.TabSize = 0 // repaired to the default
	editor.CursorWidth = -2
	editor.CursorSurroundingLines = -1
	editor.HardTabs = true
	editor.WordWrap = false

	updated, err := app.UpdateEditorSettings(editor)
	if err != nil {
		t.Fatalf("UpdateEditorSettings() error = %v", err)
	}
	if updated.TabSize != config.DefaultTabSize {
		t.Errorf("tab_size = %d, want %d", updated.TabSize, config.DefaultTabSize)
	}
	if updated.CursorWidth != config.DefaultCursorWidth {
		t.Errorf("cursor_width = %d, want %d", updated.CursorWidth, config.DefaultCursorWidth)
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

	files, err := app.UpdateFilesSettings(config.FilesSettings{
		Exclude: []string{"  node_modules  ", "", "*.log", "*.log"},
	})
	if err != nil {
		t.Fatalf("UpdateFilesSettings() error = %v", err)
	}
	if !reflect.DeepEqual(files.Exclude, []string{"node_modules", "*.log"}) {
		t.Fatalf("files exclude = %q, want trimmed unique patterns", files.Exclude)
	}

	search, err := app.UpdateSearchSettings(config.SearchSettings{Exclude: []string{" vendor ", "vendor"}})
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

	editor := config.Default().Editor
	editor.FontSize = 15
	app := NewApp()
	if _, err := app.UpdateEditorSettings(editor); err != nil {
		t.Fatalf("UpdateEditorSettings() error = %v", err)
	}

	// The file has no ui section; the write must add it without disturbing the
	// sections the user kept.
	if _, err := app.UpdateUISettings(config.UISettings{
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
	var stored config.EditorSettings
	if err := json.Unmarshal(document["editor"], &stored); err != nil {
		t.Fatal(err)
	}
	if stored.FontSize != 15 {
		t.Errorf("editor font_size = %v, want 15", stored.FontSize)
	}
	var storedUI config.UISettings
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

	got, err := app.UpdateThemeSettings(config.ThemeSettings{Mode: "dark", Light: "", Dark: "  "})
	if err != nil {
		t.Fatalf("UpdateThemeSettings() error = %v", err)
	}
	if got.Mode != "dark" || got.Light != config.DefaultThemeLight || got.Dark != config.DefaultThemeDark {
		t.Fatalf("UpdateThemeSettings() = %+v, want normalized defaults", got)
	}

	path, err := settingsFilePath()
	if err != nil {
		t.Fatal(err)
	}
	settings, err := config.Load(path)
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

	got, err := app.UpdateUISettings(config.UISettings{
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
	var sidebar config.SidebarSettings
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

func TestConfigFilesDoesNotCreateFiles(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	got, err := NewApp().ConfigFiles()
	if err != nil {
		t.Fatalf("ConfigFiles() error = %v", err)
	}

	dir := filepath.Join(home, "Library", "Application Support", appMeta.ProductName)
	want := []string{filepath.Join(dir, "settings.json"), filepath.Join(dir, "keybindings.json")}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("ConfigFiles() = %q, want %q", got, want)
	}
	for _, path := range want {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("%s exists after ConfigFiles(), err = %v", path, err)
		}
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
